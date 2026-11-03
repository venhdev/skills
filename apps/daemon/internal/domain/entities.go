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
	StatusRunning        TaskStatus = "RUNNING"         // Actively executing under an agent runner
	StatusAwaitingReview TaskStatus = "AWAITING_REVIEW" // Paused: Plan/scope drift detected, awaiting developer decision
	StatusSuspended      TaskStatus = "SUSPENDED"       // Paused: Worker process died unexpectedly or orchestrator lost
	StatusAwaitingMerge  TaskStatus = "AWAITING_MERGE"  // Implemented & tested on worktree, staged for git merge
	StatusConflict       TaskStatus = "CONFLICT"        // Git merge conflict encountered during integration
	StatusBlocked        TaskStatus = "BLOCKED"         // Blocked by upstream failure, environment, or human review
	StatusCompleted      TaskStatus = "COMPLETED"       // Verified with exit code 0 and passing tests
	StatusFailed         TaskStatus = "FAILED"          // Execution failure or test assertion failure
	StatusCancelled      TaskStatus = "CANCELLED"       // Terminated via operator abort or -PGID cascade kill
)

// IsValid checks if the status is a known domain status.
func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusReady, StatusRunning, StatusAwaitingReview,
		StatusSuspended, StatusAwaitingMerge, StatusConflict, StatusBlocked,
		StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status represents a final execution state.
