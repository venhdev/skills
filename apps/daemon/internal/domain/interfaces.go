package domain

import (
	"context"
	"io"
	"time"
)

// WindowSize conveys terminal dimensions for interactive PTY resizing.
type WindowSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// Supervisor manages OS process lifecycles and PTY allocation with zero orphan leaks.
type Supervisor interface {
	// SpawnHeadless launches an agent runner using stdio pipes (non-interactive).
	SpawnHeadless(ctx context.Context, task Task, envOverrides map[string]string) (*Run, error)

	// SpawnPTY allocates a pseudo-terminal and launches an interactive agent runner.
	SpawnPTY(ctx context.Context, task Task, win WindowSize, envOverrides map[string]string) (*Run, io.ReadWriteCloser, error)

	// Kill terminates the entire process group using -PGID (SIGTERM then SIGKILL escalation).
	Kill(runID string, gracePeriod time.Duration) error

	// ResizePTY sends a SIGWINCH window size update to the interactive PTY.
	ResizePTY(runID string, win WindowSize) error

	// GetRun retrieves the current run state snapshot.
	GetRun(runID string) (*Run, error)
}

// Scheduler resolves DAG readiness, schedules eligible workers, and audits heartbeat liveness.
type Scheduler interface {
	// SubmitPlan registers a new living plan and its decomposed task DAG.
	SubmitPlan(ctx context.Context, plan Plan, tasks []Task) error

	// GetPlan retrieves a plan and all its tasks by slug.
	GetPlan(ctx context.Context, slug string) (*Plan, []Task, error)

	// Tick evaluates ready tasks, dispatches available workers, and reaps crashed processes.
	Tick(ctx context.Context) error

	// RecordHeartbeat refreshes the liveness cursor for an active run.
	RecordHeartbeat(ctx context.Context, runID string, cursor map[string]any) error
}

// EventBroadcaster distributes real-time events to connected clients and event stores.
type EventBroadcaster interface {
	Broadcast(evt EventMessage)
}

