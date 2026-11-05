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

// PlanRepository persists plans and the plan/task bundles written by authoring
// tools.
type PlanRepository interface {
	// SavePlanWithTasks writes a plan and its tasks in a single atomic transaction.
	SavePlanWithTasks(ctx context.Context, plan Plan, tasks []Task) error

	// CreatePlan inserts or updates a single plan.
	CreatePlan(ctx context.Context, plan Plan) error

	// GetPlan retrieves a plan by slug.
	GetPlan(ctx context.Context, slug string) (*Plan, error)

	// ListPlans returns every plan registered in the plans table, including
	// plans that do not have any tasks yet.
	ListPlans(ctx context.Context) ([]Plan, error)

	// DeletePlan deletes a plan and cascades to its tasks, runs, and artifacts.
	DeletePlan(ctx context.Context, slug string) error
}

// TaskRepository persists task records and their read projections.
type TaskRepository interface {
	// GetTask retrieves a single task by its unique ID.
	GetTask(ctx context.Context, taskID string) (*Task, error)

	// GetTasksByPlan retrieves all tasks belonging to a plan.
	GetTasksByPlan(ctx context.Context, planSlug string) ([]Task, error)

	// ListTasks returns tasks filtered optionally by status.
	ListTasks(ctx context.Context, status TaskStatus) ([]Task, error)

	// GetReadyTasks returns tasks currently in READY status, oldest first.
	// It does not re-verify dependency state; readiness is owned by the
	// lifecycle transition that unlocks dependents.
	GetReadyTasks(ctx context.Context, limit int) ([]Task, error)

	// CreateTask inserts a single task.
	CreateTask(ctx context.Context, task Task) error

	// UpdateTaskDependencies updates the task dependency list.
	UpdateTaskDependencies(ctx context.Context, taskID string, deps []string) error

	// UpdateTaskNotes updates developer supplemental notes.
	UpdateTaskNotes(ctx context.Context, taskID string, notes string) error

	// AppendTaskNotes appends a note to the task's supplemental notes.
	AppendTaskNotes(ctx context.Context, taskID string, note string) error
}

// TaskLifecycle owns the task state machine, claim mutual exclusion, and the
// review handshake.
type TaskLifecycle interface {
	// UpdateTaskStatus atomically transitions task status from an expected current state.
	UpdateTaskStatus(ctx context.Context, taskID string, from TaskStatus, to TaskStatus) error

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

	// GetTaskContextPacket compiles full context packet (task, target files, skill paths, media).
	GetTaskContextPacket(ctx context.Context, taskID string) (*ContextPacket, error)
}

// ArtifactRepository persists artifact metadata following the claim-check
// pattern.
type ArtifactRepository interface {
	// RecordArtifact registers a file artifact in the database.
	RecordArtifact(ctx context.Context, artifact Artifact) error

	// GetArtifact retrieves artifact metadata by ID.
	GetArtifact(ctx context.Context, id string) (*Artifact, error)

	// GetArtifactsByTask returns all artifacts associated with a task, optionally filtered by role ("INPUT" | "OUTPUT").
	GetArtifactsByTask(ctx context.Context, taskID string, role string) ([]Artifact, error)
}

// ArtifactReader is the narrow read seam used by context packet assembly.
type ArtifactReader interface {
	GetArtifactsByTask(ctx context.Context, taskID string, role string) ([]Artifact, error)
}

// EventLog persists audit events.
type EventLog interface {
	// AppendEvent records an audit event in the database.
	AppendEvent(ctx context.Context, evt EventMessage) error
}

// StateFingerprinter summarizes plans/tasks state so callers can detect writes
// made by another process.
type StateFingerprinter interface {
	StateFingerprint(ctx context.Context) (string, error)
}

// ContextPacketBuilder assembles the prompt-ready packet for a task.
type ContextPacketBuilder interface {
	Build(ctx context.Context, task Task) (*ContextPacket, error)
}
