package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fts5"
)

// DDL for initializing the SQLite database schema.
const schemaDDL = `
CREATE TABLE IF NOT EXISTS plans (
    slug TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    dependencies TEXT NOT NULL,
    priority TEXT NOT NULL DEFAULT 'P2',
    complexity TEXT NOT NULL DEFAULT 'MEDIUM',
    risk_level TEXT NOT NULL DEFAULT 'LOW',
    required_skills TEXT NOT NULL DEFAULT '[]',
    required_tools TEXT NOT NULL DEFAULT '[]',
    target_files TEXT NOT NULL DEFAULT '[]',
    acceptance_criteria TEXT NOT NULL DEFAULT '[]',
    verification_commands TEXT NOT NULL DEFAULT '[]',
    harness TEXT NOT NULL,
    skill_command TEXT NOT NULL,
    worktree_path TEXT NOT NULL,
    status TEXT NOT NULL,
    claimed_by TEXT NOT NULL DEFAULT '',
    claimed_at DATETIME,
    orchestrator_id TEXT NOT NULL DEFAULT '',
    worker_role TEXT NOT NULL DEFAULT '',
    pid INTEGER NOT NULL DEFAULT 0,
    orchestrator_pid INTEGER NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    review_reason TEXT NOT NULL DEFAULT '',
    review_details TEXT NOT NULL DEFAULT '',
    review_options TEXT NOT NULL DEFAULT '[]',
    review_response TEXT NOT NULL DEFAULT '',
    resolved_by TEXT NOT NULL DEFAULT '',
    conflict_files TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (plan_id) REFERENCES plans (slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_tasks_plan ON tasks (plan_id);

CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
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
    run_id TEXT,
    task_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'OUTPUT',
    name TEXT NOT NULL DEFAULT '',
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

// dropLegacyDDL removes storage tables that are no longer part of the plans/tasks-only schema.
const dropLegacyDDL = `
DROP TABLE IF EXISTS tech_specs_fts;
DROP TABLE IF EXISTS tech_specs;
DROP TABLE IF EXISTS project_docs;
DROP TABLE IF EXISTS plan_revisions;
DROP TABLE IF EXISTS plan_snapshots;
DROP TABLE IF EXISTS agent_sessions;
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
	// fts5 is registered so legacy FTS tables can be dropped during migration.
	conn, err := driver.Open(dsn, fts5.Register)
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

	// Drop legacy docs/specs/snapshot/session tables (idempotent).
	if _, err := conn.Exec(dropLegacyDDL); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to drop legacy tables: %w", err)
	}

	// Apply incremental column migrations for existing databases
	_, _ = conn.Exec("ALTER TABLE artifacts ADD COLUMN role TEXT NOT NULL DEFAULT 'OUTPUT';")
	_, _ = conn.Exec("ALTER TABLE artifacts ADD COLUMN name TEXT NOT NULL DEFAULT '';")
	_, _ = conn.Exec("CREATE INDEX IF NOT EXISTS idx_artifacts_role ON artifacts (role);")

	// Apply incremental column migrations for tasks table
	taskCols := []string{
		"ALTER TABLE tasks ADD COLUMN description TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE tasks ADD COLUMN priority TEXT NOT NULL DEFAULT 'P2';",
		"ALTER TABLE tasks ADD COLUMN complexity TEXT NOT NULL DEFAULT 'MEDIUM';",
		"ALTER TABLE tasks ADD COLUMN risk_level TEXT NOT NULL DEFAULT 'LOW';",
		"ALTER TABLE tasks ADD COLUMN required_skills TEXT NOT NULL DEFAULT '[]';",
		"ALTER TABLE tasks ADD COLUMN required_tools TEXT NOT NULL DEFAULT '[]';",
		"ALTER TABLE tasks ADD COLUMN target_files TEXT NOT NULL DEFAULT '[]';",
		"ALTER TABLE tasks ADD COLUMN acceptance_criteria TEXT NOT NULL DEFAULT '[]';",
		"ALTER TABLE tasks ADD COLUMN verification_commands TEXT NOT NULL DEFAULT '[]';",
		"ALTER TABLE tasks ADD COLUMN claimed_by TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE tasks ADD COLUMN claimed_at DATETIME;",
		"ALTER TABLE tasks ADD COLUMN review_reason TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE tasks ADD COLUMN review_details TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE tasks ADD COLUMN review_options TEXT NOT NULL DEFAULT '[]';",
		"ALTER TABLE tasks ADD COLUMN review_response TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE tasks ADD COLUMN conflict_files TEXT NOT NULL DEFAULT '[]';",
		"ALTER TABLE tasks ADD COLUMN orchestrator_id TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE tasks ADD COLUMN worker_role TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE tasks ADD COLUMN pid INTEGER NOT NULL DEFAULT 0;",
		"ALTER TABLE tasks ADD COLUMN orchestrator_pid INTEGER NOT NULL DEFAULT 0;",
		"ALTER TABLE tasks ADD COLUMN notes TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE tasks ADD COLUMN resolved_by TEXT NOT NULL DEFAULT '';",
	}
	for _, colSQL := range taskCols {
		_, _ = conn.Exec(colSQL)
	}

	return &DB{sqlDB: conn}, nil
}

// RawDB returns the underlying *sql.DB instance for testing or queries.
func (d *DB) RawDB() *sql.DB {
	return d.sqlDB
}

// Query executes a read-only query that returns rows without acquiring writeMu.
func (d *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.sqlDB.QueryContext(ctx, query, args...)
}

// QueryRow executes a read-only query that returns at most one row without acquiring writeMu.
func (d *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.sqlDB.QueryRowContext(ctx, query, args...)
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
