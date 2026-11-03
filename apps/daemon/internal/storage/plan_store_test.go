package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func TestPlanStore_AtomicityAndDAGUnlock(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_plan.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Ingest Plan with 3 sequential DAG tasks
	plan := domain.Plan{
		Slug:     "zenfd-test-01",
		Title:    "ZenForge Test Plan",
		RepoPath: "/home/user/src/repo",
		Status:   "ACTIVE",
	}

	tasks := []domain.Task{
		{
			ID:           "task-01",
			PlanID:       plan.Slug,
			RepoPath:     plan.RepoPath,
			Title:        "Task 1 (No deps)",
			Dependencies: []string{},
			Status:       domain.StatusReady,
		},
		{
			ID:           "task-02",
			PlanID:       plan.Slug,
			RepoPath:     plan.RepoPath,
			Title:        "Task 2 (Depends on 1)",
			Dependencies: []string{"task-01"},
			Status:       domain.StatusPending,
		},
		{
			ID:           "task-03",
			PlanID:       plan.Slug,
			RepoPath:     plan.RepoPath,
			Title:        "Task 3 (Depends on 2)",
			Dependencies: []string{"task-02"},
			Status:       domain.StatusPending,
		},
	}

	if err := store.SavePlanWithTasks(ctx, plan, tasks); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	// 2. Verify Plan retrieval
	retrievedPlan, err := store.GetPlan(ctx, plan.Slug)
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	if retrievedPlan.Slug != plan.Slug || retrievedPlan.Title != plan.Title {
		t.Errorf("Plan mismatch: got %+v, want %+v", retrievedPlan, plan)
	}

	// 3. Verify Initial Tasks
	retrievedTasks, err := store.GetTasksByPlan(ctx, plan.Slug)
	if err != nil || len(retrievedTasks) != 3 {
		t.Fatalf("GetTasksByPlan failed: err=%v, count=%d", err, len(retrievedTasks))
	}

	readyTasks, err := store.GetReadyTasks(ctx, plan.RepoPath, 10)
	if err != nil || len(readyTasks) != 1 || readyTasks[0].ID != "task-01" {
		t.Fatalf("Expected 1 ready task (task-01), got: %v (err: %v)", readyTasks, err)
	}

	// 4. Step 1: Transition task-01 to RUNNING, then to COMPLETED
	if err := store.UpdateTaskStatus(ctx, "task-01", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("Failed transition ready->running: %v", err)
	}
	if err := store.UpdateTaskStatus(ctx, "task-01", domain.StatusRunning, domain.StatusCompleted); err != nil {
		t.Fatalf("Failed transition running->completed: %v", err)
	}

	// 5. Verify DAG Unlock: task-02 should AUTOMATICALLY be unlocked to READY!
	retrievedTasks, _ = store.GetTasksByPlan(ctx, plan.Slug)
	var task2Status, task3Status domain.TaskStatus
	for _, tsk := range retrievedTasks {
		if tsk.ID == "task-02" {
			task2Status = tsk.Status
		}
		if tsk.ID == "task-03" {
			task3Status = tsk.Status
		}
	}

	if task2Status != domain.StatusReady {
		t.Errorf("Expected task-02 to be automatically unlocked to READY, got: %s", task2Status)
	}
	if task3Status != domain.StatusPending {
		t.Errorf("Expected task-03 to remain PENDING, got: %s", task3Status)
	}

	// 6. Step 2: Transition task-02 to COMPLETED -> task-03 unlocks!
	_ = store.UpdateTaskStatus(ctx, "task-02", domain.StatusReady, domain.StatusRunning)
	_ = store.UpdateTaskStatus(ctx, "task-02", domain.StatusRunning, domain.StatusCompleted)

	retrievedTasks, _ = store.GetTasksByPlan(ctx, plan.Slug)
	for _, tsk := range retrievedTasks {
		if tsk.ID == "task-03" {
			task3Status = tsk.Status
		}
	}
	if task3Status != domain.StatusReady {
		t.Errorf("Expected task-03 to be automatically unlocked to READY, got: %s", task3Status)
	}
}

