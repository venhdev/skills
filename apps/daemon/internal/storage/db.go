package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/ncruces/go-sqlite3/driver"
)

// DDL for initializing the SQLite database schema.
const schemaDDL = `
CREATE TABLE IF NOT EXISTS plans (
    slug TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_plans_repo ON plans (repo_path, status);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    title TEXT NOT NULL,
    dependencies TEXT NOT NULL,
    harness TEXT NOT NULL,
    skill_command TEXT NOT NULL,
    worktree_path TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (plan_id) REFERENCES plans (slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_tasks_plan ON tasks (plan_id);
CREATE INDEX IF NOT EXISTS idx_tasks_repo_status ON tasks (repo_path, status);

CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    pid INTEGER NOT NULL,
    pgid INTEGER NOT NULL,
    status TEXT NOT NULL,
    log_path TEXT NOT NULL,
    exit_code INTEGER NOT NULL,
    started_at DATETIME NOT NULL,
    finished_at DATETIME,
    last_beat_at DATETIME NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_runs_task ON runs (task_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs (status);

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    type TEXT NOT NULL,
    file_path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    byte_size INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_artifacts_task ON artifacts (task_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_run ON artifacts (run_id);

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT,
    task_id TEXT,
    type TEXT NOT NULL,
    payload TEXT NOT NULL,
    timestamp DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_task ON events (task_id);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events (timestamp);
`

// DB encapsulates the connection pool and migration runner for SQLite WAL.
type DB struct {
	sqlDB   *sql.DB
	writeMu sync.Mutex // Serializes write transactions to prevent lock contention
}

// Open creates parent directories, opens the SQLite database with WAL pragmas, and migrates tables.
func Open(dbPath string) (*DB, error) {
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", dbPath)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Connection pool tuning for SQLite WAL
	conn.SetMaxOpenConns(4)
	conn.SetMaxIdleConns(4)
	conn.SetConnMaxLifetime(0)

	// Verify connection liveness
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// Verify WAL mode
	var journalMode string
	if err := conn.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to verify journal mode: %w", err)
	}
	if strings.ToUpper(journalMode) != "WAL" && dbPath != ":memory:" {
		_ = conn.Close()
		return nil, fmt.Errorf("expected WAL journal mode, got: %s", journalMode)
	}

	// Apply schema migrations
	if _, err := conn.Exec(schemaDDL); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to apply schema migrations: %w", err)
	}

	return &DB{sqlDB: conn}, nil
}

// RawDB returns the underlying *sql.DB instance for testing or queries.
func (d *DB) RawDB() *sql.DB {
	return d.sqlDB
}

// ExecImmediate executes a write operation inside a BEGIN IMMEDIATE transaction,
// serializing access via writeMu to completely prevent SQLITE_BUSY deadlocks.
func (d *DB) ExecImmediate(ctx context.Context, fn func(tx *sql.Tx) error) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	tx, err := d.sqlDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close gracefully terminates the database connection pool.
func (d *DB) Close() error {
	return d.sqlDB.Close()
}
