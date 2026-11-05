package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fts5"
)

func openRawDB(t *testing.T, path string, statements string) {
	t.Helper()
	conn, err := driver.Open("file:"+path, fts5.Register)
	if err != nil {
		t.Fatalf("failed to create legacy db: %v", err)
	}
	if _, err := conn.Exec(statements); err != nil {
		_ = conn.Close()
		t.Fatalf("failed to seed legacy db: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("failed to close legacy db: %v", err)
	}
}

func TestSchema_CreatesV2TrackingPlane(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "schema.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	for _, table := range []string{
		"projects", "plans", "tasks", "documents", "documents_fts",
		"document_tags", "document_links", "agents", "agent_sessions",
		"artifacts", "events",
	} {
		exists, err := tableExists(ctx, db.sqlDB, table)
		if err != nil {
			t.Fatalf("tableExists(%s) failed: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %s to exist", table)
		}
	}

	for _, column := range []string{
		"project_slug", "plan_slug", "agent_slug", "required_docs",
		"claim_session_id", "review_options", "conflict_files",
	} {
		exists, err := columnExists(ctx, db.sqlDB, "tasks", column)
		if err != nil {
			t.Fatalf("columnExists(tasks.%s) failed: %v", column, err)
		}
		if !exists {
			t.Errorf("expected tasks.%s to exist", column)
		}
	}
}

func TestSchema_RecordsAppliedMigration(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "ledger.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM schema_migrations WHERE version = 1").Scan(&count); err != nil {
		t.Fatalf("failed to read migration ledger: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected migration 1 to be recorded once, got %d", count)
	}
}

func TestSchema_LegacyTrackingPlaneIsReset(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	openRawDB(t, dbPath, `
		CREATE TABLE plans (
			slug TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT NOT NULL
		);
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			plan_id TEXT NOT NULL,
			title TEXT NOT NULL,
			status TEXT NOT NULL
		);
		INSERT INTO plans (slug, title, status) VALUES ('old-plan', 'old', 'ACTIVE');
		INSERT INTO tasks (id, plan_id, title, status) VALUES ('old-task', 'old-plan', 'old', 'READY');
	`)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed to reset legacy database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if exists, _ := columnExists(ctx, db.sqlDB, "plans", "project_slug"); !exists {
		t.Error("expected legacy plans table to be replaced by the v0.2 shape")
	}
	if exists, _ := columnExists(ctx, db.sqlDB, "tasks", "plan_id"); exists {
		t.Error("expected legacy tasks.plan_id column to disappear")
	}

	var planCount, taskCount int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM plans").Scan(&planCount); err != nil {
		t.Fatalf("failed to count plans: %v", err)
	}
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM tasks").Scan(&taskCount); err != nil {
		t.Fatalf("failed to count tasks: %v", err)
	}
	if planCount != 0 || taskCount != 0 {
		t.Fatalf("expected legacy rows to be dropped, got %d plans and %d tasks", planCount, taskCount)
	}
}

func TestSchema_ReopenPreservesV2Data(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reopen.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if _, err := db.sqlDB.Exec(`
		INSERT INTO projects (slug, name, created_at, updated_at)
		VALUES ('p1', 'Project One', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		_ = db.Close()
		t.Fatalf("failed to seed project: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer reopened.Close()

	var name string
	if err := reopened.QueryRow(context.Background(),
		"SELECT name FROM projects WHERE slug = 'p1'").Scan(&name); err != nil {
		t.Fatalf("expected v0.2 data to survive reopen: %v", err)
	}
	if name != "Project One" {
		t.Fatalf("name = %q, want Project One", name)
	}
}
