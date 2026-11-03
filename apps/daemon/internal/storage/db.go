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
CREATE INDEX IF NOT EXISTS idx_tasks_repo_status ON tasks (repo_path, status);

CREATE TABLE IF NOT EXISTS agent_sessions (
    session_id TEXT PRIMARY KEY,
    parent_session_id TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'SUBAGENT',
    name TEXT NOT NULL DEFAULT '',
    pid INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    claimed_tasks TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL,
    last_beat_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_agent_sessions_status ON agent_sessions (status);

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

CREATE TABLE IF NOT EXISTS project_docs (
    slug TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    git_branch TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    category TEXT NOT NULL,
    content_markdown TEXT NOT NULL,
    mermaid_diagrams TEXT NOT NULL DEFAULT '[]',
    version INTEGER NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (repo_path, slug)
);
CREATE INDEX IF NOT EXISTS idx_project_docs_repo ON project_docs (repo_path, category);

CREATE TABLE IF NOT EXISTS tech_specs (
    slug TEXT PRIMARY KEY,
    repo_path TEXT NOT NULL,
    tier INTEGER NOT NULL,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    diagram_type TEXT NOT NULL,
    diagram_code TEXT NOT NULL DEFAULT '',
    structured_payload TEXT NOT NULL DEFAULT '{}',
    notes TEXT NOT NULL DEFAULT '',
    bound_paths TEXT NOT NULL DEFAULT '[]',
    version INTEGER NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tech_specs_repo_tier ON tech_specs (repo_path, tier, kind);

CREATE VIRTUAL TABLE IF NOT EXISTS tech_specs_fts USING fts5(
    slug,
    title,
    diagram_code,
    notes,
    content='tech_specs',
    content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS trg_tech_specs_ai AFTER INSERT ON tech_specs BEGIN
  INSERT INTO tech_specs_fts(rowid, slug, title, diagram_code, notes)
  VALUES (new.rowid, new.slug, new.title, new.diagram_code, new.notes);
END;

CREATE TRIGGER IF NOT EXISTS trg_tech_specs_ad AFTER DELETE ON tech_specs BEGIN
  INSERT INTO tech_specs_fts(tech_specs_fts, rowid, slug, title, diagram_code, notes)
  VALUES ('delete', old.rowid, old.slug, old.title, old.diagram_code, old.notes);
END;

CREATE TRIGGER IF NOT EXISTS trg_tech_specs_au AFTER UPDATE ON tech_specs BEGIN
  INSERT INTO tech_specs_fts(tech_specs_fts, rowid, slug, title, diagram_code, notes)
  VALUES ('delete', old.rowid, old.slug, old.title, old.diagram_code, old.notes);
  INSERT INTO tech_specs_fts(rowid, slug, title, diagram_code, notes)
  VALUES (new.rowid, new.slug, new.title, new.diagram_code, new.notes);
END;

CREATE TABLE IF NOT EXISTS plan_revisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_slug TEXT NOT NULL,
    revision TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    FOREIGN KEY (plan_slug) REFERENCES plans (slug) ON DELETE CASCADE,
    UNIQUE (plan_slug, revision)
);
CREATE INDEX IF NOT EXISTS idx_plan_revisions_plan ON plan_revisions (plan_slug);

CREATE TABLE IF NOT EXISTS plan_snapshots (
    plan_slug TEXT NOT NULL,
    revision TEXT NOT NULL,
    file_path TEXT NOT NULL,
    content TEXT NOT NULL,
    byte_size INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (plan_slug, revision, file_path),
    FOREIGN KEY (plan_slug) REFERENCES plans (slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_plan_snapshots_slug_rev ON plan_snapshots (plan_slug, revision);
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
	conn, err := driver.Open(dsn, fts5.Register)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database with fts5: %w", err)
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
