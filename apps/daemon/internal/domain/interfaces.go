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

	// ListTasks returns tasks filtered optionally by status.
	ListTasks(ctx context.Context, status TaskStatus) ([]Task, error)

	// GetReadyTasks returns tasks whose dependencies are fully completed.
	GetReadyTasks(ctx context.Context, limit int) ([]Task, error)

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

	// CreatePlan inserts a single plan.
	CreatePlan(ctx context.Context, plan Plan) error

	// DeletePlan deletes a plan and cascades to its tasks, runs, and artifacts.
	DeletePlan(ctx context.Context, slug string) error

	// CreateTask inserts a single task.
	CreateTask(ctx context.Context, task Task) error

	// UpdateTaskDependencies updates the task dependency list.
	UpdateTaskDependencies(ctx context.Context, taskID string, deps []string) error

	// UpdateTaskNotes updates developer supplemental notes.
	UpdateTaskNotes(ctx context.Context, taskID string, notes string) error

	// AddInputAttachment adds an input artifact (media or screenshot) to a task.
	AddInputAttachment(ctx context.Context, taskID string, name string, filePath string, mimeType string, byteSize int64, sha256 string) (*Artifact, error)

	// ClaimTask atomically verifies readiness, sets claim lock (workerID & pid), transitions to RUNNING, and returns ContextPacket.
	ClaimTask(ctx context.Context, taskID string, workerID string, pid int) (*ContextPacket, error)

	// ClaimTaskWithMeta atomically claims a ready task with full metadata.
	ClaimTaskWithMeta(ctx context.Context, taskID string, workerID string, orchestratorID string, workerRole string, pid int, orchestratorPID int) (*ContextPacket, error)

	// EvictTask evicts a running or suspended task, resetting it to READY or cancelling it.
	EvictTask(ctx context.Context, taskID string, reason string, action string) error

	// RequestReview pauses execution, records review reasons/options, and transitions task to AWAITING_REVIEW.
	RequestReview(ctx context.Context, taskID string, reason string, details string, options []string) error

	// ResolveReview records the human guidance relayed by the main agent and transitions task back to READY.
	ResolveReview(ctx context.Context, taskID string, response string) error

	// AppendTaskNotes appends a note to the task's supplemental notes.
	AppendTaskNotes(ctx context.Context, taskID string, note string) error

	// GetTaskContextPacket compiles full context packet (task, target files, skill paths, media).
	GetTaskContextPacket(ctx context.Context, taskID string) (*ContextPacket, error)

	// Close safely flushes WAL checkpoints and closes the database connection pool.
	Close() error
}
