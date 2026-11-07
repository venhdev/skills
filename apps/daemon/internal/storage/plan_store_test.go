package storage_test

import (
	"errors"
	"testing"

	"zenf/internal/domain"
	"zenf/internal/storage"
)

func TestPlanStore_CreatePlanWithTasks(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")

	plan := domain.Plan{Slug: "p1", ProjectSlug: "alpha", Title: "Plan One"}
	tasks := []domain.Task{
		{ID: "A", Title: "Root"},
		{ID: "B", Title: "Child", Dependencies: []string{"A"}},
	}
	if err := store.CreatePlanWithTasks(ctx, plan, tasks); err != nil {
		t.Fatalf("CreatePlanWithTasks failed: %v", err)
	}

	gotPlan, err := store.GetPlan(ctx, "p1")
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	if gotPlan.ProjectSlug != "alpha" || gotPlan.Status != domain.PlanStatusActive {
		t.Fatalf("unexpected plan: %+v", gotPlan)
	}

	a := mustTask(t, store, ctx, "A")
	if a.ProjectSlug != "alpha" || a.PlanSlug != "p1" {
		t.Fatalf("task scope not populated: %+v", a)
	}
	if a.Status != domain.StatusReady {
		t.Fatalf("A status = %s, want READY", a.Status)
	}
	b := mustTask(t, store, ctx, "B")
	if b.Status != domain.StatusPending {
		t.Fatalf("B status = %s, want PENDING", b.Status)
	}
}

func TestPlanStore_RequiresExistingProject(t *testing.T) {
	store, ctx := newTestStore(t)

	err := store.CreatePlanWithTasks(ctx, domain.Plan{Slug: "p1", ProjectSlug: "ghost", Title: "P"}, nil)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing project, got %v", err)
	}
	if err := store.CreatePlanWithTasks(ctx, domain.Plan{Slug: "p2", Title: "P"}, nil); err == nil {
		t.Fatal("expected validation error for plan without project")
	}
}

func TestPlanStore_ReingestPreservesLifecycle(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{
		{ID: "A", Title: "Old title", Status: domain.StatusReady},
	})
	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	if err := store.CreatePlanWithTasks(ctx, domain.Plan{
		Slug: "p1", ProjectSlug: "alpha", Title: "Plan Renamed",
	}, []domain.Task{{ID: "A", Title: "New title"}}); err != nil {
		t.Fatalf("re-ingest failed: %v", err)
	}

	plan, _ := store.GetPlan(ctx, "p1")
	if plan.Title != "Plan Renamed" {
		t.Fatalf("plan title = %q, want Plan Renamed", plan.Title)
	}
	task := mustTask(t, store, ctx, "A")
	if task.Title != "New title" {
		t.Fatalf("task title = %q, want New title", task.Title)
	}
	if task.Status != domain.StatusRunning {
		t.Fatalf("re-ingest changed lifecycle status to %s", task.Status)
	}
}

func TestPlanStore_RejectsProjectDriftAndDuplicateTaskIDs(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedProject(t, store, ctx, "beta")
	seedPlan(t, store, ctx, "alpha", "p1", nil)

	if err := store.CreatePlanWithTasks(ctx, domain.Plan{
		Slug: "p1", ProjectSlug: "beta", Title: "Drifted",
	}, nil); !errors.Is(err, storage.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState on project drift, got %v", err)
	}

	err := store.CreatePlanWithTasks(ctx, domain.Plan{
		Slug: "p2", ProjectSlug: "alpha", Title: "Duplicates",
	}, []domain.Task{
		{ID: "DUP", Title: "One"},
		{ID: "DUP", Title: "Two"},
	})
	if !errors.Is(err, storage.ErrInvalidGraph) {
		t.Fatalf("expected ErrInvalidGraph for duplicate ids, got %v", err)
	}
}

func TestPlanStore_ListFilterAndArchive(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedProject(t, store, ctx, "beta")
	seedPlan(t, store, ctx, "alpha", "p1", nil)
	seedPlan(t, store, ctx, "beta", "p2", nil)

	alphaPlans, err := store.ListPlans(ctx, "alpha", false)
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if len(alphaPlans) != 1 || alphaPlans[0].Slug != "p1" {
		t.Fatalf("unexpected alpha plans: %+v", alphaPlans)
	}

	if err := store.ArchivePlan(ctx, "p1", true); err != nil {
		t.Fatalf("ArchivePlan failed: %v", err)
	}
	visible, _ := store.ListPlans(ctx, "", false)
	if len(visible) != 1 || visible[0].Slug != "p2" {
		t.Fatalf("expected archived plan hidden, got %+v", visible)
	}
	all, _ := store.ListPlans(ctx, "", true)
	if len(all) != 2 {
		t.Fatalf("expected both plans with include_archived, got %+v", all)
	}
	stored, _ := store.GetPlan(ctx, "p1")
	if stored.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}

	if err := store.ArchivePlan(ctx, "p1", false); err != nil {
		t.Fatalf("unarchive failed: %v", err)
	}
	stored, _ = store.GetPlan(ctx, "p1")
	if stored.ArchivedAt != nil {
		t.Fatal("expected archived_at to be cleared")
	}
}

func TestPlanStore_UpdateAndDelete(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{{ID: "A", Title: "A"}})

	if err := store.UpdatePlan(ctx, "p1", domain.PlanPatch{
		Title:  ptr("Renamed"),
		Status: ptr(domain.PlanStatusCompleted),
	}); err != nil {
		t.Fatalf("UpdatePlan failed: %v", err)
	}
	plan, _ := store.GetPlan(ctx, "p1")
	if plan.Title != "Renamed" || plan.Status != domain.PlanStatusCompleted {
		t.Fatalf("unexpected plan after update: %+v", plan)
	}

	if err := store.DeletePlan(ctx, "p1"); err != nil {
		t.Fatalf("DeletePlan failed: %v", err)
	}
	if _, err := store.GetPlan(ctx, "p1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected plan deleted, got %v", err)
	}
	if _, err := store.GetTask(ctx, "A"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected plan tasks cascade deleted, got %v", err)
	}
	if err := store.DeletePlan(ctx, "p1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second delete, got %v", err)
	}
}
