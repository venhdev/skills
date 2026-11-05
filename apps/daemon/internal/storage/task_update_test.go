package storage_test

import (
	"errors"
	"testing"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func strPtr(s string) *string { return &s }

func TestUpdateTaskMetadata_PatchesAndRederivesStatus(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "patch-plan",
		domain.Task{ID: "A", PlanID: "patch-plan", Title: "old", Status: domain.StatusReady},
		domain.Task{ID: "B", PlanID: "patch-plan", Title: "blocker", Status: domain.StatusReady},
	)

	deps := []string{"B"}
	priority := domain.PriorityP0
	if err := store.UpdateTaskMetadata(ctx, "A", domain.TaskPatch{
		Title:        strPtr("new"),
		Dependencies: &deps,
		Priority:     &priority,
	}); err != nil {
		t.Fatalf("UpdateTaskMetadata failed: %v", err)
	}

	task, err := store.GetTask(ctx, "A")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Title != "new" || task.Priority != domain.PriorityP0 {
		t.Errorf("metadata not applied: %+v", task)
	}
	if task.Status != domain.StatusPending {
		t.Errorf("expected PENDING after adding an unfinished dependency, got %s", task.Status)
	}

	// Completing the blocker unlocks the patched task.
	if err := store.UpdateTaskStatus(ctx, "B", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("failed to start B: %v", err)
	}
	if err := store.UpdateTaskStatus(ctx, "B", domain.StatusRunning, domain.StatusCompleted); err != nil {
		t.Fatalf("failed to complete B: %v", err)
	}
	task, err = store.GetTask(ctx, "A")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status != domain.StatusReady {
		t.Errorf("expected READY after dependency completion, got %s", task.Status)
	}
}

func TestUpdateTaskMetadata_RejectsRunningAndCycles(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "guard-plan",
		domain.Task{ID: "A", PlanID: "guard-plan", Title: "A", Status: domain.StatusReady},
	)

	if err := store.UpdateTaskMetadata(ctx, "A", domain.TaskPatch{Title: strPtr("ok")}); err != nil {
		t.Fatalf("READY patch failed: %v", err)
	}

	if _, err := store.ClaimTaskWithMeta(ctx, "A", "worker", "", "CODER"); err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if err := store.UpdateTaskMetadata(ctx, "A", domain.TaskPatch{Title: strPtr("late")}); !errors.Is(err, storage.ErrInvalidState) {
		t.Fatalf("expected RUNNING patch to be rejected, got %v", err)
	}

	// Introducing a cycle through an update is rejected before commit.
	seedPlan(t, store, ctx, "cycle-patch",
		domain.Task{ID: "C", PlanID: "cycle-patch", Title: "C", Status: domain.StatusReady},
		domain.Task{ID: "D", PlanID: "cycle-patch", Title: "D", Status: domain.StatusReady},
	)
	cDeps := []string{"D"}
	if err := store.UpdateTaskMetadata(ctx, "C", domain.TaskPatch{Dependencies: &cDeps}); err != nil {
		t.Fatalf("first edge failed: %v", err)
	}
	dDeps := []string{"C"}
	if err := store.UpdateTaskMetadata(ctx, "D", domain.TaskPatch{Dependencies: &dDeps}); !errors.Is(err, storage.ErrInvalidGraph) {
		t.Fatalf("expected cycle to be rejected, got %v", err)
	}
}

func TestDeleteArtifact_GatesInputOnInFlightTasks(t *testing.T) {
	store, _, ctx := newTestStore(t)
	seedPlan(t, store, ctx, "artifact-plan", domain.Task{
		ID: "A", PlanID: "artifact-plan", Title: "A", Status: domain.StatusReady,
	})

	art := domain.Artifact{
		ID: "art-1", TaskID: "A", Role: domain.ArtifactRoleInput,
		Type: domain.ArtifactTypeImage, FilePath: "/tmp/art-1.png",
		MimeType: "image/png", ByteSize: 1, Sha256: "x",
	}
	if err := store.RecordArtifact(ctx, art); err != nil {
		t.Fatalf("RecordArtifact failed: %v", err)
	}

	if _, err := store.ClaimTaskWithMeta(ctx, "A", "worker", "", "CODER"); err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if _, err := store.DeleteArtifact(ctx, "A", "art-1"); !errors.Is(err, storage.ErrInvalidState) {
		t.Fatalf("expected RUNNING delete to be rejected, got %v", err)
	}
}
