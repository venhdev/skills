package storage_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func TestPlans_ArchiveHidesFromDefaultList(t *testing.T) {
	store, _, ctx := newTestStore(t)
	for _, slug := range []string{"active-plan", "done-plan"} {
		if err := store.CreatePlan(ctx, domain.Plan{Slug: slug, Title: slug, Status: "ACTIVE"}); err != nil {
			t.Fatalf("CreatePlan failed: %v", err)
		}
	}

	if err := store.ArchivePlan(ctx, "done-plan", true); err != nil {
		t.Fatalf("ArchivePlan failed: %v", err)
	}

	visible, err := store.ListPlans(ctx, false)
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if len(visible) != 1 || visible[0].Slug != "active-plan" {
		t.Fatalf("archived plan leaked into default list: %+v", visible)
	}

	all, err := store.ListPlans(ctx, true)
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected both plans with include_archived, got %+v", all)
	}

	plan, err := store.GetPlan(ctx, "done-plan")
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	if plan.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}

	if err := store.ArchivePlan(ctx, "done-plan", false); err != nil {
		t.Fatalf("unarchive failed: %v", err)
	}
	plan, err = store.GetPlan(ctx, "done-plan")
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	if plan.ArchivedAt != nil {
		t.Fatal("expected archived_at to be cleared")
	}
}

func TestPlans_ArchiveUnknownReturnsNotFound(t *testing.T) {
	store, _, ctx := newTestStore(t)
	if err := store.ArchivePlan(ctx, "ghost-plan", true); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateTasks_BatchIsAtomicAndTopological(t *testing.T) {
	store, _, ctx := newTestStore(t)
	if err := store.CreatePlan(ctx, domain.Plan{Slug: "batch-plan", Title: "Batch", Status: "ACTIVE"}); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	err := store.CreateTasks(ctx, []domain.Task{
		{ID: "B-01", PlanID: "batch-plan", Title: "first"},
		{ID: "B-02", PlanID: "batch-plan", Title: "second", Dependencies: []string{"B-01"}},
		{ID: "B-03", PlanID: "batch-plan", Title: "third", Dependencies: []string{"B-02"}},
	})
	if err != nil {
		t.Fatalf("CreateTasks failed: %v", err)
	}

	first, err := store.GetTask(ctx, "B-01")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if first.Status != domain.StatusReady {
		t.Errorf("dependency-free task should be READY, got %s", first.Status)
	}
	second, err := store.GetTask(ctx, "B-02")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if second.Status != domain.StatusPending {
		t.Errorf("dependent task should be PENDING, got %s", second.Status)
	}

	// A duplicate inside the batch must roll the whole batch back.
	err = store.CreateTasks(ctx, []domain.Task{
		{ID: "B-04", PlanID: "batch-plan", Title: "new"},
		{ID: "B-01", PlanID: "batch-plan", Title: "duplicate"},
	})
	if !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
	if _, err := store.GetTask(ctx, "B-04"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("partial batch leaked task B-04: %v", err)
	}

	// A cycle across the whole batch is rejected.
	err = store.CreateTasks(ctx, []domain.Task{
		{ID: "C-01", PlanID: "batch-plan", Title: "c1", Dependencies: []string{"C-02"}},
		{ID: "C-02", PlanID: "batch-plan", Title: "c2", Dependencies: []string{"C-01"}},
	})
	if !errors.Is(err, storage.ErrInvalidGraph) {
		t.Fatalf("expected ErrInvalidGraph for a batch cycle, got %v", err)
	}
}

func TestRecordArtifact_InputMediaOnlyBeforeClaim(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "media-plan", domain.Task{
		ID: "M-01", PlanID: "media-plan", Title: "media",
	})

	artifact := func(id string) domain.Artifact {
		path := filepath.Join(t.TempDir(), id+".png")
		if err := os.WriteFile(path, []byte("not-a-real-png"), 0o644); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
		return domain.Artifact{
			ID: id, TaskID: "M-01", Role: domain.ArtifactRoleInput,
			Type: domain.ArtifactTypeImage, FilePath: path,
			MimeType: "image/png", ByteSize: 15, Sha256: "deadbeef",
		}
	}

	if err := store.RecordArtifact(ctx, artifact("art-ready")); err != nil {
		t.Fatalf("READY task should accept input media: %v", err)
	}

	if _, err := store.ClaimTaskWithMeta(ctx, "M-01", "worker", "", "CODER"); err != nil {
		t.Fatalf("claim failed: %v", err)
	}

	if err := store.RecordArtifact(ctx, artifact("art-running")); !errors.Is(err, storage.ErrInvalidState) {
		t.Fatalf("RUNNING task must reject input media, got %v", err)
	}

	output := artifact("art-output")
	output.Role = domain.ArtifactRoleOutput
	if err := store.RecordArtifact(ctx, output); err != nil {
		t.Fatalf("OUTPUT artifacts should not be gated: %v", err)
	}
}

var _ = context.Background

func TestTasks_DeriveStatusFromExistingPrerequisites(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "derive-plan",
		domain.Task{ID: "A", PlanID: "derive-plan", Title: "done upstream", Status: domain.StatusReady},
		domain.Task{ID: "F", PlanID: "derive-plan", Title: "failed upstream", Status: domain.StatusRunning},
	)

	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("failed to start A: %v", err)
	}
	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusRunning, domain.StatusCompleted); err != nil {
		t.Fatalf("failed to complete A: %v", err)
	}
	if err := store.UpdateTaskStatus(ctx, "F", domain.StatusRunning, domain.StatusFailed); err != nil {
		t.Fatalf("failed to fail F: %v", err)
	}

	// Added after its prerequisite finished: must land READY, not PENDING.
	if err := store.CreateTasks(ctx, []domain.Task{
		{ID: "B", PlanID: "derive-plan", Title: "after completed", Dependencies: []string{"A"}},
	}); err != nil {
		t.Fatalf("CreateTasks failed: %v", err)
	}
	b, err := store.GetTask(ctx, "B")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if b.Status != domain.StatusReady {
		t.Errorf("task added after its dependency completed = %s, want READY", b.Status)
	}

	// Added after its prerequisite failed: must land BLOCKED, not PENDING.
	if err := store.CreateTasks(ctx, []domain.Task{
		{ID: "C", PlanID: "derive-plan", Title: "after failure", Dependencies: []string{"F"}},
	}); err != nil {
		t.Fatalf("CreateTasks failed: %v", err)
	}
	c, err := store.GetTask(ctx, "C")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if c.Status != domain.StatusBlocked {
		t.Errorf("task added after a failed dependency = %s, want BLOCKED", c.Status)
	}

	// Re-pointing a PENDING task at a completed prerequisite unlocks it.
	if err := store.CreateTasks(ctx, []domain.Task{
		{ID: "D", PlanID: "derive-plan", Title: "forward ref", Dependencies: []string{"ghost"}},
	}); err != nil {
		t.Fatalf("CreateTasks failed: %v", err)
	}
	if err := store.UpdateTaskDependencies(ctx, "D", []string{"A"}); err != nil {
		t.Fatalf("UpdateTaskDependencies failed: %v", err)
	}
	d, err := store.GetTask(ctx, "D")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if d.Status != domain.StatusReady {
		t.Errorf("task whose dependencies became complete = %s, want READY", d.Status)
	}
}
