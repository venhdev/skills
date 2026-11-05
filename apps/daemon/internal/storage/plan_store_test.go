package storage_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
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
		Slug:   "zenfd-test-01",
		Title:  "ZenForge Test Plan",
		Status: "ACTIVE",
	}

	tasks := []domain.Task{
		{
			ID:           "task-01",
			PlanID:       plan.Slug,
			Title:        "Task 1 (No deps)",
			Dependencies: []string{},
			Status:       domain.StatusReady,
		},
		{
			ID:           "task-02",
			PlanID:       plan.Slug,
			Title:        "Task 2 (Depends on 1)",
			Dependencies: []string{"task-01"},
			Status:       domain.StatusPending,
		},
		{
			ID:           "task-03",
			PlanID:       plan.Slug,
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

	readyTasks, err := store.ListTasks(ctx, domain.StatusReady)
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
		Slug:   "fastfail-plan",
		Title:  "Fast Fail Branch Plan",
		Status: "ACTIVE",
	}

	tasks := []domain.Task{
		{ID: "task-A", PlanID: plan.Slug, Dependencies: []string{}, Status: domain.StatusReady},
		{ID: "task-B", PlanID: plan.Slug, Dependencies: []string{"task-A"}, Status: domain.StatusPending},
		{ID: "task-C", PlanID: plan.Slug, Dependencies: []string{}, Status: domain.StatusReady},
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
		Slug:   "claim-plan",
		Title:  "Claim & Review Plan",
		Status: "ACTIVE",
	}

	tasks := []domain.Task{
		{
			ID:                   "task-claim-01",
			PlanID:               plan.Slug,
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
	packet, err := store.ClaimTask(ctx, "task-claim-01", "subagent-worker-1")
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
	_, err = store.ClaimTask(ctx, "task-claim-01", "subagent-worker-2")
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

	// 3. Resolve Review: transitions to READY for clean agent re-claim
	err = store.ResolveReview(ctx, "task-claim-01", "Option A: Table Migration selected")
	if err != nil {
		t.Fatalf("ResolveReview failed: %v", err)
	}
	t2, _ := store.GetTask(ctx, "task-claim-01")
	if t2.Status != domain.StatusReady {
		t.Fatalf("Expected status resumed to READY, got: %s", t2.Status)
	}
	if t2.ReviewResponse != "Option A: Table Migration selected" {
		t.Errorf("Unexpected review response: %s", t2.ReviewResponse)
	}

	// Re-claim after review resolution to put back into RUNNING
	_, err = store.ClaimTask(ctx, "task-claim-01", "subagent-coder-2")
	if err != nil {
		t.Fatalf("Re-claiming task failed: %v", err)
	}

	// 4. Stage merge (worktree complete)
	err = store.UpdateTaskStatus(ctx, "task-claim-01", domain.StatusRunning, domain.StatusAwaitingMerge)
	if err != nil {
		t.Fatalf("UpdateTaskStatus to AWAITING_MERGE failed: %v", err)
	}
	t3, _ := store.GetTask(ctx, "task-claim-01")
	if t3.Status != domain.StatusAwaitingMerge {
		t.Fatalf("Expected status AWAITING_MERGE, got: %s", t3.Status)
	}

	// 5. Resolve merge outcome outside MCP -> COMPLETED
	err = store.UpdateTaskStatus(ctx, "task-claim-01", domain.StatusAwaitingMerge, domain.StatusCompleted)
	if err != nil {
		t.Fatalf("UpdateTaskStatus to COMPLETED failed: %v", err)
	}
	t4, _ := store.GetTask(ctx, "task-claim-01")
	if t4.Status != domain.StatusCompleted {
		t.Fatalf("Expected status COMPLETED, got: %s", t4.Status)
	}
}

func TestPlanStore_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_v24_crud.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx := context.Background()

	// 1. Create Plan via CreatePlan
	plan := domain.Plan{
		Slug:   "plan-v24",
		Title:  "V2.4 Living Plan",
		Status: "ACTIVE",
	}
	if err := store.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	// 2. Create Task via CreateTask
	task := domain.Task{
		ID:           "P24-01",
		PlanID:       "plan-v24",
		Title:        "First Slice",
		Status:       domain.StatusReady,
		Dependencies: []string{},
	}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 3. Update dependencies and notes
	if err := store.UpdateTaskDependencies(ctx, "P24-01", []string{"P24-00"}); err != nil {
		t.Fatalf("UpdateTaskDependencies failed: %v", err)
	}
	if err := store.UpdateTaskNotes(ctx, "P24-01", "Supplemental notes from developer"); err != nil {
		t.Fatalf("UpdateTaskNotes failed: %v", err)
	}
	tUpdated, _ := store.GetTask(ctx, "P24-01")
	if len(tUpdated.Dependencies) != 1 || tUpdated.Notes != "Supplemental notes from developer" {
		t.Errorf("Unexpected task update: %+v", tUpdated)
	}

	// 5. DeletePlan (cascading)
	if err := store.DeletePlan(ctx, "plan-v24"); err != nil {
		t.Fatalf("DeletePlan failed: %v", err)
	}
	_, err = store.GetPlan(ctx, "plan-v24")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
	tasks, _ := store.GetTasksByPlan(ctx, "plan-v24")
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks after cascading plan delete, got: %d", len(tasks))
	}
}

