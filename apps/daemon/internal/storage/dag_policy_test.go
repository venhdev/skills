package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"zenf/internal/domain"
)

func newInternalTestStore(t *testing.T) (*Store, *DB, context.Context) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "dag.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return NewStore(db), db, ctx
}

func seedDAG(t *testing.T, store *Store, ctx context.Context, tasks []domain.Task) {
	t.Helper()
	if err := store.CreateProject(ctx, domain.Project{Slug: "dag-project", Name: "DAG Project"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	plan := domain.Plan{
		Slug:        "dag-plan",
		ProjectSlug: "dag-project",
		Title:       "DAG Policy",
		Status:      domain.PlanStatusActive,
	}
	if err := store.CreatePlanWithTasks(ctx, plan, tasks); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
}

func taskStatuses(t *testing.T, store *Store, ctx context.Context) map[string]domain.TaskStatus {
	t.Helper()
	tasks, err := store.ListTasks(ctx, domain.TaskFilter{PlanSlug: "dag-plan"})
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	statuses := make(map[string]domain.TaskStatus, len(tasks))
	for _, task := range tasks {
		statuses[task.ID] = task.Status
	}
	return statuses
}

func TestUpdateTaskStatus_CompletedUnlocksSatisfiedDependents(t *testing.T) {
	store, _, ctx := newInternalTestStore(t)
	seedDAG(t, store, ctx, []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusRunning},
		{ID: "B", Title: "B", Status: domain.StatusPending, Dependencies: []string{"A"}},
		{ID: "C", Title: "C", Status: domain.StatusPending, Dependencies: []string{"A", "D"}},
		{ID: "D", Title: "D", Status: domain.StatusPending},
	})

	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusRunning, domain.StatusCompleted); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	statuses := taskStatuses(t, store, ctx)
	if statuses["B"] != domain.StatusReady {
		t.Errorf("expected B READY, got %s", statuses["B"])
	}
	if statuses["C"] != domain.StatusPending {
		t.Errorf("expected C to stay PENDING while D is not COMPLETED, got %s", statuses["C"])
	}
	if statuses["D"] != domain.StatusReady {
		t.Errorf("expected dependency-free PENDING task D to unlock, got %s", statuses["D"])
	}
}

func TestUpdateTaskStatus_FailedBlocksTransitiveDependents(t *testing.T) {
	store, _, ctx := newInternalTestStore(t)
	seedDAG(t, store, ctx, []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusRunning},
		{ID: "B", Title: "B", Status: domain.StatusPending, Dependencies: []string{"A"}},
		{ID: "C", Title: "C", Status: domain.StatusPending, Dependencies: []string{"B"}},
		{ID: "D", Title: "D", Status: domain.StatusPending, Dependencies: []string{"C"}},
		{ID: "E", Title: "E", Status: domain.StatusReady},
	})

	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusRunning, domain.StatusFailed); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	statuses := taskStatuses(t, store, ctx)
	for _, id := range []string{"B", "C", "D"} {
		if statuses[id] != domain.StatusBlocked {
			t.Errorf("expected transitive dependent %s BLOCKED, got %s", id, statuses[id])
		}
	}
	if statuses["E"] != domain.StatusReady {
		t.Errorf("expected unrelated task E to stay READY, got %s", statuses["E"])
	}
}

func TestUpdateTaskStatus_FailedBlocksReadyDependent(t *testing.T) {
	store, _, ctx := newInternalTestStore(t)
	seedDAG(t, store, ctx, []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusRunning},
		// B was unlocked by an earlier completion of A and is still waiting to
		// be claimed when A is re-run and fails.
		{ID: "B", Title: "B", Status: domain.StatusReady, Dependencies: []string{"A"}},
	})

	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusRunning, domain.StatusFailed); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	statuses := taskStatuses(t, store, ctx)
	if statuses["B"] != domain.StatusBlocked {
		t.Errorf("expected READY dependent B BLOCKED, got %s", statuses["B"])
	}
}

func TestUpdateTaskStatus_IgnoresCorruptDepsOnInactiveTasks(t *testing.T) {
	store, db, ctx := newInternalTestStore(t)
	seedDAG(t, store, ctx, []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusRunning},
		{ID: "B", Title: "B", Status: domain.StatusPending, Dependencies: []string{"A"}},
		{ID: "C", Title: "C", Status: domain.StatusCompleted},
	})

	// C is already COMPLETED, so its dependency list no longer participates in
	// the transition policy. A corrupt value there must not fail A's transition.
	if err := db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE tasks SET dependencies = ? WHERE id = ?", "not-json", "C")
		return err
	}); err != nil {
		t.Fatalf("failed to corrupt dependencies: %v", err)
	}

	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusRunning, domain.StatusCompleted); err != nil {
		t.Fatalf("expected transition to ignore inactive task deps, got: %v", err)
	}

	statuses := taskStatuses(t, store, ctx)
	if statuses["A"] != domain.StatusCompleted {
		t.Errorf("expected A COMPLETED, got %s", statuses["A"])
	}
	if statuses["B"] != domain.StatusReady {
		t.Errorf("expected dependent B READY, got %s", statuses["B"])
	}
	if statuses["C"] != domain.StatusCompleted {
		t.Errorf("expected unrelated C to stay COMPLETED, got %s", statuses["C"])
	}
}

func TestUpdateTaskStatus_CorruptDependenciesRollBack(t *testing.T) {
	store, db, ctx := newInternalTestStore(t)
	seedDAG(t, store, ctx, []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusRunning},
		{ID: "B", Title: "B", Status: domain.StatusPending, Dependencies: []string{"A"}},
	})

	if err := db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE tasks SET dependencies = ? WHERE id = ?", "not-json", "B")
		return err
	}); err != nil {
		t.Fatalf("failed to corrupt dependencies: %v", err)
	}

	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusRunning, domain.StatusCompleted); err == nil {
		t.Fatal("expected corrupt dependency JSON to fail the transition")
	}

	task, err := store.GetTask(ctx, "A")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status != domain.StatusRunning {
		t.Fatalf("expected rollback to keep A RUNNING, got %s", task.Status)
	}
}
