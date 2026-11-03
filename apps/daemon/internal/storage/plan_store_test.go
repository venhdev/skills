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
	"zenfd/internal/domain/prompt"
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
	_, err = store.ClaimTask(ctx, "task-claim-01", "subagent-coder-2", 48292)
	if err != nil {
		t.Fatalf("Re-claiming task failed: %v", err)
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

type mockWorktreeManager struct {
	merged    bool
	conflicts []string
	err       error
	called    bool
	lastRepo  string
	lastWt    string
}

func (m *mockWorktreeManager) MergeWorktree(ctx context.Context, repoRoot, worktreePath string) (bool, []string, error) {
	m.called = true
	m.lastRepo = repoRoot
	m.lastWt = worktreePath
	return m.merged, m.conflicts, m.err
}

func (m *mockWorktreeManager) ResolveBranchName(ctx context.Context, worktreePath string) (string, error) {
	return "feature-mock", nil
}

func TestPlanStore_MergeTask_WithWorktreeManager(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_wt_merge.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	plan := domain.Plan{
		Slug:  "wt-merge-plan",
		Title: "Worktree Merge Plan",
	}
	tasks := []domain.Task{
		{
			ID:           "task-wt-conflict",
			PlanID:       plan.Slug,
			RepoPath:     "/repo/root",
			WorktreePath: "/repo/worktree-1",
			Title:        "Conflict Task",
			Status:       domain.StatusRunning,
		},
		{
			ID:           "task-wt-clean",
			PlanID:       plan.Slug,
			RepoPath:     "/repo/root",
			WorktreePath: "/repo/worktree-2",
			Title:        "Clean Task",
			Status:       domain.StatusAwaitingMerge,
		},
	}
	if err := store.SavePlanWithTasks(ctx, plan, tasks); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	// 1. Conflict test: mock returns merged=false with conflicts
	mockMgr := &mockWorktreeManager{
		merged:    false,
		conflicts: []string{"file_a.go", "file_b.go"},
	}
	store.SetWorktreeManager(mockMgr)

	merged, conflicts, err := store.MergeTask(ctx, "task-wt-conflict")
	if err != nil {
		t.Fatalf("unexpected error on conflict: %v", err)
	}
	if merged {
		t.Fatal("expected merged == false on conflict")
	}
	if !mockMgr.called || mockMgr.lastRepo != "/repo/root" || mockMgr.lastWt != "/repo/worktree-1" {
		t.Fatalf("mock manager not invoked properly: %+v", mockMgr)
	}
	if len(conflicts) != 2 || conflicts[0] != "file_a.go" {
		t.Fatalf("unexpected conflicts returned: %v", conflicts)
	}

	// Verify task in DB is CONFLICT and has conflict_files saved
	tConflict, err := store.GetTask(ctx, "task-wt-conflict")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if tConflict.Status != domain.StatusConflict {
		t.Fatalf("expected status CONFLICT, got: %s", tConflict.Status)
	}
	if len(tConflict.ConflictFiles) != 2 || tConflict.ConflictFiles[0] != "file_a.go" {
		t.Fatalf("unexpected conflict files stored: %v", tConflict.ConflictFiles)
	}

	// 2. Clean merge test: mock returns merged=true
	mockMgrClean := &mockWorktreeManager{
		merged: true,
	}
	store.SetWorktreeManager(mockMgrClean)

	mergedClean, conflictsClean, err := store.MergeTask(ctx, "task-wt-clean")
	if err != nil {
		t.Fatalf("unexpected error on clean merge: %v", err)
	}
	if !mergedClean || len(conflictsClean) > 0 {
		t.Fatalf("expected clean merge, got merged=%v, conflicts=%v", mergedClean, conflictsClean)
	}
	if !mockMgrClean.called || mockMgrClean.lastRepo != "/repo/root" || mockMgrClean.lastWt != "/repo/worktree-2" {
		t.Fatalf("mock clean manager not invoked properly: %+v", mockMgrClean)
	}

	// Verify task in DB is COMPLETED
	tClean, err := store.GetTask(ctx, "task-wt-clean")
	if err != nil {
		t.Fatalf("failed to get clean task: %v", err)
	}
	if tClean.Status != domain.StatusCompleted {
		t.Fatalf("expected status COMPLETED, got: %s", tClean.Status)
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

func TestPlanStore_V24_ProjectsAndCRUD(t *testing.T) {
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
		Slug:     "plan-v24",
		Title:    "V2.4 Living Plan",
		RepoPath: "/test/v24-repo",
		Status:   "ACTIVE",
	}
	if err := store.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	// 2. Create Task via CreateTask
	task := domain.Task{
		ID:           "P24-01",
		PlanID:       "plan-v24",
		RepoPath:     "/test/v24-repo",
		Title:        "First Slice",
		Status:       domain.StatusReady,
		Dependencies: []string{},
	}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 3. Verify ListProjects
	projects, err := store.ListProjects(ctx)
	if err != nil || len(projects) != 1 || projects[0] != "/test/v24-repo" {
		t.Fatalf("Expected project /test/v24-repo, got: %v (err=%v)", projects, err)
	}

	// 4. Update dependencies and notes
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

func TestPlanStore_V24_ReclaimAndEvict(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_v24_reclaim.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx := context.Background()

	plan := domain.Plan{Slug: "plan-reclaim", Title: "Reclaim Plan", RepoPath: "/test/reclaim"}
	_ = store.CreatePlan(ctx, plan)

	task := domain.Task{
		ID:       "task-rec-01",
		PlanID:   "plan-reclaim",
		RepoPath: "/test/reclaim",
		Title:    "Reclaimable Task",
		Status:   domain.StatusReady,
	}
	_ = store.CreateTask(ctx, task)

	// Claim with metadata
	packet, err := store.ClaimTaskWithMeta(ctx, "task-rec-01", "worker-1", "orch-1", "CODER", 1234, 1000)
	if err != nil || packet == nil {
		t.Fatalf("ClaimTaskWithMeta failed: %v", err)
	}

	tClaimed, _ := store.GetTask(ctx, "task-rec-01")
	if tClaimed.Status != domain.StatusRunning || tClaimed.OrchestratorID != "orch-1" || tClaimed.PID != 1234 {
		t.Errorf("Unexpected claimed task meta: %+v", tClaimed)
	}

	// Evict task back to READY
	if err := store.EvictTask(ctx, "task-rec-01", "Subagent hung", "RESET_TO_READY"); err != nil {
		t.Fatalf("EvictTask failed: %v", err)
	}
	tEvicted, _ := store.GetTask(ctx, "task-rec-01")
	if tEvicted.Status != domain.StatusReady || tEvicted.ClaimedBy != "" {
		t.Errorf("Expected task back to READY and unassigned, got: %+v", tEvicted)
	}

	// Re-claim with new worker
	packet2, err := store.ReclaimTask(ctx, "task-rec-01", "worker-2", "orch-2", 5678)
	if err != nil || packet2 == nil {
		t.Fatalf("ReclaimTask failed: %v", err)
	}
	tReclaimed, _ := store.GetTask(ctx, "task-rec-01")
	if tReclaimed.Status != domain.StatusRunning || tReclaimed.ClaimedBy != "worker-2" || tReclaimed.OrchestratorID != "orch-2" {
		t.Errorf("Unexpected reclaimed task meta: %+v", tReclaimed)
	}
}

func TestPlanStore_V24_AgentSessions(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_v24_sessions.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx := context.Background()

	sess1 := domain.AgentSession{
		SessionID: "orch-root-01",
		Role:      "ORCHESTRATOR",
		Name:      "Lead Architect",
		PID:       48100,
	}
	sess2 := domain.AgentSession{
		SessionID:       "worker-sub-01",
		ParentSessionID: "orch-root-01",
		Role:            "SUBAGENT",
		Name:            "Coder 1",
		PID:             48105,
		ClaimedTasks:    []string{"P-01"},
	}

	if err := store.RegisterSession(ctx, sess1); err != nil {
		t.Fatalf("RegisterSession 1 failed: %v", err)
	}
	if err := store.RegisterSession(ctx, sess2); err != nil {
		t.Fatalf("RegisterSession 2 failed: %v", err)
	}

	sessions, err := store.ListSessions(ctx)
	if err != nil || len(sessions) != 2 {
		t.Fatalf("ListSessions failed: len=%d, err=%v", len(sessions), err)
	}

	if err := store.HeartbeatSession(ctx, "orch-root-01"); err != nil {
		t.Fatalf("HeartbeatSession failed: %v", err)
	}
}

func TestPlanStore_TechSpecs_CRUD_FTS5(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_specs_fts.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx := context.Background()

	// 1. Upsert Spec
	spec1 := domain.TechSpec{
		Slug:              "spec-auth-flow",
		RepoPath:          "/home/user/repo",
		Tier:              2, // Dynamics
		Kind:              "SEQUENCE",
		Title:             "Authentication Flow Sequence",
		DiagramType:       "mermaid:sequenceDiagram",
		DiagramCode:       "sequenceDiagram\nClient->>Server: POST /login\nServer-->>Client: 200 OK JWT",
		StructuredPayload: map[string]any{"timeout_ms": 5000, "algorithm": "RS256"},
		Notes:             "Governs auth subsystem",
		BoundPaths:        []string{"internal/auth/", "cmd/login"},
	}

	saved, err := store.UpsertSpec(ctx, spec1)
	if err != nil {
		t.Fatalf("UpsertSpec failed: %v", err)
	}
	if saved.Version != 1 {
		t.Fatalf("expected version 1, got %d", saved.Version)
	}

	// 2. Update Spec -> in-place version bump
	spec1.Notes = "Updated auth subsystem governance with refresh tokens"
	spec1.StructuredPayload["refresh_enabled"] = true
	updated, err := store.UpsertSpec(ctx, spec1)
	if err != nil {
		t.Fatalf("UpsertSpec update failed: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}

	// 3. FTS5 Search
	results, err := store.SearchSpecs(ctx, "refresh")
	if err != nil {
		t.Fatalf("SearchSpecs failed: %v", err)
	}
	if len(results) != 1 || results[0].Slug != "spec-auth-flow" {
		t.Fatalf("expected 1 result matching 'refresh', got %d", len(results))
	}

	// 4. AutoLink by target files
	targetFiles := []string{"internal/auth/handler.go", "pkg/utils.go"}
	linked, err := store.AutoLinkSpecs(ctx, targetFiles)
	if err != nil {
		t.Fatalf("AutoLinkSpecs failed: %v", err)
	}
	if len(linked) != 1 || linked[0] != "spec-auth-flow" {
		t.Fatalf("expected auto-linked spec 'spec-auth-flow', got %v", linked)
	}

	// 5. Zero-Round-Trip claim test
	plan := domain.Plan{Slug: "plan-auth", Title: "Auth Plan", RepoPath: "/home/user/repo", Status: "ACTIVE"}
	task := domain.Task{
		ID:          "AUTH-01",
		PlanID:      plan.Slug,
		RepoPath:    plan.RepoPath,
		Title:       "Implement Auth Handler",
		TargetFiles: targetFiles,
		Status:      domain.StatusReady,
	}
	_ = store.CreatePlan(ctx, plan)
	_ = store.CreateTask(ctx, task)

	packet, err := store.ClaimTask(ctx, "AUTH-01", "worker-1", 1234)
	if err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}
	if len(packet.LinkedSpecs) != 1 {
		t.Fatalf("expected 1 linked spec in ContextPacket, got %d", len(packet.LinkedSpecs))
	}
	if packet.LinkedSpecs[0].Slug != "spec-auth-flow" {
		t.Fatalf("expected spec 'spec-auth-flow', got %s", packet.LinkedSpecs[0].Slug)
	}
	if packet.LinkedSpecs[0].Kind != "SEQUENCE" {
		t.Fatalf("expected kind 'SEQUENCE', got %s", packet.LinkedSpecs[0].Kind)
	}
}

func TestPlanStore_PlanSnapshots_Revisions_Diff(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_snapshots.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx := context.Background()

	plan := domain.Plan{Slug: "plan-diff-test", Title: "Snapshot Test", RepoPath: "/repo", Status: "ACTIVE"}
	if err := store.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	// 1. Save Snapshot v1.0
	v1Files := []domain.SnapshotFile{
		{FilePath: "README.md", Content: "# Plan v1.0\nInitial plan."},
		{FilePath: "01-contracts.md", Content: "type API struct {}"},
	}
	err = store.SavePlanSnapshot(ctx, plan.Slug, "v1.0", "Architect", "Initial baseline", v1Files)
	if err != nil {
		t.Fatalf("SavePlanSnapshot v1.0 failed: %v", err)
	}

	// 2. Save Snapshot v2.0
	v2Files := []domain.SnapshotFile{
		{FilePath: "README.md", Content: "# Plan v2.0\nExpanded scope."},
		{FilePath: "02-ui.md", Content: "UI Layout spec"},
	}
	err = store.SavePlanSnapshot(ctx, plan.Slug, "v2.0", "Architect", "V2 revision", v2Files)
	if err != nil {
		t.Fatalf("SavePlanSnapshot v2.0 failed: %v", err)
	}

	// 3. List Revisions
	revs, err := store.ListPlanRevisions(ctx, plan.Slug)
	if err != nil || len(revs) != 2 {
		t.Fatalf("ListPlanRevisions failed: len=%d, err=%v", len(revs), err)
	}
	if revs[0].Revision != "v1.0" || revs[1].Revision != "v2.0" {
		t.Fatalf("unexpected revisions order: %v", revs)
	}

	// 4. Get Snapshot Files
	loadedV1, err := store.GetPlanSnapshot(ctx, plan.Slug, "v1.0")
	if err != nil || len(loadedV1) != 2 {
		t.Fatalf("GetPlanSnapshot v1.0 failed: len=%d, err=%v", len(loadedV1), err)
	}

	// 5. Get Diff between v1.0 and v2.0
	diffs, err := store.GetPlanDiff(ctx, plan.Slug, "v1.0", "v2.0")
	if err != nil {
		t.Fatalf("GetPlanDiff failed: %v", err)
	}

	// Expect:
	// 01-contracts.md -> DELETED
	// 02-ui.md -> ADDED
	// README.md -> MODIFIED
	diffMap := make(map[string]domain.FileDiff)
	for _, d := range diffs {
		diffMap[d.FilePath] = d
	}

	if d, ok := diffMap["01-contracts.md"]; !ok || d.ChangeType != "DELETED" {
		t.Fatalf("expected 01-contracts.md DELETED, got %v", d)
	}
	if d, ok := diffMap["02-ui.md"]; !ok || d.ChangeType != "ADDED" {
		t.Fatalf("expected 02-ui.md ADDED, got %v", d)
	}
	if d, ok := diffMap["README.md"]; !ok || d.ChangeType != "MODIFIED" {
		t.Fatalf("expected README.md MODIFIED, got %v", d)
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

	// Seed initial data: plan, revision, snapshot, spec
	plan := domain.Plan{
		Slug:     "plan-concurrent",
		Title:    "Concurrent Plan",
		RepoPath: "/workspace/concurrent",
		Status:   "ACTIVE",
	}
	if err := store.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	snapFiles := []domain.SnapshotFile{
		{FilePath: "main.go", Content: "package main"},
	}
	if err := store.SavePlanSnapshot(ctx, plan.Slug, "v1.0", "Author", "Initial", snapFiles); err != nil {
		t.Fatalf("SavePlanSnapshot failed: %v", err)
	}

	spec := domain.TechSpec{
		Slug:              "spec-concurrent",
		RepoPath:          "/workspace/concurrent",
		Tier:              1,
		Kind:              "ARCHITECTURE",
		Title:             "Concurrent Spec",
		DiagramType:       "mermaid:flowchart",
		DiagramCode:       "graph TD; A-->B;",
		StructuredPayload: map[string]any{"key": "value"},
		Notes:             "Testing concurrency",
		BoundPaths:        []string{"main.go"},
	}
	if _, err := store.UpsertSpec(ctx, spec); err != nil {
		t.Fatalf("UpsertSpec failed: %v", err)
	}

	// 1. Proof of non-blocking: While a writer transaction is open and holding writeMu,
	// concurrent readers query GetSpec, ListSpecs, and ListPlanRevisions.
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
	retrievedSpec, err := store.GetSpec(ctx, "spec-concurrent")
	if err != nil {
		t.Fatalf("GetSpec failed while writer active: %v", err)
	}
	if retrievedSpec.Slug != "spec-concurrent" {
		t.Errorf("expected spec-concurrent, got: %s", retrievedSpec.Slug)
	}

	specs, err := store.ListSpecs(ctx, "/workspace/concurrent", 0)
	if err != nil {
		t.Fatalf("ListSpecs failed while writer active: %v", err)
	}
	if len(specs) != 1 {
		t.Errorf("expected 1 spec, got: %d", len(specs))
	}

	revs, err := store.ListPlanRevisions(ctx, plan.Slug)
	if err != nil {
		t.Fatalf("ListPlanRevisions failed while writer active: %v", err)
	}
	if len(revs) != 1 || revs[0].Revision != "v1.0" {
		t.Errorf("unexpected revisions: %v", revs)
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
					ID:       fmt.Sprintf("stress-task-%d-%d", workerID, i),
					PlanID:   plan.Slug,
					RepoPath: plan.RepoPath,
					Title:    fmt.Sprintf("Stress Task %d %d", workerID, i),
					Status:   domain.StatusReady,
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
				if _, err := store.GetSpec(ctx, "spec-concurrent"); err != nil {
					errCh <- fmt.Errorf("reader %d GetSpec failed: %w", readerID, err)
					return
				}
				if _, err := store.ListSpecs(ctx, "", 0); err != nil {
					errCh <- fmt.Errorf("reader %d ListSpecs failed: %w", readerID, err)
					return
				}
				if _, err := store.ListPlanRevisions(ctx, plan.Slug); err != nil {
					errCh <- fmt.Errorf("reader %d ListPlanRevisions failed: %w", readerID, err)
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

var _ prompt.Compiler = (*mockPromptCompiler)(nil)

type mockPromptCompiler struct {
	called           bool
	task             domain.Task
	upstreamDiffs    []string
	inputAttachments []domain.Artifact
	specSnippets     []string
}

func (m *mockPromptCompiler) Compile(ctx context.Context, task domain.Task, upstreamDiffs []string, inputAttachments []domain.Artifact, specSnippets []string) (*domain.ContextPacket, error) {
	m.called = true
	m.task = task
	m.upstreamDiffs = upstreamDiffs
	m.inputAttachments = inputAttachments
	m.specSnippets = specSnippets
	return &domain.ContextPacket{
		Task:           task,
		TargetFiles:    task.TargetFiles,
		PromptMarkdown: fmt.Sprintf("# MOCK PROMPT: %s", task.Title),
	}, nil
}

func TestPlanStore_PromptCompilerDelegation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_compiler_delegation.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mock := &mockPromptCompiler{}
	store.SetPromptCompiler(mock)

	plan := domain.Plan{
		Slug:     "plan-compiler-test",
		Title:    "Compiler Delegation Plan",
		RepoPath: "/test/repo",
		Status:   "ACTIVE",
	}
	task := domain.Task{
		ID:          "T-DELEGATE-01",
		PlanID:      plan.Slug,
		RepoPath:    plan.RepoPath,
		Title:       "Test Compiler Delegation",
		TargetFiles: []string{"a.go"},
		Status:      domain.StatusReady,
	}

	if err := store.SavePlanWithTasks(ctx, plan, []domain.Task{task}); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	packet, err := store.GetTaskContextPacket(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskContextPacket failed: %v", err)
	}

	if !mock.called {
		t.Fatal("expected mock prompt compiler to be called")
	}
	if mock.task.ID != "T-DELEGATE-01" {
		t.Errorf("expected task ID 'T-DELEGATE-01', got: %s", mock.task.ID)
	}
	if packet.PromptMarkdown != "# MOCK PROMPT: Test Compiler Delegation" {
		t.Errorf("unexpected PromptMarkdown: %s", packet.PromptMarkdown)
	}
}

