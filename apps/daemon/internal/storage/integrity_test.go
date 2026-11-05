package storage_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/storage"

	"database/sql"
)

func newTestStore(t *testing.T) (*storage.Store, *storage.DB, context.Context) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "integrity.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return storage.NewStore(db), db, ctx
}

func seedPlan(t *testing.T, store *storage.Store, ctx context.Context, slug string, tasks ...domain.Task) {
	t.Helper()
	plan := domain.Plan{Slug: slug, Title: slug, Status: "ACTIVE"}
	if err := store.SavePlanWithTasks(ctx, plan, tasks); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}
}

// TestCreateTask_RejectsDuplicateWithoutTouchingState locks the create-only
// contract: a second create for an existing ID must fail and leave the
// lifecycle columns exactly as they were.
func TestCreateTask_RejectsDuplicateWithoutTouchingState(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "dup-plan", domain.Task{
		ID: "T1", PlanID: "dup-plan", Title: "Original", Status: domain.StatusReady,
	})

	packet, err := store.ClaimTaskWithMeta(ctx, "T1", "worker-1", "orch-1", "CODER")
	if err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if packet.Task.Status != domain.StatusRunning {
		t.Fatalf("expected RUNNING after claim, got %s", packet.Task.Status)
	}
	if err := store.AppendTaskNotes(ctx, "T1", "note-before-retry"); err != nil {
		t.Fatalf("AppendTaskNotes failed: %v", err)
	}
	before, err := store.GetTask(ctx, "T1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	duplicate := domain.Task{
		ID: "T1", PlanID: "dup-plan", Title: "Overwritten", Harness: "orchestrator", Status: domain.StatusReady,
	}
	if err := store.CreateTask(ctx, duplicate); !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	after, err := store.GetTask(ctx, "T1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if after.Status != domain.StatusRunning {
		t.Errorf("status was reset to %s", after.Status)
	}
	if after.ClaimedBy != "worker-1" || after.OrchestratorID != "orch-1" {
		t.Errorf("claim metadata was reset: %+v", after)
	}
	if after.Notes != before.Notes {
		t.Errorf("notes changed: %q -> %q", before.Notes, after.Notes)
	}
	if !after.CreatedAt.Equal(before.CreatedAt) {
		t.Errorf("created_at changed: %v -> %v", before.CreatedAt, after.CreatedAt)
	}
}

// TestSavePlanWithTasks_PatchesMetadataOnly proves a dossier re-ingest can
// change authoring metadata without regressing the task lifecycle.
func TestSavePlanWithTasks_PatchesMetadataOnly(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "patch-plan", domain.Task{
		ID: "T1", PlanID: "patch-plan", Title: "Old title", Description: "old",
		Priority: domain.PriorityP2, Status: domain.StatusReady,
	})
	if _, err := store.ClaimTaskWithMeta(ctx, "T1", "worker-1", "orch-1", "CODER"); err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	before, err := store.GetTask(ctx, "T1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	seedPlan(t, store, ctx, "patch-plan", domain.Task{
		ID: "T1", PlanID: "patch-plan", Title: "New title", Description: "new",
		Priority: domain.PriorityP0, Status: domain.StatusPending,
	})

	after, err := store.GetTask(ctx, "T1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if after.Title != "New title" || after.Description != "new" || after.Priority != domain.PriorityP0 {
		t.Errorf("metadata was not patched: %+v", after)
	}
	if after.Status != before.Status {
		t.Errorf("status was overwritten by re-ingest: %s -> %s", before.Status, after.Status)
	}
	if after.ClaimedBy != before.ClaimedBy {
		t.Errorf("claim metadata was overwritten: %+v", after)
	}
	if !after.CreatedAt.Equal(before.CreatedAt) {
		t.Errorf("created_at was overwritten: %v -> %v", before.CreatedAt, after.CreatedAt)
	}
}

func TestSavePlanWithTasks_RejectsUnknownDependency(t *testing.T) {
	store, _, ctx := newTestStore(t)
	err := store.SavePlanWithTasks(ctx, domain.Plan{Slug: "bad-plan", Title: "bad", Status: "ACTIVE"}, []domain.Task{
		{ID: "T1", PlanID: "bad-plan", Title: "T1", Status: domain.StatusPending, Dependencies: []string{"ghost"}},
	})
	if !errors.Is(err, storage.ErrInvalidGraph) {
		t.Fatalf("expected ErrInvalidGraph, got %v", err)
	}
	if _, getErr := store.GetPlan(ctx, "bad-plan"); !errors.Is(getErr, storage.ErrNotFound) {
		t.Errorf("rejected plan should have rolled back, got %v", getErr)
	}
}

func TestSavePlanWithTasks_RejectsCycle(t *testing.T) {
	store, _, ctx := newTestStore(t)
	err := store.SavePlanWithTasks(ctx, domain.Plan{Slug: "cycle-plan", Title: "cycle", Status: "ACTIVE"}, []domain.Task{
		{ID: "A", PlanID: "cycle-plan", Title: "A", Status: domain.StatusPending, Dependencies: []string{"B"}},
		{ID: "B", PlanID: "cycle-plan", Title: "B", Status: domain.StatusPending, Dependencies: []string{"A"}},
	})
	if !errors.Is(err, storage.ErrInvalidGraph) {
		t.Fatalf("expected ErrInvalidGraph for a cycle, got %v", err)
	}
}

