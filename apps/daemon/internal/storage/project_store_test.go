package storage_test

import (
	"errors"
	"testing"

	"zenf/internal/domain"
	"zenf/internal/storage"
)

func TestProjectStore_CreateGetList(t *testing.T) {
	store, ctx := newTestStore(t)

	project := domain.Project{Slug: "alpha", Name: "Alpha", RootPath: "/tmp/alpha", Description: "first"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	got, err := store.GetProject(ctx, "alpha")
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if got.Name != "Alpha" || got.RootPath != "/tmp/alpha" || got.Description != "first" {
		t.Fatalf("unexpected project: %+v", got)
	}
	if got.Status != domain.ProjectStatusActive {
		t.Fatalf("status = %s, want ACTIVE", got.Status)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatal("expected timestamps to be set")
	}

	seedProject(t, store, ctx, "beta")
	projects, err := store.ListProjects(ctx, false)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 2 || projects[0].Slug != "alpha" || projects[1].Slug != "beta" {
		t.Fatalf("unexpected project list: %+v", projects)
	}
}

func TestProjectStore_DuplicateAndNotFound(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")

	err := store.CreateProject(ctx, domain.Project{Slug: "alpha", Name: "Duplicate"})
	if !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	if _, err := store.GetProject(ctx, "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProjectStore_Validation(t *testing.T) {
	store, ctx := newTestStore(t)

	if err := store.CreateProject(ctx, domain.Project{Name: "No slug"}); err == nil {
		t.Fatal("expected error for missing slug")
	}
	if err := store.CreateProject(ctx, domain.Project{Slug: "no-name"}); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestProjectStore_UpdateArchiveAndDelete(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")

	patch := domain.ProjectPatch{
		Name:        ptr("Alpha Updated"),
		Description: ptr("updated description"),
	}
	if err := store.UpdateProject(ctx, "alpha", patch); err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}
	got, _ := store.GetProject(ctx, "alpha")
	if got.Name != "Alpha Updated" || got.Description != "updated description" {
		t.Fatalf("patch not applied: %+v", got)
	}

	if err := store.ArchiveProject(ctx, "alpha", true); err != nil {
		t.Fatalf("ArchiveProject failed: %v", err)
	}
	visible, _ := store.ListProjects(ctx, false)
	if len(visible) != 0 {
		t.Fatalf("expected archived project hidden, got %+v", visible)
	}
	all, _ := store.ListProjects(ctx, true)
	if len(all) != 1 || all[0].Status != domain.ProjectStatusArchived {
		t.Fatalf("expected archived project listed, got %+v", all)
	}

	if err := store.ArchiveProject(ctx, "alpha", false); err != nil {
		t.Fatalf("unarchive failed: %v", err)
	}
	visible, _ = store.ListProjects(ctx, false)
	if len(visible) != 1 || visible[0].Status != domain.ProjectStatusActive {
		t.Fatalf("expected active project listed, got %+v", visible)
	}

	if err := store.DeleteProject(ctx, "alpha"); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}
	if _, err := store.GetProject(ctx, "alpha"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected project deleted, got %v", err)
	}
	if err := store.DeleteProject(ctx, "alpha"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second delete, got %v", err)
	}
}

func TestProjectStore_DeleteCascadesPlansAndTasks(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "plan-alpha", []domain.Task{
		{ID: "T1", Title: "Task one"},
	})

	if err := store.DeleteProject(ctx, "alpha"); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}
	if _, err := store.GetPlan(ctx, "plan-alpha"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected plan cascade delete, got %v", err)
	}
	if _, err := store.GetTask(ctx, "T1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected task cascade delete, got %v", err)
	}
}