func TestPlanStore_ListPlansIncludesEmptyPlan(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_list_plans.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx := context.Background()

	// A plan with a task and a plan without any tasks.
	if err := store.CreatePlan(ctx, domain.Plan{Slug: "plan-with-tasks", Title: "With tasks", Status: "ACTIVE"}); err != nil {
		t.Fatalf("CreatePlan (with tasks) failed: %v", err)
	}
	if err := store.CreatePlan(ctx, domain.Plan{Slug: "plan-empty", Title: "Empty", Status: "ACTIVE"}); err != nil {
		t.Fatalf("CreatePlan (empty) failed: %v", err)
	}
	if err := store.CreateTask(ctx, domain.Task{
		ID:           "LT-01",
		PlanID:       "plan-with-tasks",
		Title:        "First slice",
		Status:       domain.StatusReady,
		Dependencies: []string{},
	}); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	plans, err := store.ListPlans(ctx, false)
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}

	got := make(map[string]bool, len(plans))
	for _, p := range plans {
		got[p.Slug] = true
	}
	if !got["plan-with-tasks"] || !got["plan-empty"] {
		t.Fatalf("expected ListPlans to include empty plan, got: %+v", plans)
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_concurrent_read_write.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Seed initial data: plan and task
	plan := domain.Plan{
		Slug:   "plan-concurrent",
		Title:  "Concurrent Plan",
		Status: "ACTIVE",
	}
	if err := store.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	seedTask := domain.Task{
		ID:     "concurrent-task",
		PlanID: plan.Slug,
		Title:  "Concurrent Task",
		Status: domain.StatusReady,
	}
	if err := store.CreateTask(ctx, seedTask); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 1. Proof of non-blocking: While a writer transaction is open and holding writeMu,
	// concurrent readers query GetTask, GetTasksByPlan, and GetPlan.
	writerHolding := make(chan struct{})
	readerDone := make(chan struct{})
	writerDone := make(chan error, 1)

	go func() {
		writerDone <- db.ExecImmediate(ctx, func(tx *sql.Tx) error {
			// Write something in the transaction
			_, err := tx.ExecContext(ctx, "UPDATE plans SET title = 'Concurrent In-Progress' WHERE slug = ?", plan.Slug)
			if err != nil {
				return err
			}

			// Signal writer has transaction open and writeMu acquired
			close(writerHolding)

			// Wait for reader to perform its queries concurrently
			select {
			case <-readerDone:
				return nil
			case <-time.After(3 * time.Second):
				return errors.New("timed out waiting for readers - likely deadlock on writeMu")
			}
		})
	}()

	// Wait for writer to enter transaction and acquire writeMu
	select {
	case <-writerHolding:
	case <-time.After(2 * time.Second):
		t.Fatal("writer did not enter transaction in time")
	}

	// Now writer holds writeMu and has an open write transaction.
	// Concurrent readers MUST succeed without deadlock.
	retrievedTask, err := store.GetTask(ctx, "concurrent-task")
	if err != nil {
		t.Fatalf("GetTask failed while writer active: %v", err)
	}
	if retrievedTask.ID != "concurrent-task" {
		t.Errorf("expected concurrent-task, got: %s", retrievedTask.ID)
	}

	planTasks, err := store.GetTasksByPlan(ctx, plan.Slug)
	if err != nil {
		t.Fatalf("GetTasksByPlan failed while writer active: %v", err)
	}
	if len(planTasks) != 1 {
		t.Errorf("expected 1 task, got: %d", len(planTasks))
	}

	retrievedPlan, err := store.GetPlan(ctx, plan.Slug)
	if err != nil {
		t.Fatalf("GetPlan failed while writer active: %v", err)
	}
	if retrievedPlan.Slug != plan.Slug {
		t.Errorf("unexpected plan: %v", retrievedPlan)
	}

	// Signal writer that reader is done
	close(readerDone)

	// Wait for writer goroutine to complete
	select {
	case err := <-writerDone:
		if err != nil {
			t.Fatalf("writer failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("writer did not complete in time")
	}

	// 2. High-concurrency stress test with multiple goroutines
	const numWriters = 3
	const numReaders = 8
	const iterations = 20

	var wg sync.WaitGroup
	errCh := make(chan error, numWriters+numReaders)

	// Writers
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				task := domain.Task{
					ID:     fmt.Sprintf("stress-task-%d-%d", workerID, i),
					PlanID: plan.Slug,
					Title:  fmt.Sprintf("Stress Task %d %d", workerID, i),
					Status: domain.StatusReady,
				}
				if err := store.CreateTask(ctx, task); err != nil {
					errCh <- fmt.Errorf("writer %d CreateTask failed: %w", workerID, err)
					return
				}
			}
		}(w)
	}

	// Readers
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if _, err := store.GetTask(ctx, "concurrent-task"); err != nil {
					errCh <- fmt.Errorf("reader %d GetTask failed: %w", readerID, err)
					return
				}
				if _, err := store.GetTasksByPlan(ctx, plan.Slug); err != nil {
					errCh <- fmt.Errorf("reader %d GetTasksByPlan failed: %w", readerID, err)
					return
				}
				if _, err := store.GetPlan(ctx, plan.Slug); err != nil {
					errCh <- fmt.Errorf("reader %d GetPlan failed: %w", readerID, err)
					return
				}
			}
		}(r)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent test error: %v", err)
	}
}