func (s TaskStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusCancelled
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
		return target == StatusAwaitingReview || target == StatusSuspended ||
			target == StatusAwaitingMerge || target == StatusCompleted ||
			target == StatusFailed || target == StatusCancelled || target == StatusBlocked || target == StatusReady
	case StatusAwaitingReview:
		// Resumed back to READY (recommended) or RUNNING, or failed/cancelled
		return target == StatusReady || target == StatusRunning || target == StatusFailed || target == StatusCancelled
	case StatusSuspended:
		// Resumed via reclaim (RUNNING), reset to READY, or cancelled
		return target == StatusReady || target == StatusRunning || target == StatusCancelled || target == StatusFailed
	case StatusAwaitingMerge:
		// Merged to COMPLETED, hit CONFLICT, rejected back to RUNNING, or cancelled
		return target == StatusCompleted || target == StatusConflict ||
			target == StatusRunning || target == StatusCancelled
	case StatusConflict:
		// Assigned back to RUNNING to resolve conflict, or failed/cancelled
		return target == StatusRunning || target == StatusReady || target == StatusFailed || target == StatusCancelled
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
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	RepoPath  string    `json:"repo_path"`
	Status    string    `json:"status"` // ACTIVE, COMPLETED, DRIFTED
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Task represents an atomic vertical slice of execution in the DAG.
type Task struct {
	// 1. Identity & DAG Topology
	ID           string   `json:"id"`
	PlanID       string   `json:"plan_id"`
	RepoPath     string   `json:"repo_path"`
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
	SkillCommand         string         `json:"skill_command"`
	WorktreePath         string         `json:"worktree_path,omitempty"`

	// 3. State & Claim Mutual Exclusion
	Status          TaskStatus `json:"status"`
	ClaimedBy       string     `json:"claimed_by,omitempty"`
	OrchestratorID  string     `json:"orchestrator_id,omitempty"`
	WorkerRole      string     `json:"worker_role,omitempty"`
	PID             int        `json:"pid,omitempty"`
	OrchestratorPID int        `json:"orchestrator_pid,omitempty"`
	ClaimedAt       *time.Time `json:"claimed_at,omitempty"`

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
}

// AgentSession tracks active live agent sessions in the local operating environment.
type AgentSession struct {
	SessionID       string    `json:"session_id"`
	ParentSessionID string    `json:"parent_session_id,omitempty"`
	Role            string    `json:"role"`
	Name            string    `json:"name"`
	PID             int       `json:"pid"`
	Status          string    `json:"status"` // 'ACTIVE', 'DISCONNECTED', 'TERMINATED'
	ClaimedTasks    []string  `json:"claimed_tasks"`
	CreatedAt       time.Time `json:"created_at"`
	LastBeatAt      time.Time `json:"last_beat_at"`
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

// ProjectDoc represents a living architectural document or schema.
type ProjectDoc struct {
	Slug            string    `json:"slug"`
	RepoPath        string    `json:"repo_path"`
	GitBranch       string    `json:"git_branch"`
	Title           string    `json:"title"`
	Category        string    `json:"category"` // ARCHITECTURE, SCHEMA, ADR, GUIDE, MEMORY
	ContentMarkdown string    `json:"content_markdown"`
	MermaidDiagrams []string  `json:"mermaid_diagrams,omitempty"`
	Version         int       `json:"version"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ContextPacket is the prompt-ready manifest delivered to an agent upon claim.
type ContextPacket struct {
	Task                 Task                 `json:"task"`
	TargetFiles          []string             `json:"target_files"`
	SkillPaths           []string             `json:"skill_paths"`
	UpstreamDiffs        []string             `json:"upstream_diffs"`
	InputAttachments     []Artifact           `json:"input_attachments"`
	VerificationCommands []string             `json:"verification_commands"`
	PromptMarkdown       string               `json:"prompt_markdown"`
	LinkedSpecs          []CompactSpecPayload `json:"linked_specs,omitempty"`
}

// CompactSpecPayload represents eagerly resolved architecture context injected into a claimed task.
type CompactSpecPayload struct {
	Slug        string         `json:"slug"`
	Kind        string         `json:"kind"`
	Title       string         `json:"title"`
	DiagramCode string         `json:"diagram_code"`
	Payload     map[string]any `json:"payload"`
}

// TechSpec represents a multi-tiered technical specification (Topology, Dynamics, Persistence, Contracts, ADRs).
type TechSpec struct {
	Slug              string         `json:"slug"`
	RepoPath          string         `json:"repo_path"`
	Tier              int            `json:"tier"` // 1: Topology, 2: Dynamics, 3: Data, 4: Contracts, 5: ADR
	Kind              string         `json:"kind"` // 'CONTEXT', 'FLOWCHART', 'SEQUENCE', 'STATE', 'ERD', 'CONTRACT', 'ADR'
	Title             string         `json:"title"`
	DiagramType       string         `json:"diagram_type"` // 'mermaid:flowchart', 'mermaid:sequenceDiagram', 'mermaid:erDiagram', 'json:schema'
	DiagramCode       string         `json:"diagram_code"` // Pure Mermaid code or schema text
	StructuredPayload map[string]any `json:"structured_payload"`
	Notes             string         `json:"notes"`
	BoundPaths        []string       `json:"bound_paths"` // File or folder paths bound to this spec for auto-linking
	Version           int            `json:"version"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// PlanRevision records an immutable revision record for a plan folder snapshot.
type PlanRevision struct {
	ID        int64     `json:"id"`
	PlanSlug  string    `json:"plan_slug"`
	Revision  string    `json:"revision"` // e.g. "v1.0", "v2.5"
	Author    string    `json:"author"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

// SnapshotFile records an individual file preserved within a plan revision snapshot.
type SnapshotFile struct {
	PlanSlug  string    `json:"plan_slug"`
	Revision  string    `json:"revision"`
	FilePath  string    `json:"file_path"` // Relative path inside plan directory (e.g. "README.md", "01-contracts.md")
	Content   string    `json:"content"`
	ByteSize  int64     `json:"byte_size"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}

// FileDiff represents a computed difference between two plan revision file versions.
type FileDiff struct {
	FilePath   string `json:"file_path"`
	ChangeType string `json:"change_type"` // "ADDED", "MODIFIED", "DELETED", "IDENTICAL"
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

// Run tracks an individual execution attempt of a task under an agent runner.
type Run struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"task_id"`
	RepoPath   string     `json:"repo_path"`
	PID        int        `json:"pid"`
	PGID       int        `json:"pgid"` // Negative PGID used for cascade kill
	Status     TaskStatus `json:"status"`
	LogPath    string     `json:"log_path"` // Claim-check path on disk
	ExitCode   int        `json:"exit_code"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	LastBeatAt time.Time  `json:"last_beat_at"` // Heartbeat timestamp
}

// ArtifactType categorizes concrete outputs produced during task execution.
type ArtifactType string

const (
	ArtifactTypeDiff       ArtifactType = "DIFF"
	ArtifactTypeImage      ArtifactType = "IMAGE"
	ArtifactTypeLog        ArtifactType = "LOG"
	ArtifactTypeTestResult ArtifactType = "TEST_RESULT"
	ArtifactTypeMarkdown   ArtifactType = "MARKDOWN"
	ArtifactTypeDoc        ArtifactType = "DOC"
	ArtifactTypeFile       ArtifactType = "FILE"
)

const (
	ArtifactRoleInput  = "INPUT"
	ArtifactRoleOutput = "OUTPUT"
)

// IsValid checks if the artifact type is supported.
func (a ArtifactType) IsValid() bool {
	switch a {
	case ArtifactTypeDiff, ArtifactTypeImage, ArtifactTypeLog,
		ArtifactTypeTestResult, ArtifactTypeMarkdown, ArtifactTypeDoc, ArtifactTypeFile:
		return true
	default:
		return false
	}
}

// Artifact represents a concrete file input or output tracked via the Claim-Check pattern.
type Artifact struct {
	ID        string       `json:"id"`
	RunID     string       `json:"run_id,omitempty"`
	TaskID    string       `json:"task_id"`
	Role      string       `json:"role"` // "INPUT" (spec/mockup) or "OUTPUT" (diff/log/result)
	Name      string       `json:"name"` // Display filename or caption
	Type      ArtifactType `json:"type"`
	FilePath  string       `json:"file_path"`
	MimeType  string       `json:"mime_type"`
	ByteSize  int64        `json:"byte_size"`
	Sha256    string       `json:"sha256"`
	CreatedAt time.Time    `json:"created_at"`
}

// EventMessage represents an append-only event broadcasted over SSE.
type EventMessage struct {
	ID        int64     `json:"id"`
	RunID     string    `json:"run_id,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	Type      string    `json:"type"` // e.g. "task:status", "run:heartbeat", "artifact:created"
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}
