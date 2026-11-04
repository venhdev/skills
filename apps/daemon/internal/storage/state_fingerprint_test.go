package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"zenfd/internal/storage"
)

func TestStore_StateFingerprintDetectsExternalWrites(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fingerprint.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	before, err := store.StateFingerprint(ctx)
	if err != nil {
		t.Fatalf("StateFingerprint failed: %v", err)
	}

	now := time.Now().UTC()
	if _, err := db.RawDB().ExecContext(ctx,
		`INSERT INTO plans (slug, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"external-plan", "External Plan", "ACTIVE", now, now,
	); err != nil {
		t.Fatalf("failed to insert plan: %v", err)
	}

	afterPlan, err := store.StateFingerprint(ctx)
	if err != nil {
		t.Fatalf("StateFingerprint failed: %v", err)
	}
	if afterPlan == before {
		t.Fatalf("fingerprint did not change after plan insert: %q", afterPlan)
	}

	if _, err := db.RawDB().ExecContext(ctx,
		`INSERT INTO tasks (id, plan_id, title, dependencies, harness, skill_command, worktree_path, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"EXT-01", "external-plan", "External Task", "[]", "orchestrator", "", "", "READY", now, now,
	); err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}

	afterTask, err := store.StateFingerprint(ctx)
	if err != nil {
		t.Fatalf("StateFingerprint failed: %v", err)
	}
	if afterTask == afterPlan {
		t.Fatalf("fingerprint did not change after task insert: %q", afterTask)
	}

	if _, err := db.RawDB().ExecContext(ctx,
		`UPDATE tasks SET notes = ?, updated_at = ? WHERE id = ?`,
		"external note", now.Add(time.Second), "EXT-01",
	); err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

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