func TestCreateTask_RejectsSelfDependency(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "self-plan")
	err := store.CreateTask(ctx, domain.Task{
		ID: "T1", PlanID: "self-plan", Title: "T1", Status: domain.StatusPending,
		Dependencies: []string{"T1"},
	})
	if !errors.Is(err, storage.ErrInvalidGraph) {
		t.Fatalf("expected ErrInvalidGraph, got %v", err)
	}
}

// TestCreateTask_AllowsForwardReference keeps incremental authoring working:
// tasks may be created before the prerequisite they name.
func TestCreateTask_AllowsForwardReference(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "forward-plan")
	if err := store.CreateTask(ctx, domain.Task{
		ID: "T2", PlanID: "forward-plan", Title: "T2", Status: domain.StatusPending,
		Dependencies: []string{"T1"},
	}); err != nil {
		t.Fatalf("forward reference should be accepted, got %v", err)
	}
	if err := store.CreateTask(ctx, domain.Task{
		ID: "T1", PlanID: "forward-plan", Title: "T1", Status: domain.StatusReady,
	}); err != nil {
		t.Fatalf("failed to create the prerequisite: %v", err)
	}
}

// TestEvictTask_ClearsClaimMetadata checks that a reset leaves no stale worker
// identity behind.
func TestEvictTask_ClearsClaimMetadata(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "evict-plan", domain.Task{
		ID: "T1", PlanID: "evict-plan", Title: "T1", Status: domain.StatusReady,
	})
	if _, err := store.ClaimTaskWithMeta(ctx, "T1", "worker-1", "orch-1", "CODER"); err != nil {
		t.Fatalf("claim failed: %v", err)
	}

	status, err := store.EvictTask(ctx, "T1", "worker died", "RESET_TO_READY")
	if err != nil {
		t.Fatalf("EvictTask failed: %v", err)
	}
	if status != domain.StatusReady {
		t.Fatalf("expected READY, got %s", status)
	}

	task, err := store.GetTask(ctx, "T1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status != domain.StatusReady {
		t.Errorf("status = %s, want READY", task.Status)
	}
	if task.ClaimedBy != "" || task.OrchestratorID != "" || task.WorkerRole != "" {
		t.Errorf("claim identity not cleared: %+v", task)
	}
	if task.ClaimedAt != nil {
		t.Errorf("claimed_at not cleared: %v", task.ClaimedAt)
	}
}

// TestEvictTask_BlocksWhenPrerequisitesUnfinished keeps a reset task out of the
// READY pool when an upstream dependency failed.
func TestEvictTask_BlocksWhenPrerequisitesUnfinished(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "evict-dep-plan",
		domain.Task{ID: "A", PlanID: "evict-dep-plan", Title: "A", Status: domain.StatusRunning},
		domain.Task{ID: "B", PlanID: "evict-dep-plan", Title: "B", Status: domain.StatusRunning, Dependencies: []string{"A"}},
	)
	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusRunning, domain.StatusFailed); err != nil {
		t.Fatalf("failed to fail A: %v", err)
	}

	status, err := store.EvictTask(ctx, "B", "reset", "RESET_TO_READY")
	if err != nil {
		t.Fatalf("EvictTask failed: %v", err)
	}
	if status != domain.StatusBlocked {
		t.Fatalf("expected BLOCKED, got %s", status)
	}
	task, err := store.GetTask(ctx, "B")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status != domain.StatusBlocked {
		t.Errorf("persisted status = %s, want BLOCKED", task.Status)
	}
}

func TestTaskWrites_UnknownTaskReturnsNotFound(t *testing.T) {
	store, _, ctx := newTestStore(t)

	if err := store.UpdateTaskDependencies(ctx, "ghost", []string{"A"}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("UpdateTaskDependencies: expected ErrNotFound, got %v", err)
	}
	if err := store.UpdateTaskNotes(ctx, "ghost", "note"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("UpdateTaskNotes: expected ErrNotFound, got %v", err)
	}
	if err := store.AppendTaskNotes(ctx, "ghost", "note"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("AppendTaskNotes: expected ErrNotFound, got %v", err)
	}
}

// TestClaimTask_RejectsCorruptDependencies prevents a task whose DAG edges are
// unreadable from being claimed as if it had no prerequisites.
func TestClaimTask_RejectsCorruptDependencies(t *testing.T) {
	store, db, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "corrupt-plan", domain.Task{
		ID: "T1", PlanID: "corrupt-plan", Title: "T1", Status: domain.StatusReady,
	})
	if err := db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE tasks SET dependencies = ? WHERE id = ?", "not-json", "T1")
		return err
	}); err != nil {
		t.Fatalf("failed to corrupt dependencies: %v", err)
	}

	if _, err := store.ClaimTaskWithMeta(ctx, "T1", "worker", "", ""); !errors.Is(err, storage.ErrInvalidGraph) {
		t.Fatalf("expected ErrInvalidGraph, got %v", err)
	}
	task, err := store.GetTask(ctx, "T1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status != domain.StatusReady {
		t.Errorf("corrupt task was claimed anyway: %s", task.Status)
	}
}