func TestPlanStore_FastFailBranch(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_fastfail.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ingest:
	// Branch 1: task-A -> task-B (dependent)
	// Branch 2: task-C (independent parallel branch)
	plan := domain.Plan{
		Slug:     "fastfail-plan",
		Title:    "Fast Fail Branch Plan",
		RepoPath: "/repo",
		Status:   "ACTIVE",
	}

	tasks := []domain.Task{
		{ID: "task-A", PlanID: plan.Slug, RepoPath: plan.RepoPath, Dependencies: []string{}, Status: domain.StatusReady},
		{ID: "task-B", PlanID: plan.Slug, RepoPath: plan.RepoPath, Dependencies: []string{"task-A"}, Status: domain.StatusPending},
		{ID: "task-C", PlanID: plan.Slug, RepoPath: plan.RepoPath, Dependencies: []string{}, Status: domain.StatusReady},
	}

	if err := store.SavePlanWithTasks(ctx, plan, tasks); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	// Fail task-A: READY -> RUNNING -> FAILED
	_ = store.UpdateTaskStatus(ctx, "task-A", domain.StatusReady, domain.StatusRunning)
	if err := store.UpdateTaskStatus(ctx, "task-A", domain.StatusRunning, domain.StatusFailed); err != nil {
		t.Fatalf("Failed to transition task-A to FAILED: %v", err)
	}

	// Verify: task-B is now BLOCKED, but task-C is STILL READY!
	allTasks, _ := store.GetTasksByPlan(ctx, plan.Slug)
	var statusB, statusC domain.TaskStatus
	for _, tsk := range allTasks {
		if tsk.ID == "task-B" {
			statusB = tsk.Status
		}
		if tsk.ID == "task-C" {
			statusC = tsk.Status
		}
	}

	if statusB != domain.StatusBlocked {
		t.Errorf("Expected downstream task-B to be BLOCKED, got: %s", statusB)
	}
	if statusC != domain.StatusReady {
		t.Errorf("Expected orthogonal task-C to remain READY, got: %s", statusC)
	}
}

func TestPlanStore_ClaimAndReviewMerge(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_claim.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	plan := domain.Plan{
		Slug:     "claim-plan",
		Title:    "Claim & Review Plan",
		RepoPath: tempDir,
		Status:   "ACTIVE",
	}

	tasks := []domain.Task{
		{
			ID:                   "task-claim-01",
			PlanID:               plan.Slug,
			RepoPath:             plan.RepoPath,
			Title:                "Test Slice Implementation",
			Description:          "Implement atomic review and merge gate",
			Priority:             domain.PriorityP1,
			Complexity:           domain.ComplexityMedium,
			RiskLevel:            domain.RiskLow,
			RequiredSkills:       []string{"tdd", "recon"},
			TargetFiles:          []string{"internal/storage/plan_store.go"},
			AcceptanceCriteria:   []string{"Review state transitions properly", "Merge gate completes"},
			VerificationCommands: []string{"go test -v ./..."},
			Dependencies:         []string{},
			Status:               domain.StatusReady,
		},
	}

	if err := store.SavePlanWithTasks(ctx, plan, tasks); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	// 1. Claim task
	packet, err := store.ClaimTask(ctx, "task-claim-01", "subagent-worker-1", 12345)
	if err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}
	if packet == nil || packet.Task.Status != domain.StatusRunning {
		t.Fatalf("Expected claimed task to be RUNNING, got: %+v", packet)
	}
	if packet.Task.ClaimedBy != "subagent-worker-1" {
		t.Errorf("Expected ClaimedBy subagent-worker-1, got: %s", packet.Task.ClaimedBy)
	}

	// Double claim should fail (already RUNNING)
	_, err = store.ClaimTask(ctx, "task-claim-01", "subagent-worker-2", 67890)
	if err == nil {
		t.Fatal("Expected second claim to fail, but it succeeded")
	}

	// 2. Request Review (architectural drift detected)
	err = store.RequestReview(ctx, "task-claim-01", "Scope Drift", "Need to clarify SQLite schema", []string{"Option A: Table Migration", "Option B: Rebuild DB"})
	if err != nil {
		t.Fatalf("RequestReview failed: %v", err)
	}
	t1, _ := store.GetTask(ctx, "task-claim-01")
	if t1.Status != domain.StatusAwaitingReview {
		t.Fatalf("Expected status AWAITING_REVIEW, got: %s", t1.Status)
	}
	if len(t1.ReviewOptions) != 2 || t1.ReviewReason != "Scope Drift" {
		t.Errorf("Unexpected review details: %+v", t1)
	}

	// 3. Resolve Review
	err = store.ResolveReview(ctx, "task-claim-01", "Option A: Table Migration selected")
	if err != nil {
		t.Fatalf("ResolveReview failed: %v", err)
	}
	t2, _ := store.GetTask(ctx, "task-claim-01")
	if t2.Status != domain.StatusRunning {
		t.Fatalf("Expected status resumed to RUNNING, got: %s", t2.Status)
	}
	if t2.ReviewResponse != "Option A: Table Migration selected" {
		t.Errorf("Unexpected review response: %s", t2.ReviewResponse)
	}

	// 4. Stage Merge (worktree complete)
	err = store.StageMerge(ctx, "task-claim-01")
	if err != nil {
		t.Fatalf("StageMerge failed: %v", err)
	}
	t3, _ := store.GetTask(ctx, "task-claim-01")
	if t3.Status != domain.StatusAwaitingMerge {
		t.Fatalf("Expected status AWAITING_MERGE, got: %s", t3.Status)
	}

	// 5. Merge Task (no worktree path specified -> directly COMPLETED)
	merged, conflicts, err := store.MergeTask(ctx, "task-claim-01")
	if err != nil || !merged || len(conflicts) > 0 {
		t.Fatalf("MergeTask failed: merged=%v, conflicts=%v, err=%v", merged, conflicts, err)
	}
	t4, _ := store.GetTask(ctx, "task-claim-01")
	if t4.Status != domain.StatusCompleted {
		t.Fatalf("Expected status COMPLETED, got: %s", t4.Status)
	}
}

