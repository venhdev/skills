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
