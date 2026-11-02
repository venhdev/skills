package domain

import (
	"fmt"
	"time"
)

// TaskStatus defines the state machine states for a DAG task slice.
type TaskStatus string

const (
	StatusPending   TaskStatus = "PENDING"   // Waiting for upstream dependencies
	StatusReady     TaskStatus = "READY"     // Dependencies fulfilled, ready to claim
	StatusRunning   TaskStatus = "RUNNING"   // Actively executing under an agent runner
	StatusBlocked   TaskStatus = "BLOCKED"   // Blocked by upstream failure, environment, or human review
	StatusCompleted TaskStatus = "COMPLETED" // Verified with exit code 0 and passing tests
	StatusFailed    TaskStatus = "FAILED"    // Execution failure or test assertion failure
	StatusCancelled TaskStatus = "CANCELLED" // Terminated via operator abort or -PGID cascade kill
)

// IsValid checks if the status is a known domain status.
func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusReady, StatusRunning, StatusBlocked,
		StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status represents a final execution state.
func (s TaskStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCancelled
}

// CanTransitionTo enforces the state machine transition rules.
func (s TaskStatus) CanTransitionTo(target TaskStatus) bool {
	if !s.IsValid() || !target.IsValid() {
		return false
	}
	if s == target {
		return true // No-op idempotent transition
	}
	if s.IsTerminal() {
		return false // Terminal states are immutable
	}

	switch s {
	case StatusPending:
		return target == StatusReady || target == StatusBlocked || target == StatusCancelled
	case StatusReady:
		return target == StatusRunning || target == StatusBlocked || target == StatusCancelled
	case StatusRunning:
		return target == StatusCompleted || target == StatusFailed || target == StatusCancelled || target == StatusBlocked
	case StatusBlocked:
		return target == StatusReady || target == StatusPending || target == StatusCancelled
	default:
		return false
	}
}

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
	ID           string     `json:"id"`
	PlanID       string     `json:"plan_id"`
	RepoPath     string     `json:"repo_path"`
	Title        string     `json:"title"`
	Dependencies []string   `json:"dependencies"` // List of prerequisite Task IDs
	Harness      string     `json:"harness"`      // claude-code, antigravity, shell-script
	SkillCommand string     `json:"skill_command"` // e.g. "/forge task-01"
	WorktreePath string     `json:"worktree_path"` // Isolated git worktree directory
	Status       TaskStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
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
)

// IsValid checks if the artifact type is supported.
func (a ArtifactType) IsValid() bool {
	switch a {
	case ArtifactTypeDiff, ArtifactTypeImage, ArtifactTypeLog,
		ArtifactTypeTestResult, ArtifactTypeMarkdown:
		return true
	default:
		return false
	}
}

// Artifact represents a concrete file output tracked via the Claim-Check pattern.
type Artifact struct {
	ID        string       `json:"id"`
	RunID     string       `json:"run_id"`
	TaskID    string       `json:"task_id"`
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
