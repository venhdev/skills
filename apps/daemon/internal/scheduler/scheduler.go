package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"zenfd/internal/domain"
)

// SchedulerConfig holds operational tuning for the DAG scheduler.
type SchedulerConfig struct {
	MaxConcurrentWorkers int           // Maximum parallel active agent runs
	HeartbeatTimeout     time.Duration // Time without heartbeat before marking a run hung
	TickInterval         time.Duration // Interval between scheduler ticks
}

// DefaultConfig returns sane defaults for local development.
func DefaultConfig() SchedulerConfig {
	return SchedulerConfig{
		MaxConcurrentWorkers: 3,
		HeartbeatTimeout:     60 * time.Second,
		TickInterval:         10 * time.Second,
	}
}

// DAGScheduler coordinates task dispatch, DAG dependency unlocking, and worker liveness.
type DAGScheduler struct {
	cfg        SchedulerConfig
	storage    domain.StorageEngine
	supervisor domain.Supervisor
	mu         sync.Mutex
	stopCh     chan struct{}
}

// NewDAGScheduler creates a new scheduler instance.
func NewDAGScheduler(cfg SchedulerConfig, store domain.StorageEngine, sup domain.Supervisor) *DAGScheduler {
	if cfg.MaxConcurrentWorkers <= 0 {
		cfg.MaxConcurrentWorkers = 3
	}
	if cfg.HeartbeatTimeout <= 0 {
		cfg.HeartbeatTimeout = 60 * time.Second
	}
	if cfg.TickInterval <= 0 {
		cfg.TickInterval = 10 * time.Second
	}

	return &DAGScheduler{
		cfg:        cfg,
		storage:    store,
		supervisor: sup,
		stopCh:     make(chan struct{}),
	}
}

// SubmitPlan registers a new plan and stores its DAG in storage.
func (s *DAGScheduler) SubmitPlan(ctx context.Context, plan domain.Plan, tasks []domain.Task) error {
	return s.storage.SavePlanWithTasks(ctx, plan, tasks)
}

// GetPlan retrieves a plan and its tasks by slug.
func (s *DAGScheduler) GetPlan(ctx context.Context, slug string) (*domain.Plan, []domain.Task, error) {
	plan, err := s.storage.GetPlan(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	tasks, err := s.storage.GetTasksByPlan(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	return plan, tasks, nil
}

// RecordHeartbeat records an active heartbeat for a running agent.
func (s *DAGScheduler) RecordHeartbeat(ctx context.Context, runID string, cursor map[string]any) error {
	return s.storage.UpdateRunHeartbeat(ctx, runID, time.Now().UTC())
}

// Tick audits active workers for timeouts, and dispatches available ready tasks into slots.
func (s *DAGScheduler) Tick(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	// 1. Audit active runs for liveness / crash reaping
	activeRuns, err := s.storage.GetActiveRuns(ctx)
	if err == nil {
		for _, run := range activeRuns {
			if now.Sub(run.LastBeatAt) > s.cfg.HeartbeatTimeout {
				// Run is hung or dead: kill process and fail task
				_ = s.supervisor.Kill(run.ID, 1*time.Second)
				_ = s.storage.FinishRun(ctx, run.ID, domain.StatusFailed, -1)
				_ = s.storage.UpdateTaskStatus(ctx, run.TaskID, domain.StatusRunning, domain.StatusFailed)
			}
		}
	}

	// 2. Query currently active runs count after reaping
	activeRuns, err = s.storage.GetActiveRuns(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active runs: %w", err)
	}

	availableSlots := s.cfg.MaxConcurrentWorkers - len(activeRuns)
	if availableSlots <= 0 {
		return nil // Max workers saturated
	}

	// 3. Claim and dispatch ready tasks
	readyTasks, err := s.storage.GetReadyTasks(ctx, "", availableSlots)
	if err != nil {
		return fmt.Errorf("failed to get ready tasks: %w", err)
	}

	for _, task := range readyTasks {
		// Transition task: READY -> RUNNING
		if err := s.storage.UpdateTaskStatus(ctx, task.ID, domain.StatusReady, domain.StatusRunning); err != nil {
			continue // Claim collision, skip
		}

		// Spawn process via supervisor
		run, err := s.supervisor.SpawnHeadless(ctx, task, nil)
		if err != nil {
			// Spawn failed: revert or mark failed
			_ = s.storage.UpdateTaskStatus(ctx, task.ID, domain.StatusRunning, domain.StatusFailed)
			continue
		}

		// Record run in storage
		_ = s.storage.CreateRun(ctx, *run)
	}

	return nil
}
