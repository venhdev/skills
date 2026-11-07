package storage_test

import (
	"errors"
	"strings"
	"testing"

	"zenf/internal/domain"
	"zenf/internal/storage"
)

func TestTaskLifecycle_TransitionRules(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusReady},
	})

	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusCompleted, domain.StatusRunning); !errors.Is(err, storage.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState for invalid transition, got %v", err)
	}
	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusRunning, domain.StatusRunning); !errors.Is(err, storage.ErrConflictStatus) {
		t.Fatalf("expected ErrConflictStatus for stale expected status, got %v", err)
	}
	if err := store.UpdateTaskStatus(ctx, "missing", domain.StatusReady, domain.StatusRunning); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("valid transition failed: %v", err)
	}
	if got := mustTask(t, store, ctx, "A").Status; got != domain.StatusRunning {
		t.Fatalf("status = %s, want RUNNING", got)
	}
}

func TestTaskLifecycle_TransitionNoteIsAtomic(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusRunning},
	})

	if err := store.UpdateTaskStatusWithNote(ctx, "A", domain.StatusRunning, domain.StatusCompleted, "Summary: done"); err != nil {
		t.Fatalf("UpdateTaskStatusWithNote failed: %v", err)
	}
	task := mustTask(t, store, ctx, "A")
	if task.Status != domain.StatusCompleted {
		t.Fatalf("status = %s, want COMPLETED", task.Status)
	}
	if !strings.Contains(task.Notes, "Summary: done") {
		t.Fatalf("expected note to be persisted, got %q", task.Notes)
	}
}

func TestTaskLifecycle_EvictClearsClaimAndRecomputesReadiness(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusReady},
		{ID: "B", Title: "B", Dependencies: []string{"A"}},
	})

	if _, err := store.ClaimTask(ctx, "B", domain.ClaimOptions{WorkerID: "w"}); !errors.Is(err, storage.ErrConflictStatus) {
		t.Fatalf("expected dependent B to be unclaimable, got %v", err)
	}

	if _, err := store.ClaimTask(ctx, "A", domain.ClaimOptions{
		WorkerID: "w", OrchestratorID: "main", WorkerRole: "CODER", SessionID: "s1",
	}); err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}

	status, err := store.EvictTask(ctx, "A", "worker lost", "RESET_TO_READY")
	if err != nil {
		t.Fatalf("EvictTask failed: %v", err)
	}
	if status != domain.StatusReady {
		t.Fatalf("evicted status = %s, want READY", status)
	}
	task := mustTask(t, store, ctx, "A")
	if task.Status != domain.StatusReady || task.ClaimedBy != "" || task.ClaimedAt != nil ||
		task.OrchestratorID != "" || task.WorkerRole != "" || task.ClaimSessionID != "" {
		t.Fatalf("claim metadata not cleared: %+v", task)
	}
	if !strings.Contains(task.Notes, "Eviction: worker lost") {
		t.Fatalf("expected eviction note, got %q", task.Notes)
	}
}

func TestTaskLifecycle_EvictBlocksWhenUpstreamIncomplete(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{
		{ID: "A", Title: "A"},
		{ID: "B", Title: "B", Dependencies: []string{"A"}},
	})

	// Force B into RUNNING directly because its dependency is incomplete.
	if err := store.UpdateTaskStatus(ctx, "B", domain.StatusPending, domain.StatusReady); err != nil {
		t.Fatalf("failed to force B ready: %v", err)
	}
	if _, err := store.ClaimTask(ctx, "B", domain.ClaimOptions{WorkerID: "w"}); err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}

	status, err := store.EvictTask(ctx, "B", "retry", "RESET_TO_READY")
	if err != nil {
		t.Fatalf("EvictTask failed: %v", err)
	}
	if status != domain.StatusBlocked {
		t.Fatalf("evicted status = %s, want BLOCKED", status)
	}
}

func TestTaskLifecycle_EvictCancel(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{{ID: "A", Title: "A", Status: domain.StatusRunning}})

	status, err := store.EvictTask(ctx, "A", "operator abort", "CANCEL")
	if err != nil {
		t.Fatalf("EvictTask failed: %v", err)
	}
	if status != domain.StatusCancelled {
		t.Fatalf("status = %s, want CANCELLED", status)
	}
	if _, err := store.EvictTask(ctx, "A", "again", "CANCEL"); !errors.Is(err, storage.ErrConflictStatus) {
		t.Fatalf("expected ErrConflictStatus when evicting non-running task, got %v", err)
	}
}

func TestTaskLifecycle_ReviewHandshake(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusReady},
	})

	if err := store.RequestReview(ctx, "A", "early", "not running", nil); !errors.Is(err, storage.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState before claim, got %v", err)
	}
	if _, err := store.ClaimTask(ctx, "A", domain.ClaimOptions{WorkerID: "w", SessionID: "s"}); err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}
	if err := store.RequestReview(ctx, "A", "scope drift", "needs decision", []string{"continue", "stop"}); err != nil {
		t.Fatalf("RequestReview failed: %v", err)
	}
	task := mustTask(t, store, ctx, "A")
	if task.Status != domain.StatusAwaitingReview {
		t.Fatalf("status = %s, want AWAITING_REVIEW", task.Status)
	}
	if task.ReviewReason != "scope drift" || task.ReviewDetails != "needs decision" || len(task.ReviewOptions) != 2 {
		t.Fatalf("review fields not persisted: %+v", task)
	}

	if err := store.ResolveReview(ctx, "A", "continue with option A"); err != nil {
		t.Fatalf("ResolveReview failed: %v", err)
	}
	task = mustTask(t, store, ctx, "A")
	if task.Status != domain.StatusReady {
		t.Fatalf("status = %s, want READY", task.Status)
	}
	if task.ReviewResponse != "continue with option A" || task.ResolvedBy != "main-agent" {
		t.Fatalf("review resolution not persisted: %+v", task)
	}
	if task.ClaimedBy != "" || task.ClaimSessionID != "" {
		t.Fatalf("expected claim cleared after review: %+v", task)
	}
}

func TestTaskLifecycle_ContainerIsolation(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{
		{ID: "P1-A", Title: "A", Status: domain.StatusRunning},
	})
	if err := store.CreateTasks(ctx, []domain.Task{
		{ID: "PROJ-A", ProjectSlug: "alpha", Title: "Project A", Status: domain.StatusRunning},
	}); err != nil {
		t.Fatalf("failed to create project task: %v", err)
	}

	if err := store.UpdateTaskStatus(ctx, "P1-A", domain.StatusRunning, domain.StatusCompleted); err != nil {
		t.Fatalf("complete plan task failed: %v", err)
	}
	if got := mustTask(t, store, ctx, "PROJ-A").Status; got != domain.StatusRunning {
		t.Fatalf("project task changed status across containers: %s", got)
	}
}
