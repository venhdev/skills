package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func TestArtifactStore_ClaimCheck(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_artifact.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Seed plan and task
	plan := domain.Plan{Slug: "p1", Title: "P1", Status: "ACTIVE"}
	task := domain.Task{ID: "t1", PlanID: "p1", Title: "T1", Status: domain.StatusReady}
	_ = store.SavePlanWithTasks(ctx, plan, []domain.Task{task})

	// Create output artifact
	artifact := domain.Artifact{
		ID:        "art-123",
		RunID:     "run-dummy",
		TaskID:    "t1",
		Role:      domain.ArtifactRoleOutput,
		Name:      "diff.png",
		Type:      domain.ArtifactTypeImage,
		FilePath:  "/tmp/diff.png",
		MimeType:  "image/png",
		ByteSize:  2048,
		Sha256:    "hash456",
		CreatedAt: time.Now().UTC(),
	}

	if err := store.RecordArtifact(ctx, artifact); err != nil {
		t.Fatalf("RecordArtifact failed: %v", err)
	}

	// Create input attachment (no RunID)
	inputArt := domain.Artifact{
		ID:        "art-input-mockup",
		TaskID:    "t1",
		Role:      domain.ArtifactRoleInput,
		Name:      "mockup-design.png",
		Type:      domain.ArtifactTypeImage,
		FilePath:  "/tmp/mockup-design.png",
		MimeType:  "image/png",
		ByteSize:  1024,
		Sha256:    "hash123",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.RecordArtifact(ctx, inputArt); err != nil {
		t.Fatalf("RecordArtifact input failed: %v", err)
	}

	// Retrieve artifact by ID
	retrieved, err := store.GetArtifact(ctx, "art-123")
	if err != nil {
		t.Fatalf("GetArtifact failed: %v", err)
	}
	if retrieved.ID != "art-123" || retrieved.Type != domain.ArtifactTypeImage || retrieved.ByteSize != 2048 || retrieved.Role != domain.ArtifactRoleOutput || retrieved.Name != "diff.png" {
		t.Errorf("Artifact mismatch: %+v", retrieved)
	}

	// Test GetArtifactsByTask (all)
	allArts, err := store.GetArtifactsByTask(ctx, "t1", "")
	if err != nil || len(allArts) != 2 {
		t.Errorf("Expected 2 artifacts for task t1, got %d (err: %v)", len(allArts), err)
	}

	// Test GetArtifactsByTask (INPUT)
	inputArts, err := store.GetArtifactsByTask(ctx, "t1", "INPUT")
	if err != nil || len(inputArts) != 1 || inputArts[0].ID != "art-input-mockup" {
		t.Errorf("Expected 1 input artifact, got %+v", inputArts)
	}

	// Test GetArtifactsByTask (OUTPUT)
	outputArts, err := store.GetArtifactsByTask(ctx, "t1", "OUTPUT")
	if err != nil || len(outputArts) != 1 || outputArts[0].ID != "art-123" {
		t.Errorf("Expected 1 output artifact, got %+v", outputArts)
	}

	// Test non-existent artifact
	_, err = store.GetArtifact(ctx, "non-existent")
	if err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}
