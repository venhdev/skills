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
	name string
	ddl  string // full column definition used in CREATE TABLE
	// patchable marks authoring metadata. Plan re-ingestion may update these
	// columns on an existing row; lifecycle columns (status, claim, notes,
	// review, created_at) are owned by the task state machine and are never
	// overwritten by a re-ingest.
	patchable bool
}

// taskColumns is ordered. scanTask and taskArgs must follow this exact order.
var taskColumns = []column{
	{name: "id", ddl: "id TEXT PRIMARY KEY"},
	{name: "plan_id", ddl: "plan_id TEXT NOT NULL"},
	{name: "title", ddl: "title TEXT NOT NULL", patchable: true},
	{name: "description", ddl: "description TEXT NOT NULL DEFAULT ''", patchable: true},
	{name: "dependencies", ddl: "dependencies TEXT NOT NULL", patchable: true},
	{name: "priority", ddl: "priority TEXT NOT NULL DEFAULT 'P2'", patchable: true},
	{name: "complexity", ddl: "complexity TEXT NOT NULL DEFAULT 'MEDIUM'", patchable: true},
	{name: "risk_level", ddl: "risk_level TEXT NOT NULL DEFAULT 'LOW'", patchable: true},
	{name: "required_skills", ddl: "required_skills TEXT NOT NULL DEFAULT '[]'", patchable: true},
	{name: "required_tools", ddl: "required_tools TEXT NOT NULL DEFAULT '[]'", patchable: true},
	{name: "target_files", ddl: "target_files TEXT NOT NULL DEFAULT '[]'", patchable: true},
	{name: "acceptance_criteria", ddl: "acceptance_criteria TEXT NOT NULL DEFAULT '[]'", patchable: true},
	{name: "verification_commands", ddl: "verification_commands TEXT NOT NULL DEFAULT '[]'", patchable: true},
	{name: "harness", ddl: "harness TEXT NOT NULL", patchable: true},
	{name: "worktree_path", ddl: "worktree_path TEXT NOT NULL", patchable: true},
	{name: "status", ddl: "status TEXT NOT NULL"},
	{name: "claimed_by", ddl: "claimed_by TEXT NOT NULL DEFAULT ''"},
	{name: "claimed_at", ddl: "claimed_at DATETIME"},
	{name: "orchestrator_id", ddl: "orchestrator_id TEXT NOT NULL DEFAULT ''"},
	{name: "worker_role", ddl: "worker_role TEXT NOT NULL DEFAULT ''"},
	{name: "notes", ddl: "notes TEXT NOT NULL DEFAULT ''"},
	{name: "review_reason", ddl: "review_reason TEXT NOT NULL DEFAULT ''"},
	{name: "review_details", ddl: "review_details TEXT NOT NULL DEFAULT ''"},
	{name: "review_options", ddl: "review_options TEXT NOT NULL DEFAULT '[]'"},
	{name: "review_response", ddl: "review_response TEXT NOT NULL DEFAULT ''"},
	{name: "resolved_by", ddl: "resolved_by TEXT NOT NULL DEFAULT ''"},
	{name: "conflict_files", ddl: "conflict_files TEXT NOT NULL DEFAULT '[]'"},
	{name: "created_at", ddl: "created_at DATETIME NOT NULL"},
	{name: "updated_at", ddl: "updated_at DATETIME NOT NULL", patchable: true},
}

// artifactColumns are the columns added after the original artifacts table.
var artifactColumns = []column{
	{name: "role", ddl: "role TEXT NOT NULL DEFAULT 'OUTPUT'"},
	{name: "name", ddl: "name TEXT NOT NULL DEFAULT ''"},
}

// eventColumns are the columns added after the original events table.
var eventColumns = []column{
	{name: "plan_id", ddl: "plan_id TEXT"},
}

// planColumns are the columns added after the original plans table.
var planColumns = []column{
	{name: "archived_at", ddl: "archived_at DATETIME"},
}

const plansDDL = `
CREATE TABLE IF NOT EXISTS plans (
    slug TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    archived_at DATETIME,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);`

const taskIndexesDDL = `
CREATE INDEX IF NOT EXISTS idx_tasks_plan ON tasks (plan_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);
CREATE INDEX IF NOT EXISTS idx_tasks_updated ON tasks (updated_at);
CREATE INDEX IF NOT EXISTS idx_tasks_plan_status ON tasks (plan_id, status);`

