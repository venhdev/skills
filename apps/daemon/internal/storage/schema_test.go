package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fts5"

	"zenfd/internal/domain"
)

func TestSchema_TaskColumnsMatchLiveTable(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "schema.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	cols, err := tableColumns(context.Background(), db.sqlDB, "tasks")
	if err != nil {
		t.Fatalf("tableColumns failed: %v", err)
	}
	if len(cols) != len(taskColumns) {
		t.Fatalf("live tasks table has %d columns, registry has %d", len(cols), len(taskColumns))
	}
	for _, c := range taskColumns {
		if !cols[c.name] {
			t.Errorf("registry column %s is missing from the live table", c.name)
		}
	}
}

func TestSchema_TaskArgsMatchesColumnCount(t *testing.T) {
	args := taskArgs(&domain.Task{})
	if len(args) != len(taskColumns) {
		t.Fatalf("taskArgs returns %d values for %d columns", len(args), len(taskColumns))
	}
}

func TestSchema_LegacyDatabaseMigrates(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	conn, err := driver.Open("file:"+dbPath, fts5.Register)
	if err != nil {
		t.Fatalf("failed to create legacy db: %v", err)
	}
	if _, err := conn.Exec(`
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			plan_id TEXT NOT NULL,
			title TEXT NOT NULL,
			dependencies TEXT NOT NULL,
			harness TEXT NOT NULL,
			skill_command TEXT NOT NULL,
			worktree_path TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
	`); err != nil {
		_ = conn.Close()
		t.Fatalf("failed to create legacy tasks table: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("failed to close legacy db: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed to migrate legacy database: %v", err)
	}
	defer db.Close()

	cols, err := tableColumns(context.Background(), db.sqlDB, "tasks")
	if err != nil {
		t.Fatalf("tableColumns failed: %v", err)
	}
	for _, c := range taskColumns {
		if !cols[c.name] {
			t.Errorf("legacy migration did not add column %s", c.name)
		}
	}
	if cols["skill_command"] {
		t.Error("legacy migration should drop the unused skill_command column")
	}
}

// TestSchema_MigratesSuspendedTasks parks runtime-era SUSPENDED rows in BLOCKED
// so they are visible to an operator instead of silently disappearing.
func TestSchema_MigratesSuspendedTasks(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "suspended.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := db.ExecImmediate(context.Background(), func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO plans (slug, title, status, created_at, updated_at)
			VALUES ('legacy-plan', 'legacy', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
			return err
		}
		_, err := tx.Exec(`
			INSERT INTO tasks (id, plan_id, title, dependencies, harness, worktree_path, status, notes, created_at, updated_at)
			VALUES ('S-01', 'legacy-plan', 'stalled', '[]', 'orchestrator', '', 'SUSPENDED', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`)
		return err
	}); err != nil {
		_ = db.Close()
		t.Fatalf("failed to seed suspended task: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer reopened.Close()

	var status, notes string
	if err := reopened.QueryRow(context.Background(),
		"SELECT status, notes FROM tasks WHERE id = 'S-01'").Scan(&status, &notes); err != nil {
		t.Fatalf("failed to read migrated task: %v", err)
	}
	if status != "BLOCKED" {
		t.Errorf("status = %s, want BLOCKED", status)
	}
	if !strings.Contains(notes, "Migrated from SUSPENDED") {
		t.Errorf("expected migration note, got %q", notes)
	}
}

// TestSchema_MigratesPopulatedLegacyTable guards the migration path against
// SQLite's rule that ADD COLUMN NOT NULL requires a default on a populated
// table.
func TestSchema_MigratesPopulatedLegacyTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy-populated.db")

	conn, err := driver.Open("file:"+dbPath, fts5.Register)
	if err != nil {
		t.Fatalf("failed to create legacy db: %v", err)
	}
	if _, err := conn.Exec(`
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			plan_id TEXT NOT NULL,
			title TEXT NOT NULL
		);
		INSERT INTO tasks (id, plan_id, title) VALUES ('old', 'legacy', 'legacy task');
	`); err != nil {
		_ = conn.Close()
		t.Fatalf("failed to seed legacy tasks table: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("failed to close legacy db: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed to migrate a populated legacy database: %v", err)
	}
	defer db.Close()

	var (
		status    string
		createdAt string
		deps      string
		planID    string
	)
	if err := db.QueryRow(context.Background(),
		"SELECT status, created_at, dependencies, plan_id FROM tasks WHERE id = 'old'").
		Scan(&status, &createdAt, &deps, &planID); err != nil {
		t.Fatalf("backfilled row is unreadable: %v", err)
	}
	if status != "PENDING" {
		t.Errorf("expected backfilled status default, got %q", status)
	}
	if createdAt == "" {
		t.Error("expected created_at to be backfilled")
	}
	if deps != "[]" {
		t.Errorf("expected dependencies default [], got %q", deps)
	}
	if planID != "legacy" {
		t.Errorf("pre-existing value changed: plan_id=%q", planID)
	}
}
