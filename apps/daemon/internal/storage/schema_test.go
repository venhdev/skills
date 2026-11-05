package storage

import (
	"context"
	"path/filepath"
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
	args := taskArgs(&domain.Task{}, "plan")
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
}
