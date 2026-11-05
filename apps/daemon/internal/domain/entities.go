package domain

import (
	"fmt"
	"time"
)

// TaskStatus defines the state machine states for a DAG task slice.
type TaskStatus string

const (
	StatusPending        TaskStatus = "PENDING"         // Waiting for upstream dependencies
	StatusReady          TaskStatus = "READY"           // Dependencies fulfilled, ready to claim
	StatusRunning        TaskStatus = "RUNNING"         // Actively being executed by a worker
	StatusAwaitingReview TaskStatus = "AWAITING_REVIEW" // Paused: Plan/scope drift detected, awaiting developer decision
	StatusAwaitingMerge  TaskStatus = "AWAITING_MERGE"  // Implemented & tested on worktree, staged for git merge
	StatusConflict       TaskStatus = "CONFLICT"        // Git merge conflict encountered during integration
	StatusBlocked        TaskStatus = "BLOCKED"         // Blocked by upstream failure, environment, or human review
	StatusCompleted      TaskStatus = "COMPLETED"       // Verified with exit code 0 and passing tests
	StatusFailed         TaskStatus = "FAILED"          // Execution failure or test assertion failure
	StatusCancelled      TaskStatus = "CANCELLED"       // Terminated by an explicit operator abort
)

// IsValid checks if the status is a known domain status.
func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusReady, StatusRunning, StatusAwaitingReview,
		StatusAwaitingMerge, StatusConflict, StatusBlocked,
		StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionTo enforces the state machine transition rules.
func (s TaskStatus) CanTransitionTo(target TaskStatus) bool {
	if !s.IsValid() || !target.IsValid() {
		return false
	}
	if s == target {
		return true // No-op idempotent transition
	}
	if s == StatusCancelled {
		return false // Cancelled is strictly terminal
	}
	if s == StatusCompleted {
		// Completed tasks can only be transitioned back to READY by explicit operator re-open
		return target == StatusReady
	}

	switch s {
	case StatusPending:
		return target == StatusReady || target == StatusBlocked || target == StatusCancelled
	case StatusReady:
		return target == StatusRunning || target == StatusBlocked || target == StatusCancelled
	case StatusRunning:
		return target == StatusAwaitingReview ||
			target == StatusAwaitingMerge || target == StatusCompleted ||
			target == StatusFailed || target == StatusCancelled || target == StatusBlocked || target == StatusReady
	case StatusAwaitingReview:
		// Resumed back to READY (recommended) or RUNNING, or failed/cancelled
		return target == StatusReady || target == StatusRunning || target == StatusFailed || target == StatusCancelled
	case StatusAwaitingMerge:
		// Merged to COMPLETED, hit CONFLICT, rejected back to RUNNING, or cancelled
		return target == StatusCompleted || target == StatusConflict ||
			target == StatusRunning || target == StatusCancelled
	case StatusConflict:
		// Merged after manual conflict resolution, assigned back to RUNNING, or failed/cancelled
		return target == StatusCompleted || target == StatusRunning ||
			target == StatusReady || target == StatusFailed || target == StatusCancelled
	case StatusBlocked:
		return target == StatusReady || target == StatusPending || target == StatusCancelled
	case StatusFailed:
		// Allow retry back to READY or directly to RUNNING
		return target == StatusReady || target == StatusRunning || target == StatusCancelled
	default:
		return false
	}
}

type TaskPriority string

const (
	PriorityP0 TaskPriority = "P0"
	PriorityP1 TaskPriority = "P1"
	PriorityP2 TaskPriority = "P2"
	PriorityP3 TaskPriority = "P3"
)

type TaskComplexity string

const (
	ComplexityLow    TaskComplexity = "LOW"
	ComplexityMedium TaskComplexity = "MEDIUM"
	ComplexityHigh   TaskComplexity = "HIGH"
	ComplexitySpike  TaskComplexity = "SPIKE"
)

type TaskRisk string

const (
	RiskLow      TaskRisk = "LOW"
	RiskMedium   TaskRisk = "MEDIUM"
	RiskHigh     TaskRisk = "HIGH"
	RiskCritical TaskRisk = "CRITICAL"
)