// StorageEngine provides atomic SQLite WAL operations with connection serialization.
type StorageEngine interface {
	// SavePlanWithTasks writes a plan and its tasks in a single atomic transaction.
	SavePlanWithTasks(ctx context.Context, plan Plan, tasks []Task) error

	// GetPlan retrieves a plan by slug.
	GetPlan(ctx context.Context, slug string) (*Plan, error)

	// GetTasksByPlan retrieves all tasks belonging to a plan.
	GetTasksByPlan(ctx context.Context, planSlug string) ([]Task, error)

	// ListTasks returns tasks filtered optionally by repoPath and/or status.
	ListTasks(ctx context.Context, repoPath string, status TaskStatus) ([]Task, error)

	// GetReadyTasks returns tasks whose dependencies are fully completed.
	GetReadyTasks(ctx context.Context, repoPath string, limit int) ([]Task, error)

	// UpdateTaskStatus atomically transitions task status from an expected current state.
	UpdateTaskStatus(ctx context.Context, taskID string, from TaskStatus, to TaskStatus) error

	// CreateRun registers a new execution attempt.
	CreateRun(ctx context.Context, run Run) error

	// GetActiveRuns returns all currently active runs (Status == RUNNING).
	GetActiveRuns(ctx context.Context) ([]Run, error)

	// UpdateRunHeartbeat updates the last heartbeat timestamp for an active run.
	UpdateRunHeartbeat(ctx context.Context, runID string, now time.Time) error

	// FinishRun marks a run as completed or failed with an exit code.
	FinishRun(ctx context.Context, runID string, status TaskStatus, exitCode int) error

	// RecordArtifact registers a file artifact in the database.
	RecordArtifact(ctx context.Context, artifact Artifact) error

	// GetArtifact retrieves artifact metadata by ID.
	GetArtifact(ctx context.Context, id string) (*Artifact, error)

	// GetArtifactsByTask returns all artifacts associated with a task, optionally filtered by role ("INPUT" | "OUTPUT").
	GetArtifactsByTask(ctx context.Context, taskID string, role string) ([]Artifact, error)

	// AppendEvent records an audit event in the database.
	AppendEvent(ctx context.Context, evt EventMessage) error

	// GetTask retrieves a single task by its unique ID.
	GetTask(ctx context.Context, taskID string) (*Task, error)

	// ListProjects returns all distinct repo_paths known to the store.
	ListProjects(ctx context.Context) ([]string, error)

	// CreatePlan inserts a single plan.
	CreatePlan(ctx context.Context, plan Plan) error

	// DeletePlan deletes a plan and cascades to its tasks, runs, artifacts, and docs.
	DeletePlan(ctx context.Context, slug string) error

	// CreateTask inserts a single task.
	CreateTask(ctx context.Context, task Task) error

	// UpdateTaskDependencies updates the task dependency list.
	UpdateTaskDependencies(ctx context.Context, taskID string, deps []string) error

	// UpdateTaskNotes updates developer supplemental notes.
	UpdateTaskNotes(ctx context.Context, taskID string, notes string) error

	// AddInputAttachment adds an input artifact (media, screenshot, or spec) to a task.
	AddInputAttachment(ctx context.Context, taskID string, name string, filePath string, mimeType string, byteSize int64, sha256 string) (*Artifact, error)

	// ClaimTask atomically verifies readiness, sets claim lock (workerID & pid), transitions to RUNNING, and returns ContextPacket.
	ClaimTask(ctx context.Context, taskID string, workerID string, pid int) (*ContextPacket, error)

	// ClaimTaskWithMeta atomically claims a ready task with full metadata.
	ClaimTaskWithMeta(ctx context.Context, taskID string, workerID string, orchestratorID string, workerRole string, pid int, orchestratorPID int) (*ContextPacket, error)

	// ReclaimTask re-claims a SUSPENDED task with new worker and orchestrator.
	ReclaimTask(ctx context.Context, taskID string, newWorkerID string, newOrchestratorID string, newPID int) (*ContextPacket, error)

	// EvictTask evicts a running or suspended task, resetting it to READY or cancelling it.
	EvictTask(ctx context.Context, taskID string, reason string, action string) error

	// RegisterSession registers or updates an agent session.
	RegisterSession(ctx context.Context, session AgentSession) error

	// ListSessions returns all active or recently seen sessions.
	ListSessions(ctx context.Context) ([]AgentSession, error)

	// HeartbeatSession updates the last heartbeat timestamp for a session.
	HeartbeatSession(ctx context.Context, sessionID string) error

	// RequestReview pauses execution, records review reasons/options, and transitions task to AWAITING_REVIEW.
	RequestReview(ctx context.Context, taskID string, reason string, details string, options []string) error

	// ResolveReview answers an awaiting review, records resolution, and transitions task back to READY.
	ResolveReview(ctx context.Context, taskID string, response string) error

	// StageMerge marks a worktree task as complete and transitions it to AWAITING_MERGE.
	StageMerge(ctx context.Context, taskID string) error

	// MergeTask executes git merge from worktree to target branch, transitioning to COMPLETED or CONFLICT.
	MergeTask(ctx context.Context, taskID string) (bool, []string, error)

	// GetTaskContextPacket compiles full context packet (task, target files, skill paths, upstream diffs, media).
	GetTaskContextPacket(ctx context.Context, taskID string) (*ContextPacket, error)

	// UpsertDoc creates or updates a project document or living architecture spec.
	UpsertDoc(ctx context.Context, doc ProjectDoc) error

	// GetDoc retrieves a project document by repoPath and slug.
	GetDoc(ctx context.Context, repoPath, slug string) (*ProjectDoc, error)

	// ListDocs lists project documents filtered optionally by category.
	ListDocs(ctx context.Context, repoPath, category string) ([]ProjectDoc, error)

	// UpsertSpec creates or updates a multi-tiered technical specification.
	UpsertSpec(ctx context.Context, spec TechSpec) (*TechSpec, error)

	// GetSpec retrieves a technical spec by slug.
	GetSpec(ctx context.Context, slug string) (*TechSpec, error)

	// ListSpecs lists technical specs filtered optionally by repoPath and/or tier.
	ListSpecs(ctx context.Context, repoPath string, tier int) ([]TechSpec, error)

	// SearchSpecs performs a full-text FTS5 search across all technical specs.
	SearchSpecs(ctx context.Context, query string) ([]TechSpec, error)

	// AutoLinkSpecs evaluates targetFiles against bound_paths of all registered specs.
	AutoLinkSpecs(ctx context.Context, targetFiles []string) ([]string, error)

	// SavePlanSnapshot saves an atomic directory snapshot for a plan at a given revision.
	SavePlanSnapshot(ctx context.Context, planSlug, revision, author, summary string, files []SnapshotFile) error

	// GetPlanSnapshot retrieves all snapshot files for a plan at a given revision.
	GetPlanSnapshot(ctx context.Context, planSlug, revision string) ([]SnapshotFile, error)

	// ListPlanRevisions lists all revision records for a given plan.
	ListPlanRevisions(ctx context.Context, planSlug string) ([]PlanRevision, error)

	// GetPlanDiff computes differences between two plan revisions.
	GetPlanDiff(ctx context.Context, planSlug, fromRev, toRev string) ([]FileDiff, error)

	// Close safely flushes WAL checkpoints and closes the database connection pool.
	Close() error
}
