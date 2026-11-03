package scheduler_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/scheduler"
	"zenfd/internal/storage"
	"zenfd/internal/supervisor"
)

func TestScheduler_SubmitAndGetPlan(t *testing.T) {
	tempDir := t.TempDir()
	db, err := storage.Open(filepath.Join(tempDir, "state.db"))
	if err != nil {
		t.Fatalf("Open DB failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	sup := supervisor.NewProcessSupervisor(tempDir)
	cfg := scheduler.DefaultConfig()

	sched := scheduler.NewDAGScheduler(cfg, store, sup)
	ctx := context.Background()

	plan := domain.Plan{Slug: "plan-sched", Title: "Scheduler Test", RepoPath: "/repo", Status: "ACTIVE"}
	tasks := []domain.Task{
		{ID: "t-1", PlanID: plan.Slug, RepoPath: plan.RepoPath, Title: "Task 1", Status: domain.StatusReady},
	}

	if err := sched.SubmitPlan(ctx, plan, tasks); err != nil {
		t.Fatalf("SubmitPlan failed: %v", err)
	}

	p, ts, err := sched.GetPlan(ctx, "plan-sched")
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	if p.Slug != "plan-sched" || len(ts) != 1 {
		t.Errorf("Unexpected plan/tasks: %+v, %+v", p, ts)
	}
}

func TestScheduler_Tick_DispatchSlots(t *testing.T) {
	tempDir := t.TempDir()
	db, err := storage.Open(filepath.Join(tempDir, "state.db"))
	if err != nil {
		t.Fatalf("Open DB failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	sup := supervisor.NewProcessSupervisor(tempDir)
	cfg := scheduler.SchedulerConfig{
		MaxConcurrentWorkers: 2, // Limit to 2 parallel workers
		HeartbeatTimeout:     10 * time.Second,
		TickInterval:         1 * time.Second,
	}

	sched := scheduler.NewDAGScheduler(cfg, store, sup)
	ctx := context.Background()

	// 3 ready tasks, but max workers is 2
	plan := domain.Plan{Slug: "p2", Title: "P2", RepoPath: "/repo", Status: "ACTIVE"}
	tasks := []domain.Task{
		{ID: "t-1", PlanID: plan.Slug, RepoPath: plan.RepoPath, Title: "T1", SkillCommand: "sleep 1", Status: domain.StatusReady},
		{ID: "t-2", PlanID: plan.Slug, RepoPath: plan.RepoPath, Title: "T2", SkillCommand: "sleep 1", Status: domain.StatusReady},
		{ID: "t-3", PlanID: plan.Slug, RepoPath: plan.RepoPath, Title: "T3", SkillCommand: "sleep 1", Status: domain.StatusReady},
	}
	_ = sched.SubmitPlan(ctx, plan, tasks)

	// Tick: Should dispatch exactly 2 tasks
	if err := sched.Tick(ctx); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}

	activeRuns, err := store.GetActiveRuns(ctx)
	if err != nil {
		t.Fatalf("GetActiveRuns failed: %v", err)
	}
	if len(activeRuns) != 2 {
		t.Errorf("Expected 2 active runs dispatched, got: %d", len(activeRuns))
	}

	// Verify the 3rd task is still READY
	readyTasks, err := store.GetReadyTasks(ctx, "/repo", 10)
	if err != nil || len(readyTasks) != 1 || readyTasks[0].ID != "t-3" {
		t.Errorf("Expected t-3 to remain READY, got: %v", readyTasks)
	}

	// Cleanup active runs
	for _, r := range activeRuns {
		_ = sup.Kill(r.ID, 50*time.Millisecond)
	}
}

func TestScheduler_Tick_ReapHungRuns(t *testing.T) {
	tempDir := t.TempDir()
	db, err := storage.Open(filepath.Join(tempDir, "state.db"))
	if err != nil {
		t.Fatalf("Open DB failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	sup := supervisor.NewProcessSupervisor(tempDir)
	cfg := scheduler.SchedulerConfig{
		MaxConcurrentWorkers: 3,
		HeartbeatTimeout:     100 * time.Millisecond, // Fast timeout for test
		TickInterval:         1 * time.Second,
	}

	sched := scheduler.NewDAGScheduler(cfg, store, sup)
	ctx := context.Background()

	plan := domain.Plan{Slug: "p3", Title: "P3", RepoPath: "/repo", Status: "ACTIVE"}
	task := domain.Task{ID: "t-hang", PlanID: plan.Slug, RepoPath: plan.RepoPath, Title: "Hang Task", SkillCommand: "sleep 50", Status: domain.StatusReady}
	_ = sched.SubmitPlan(ctx, plan, []domain.Task{task})

	// Dispatch task
	_ = sched.Tick(ctx)

	// Wait for heartbeat timeout to expire (150ms > 100ms)
	time.Sleep(150 * time.Millisecond)

	// Tick should reap the hung run
	if err := sched.Tick(ctx); err != nil {
		t.Fatalf("Tick reap failed: %v", err)
	}

	// Verify task is now FAILED
	ts, _ := store.GetTasksByPlan(ctx, "p3")
	if len(ts) != 1 || ts[0].Status != domain.StatusFailed {
		t.Errorf("Expected task to be reaped to FAILED, got status: %s", ts[0].Status)
	}
}

type mockBroadcaster struct {
	events []domain.EventMessage
}

func (m *mockBroadcaster) Broadcast(evt domain.EventMessage) {
	m.events = append(m.events, evt)
}

func TestScheduler_Tick_ExternalOrchestratorHarness(t *testing.T) {
	tempDir := t.TempDir()
	db, err := storage.Open(filepath.Join(tempDir, "state.db"))
	if err != nil {
		t.Fatalf("Open DB failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	sup := supervisor.NewProcessSupervisor(tempDir)
	cfg := scheduler.DefaultConfig()

	sched := scheduler.NewDAGScheduler(cfg, store, sup)
	mb := &mockBroadcaster{}
	sched.SetBroadcaster(mb)

	ctx := context.Background()

	plan := domain.Plan{Slug: "orch-plan", Title: "Orchestrator Plan", RepoPath: "/repo", Status: "ACTIVE"}
	task := domain.Task{
		ID:           "t-orch",
		PlanID:       plan.Slug,
		RepoPath:     plan.RepoPath,
		Title:        "External Agent Task",
		Harness:      "orchestrator",
		SkillCommand: "/forge t-orch",
		Status:       domain.StatusReady,
	}
	_ = sched.SubmitPlan(ctx, plan, []domain.Task{task})

	if err := sched.Tick(ctx); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}

	// Task must remain in READY status (never spawned as OS process)
	ts, err := store.GetTasksByPlan(ctx, "orch-plan")
	if err != nil || len(ts) != 1 {
		t.Fatalf("Failed to retrieve tasks: %v", err)
	}
	if ts[0].Status != domain.StatusReady {
		t.Errorf("Expected status READY, got %s", ts[0].Status)
	}

	// Active runs in storage should be 0
	activeRuns, err := store.GetActiveRuns(ctx)
	if err != nil || len(activeRuns) != 0 {
		t.Errorf("Expected 0 active runs, got %d", len(activeRuns))
	}

	// Event broadcaster should have received task:ready_for_agent
	foundEvent := false
	for _, evt := range mb.events {
		if evt.Type == "task:ready_for_agent" && evt.TaskID == "t-orch" {
			foundEvent = true
			break
		}
	}
	if !foundEvent {
		t.Errorf("Expected task:ready_for_agent event, received: %+v", mb.events)
	}
}
