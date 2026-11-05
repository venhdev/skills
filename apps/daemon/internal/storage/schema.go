package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// column is one table column. It is the single source of truth for the CREATE
// TABLE statement, the SELECT projection, and the upsert statement, so adding a
// column cannot drift between those three places.
type column struct {
	name      string
	ddl       string // full column definition used in CREATE TABLE
	updatable bool   // included in ON CONFLICT DO UPDATE
}

// taskColumns is ordered. scanTask and taskArgs must follow this exact order.
var taskColumns = []column{
	{"id", "id TEXT PRIMARY KEY", false},
	{"plan_id", "plan_id TEXT NOT NULL", false},
	{"title", "title TEXT NOT NULL", true},
	{"description", "description TEXT NOT NULL DEFAULT ''", true},
	{"dependencies", "dependencies TEXT NOT NULL", true},
	{"priority", "priority TEXT NOT NULL DEFAULT 'P2'", true},
	{"complexity", "complexity TEXT NOT NULL DEFAULT 'MEDIUM'", true},
	{"risk_level", "risk_level TEXT NOT NULL DEFAULT 'LOW'", true},
	{"required_skills", "required_skills TEXT NOT NULL DEFAULT '[]'", true},
	{"required_tools", "required_tools TEXT NOT NULL DEFAULT '[]'", true},
	{"target_files", "target_files TEXT NOT NULL DEFAULT '[]'", true},
	{"acceptance_criteria", "acceptance_criteria TEXT NOT NULL DEFAULT '[]'", true},
	{"verification_commands", "verification_commands TEXT NOT NULL DEFAULT '[]'", true},
	{"harness", "harness TEXT NOT NULL", true},
	{"skill_command", "skill_command TEXT NOT NULL", true},
	{"worktree_path", "worktree_path TEXT NOT NULL", true},
	{"status", "status TEXT NOT NULL", true},
	{"claimed_by", "claimed_by TEXT NOT NULL DEFAULT ''", true},
	{"claimed_at", "claimed_at DATETIME", true},
	{"orchestrator_id", "orchestrator_id TEXT NOT NULL DEFAULT ''", true},
	{"worker_role", "worker_role TEXT NOT NULL DEFAULT ''", true},
	{"pid", "pid INTEGER NOT NULL DEFAULT 0", true},
	{"orchestrator_pid", "orchestrator_pid INTEGER NOT NULL DEFAULT 0", true},
	{"notes", "notes TEXT NOT NULL DEFAULT ''", true},
	{"review_reason", "review_reason TEXT NOT NULL DEFAULT ''", true},
	{"review_details", "review_details TEXT NOT NULL DEFAULT ''", true},
	{"review_options", "review_options TEXT NOT NULL DEFAULT '[]'", true},
	{"review_response", "review_response TEXT NOT NULL DEFAULT ''", true},
	{"resolved_by", "resolved_by TEXT NOT NULL DEFAULT ''", true},
	{"conflict_files", "conflict_files TEXT NOT NULL DEFAULT '[]'", true},
	{"created_at", "created_at DATETIME NOT NULL", false},
	{"updated_at", "updated_at DATETIME NOT NULL", true},
}

// artifactColumns are the columns added after the original artifacts table.
var artifactColumns = []column{
	{"role", "role TEXT NOT NULL DEFAULT 'OUTPUT'", false},
	{"name", "name TEXT NOT NULL DEFAULT ''", false},
}

const plansDDL = `
CREATE TABLE IF NOT EXISTS plans (
    slug TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);`

const taskIndexesDDL = `
CREATE INDEX IF NOT EXISTS idx_tasks_plan ON tasks (plan_id);`

const runsDDL = `
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
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs (status);`

const artifactsDDL = `
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
CREATE INDEX IF NOT EXISTS idx_artifacts_run ON artifacts (run_id);`

const eventsDDL = `
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT,
    task_id TEXT,
    type TEXT NOT NULL,
    payload TEXT NOT NULL,
    timestamp DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_task ON events (task_id);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events (timestamp);`

// dropLegacyDDL removes storage tables that are no longer part of the
// plans/tasks-only schema.
const dropLegacyDDL = `
DROP TABLE IF EXISTS tech_specs_fts;
DROP TABLE IF EXISTS tech_specs;
DROP TABLE IF EXISTS project_docs;
DROP TABLE IF EXISTS plan_revisions;
DROP TABLE IF EXISTS plan_snapshots;
DROP TABLE IF EXISTS agent_sessions;
`

var (
	taskSelectCols  = joinColumnNames(taskColumns)
	taskUpsertQuery = buildTaskUpsert(taskColumns)
	schemaDDL       = plansDDL + "\n" + buildCreateTable("tasks", taskColumns,
		"FOREIGN KEY (plan_id) REFERENCES plans (slug) ON DELETE CASCADE") +
		taskIndexesDDL + runsDDL + artifactsDDL + eventsDDL
)

func joinColumnNames(cols []column) string {
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.name
	}
	return strings.Join(names, ", ")
}

func buildCreateTable(name string, cols []column, footer string) string {
	defs := make([]string, 0, len(cols)+1)
	for _, c := range cols {
		defs = append(defs, "    "+c.ddl)
	}
	if footer != "" {
		defs = append(defs, "    "+footer)
	}
	return "CREATE TABLE IF NOT EXISTS " + name + " (\n" + strings.Join(defs, ",\n") + "\n);"
}

func buildTaskUpsert(cols []column) string {
	names := make([]string, len(cols))
	marks := make([]string, len(cols))
	updates := make([]string, 0, len(cols))
	for i, c := range cols {
		names[i] = c.name
		marks[i] = "?"
		if c.updatable {
			updates = append(updates, c.name+" = excluded."+c.name)
		}
	}
	return "INSERT INTO tasks (" + strings.Join(names, ", ") + ") VALUES (" +
		strings.Join(marks, ", ") + ") ON CONFLICT(id) DO UPDATE SET " +
		strings.Join(updates, ", ")
}

// applyIncrementalMigrations adds columns that a legacy database is missing.
// It inspects the live schema instead of swallowing duplicate-column errors, so
// a genuine migration failure surfaces to the caller.
func applyIncrementalMigrations(ctx context.Context, conn *sql.DB) error {
	if err := addMissingColumns(ctx, conn, "tasks", taskColumns); err != nil {
		return err
	}
	if err := addMissingColumns(ctx, conn, "artifacts", artifactColumns); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_artifacts_role ON artifacts (role);"); err != nil {
		return fmt.Errorf("failed to create artifacts role index: %w", err)
	}
	return nil
}

func addMissingColumns(ctx context.Context, conn *sql.DB, table string, cols []column) error {
	existing, err := tableColumns(ctx, conn, table)
	if err != nil {
		return err
	}
	for _, c := range cols {
		if existing[c.name] {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, c.ddl)
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to add column %s.%s: %w", table, c.name, err)
		}
	}
	return nil
}

func tableColumns(ctx context.Context, conn *sql.DB, table string) (map[string]bool, error) {
	rows, err := conn.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return nil, fmt.Errorf("failed to inspect table %s: %w", table, err)
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultV   sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &primaryKey); err != nil {
			return nil, fmt.Errorf("failed to scan table info for %s: %w", table, err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read table info for %s: %w", table, err)
	}
	return cols, nil
}