// revisionDDL maintains an O(1) change counter for the whole plans/tasks state.
// The gateway watcher polls it to notice writes from other processes; counting
// rows and MAX(updated_at) required a full table scan every interval.
const revisionDDL = `
CREATE TABLE IF NOT EXISTS state_revision (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    revision INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL
);
INSERT OR IGNORE INTO state_revision (id, revision, updated_at) VALUES (1, 0, CURRENT_TIMESTAMP);
CREATE TRIGGER IF NOT EXISTS trg_tasks_revision_insert AFTER INSERT ON tasks BEGIN
    UPDATE state_revision SET revision = revision + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
END;
CREATE TRIGGER IF NOT EXISTS trg_tasks_revision_update AFTER UPDATE ON tasks BEGIN
    UPDATE state_revision SET revision = revision + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
END;
CREATE TRIGGER IF NOT EXISTS trg_tasks_revision_delete AFTER DELETE ON tasks BEGIN
    UPDATE state_revision SET revision = revision + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
END;
CREATE TRIGGER IF NOT EXISTS trg_plans_revision_insert AFTER INSERT ON plans BEGIN
    UPDATE state_revision SET revision = revision + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
END;
CREATE TRIGGER IF NOT EXISTS trg_plans_revision_update AFTER UPDATE ON plans BEGIN
    UPDATE state_revision SET revision = revision + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
END;
CREATE TRIGGER IF NOT EXISTS trg_plans_revision_delete AFTER DELETE ON plans BEGIN
    UPDATE state_revision SET revision = revision + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
END;`

const artifactsDDL = `
CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
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
CREATE INDEX IF NOT EXISTS idx_artifacts_task ON artifacts (task_id);`

const eventsDDL = `
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id TEXT,
    task_id TEXT,
    type TEXT NOT NULL,
    payload TEXT NOT NULL,
    timestamp DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_task ON events (task_id);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events (timestamp);`

// auditTriggersDDL records every mutation in the activity log, including writes
// made by the stdio MCP server or the CLI in another process. The daemon never
// writes these rows itself, so the log cannot miss an external writer.
const auditTriggersDDL = `
CREATE TRIGGER IF NOT EXISTS trg_audit_task_insert AFTER INSERT ON tasks BEGIN
    INSERT INTO events (task_id, plan_id, type, payload, timestamp)
    VALUES (NEW.id, NEW.plan_id, 'task.created',
        json_object('task_id', NEW.id, 'plan_id', NEW.plan_id, 'title', NEW.title, 'status', NEW.status),
        CURRENT_TIMESTAMP);
END;
CREATE TRIGGER IF NOT EXISTS trg_audit_task_update AFTER UPDATE ON tasks BEGIN
    INSERT INTO events (task_id, plan_id, type, payload, timestamp)
    VALUES (NEW.id, NEW.plan_id, 'task.updated',
        json_object(
            'task_id', NEW.id, 'plan_id', NEW.plan_id,
            'kind', CASE
                WHEN OLD.status <> NEW.status THEN 'status'
                WHEN OLD.notes <> NEW.notes THEN 'notes'
                WHEN OLD.dependencies <> NEW.dependencies THEN 'dependencies'
                WHEN OLD.claimed_by <> NEW.claimed_by THEN 'claim'
                ELSE 'metadata' END,
            'from', OLD.status, 'to', NEW.status,
            'claimed_by', NEW.claimed_by,
            'worktree_path', NEW.worktree_path),
        CURRENT_TIMESTAMP);
END;
CREATE TRIGGER IF NOT EXISTS trg_audit_task_delete AFTER DELETE ON tasks BEGIN
    INSERT INTO events (task_id, plan_id, type, payload, timestamp)
    VALUES (OLD.id, OLD.plan_id, 'task.deleted',
        json_object('task_id', OLD.id, 'plan_id', OLD.plan_id, 'title', OLD.title),
        CURRENT_TIMESTAMP);
END;
CREATE TRIGGER IF NOT EXISTS trg_audit_plan_insert AFTER INSERT ON plans BEGIN
    INSERT INTO events (plan_id, type, payload, timestamp)
    VALUES (NEW.slug, 'plan.created',
        json_object('slug', NEW.slug, 'title', NEW.title, 'status', NEW.status),
        CURRENT_TIMESTAMP);
END;
CREATE TRIGGER IF NOT EXISTS trg_audit_plan_update AFTER UPDATE ON plans BEGIN
    INSERT INTO events (plan_id, type, payload, timestamp)
    VALUES (NEW.slug, 'plan.updated',
        json_object('slug', NEW.slug,
            'kind', CASE
                WHEN OLD.status <> NEW.status THEN 'status'
                WHEN OLD.archived_at IS NULL AND NEW.archived_at IS NOT NULL THEN 'archived'
                WHEN OLD.archived_at IS NOT NULL AND NEW.archived_at IS NULL THEN 'unarchived'
                WHEN OLD.title <> NEW.title THEN 'title'
                ELSE 'metadata' END,
            'from', OLD.status, 'to', NEW.status,
            'archived', NEW.archived_at IS NOT NULL),
        CURRENT_TIMESTAMP);
END;
CREATE TRIGGER IF NOT EXISTS trg_audit_plan_delete AFTER DELETE ON plans BEGIN
    INSERT INTO events (plan_id, type, payload, timestamp)
    VALUES (OLD.slug, 'plan.deleted',
        json_object('slug', OLD.slug, 'title', OLD.title),
        CURRENT_TIMESTAMP);
END;
CREATE TRIGGER IF NOT EXISTS trg_audit_artifact_insert AFTER INSERT ON artifacts BEGIN
    INSERT INTO events (task_id, plan_id, type, payload, timestamp)
    VALUES (NEW.task_id, (SELECT plan_id FROM tasks WHERE id = NEW.task_id), 'artifact.added',
        json_object('artifact_id', NEW.id, 'task_id', NEW.task_id, 'role', NEW.role, 'name', NEW.name, 'type', NEW.type, 'mime_type', NEW.mime_type),
        CURRENT_TIMESTAMP);
END;`