func TestPlanStore_ProjectDocs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_docs.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repoPath := "/workspace/test"

	doc1 := domain.ProjectDoc{
		Slug:            "architecture-overview",
		RepoPath:        repoPath,
		GitBranch:       "main",
		Title:           "Architecture Overview",
		Category:        "ARCHITECTURE",
		ContentMarkdown: "# Architecture Overview\nSystem topology and daemon specs.",
		MermaidDiagrams: []string{"graph TD; A-->B;"},
	}

	doc2 := domain.ProjectDoc{
		Slug:            "memory-scratchpad",
		RepoPath:        repoPath,
		GitBranch:       "main",
		Title:           "Project Memory",
		Category:        "MEMORY",
		ContentMarkdown: "Active memory of decisions and design principles.",
	}

	if err := store.UpsertDoc(ctx, doc1); err != nil {
		t.Fatalf("UpsertDoc 1 failed: %v", err)
	}
	if err := store.UpsertDoc(ctx, doc2); err != nil {
		t.Fatalf("UpsertDoc 2 failed: %v", err)
	}

	// Verify GetDoc
	retrieved, err := store.GetDoc(ctx, repoPath, "architecture-overview")
	if err != nil {
		t.Fatalf("GetDoc failed: %v", err)
	}
	if retrieved.Title != doc1.Title || len(retrieved.MermaidDiagrams) != 1 {
		t.Errorf("Mismatch in retrieved doc: %+v", retrieved)
	}

	// Update Doc
	doc1.Title = "Architecture Overview (v2)"
	if err := store.UpsertDoc(ctx, doc1); err != nil {
		t.Fatalf("UpsertDoc update failed: %v", err)
	}
	updated, _ := store.GetDoc(ctx, repoPath, "architecture-overview")
	if updated.Title != "Architecture Overview (v2)" || updated.Version != 2 {
		t.Errorf("Expected updated title and version=2, got: title=%s version=%d", updated.Title, updated.Version)
	}

	// Verify ListDocs
	allDocs, err := store.ListDocs(ctx, repoPath, "")
	if err != nil || len(allDocs) != 2 {
		t.Fatalf("ListDocs all failed: len=%d, err=%v", len(allDocs), err)
	}

	archDocs, err := store.ListDocs(ctx, repoPath, "ARCHITECTURE")
	if err != nil || len(archDocs) != 1 || archDocs[0].Slug != "architecture-overview" {
		t.Fatalf("ListDocs filtered failed: count=%d, err=%v", len(archDocs), err)
	}
}
