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

// TaskScope is the ownership container of a task.
type TaskScope string

const (
	ScopePlan    TaskScope = "PLAN"
	ScopeProject TaskScope = "PROJECT"
	ScopeOrphan  TaskScope = "ORPHAN"
)

// Task represents an atomic vertical slice of execution in a DAG.
type Task struct {
	// 1. Identity, Scope & DAG Topology
	ID           string   `json:"id"`
	ProjectSlug  string   `json:"project_slug,omitempty"`
	PlanSlug     string   `json:"plan_slug,omitempty"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Dependencies []string `json:"dependencies"` // Prerequisite task IDs inside the same scope

	// 2. Triage & Fan-out Attributes
	Priority             TaskPriority   `json:"priority,omitempty"`
	Complexity           TaskComplexity `json:"complexity,omitempty"`
	RiskLevel            TaskRisk       `json:"risk_level,omitempty"`
	AgentSlug            string         `json:"agent_slug,omitempty"`
	RequiredSkills       []string       `json:"required_skills,omitempty"`
	RequiredTools        []string       `json:"required_tools,omitempty"`
	RequiredDocs         []string       `json:"required_docs,omitempty"` // document keys
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
	ClaimSessionID string     `json:"claim_session_id,omitempty"`
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

// Scope derives the ownership container from the persisted slugs.
func (t Task) Scope() TaskScope {
	switch {
	case t.PlanSlug != "":
		return ScopePlan
	case t.ProjectSlug != "":
		return ScopeProject
	default:
		return ScopeOrphan
	}
}

// ContainerKey identifies the dependency namespace a task belongs to.
func (t Task) ContainerKey() string {
	switch t.Scope() {
	case ScopePlan:
		return "plan:" + t.PlanSlug
	case ScopeProject:
		return "project:" + t.ProjectSlug
	default:
		return "orphan"
	}
}

// Validate checks task integrity.
func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if t.Title == "" {
		return fmt.Errorf("task title cannot be empty")
	}
	if t.PlanSlug != "" && t.ProjectSlug == "" {
		return fmt.Errorf("task %s belongs to plan %s without a project", t.ID, t.PlanSlug)
	}
	if !t.Status.IsValid() {
		return fmt.Errorf("invalid task status: %s", t.Status)
	}
	return nil
}

// TaskPatch carries authoring metadata for a partial update. Nil fields are
// left untouched; a non-nil empty slice clears a list, and a non-nil empty
// string clears a text field.
type TaskPatch struct {
	Title                *string
	Description          *string
	Dependencies         *[]string
	Priority             *TaskPriority
	Complexity           *TaskComplexity
	RiskLevel            *TaskRisk
	AgentSlug            *string
	RequiredSkills       *[]string
	RequiredTools        *[]string
	RequiredDocs         *[]string
	TargetFiles          *[]string
	AcceptanceCriteria   *[]string
	VerificationCommands *[]string
	Harness              *string
	WorktreePath         *string
}

// TaskFilter narrows a task query. Empty fields are not applied.
type TaskFilter struct {
	ProjectSlug string
	PlanSlug    string
	Status      TaskStatus
	AgentSlug   string
	OrphanOnly  bool
}

// ClaimOptions carries the identity of the claimer. SessionID links the claim
// to an observed MAIN/SUBAGENT session when the client registers one.
type ClaimOptions struct {
	WorkerID       string
	OrchestratorID string
	WorkerRole     string
	SessionID      string
}

// ContextPacket is the prompt-ready manifest delivered to an agent upon claim.
type ContextPacket struct {
	Task                 Task            `json:"task"`
	Project              *Project        `json:"project,omitempty"`
	Plan                 *Plan           `json:"plan,omitempty"`
	Agent                *Agent          `json:"agent,omitempty"`
	TargetFiles          []string        `json:"target_files"`
	SkillPaths           []string        `json:"skill_paths"`
	InputAttachments     []Artifact      `json:"input_attachments"`
	LinkedDocuments      []DocumentBrief `json:"linked_documents,omitempty"`
	VerificationCommands []string        `json:"verification_commands"`
	PromptMarkdown       string          `json:"prompt_markdown"`
}