// dropLegacyDDL removes storage tables that are no longer part of the
// plans/tasks/artifacts/events schema.
const dropLegacyDDL = `
DROP TABLE IF EXISTS runs;
DROP TABLE IF EXISTS tech_specs_fts;
DROP TABLE IF EXISTS tech_specs;
DROP TABLE IF EXISTS project_docs;
DROP TABLE IF EXISTS plan_revisions;
DROP TABLE IF EXISTS plan_snapshots;
DROP TABLE IF EXISTS agent_sessions;
`

var (
	taskSelectCols = joinColumnNames(taskColumns)
	// taskInsertQuery creates a task and refuses to touch an existing row.
	taskInsertQuery = buildTaskInsert(taskColumns)
	// taskPatchQuery creates a missing task and patches authoring metadata of an
	// existing one without disturbing its lifecycle state.
	taskPatchQuery = buildTaskPatch(taskColumns)
	schemaDDL      = plansDDL + "\n" + buildCreateTable("tasks", taskColumns,
		"FOREIGN KEY (plan_id) REFERENCES plans (slug) ON DELETE CASCADE") +
		artifactsDDL + eventsDDL + revisionDDL
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

func buildTaskInsert(cols []column) string {
	names := make([]string, len(cols))
	marks := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.name
		marks[i] = "?"
	}
	return "INSERT INTO tasks (" + strings.Join(names, ", ") + ") VALUES (" +
		strings.Join(marks, ", ") + ") ON CONFLICT(id) DO NOTHING"
}

func buildTaskPatch(cols []column) string {
	names := make([]string, len(cols))
	marks := make([]string, len(cols))
	updates := make([]string, 0, len(cols))
	for i, c := range cols {
		names[i] = c.name
		marks[i] = "?"
		if c.patchable {
			updates = append(updates, c.name+" = excluded."+c.name)
		}
	}
	return "INSERT INTO tasks (" + strings.Join(names, ", ") + ") VALUES (" +
		strings.Join(marks, ", ") + ") ON CONFLICT(id) DO UPDATE SET " +
		strings.Join(updates, ", ")
}