// Plan represents a living plan dossier and its associated task DAG.
type Plan struct {
	Slug       string     `json:"slug"`
	Title      string     `json:"title"`
	Status     string     `json:"status"` // ACTIVE, COMPLETED, DRIFTED
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Task represents an atomic vertical slice of execution in the DAG.
type Task struct {
	// 1. Identity & DAG Topology
	ID           string   `json:"id"`
	PlanID       string   `json:"plan_id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Dependencies []string `json:"dependencies"` // List of prerequisite Task IDs

	// 2. Triage & Fan-out Attributes
	Priority             TaskPriority   `json:"priority,omitempty"`
	Complexity           TaskComplexity `json:"complexity,omitempty"`
	RiskLevel            TaskRisk       `json:"risk_level,omitempty"`
	RequiredSkills       []string       `json:"required_skills,omitempty"`
	RequiredTools        []string       `json:"required_tools,omitempty"`
	TargetFiles          []string       `json:"target_files,omitempty"`
	AcceptanceCriteria   []string       `json:"acceptance_criteria,omitempty"`
	VerificationCommands []string       `json:"verification_commands,omitempty"`
	Harness              string         `json:"harness"`
	WorktreePath         string         `json:"worktree_path,omitempty"`

	// 3. State & Claim Mutual Exclusion
	Status         TaskStatus `json:"status"`
	ClaimedBy      string     `json:"claimed_by,omitempty"`
	OrchestratorID string     `json:"orchestrator_id,omitempty"`
	WorkerRole     string     `json:"worker_role,omitempty"`
	ClaimedAt      *time.Time `json:"claimed_at,omitempty"`

	// 4. Developer Seeding & Supplemental Notes
	Notes string `json:"notes,omitempty"`

	// 5. Review & Architectural Drift Handshake
	ReviewReason   string   `json:"review_reason,omitempty"`
	ReviewDetails  string   `json:"review_details,omitempty"`
	ReviewOptions  []string `json:"review_options,omitempty"`
	ReviewResponse string   `json:"review_response,omitempty"`
	ResolvedBy     string   `json:"resolved_by,omitempty"`

	// 6. Integration & Merge Gate
	ConflictFiles []string `json:"conflict_files,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// DependenciesCorrupt reports that the persisted dependency list could not
	// be decoded. It is internal to storage and never serialized; topology-aware
	// operations (claim, transitions) must refuse to act on such a task.
	DependenciesCorrupt bool `json:"-"`
}

// Validate checks task integrity.
func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if t.PlanID == "" {
		return fmt.Errorf("task plan_id cannot be empty")
	}
	if !t.Status.IsValid() {
		return fmt.Errorf("invalid task status: %s", t.Status)
	}
	return nil
}

// ContextPacket is the prompt-ready manifest delivered to an agent upon claim.
type ContextPacket struct {
	Task                 Task       `json:"task"`
	TargetFiles          []string   `json:"target_files"`
	SkillPaths           []string   `json:"skill_paths"`
	InputAttachments     []Artifact `json:"input_attachments"`
	VerificationCommands []string   `json:"verification_commands"`
	PromptMarkdown       string     `json:"prompt_markdown"`
}

// ArtifactType categorizes concrete outputs produced during task execution.
type ArtifactType string

const (
	ArtifactTypeImage    ArtifactType = "IMAGE"
	ArtifactTypeMarkdown ArtifactType = "MARKDOWN"
	ArtifactTypeDoc      ArtifactType = "DOC"
	ArtifactTypeFile     ArtifactType = "FILE"
)

const (
	ArtifactRoleInput  = "INPUT"
	ArtifactRoleOutput = "OUTPUT"
)

// IsValid checks if the artifact type is supported.
func (a ArtifactType) IsValid() bool {
	switch a {
	case ArtifactTypeImage, ArtifactTypeMarkdown, ArtifactTypeDoc, ArtifactTypeFile:
		return true
	default:
		return false
	}
}

// Artifact represents a concrete file input or output tracked via the Claim-Check pattern.
type Artifact struct {
	ID        string       `json:"id"`
	TaskID    string       `json:"task_id"`
	Role      string       `json:"role"` // "INPUT" (spec/mockup) or "OUTPUT" (log/result)
	Name      string       `json:"name"` // Display filename or caption
	Type      ArtifactType `json:"type"`
	FilePath  string       `json:"file_path"`
	MimeType  string       `json:"mime_type"`
	ByteSize  int64        `json:"byte_size"`
	Sha256    string       `json:"sha256"`
	CreatedAt time.Time    `json:"created_at"`
}

// EventMessage is one entry of the append-only activity log. Rows are produced
// by database triggers, so every writer (MCP, CLI, HTTP, raw SQL) is recorded
// and the daemon can stream the same log it shows in the UI.
type EventMessage struct {
	ID        int64     `json:"id"`
	PlanID    string    `json:"plan_id,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	Type      string    `json:"type"` // e.g. "task.created", "task.updated", "plan.archived"
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}
