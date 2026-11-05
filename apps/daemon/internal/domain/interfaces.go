package domain

import (
	"context"
)

// PlanRepository persists plans and the plan/task bundles written by authoring
// tools.
type PlanRepository interface {
	// SavePlanWithTasks writes a plan and its tasks in a single atomic transaction.
	SavePlanWithTasks(ctx context.Context, plan Plan, tasks []Task) error

	// CreatePlan inserts or updates a single plan.
	CreatePlan(ctx context.Context, plan Plan) error

	// GetPlan retrieves a plan by slug.
	GetPlan(ctx context.Context, slug string) (*Plan, error)

	// ListPlans returns plans ordered by archive state and slug. Archived plans
	// are included only when includeArchived is true.
	ListPlans(ctx context.Context, includeArchived bool) ([]Plan, error)

	// ArchivePlan sets or clears archived_at. Archived plans stay readable but
	// are hidden from the default selector and DAG.
	ArchivePlan(ctx context.Context, slug string, archived bool) error

	// DeletePlan deletes a plan and cascades to its tasks and artifacts.
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

	// CreateTask inserts a single task and fails when the ID is taken.
	CreateTask(ctx context.Context, task Task) error

	// CreateTasks inserts a batch atomically and validates the plan graph once.
	CreateTasks(ctx context.Context, tasks []Task) error

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

	// UpdateTaskStatusWithNote applies a transition and appends an audit note in
	// the same transaction.
	UpdateTaskStatusWithNote(ctx context.Context, taskID string, from TaskStatus, to TaskStatus, note string) error

	// ClaimTask atomically verifies readiness, sets the claim lock, transitions to RUNNING, and returns the ContextPacket.
	ClaimTask(ctx context.Context, taskID string, workerID string) (*ContextPacket, error)

	// ClaimTaskWithMeta atomically claims a ready task with full metadata.
	ClaimTaskWithMeta(ctx context.Context, taskID string, workerID string, orchestratorID string, workerRole string) (*ContextPacket, error)

	// EvictTask evicts a running task, resetting it to READY (or
	// BLOCKED when prerequisites are unmet) or cancelling it. It returns the
	// status actually written.
	EvictTask(ctx context.Context, taskID string, reason string, action string) (TaskStatus, error)

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

// EventRepository reads the trigger-produced activity log.
type EventRepository interface {
	// ListEvents returns up to limit events newer than afterID in ascending
	// order. When afterID is zero it returns the most recent window.
	ListEvents(ctx context.Context, afterID int64, limit int) ([]EventMessage, error)
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