// applyIncrementalMigrations adds columns that a legacy database is missing.
// It inspects the live schema instead of swallowing duplicate-column errors, so
// a genuine migration failure surfaces to the caller. All statements run inside
// one transaction so a partial migration can never be left behind.
func applyIncrementalMigrations(ctx context.Context, conn *sql.DB) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback()

	added, err := addMissingColumns(ctx, tx, "tasks", taskColumns)
	if err != nil {
		return err
	}
	if _, err := addMissingColumns(ctx, tx, "artifacts", artifactColumns); err != nil {
		return err
	}
	if _, err := addMissingColumns(ctx, tx, "events", eventColumns); err != nil {
		return err
	}
	if _, err := addMissingColumns(ctx, tx, "plans", planColumns); err != nil {
		return err
	}

	// Runtime-era columns with no producer or consumer.
	for _, column := range []string{"skill_command", "pid", "orchestrator_pid"} {
		if err := dropColumnIfExists(ctx, tx, "tasks", column); err != nil {
			return err
		}
	}
	// The run index would block DROP COLUMN on older databases.
	if _, err := tx.ExecContext(ctx, "DROP INDEX IF EXISTS idx_artifacts_run"); err != nil {
		return fmt.Errorf("failed to drop artifacts run index: %w", err)
	}
	for _, column := range []string{"run_id"} {
		if err := dropColumnIfExists(ctx, tx, "artifacts", column); err != nil {
			return err
		}
		if err := dropColumnIfExists(ctx, tx, "events", column); err != nil {
			return err
		}
	}
	for _, name := range added {
		// created_at was added with an epoch placeholder because SQLite only
		// accepts constant defaults in ADD COLUMN; stamp the real migration time.
		if name == "created_at" {
			if _, err := tx.ExecContext(ctx,
				"UPDATE tasks SET created_at = CURRENT_TIMESTAMP WHERE created_at = '1970-01-01 00:00:00'"); err != nil {
				return fmt.Errorf("failed to backfill created_at: %w", err)
			}
		}
	}

	// Task indexes reference columns that a legacy table may be missing, so they
	// are created only after addMissingColumns has run.
	if _, err := tx.ExecContext(ctx, taskIndexesDDL); err != nil {
		return fmt.Errorf("failed to create task indexes: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_artifacts_role ON artifacts (role);"); err != nil {
		return fmt.Errorf("failed to create artifacts role index: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_events_plan ON events (plan_id);"); err != nil {
		return fmt.Errorf("failed to create events plan index: %w", err)
	}

	// Audit triggers reference columns that legacy tables may be missing, so
	// they are installed only after addMissingColumns has run.
	if _, err := tx.ExecContext(ctx, auditTriggersDDL); err != nil {
		return fmt.Errorf("failed to create audit triggers: %w", err)
	}

	// SUSPENDED was a runtime-only state with no producer once the daemon
	// stopped supervising processes. Park legacy rows in BLOCKED so a human or
	// the orchestrator explicitly decides the next step.
	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET status = 'BLOCKED',
		    notes = CASE WHEN notes = '' THEN 'Migrated from SUSPENDED' ELSE notes || char(10) || 'Migrated from SUSPENDED' END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE status = 'SUSPENDED'`); err != nil {
		return fmt.Errorf("failed to migrate suspended tasks: %w", err)
	}

	// The previous daemon persisted a colon-named event vocabulary that no
	// longer exists; keep only trigger-produced rows.
	if _, err := tx.ExecContext(ctx, "DELETE FROM events WHERE type LIKE '%:%'"); err != nil {
		return fmt.Errorf("failed to prune legacy events: %w", err)
	}

	// Bound the audit table. Nothing needs more than a rolling month of
	// activity, so old rows are pruned instead of growing the file forever.
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM events WHERE julianday(timestamp) < julianday('now', '-30 days')"); err != nil {
		return fmt.Errorf("failed to prune old events: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}
	return nil
}

func addMissingColumns(ctx context.Context, tx *sql.Tx, table string, cols []column) ([]string, error) {
	existing, err := tableColumns(ctx, tx, table)
	if err != nil {
		return nil, err
	}
	var added []string
	for _, c := range cols {
		if existing[c.name] {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, migrationDDL(c))
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return nil, fmt.Errorf("failed to add column %s.%s: %w", table, c.name, err)
		}
		added = append(added, c.name)
	}
	return added, nil
}

// migrationDefaults are semantic backfills for NOT NULL columns that have no
// DEFAULT in the table definition. A blank string would create rows that the
// domain considers invalid (for example an unknown status).
var migrationDefaults = map[string]string{
	"status":       "'PENDING'",
	"dependencies": "'[]'",
	"harness":      "'orchestrator'",
	"created_at":   "'1970-01-01 00:00:00'",
}

// migrationDDL makes a column definition legal for ALTER TABLE ADD COLUMN on a
// populated table. SQLite rejects NOT NULL without a constant default there, so
// a backfill default is synthesized from the column semantics.
func migrationDDL(c column) string {
	ddl := c.ddl
	if !strings.Contains(ddl, "NOT NULL") || strings.Contains(ddl, "DEFAULT") {
		return ddl
	}
	if def, ok := migrationDefaults[c.name]; ok {
		return ddl + " DEFAULT " + def
	}
	switch {
	case strings.Contains(ddl, "INTEGER"):
		return ddl + " DEFAULT 0"
	case strings.Contains(ddl, "DATETIME"):
		return ddl + " DEFAULT '1970-01-01 00:00:00'"
	default:
		return ddl + " DEFAULT ''"
	}
}

// queryer is satisfied by both *sql.DB and *sql.Tx so schema inspection can run
// inside or outside the migration transaction.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// dropColumnIfExists removes a column from a table when the live schema still
// carries it. SQLite 3.35+ supports DROP COLUMN and fails safely if the column
// is indexed or referenced by a trigger.
func dropColumnIfExists(ctx context.Context, tx *sql.Tx, table, column string) error {
	existing, err := tableColumns(ctx, tx, table)
	if err != nil {
		return err
	}
	if !existing[column] {
		return nil
	}
	stmt := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, column)
	if _, err := tx.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop column %s.%s: %w", table, column, err)
	}
	return nil
}

func tableColumns(ctx context.Context, q queryer, table string) (map[string]bool, error) {
	rows, err := q.QueryContext(ctx, "PRAGMA table_info("+table+")")
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
