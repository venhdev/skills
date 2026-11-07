package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"zenf/internal/domain"
	"zenf/internal/storage"
)

// newTestStore opens a fresh v0.2 database in t.TempDir().
func newTestStore(t *testing.T) (*storage.Store, context.Context) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return storage.NewStore(db), ctx
}

func seedProject(t *testing.T, store *storage.Store, ctx context.Context, slug string) {
	t.Helper()
	if err := store.CreateProject(ctx, domain.Project{
		Slug:        slug,
		Name:        "Project " + slug,
		RootPath:    "/tmp/" + slug,
		Description: "Test project",
	}); err != nil {
		t.Fatalf("CreateProject(%s) failed: %v", slug, err)
	}
}

func seedPlan(t *testing.T, store *storage.Store, ctx context.Context, projectSlug, planSlug string, tasks []domain.Task) {
	t.Helper()
	if err := store.CreatePlanWithTasks(ctx, domain.Plan{
		Slug:        planSlug,
		ProjectSlug: projectSlug,
		Title:       "Plan " + planSlug,
		Status:      domain.PlanStatusActive,
	}, tasks); err != nil {
		t.Fatalf("CreatePlanWithTasks(%s) failed: %v", planSlug, err)
	}
}

func ptr[T any](value T) *T {
	return &value
}

func mustTask(t *testing.T, store *storage.Store, ctx context.Context, id string) *domain.Task {
	t.Helper()
	task, err := store.GetTask(ctx, id)
	if err != nil {
		t.Fatalf("GetTask(%s) failed: %v", id, err)
	}
	return task
}
