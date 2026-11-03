package scheduler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/harness"
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
	cfg             SchedulerConfig
	storage         domain.StorageEngine
	supervisor      domain.Supervisor
	broadcaster     domain.EventBroadcaster
	harnessRegistry *harness.Registry
	mu              sync.Mutex
	stopCh          chan struct{}
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
		cfg:             cfg,
		storage:         store,
		supervisor:      sup,
		harnessRegistry: harness.DefaultRegistry(),
		stopCh:          make(chan struct{}),
	}
}

// SetBroadcaster configures the event broadcaster for real-time SSE task notifications.
func (s *DAGScheduler) SetBroadcaster(b domain.EventBroadcaster) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.broadcaster = b
}

// SetHarnessRegistry configures the execution harness adapter registry.
func (s *DAGScheduler) SetHarnessRegistry(reg *harness.Registry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.harnessRegistry = reg
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

	// 1. Audit active runs for completion and liveness / crash reaping
	activeRuns, err := s.storage.GetActiveRuns(ctx)
	if err == nil {
		for _, run := range activeRuns {
			supRun, err := s.supervisor.GetRun(run.ID)
			if err == nil && supRun != nil && supRun.FinishedAt != nil {
				// Process finished naturally in supervisor
				_ = s.storage.FinishRun(ctx, run.ID, supRun.Status, supRun.ExitCode)
				_ = s.storage.UpdateTaskStatus(ctx, run.TaskID, domain.StatusRunning, supRun.Status)
				if s.broadcaster != nil {
					s.broadcaster.Broadcast(domain.EventMessage{
						Type:      "task:status",
						TaskID:    run.TaskID,
						Payload:   fmt.Sprintf(`{"task_id":"%s","status":"%s"}`, run.TaskID, supRun.Status),
						Timestamp: now,
					})
				}
				s.captureGitDiff(ctx, run)
				continue
			}

			if now.Sub(run.LastBeatAt) > s.cfg.HeartbeatTimeout {
				// Run is hung or dead: kill process and fail task
				_ = s.supervisor.Kill(run.ID, 1*time.Second)
				_ = s.storage.FinishRun(ctx, run.ID, domain.StatusFailed, -1)
				_ = s.storage.UpdateTaskStatus(ctx, run.TaskID, domain.StatusRunning, domain.StatusFailed)
				if s.broadcaster != nil {
					s.broadcaster.Broadcast(domain.EventMessage{
						Type:      "task:status",
						TaskID:    run.TaskID,
						Payload:   fmt.Sprintf(`{"task_id":"%s","status":"%s"}`, run.TaskID, domain.StatusFailed),
						Timestamp: now,
					})
				}
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
		// Resolve execution harness adapter
		if s.harnessRegistry != nil {
			adapter, err := s.harnessRegistry.Resolve(task)
			if err == nil && (!adapter.IsAvailable() || adapter.Name() == "orchestrator") {
				// Task requires external orchestrator or toolchain not present on host.
				// Broadcast notification and leave in READY status (do not spawn failing OS process).
				if s.broadcaster != nil {
					s.broadcaster.Broadcast(domain.EventMessage{
						Type:      "task:ready_for_agent",
						TaskID:    task.ID,
						Payload:   fmt.Sprintf(`{"task_id":"%s","harness":"%s","command":"%s"}`, task.ID, adapter.Name(), task.SkillCommand),
						Timestamp: now,
					})
				}
				continue
			}
		}

		// Transition task: READY -> RUNNING
		if err := s.storage.UpdateTaskStatus(ctx, task.ID, domain.StatusReady, domain.StatusRunning); err != nil {
			continue // Claim collision, skip
		}
		if s.broadcaster != nil {
			s.broadcaster.Broadcast(domain.EventMessage{
				Type:      "task:status",
				TaskID:    task.ID,
				Payload:   fmt.Sprintf(`{"task_id":"%s","status":"%s"}`, task.ID, domain.StatusRunning),
				Timestamp: now,
			})
		}

		// Spawn process via supervisor
		run, err := s.supervisor.SpawnHeadless(ctx, task, nil)
		if err != nil {
			// Spawn failed: revert or mark failed
			_ = s.storage.UpdateTaskStatus(ctx, task.ID, domain.StatusRunning, domain.StatusFailed)
			if s.broadcaster != nil {
				s.broadcaster.Broadcast(domain.EventMessage{
					Type:      "task:status",
					TaskID:    task.ID,
					Payload:   fmt.Sprintf(`{"task_id":"%s","status":"%s"}`, task.ID, domain.StatusFailed),
					Timestamp: now,
				})
			}
			continue
		}

		// Record run in storage
		_ = s.storage.CreateRun(ctx, *run)
	}

	return nil
}

// captureGitDiff captures unified diffs generated by a completed task and saves an artifact.
func (s *DAGScheduler) captureGitDiff(ctx context.Context, run domain.Run) {
	workDir := run.RepoPath
	if workDir == "" {
		return
	}
	// Verify git presence in directory
	if _, err := os.Stat(filepath.Join(workDir, ".git")); err != nil {
		return
	}

	// 1. Stage untracked files with intent-to-add
	_ = exec.CommandContext(ctx, "git", "-C", workDir, "add", "-N", ".").Run()

	// 2. Capture unified diff
	out, err := exec.CommandContext(ctx, "git", "-C", workDir, "diff", "HEAD").Output()
	if err != nil || len(bytes.TrimSpace(out)) == 0 {
		return
	}

	// 3. Save diff to artifact store
	home, _ := os.UserHomeDir()
	artDir := filepath.Join(home, ".zenforge", "artifacts", run.ID)
	_ = os.MkdirAll(artDir, 0755)
	diffPath := filepath.Join(artDir, "changes.diff")
	if err := os.WriteFile(diffPath, out, 0644); err != nil {
		return
	}

	hash := sha256.Sum256(out)
	art := domain.Artifact{
		ID:        fmt.Sprintf("art-%s-diff", run.ID),
		RunID:     run.ID,
		TaskID:    run.TaskID,
		Role:      domain.ArtifactRoleOutput,
		Name:      "changes.diff",
		Type:      domain.ArtifactTypeDiff,
		FilePath:  diffPath,
		MimeType:  "text/x-diff",
		ByteSize:  int64(len(out)),
		Sha256:    hex.EncodeToString(hash[:]),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.storage.RecordArtifact(ctx, art); err == nil {
		if s.broadcaster != nil {
			s.broadcaster.Broadcast(domain.EventMessage{
				Type:      "task:artifact_added",
				TaskID:    run.TaskID,
				RunID:     run.ID,
				Payload:   fmt.Sprintf(`{"artifact_id":"%s","role":"OUTPUT","name":"changes.diff","type":"DIFF"}`, art.ID),
				Timestamp: time.Now().UTC(),
			})
		}
	}
}
