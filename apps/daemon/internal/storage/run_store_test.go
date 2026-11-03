package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func TestRunStore_Lifecycle(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_run.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Seed plan and task first for foreign key
	plan := domain.Plan{Slug: "p1", Title: "P1", RepoPath: "/repo", Status: "ACTIVE"}
	task := domain.Task{ID: "t1", PlanID: "p1", RepoPath: "/repo", Title: "T1", Status: domain.StatusReady}
	_ = store.SavePlanWithTasks(ctx, plan, []domain.Task{task})

	// 1. Create Run
	run := domain.Run{
		ID:        "run-001",
		TaskID:    "t1",
		RepoPath:  "/repo",
		PID:       1234,
		PGID:      1234,
		Status:    domain.StatusRunning,
		LogPath:   "/tmp/run-001.log",
		ExitCode:  -1,
		StartedAt: time.Now().UTC(),
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	// 2. Query Active Runs
	activeRuns, err := store.GetActiveRuns(ctx)
	if err != nil || len(activeRuns) != 1 {
		t.Fatalf("GetActiveRuns failed: err=%v, count=%d", err, len(activeRuns))
	}
	if activeRuns[0].ID != "run-001" || activeRuns[0].PID != 1234 {
		t.Errorf("Unexpected active run: %+v", activeRuns[0])
	}

	// 3. Update Heartbeat
	newBeat := time.Now().UTC().Add(10 * time.Second)
	if err := store.UpdateRunHeartbeat(ctx, "run-001", newBeat); err != nil {
		t.Fatalf("UpdateRunHeartbeat failed: %v", err)
	}

	// 4. Finish Run
	if err := store.FinishRun(ctx, "run-001", domain.StatusCompleted, 0); err != nil {
		t.Fatalf("FinishRun failed: %v", err)
	}

	// Verify no more active runs
	activeRunsAfter, err := store.GetActiveRuns(ctx)
	if err != nil || len(activeRunsAfter) != 0 {
		t.Errorf("Expected 0 active runs, got %d", len(activeRunsAfter))
	}
}

func TestRunStore_ArtifactClaimCheck(t *testing.T) {
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
	plan := domain.Plan{Slug: "p1", Title: "P1", RepoPath: "/repo", Status: "ACTIVE"}
	task := domain.Task{ID: "t1", PlanID: "p1", RepoPath: "/repo", Title: "T1", Status: domain.StatusReady}
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
