package storage_test

import (
	"errors"
	"testing"

	"zenf/internal/domain"
	"zenf/internal/storage"
)

func TestArtifactStore_RecordListDelete(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{{ID: "A", Title: "A"}})

	inputs := []domain.Artifact{
		{
			ID: "in-1", TaskID: "A", Role: domain.ArtifactRoleInput, Name: "spec.md",
			Type: domain.ArtifactTypeMarkdown, FilePath: "/tmp/in-1.md",
			MimeType: "text/markdown", ByteSize: 10, Sha256: "aaa",
		},
		{
			ID: "out-1", TaskID: "A", Role: domain.ArtifactRoleOutput, Name: "log.txt",
			Type: domain.ArtifactTypeFile, FilePath: "/tmp/out-1.txt",
			MimeType: "text/plain", ByteSize: 20, Sha256: "bbb",
		},
	}
	for _, artifact := range inputs {
		if err := store.RecordArtifact(ctx, artifact); err != nil {
			t.Fatalf("RecordArtifact(%s) failed: %v", artifact.ID, err)
		}
	}

	all, err := store.GetArtifactsByTask(ctx, "A", "")
	if err != nil || len(all) != 2 {
		t.Fatalf("unexpected artifact list: %+v, err=%v", all, err)
	}
	onlyInput, err := store.GetArtifactsByTask(ctx, "A", domain.ArtifactRoleInput)
	if err != nil || len(onlyInput) != 1 || onlyInput[0].ID != "in-1" {
		t.Fatalf("unexpected input artifact list: %+v, err=%v", onlyInput, err)
	}

	got, err := store.GetArtifact(ctx, "out-1")
	if err != nil || got.ByteSize != 20 || got.Sha256 != "bbb" {
		t.Fatalf("GetArtifact mismatch: %+v, err=%v", got, err)
	}
	if _, err := store.GetArtifact(ctx, "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if _, err := store.DeleteArtifact(ctx, "A", "out-1"); err != nil {
		t.Fatalf("DeleteArtifact failed: %v", err)
	}
	if _, err := store.DeleteArtifact(ctx, "A", "out-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second delete, got %v", err)
	}
}

func TestArtifactStore_InputRequiresPreparedTask(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{
		{ID: "RUNNING", Title: "Running", Status: domain.StatusRunning},
		{ID: "READY", Title: "Ready", Status: domain.StatusReady},
	})

	err := store.RecordArtifact(ctx, domain.Artifact{
		ID: "in-running", TaskID: "RUNNING", Role: domain.ArtifactRoleInput,
		Type: domain.ArtifactTypeImage, FilePath: "/tmp/x.png", MimeType: "image/png",
	})
	if !errors.Is(err, storage.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState for RUNNING input, got %v", err)
	}

	if err := store.RecordArtifact(ctx, domain.Artifact{
		ID: "in-ready", TaskID: "READY", Role: domain.ArtifactRoleInput,
		Type: domain.ArtifactTypeImage, FilePath: "/tmp/y.png", MimeType: "image/png",
	}); err != nil {
		t.Fatalf("expected READY input to be allowed: %v", err)
	}
}

func TestArtifactStore_InvalidTypeAndCascade(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{{ID: "A", Title: "A"}})

	if err := store.RecordArtifact(ctx, domain.Artifact{
		ID: "bad", TaskID: "A", Type: "NOPE", FilePath: "/tmp/x",
	}); err == nil {
		t.Fatal("expected invalid artifact type to be rejected")
	}

	if err := store.RecordArtifact(ctx, domain.Artifact{
		ID: "out", TaskID: "A", Type: domain.ArtifactTypeFile, FilePath: "/tmp/out",
	}); err != nil {
		t.Fatalf("RecordArtifact failed: %v", err)
	}
	if err := store.DeletePlan(ctx, "p1"); err != nil {
		t.Fatalf("DeletePlan failed: %v", err)
	}
	if _, err := store.GetArtifact(ctx, "out"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected artifact cascade delete, got %v", err)
	}
}
