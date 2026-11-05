package storage_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"zenfd/internal/storage"
)

// execExternal simulates a writer outside the daemon: it opens its own
// connection and never touches the store's write mutex.
func execExternal(t *testing.T, db *storage.DB, ctx context.Context, query string, args ...any) {
	t.Helper()
	if err := db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		t.Fatalf("external write failed: %v", err)
	}
}

func TestStore_StateFingerprintDetectsExternalWrites(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fingerprint.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Second connection stands in for the stdio MCP server or a CLI writer.
	external, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open external DB: %v", err)
	}
	defer external.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	before, err := store.StateFingerprint(ctx)
	if err != nil {
		t.Fatalf("StateFingerprint failed: %v", err)
	}

	now := time.Now().UTC()
	execExternal(t, external, ctx,
		`INSERT INTO plans (slug, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"external-plan", "External Plan", "ACTIVE", now, now,
	)

	afterPlan, err := store.StateFingerprint(ctx)
	if err != nil {
		t.Fatalf("StateFingerprint failed: %v", err)
	}
	if afterPlan == before {
		t.Fatalf("fingerprint did not change after plan insert: %q", afterPlan)
	}

	execExternal(t, external, ctx,
		`INSERT INTO tasks (id, plan_id, title, dependencies, harness, worktree_path, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"EXT-01", "external-plan", "External Task", "[]", "orchestrator", "", "READY", now, now,
	)

	afterTask, err := store.StateFingerprint(ctx)
	if err != nil {
		t.Fatalf("StateFingerprint failed: %v", err)
	}
	if afterTask == afterPlan {
		t.Fatalf("fingerprint did not change after task insert: %q", afterTask)
	}

	execExternal(t, external, ctx,
		`UPDATE tasks SET notes = ?, updated_at = ? WHERE id = ?`,
		"external note", now.Add(time.Second), "EXT-01",
	)

	afterUpdate, err := store.StateFingerprint(ctx)
	if err != nil {
		t.Fatalf("StateFingerprint failed: %v", err)
	}
	if afterUpdate == afterTask {
		t.Fatalf("fingerprint did not change after task update: %q", afterUpdate)
	}

	if err := store.DeletePlan(ctx, "external-plan"); err != nil {
		t.Fatalf("DeletePlan failed: %v", err)
	}

	afterDelete, err := store.StateFingerprint(ctx)
	if err != nil {
		t.Fatalf("StateFingerprint failed: %v", err)
	}
	if afterDelete == afterUpdate {
		t.Fatalf("fingerprint did not change after plan delete: %q", afterDelete)
	}
}
