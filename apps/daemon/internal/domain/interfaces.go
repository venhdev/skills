package domain

import "context"

// ProjectRepository persists the top-level working environments.
type ProjectRepository interface {
	CreateProject(ctx context.Context, project Project) error
	GetProject(ctx context.Context, slug string) (*Project, error)
	ListProjects(ctx context.Context, includeArchived bool) ([]Project, error)
	UpdateProject(ctx context.Context, slug string, patch ProjectPatch) error
	ArchiveProject(ctx context.Context, slug string, archived bool) error
	DeleteProject(ctx context.Context, slug string) error
}

// PlanRepository persists plans and the atomic plan/task authoring bundle.
type PlanRepository interface {
	// CreatePlanWithTasks writes a plan and its tasks in one atomic transaction.
	CreatePlanWithTasks(ctx context.Context, plan Plan, tasks []Task) error
	GetPlan(ctx context.Context, slug string) (*Plan, error)
	// ListPlans filters by project when projectSlug is non-empty.
	ListPlans(ctx context.Context, projectSlug string, includeArchived bool) ([]Plan, error)
	UpdatePlan(ctx context.Context, slug string, patch PlanPatch) error
	ArchivePlan(ctx context.Context, slug string, archived bool) error
	DeletePlan(ctx context.Context, slug string) error
}

// TaskRepository persists task records and their read projections.
type TaskRepository interface {
	GetTask(ctx context.Context, taskID string) (*Task, error)
	GetTasksByPlan(ctx context.Context, planSlug string) ([]Task, error)
	ListTasks(ctx context.Context, filter TaskFilter) ([]Task, error)
	CreateTasks(ctx context.Context, tasks []Task) error
	UpdateTaskMetadata(ctx context.Context, taskID string, patch TaskPatch) error
	UpdateTaskDependencies(ctx context.Context, taskID string, deps []string) error
	UpdateTaskNotes(ctx context.Context, taskID string, notes string) error
	AppendTaskNotes(ctx context.Context, taskID string, note string) error
}

// TaskLifecycle owns the task state machine, claim mutual exclusion, and the
// review handshake.
type TaskLifecycle interface {
	UpdateTaskStatus(ctx context.Context, taskID string, from TaskStatus, to TaskStatus) error
	UpdateTaskStatusWithNote(ctx context.Context, taskID string, from TaskStatus, to TaskStatus, note string) error
	ClaimTask(ctx context.Context, taskID string, opts ClaimOptions) (*ContextPacket, error)
	EvictTask(ctx context.Context, taskID string, reason string, action string) (TaskStatus, error)
	RequestReview(ctx context.Context, taskID string, reason string, details string, options []string) error
	ResolveReview(ctx context.Context, taskID string, response string) error
	GetTaskContextPacket(ctx context.Context, taskID string) (*ContextPacket, error)
}

// DocumentRepository persists the file-canonical document registry.
type DocumentRepository interface {
	CreateDocument(ctx context.Context, document Document) error
	GetDocument(ctx context.Context, id string) (*Document, error)
	GetDocumentByKey(ctx context.Context, projectSlug string, key string) (*Document, error)
	GetDocumentBySlug(ctx context.Context, projectSlug string, slug string) (*Document, error)
	ListDocuments(ctx context.Context, filter DocumentFilter) ([]Document, error)
	UpdateDocument(ctx context.Context, id string, patch DocumentPatch) error
	RefreshDocument(ctx context.Context, id string, content string, contentHash string, byteSize int64) error
	ArchiveDocument(ctx context.Context, id string, archived bool) error
	DeleteDocument(ctx context.Context, id string) error
	SearchDocuments(ctx context.Context, query string, filter DocumentFilter) ([]DocumentSearchResult, error)
	CreateDocumentLink(ctx context.Context, link DocumentLink) error
	ListDocumentLinks(ctx context.Context, fromID string) ([]DocumentLink, error)
}

// AgentRepository persists reusable worker definitions.
type AgentRepository interface {
	CreateAgent(ctx context.Context, agent Agent) error
	GetAgent(ctx context.Context, slug string) (*Agent, error)
	ListAgents(ctx context.Context, includeDisabled bool) ([]Agent, error)
	UpdateAgent(ctx context.Context, slug string, patch AgentPatch) error
	DeleteAgent(ctx context.Context, slug string) error
}

// SessionRepository persists observed MAIN/SUBAGENT sessions and WORKER
// sessions registered by external runners.
type SessionRepository interface {
	CreateSession(ctx context.Context, session AgentSession) error
	GetSession(ctx context.Context, id string) (*AgentSession, error)
	ListSessions(ctx context.Context, filter SessionFilter) ([]AgentSession, error)
	HeartbeatSession(ctx context.Context, id string, status SessionStatus) error
	CloseSession(ctx context.Context, id string) error
}

// ArtifactRepository persists artifact metadata following the claim-check
// pattern.
type ArtifactRepository interface {
	RecordArtifact(ctx context.Context, artifact Artifact) error
	GetArtifact(ctx context.Context, id string) (*Artifact, error)
	GetArtifactsByTask(ctx context.Context, taskID string, role string) ([]Artifact, error)
	DeleteArtifact(ctx context.Context, taskID string, artifactID string) (*Artifact, error)
}

// ArtifactReader is the narrow read seam used by context packet assembly.
type ArtifactReader interface {
	GetArtifactsByTask(ctx context.Context, taskID string, role string) ([]Artifact, error)
}

// ProjectReader is the narrow read seam used by context packet assembly.
type ProjectReader interface {
	GetProject(ctx context.Context, slug string) (*Project, error)
}

// PlanReader is the narrow read seam used by context packet assembly.
type PlanReader interface {
	GetPlan(ctx context.Context, slug string) (*Plan, error)
}

// DocumentReader is the narrow read seam used by context packet assembly.
type DocumentReader interface {
	GetDocumentByKey(ctx context.Context, projectSlug string, key string) (*Document, error)
}

// AgentReader is the narrow read seam used by context packet assembly.
type AgentReader interface {
	GetAgent(ctx context.Context, slug string) (*Agent, error)
}

// EventRepository reads the trigger-produced activity log.
type EventRepository interface {
	ListEvents(ctx context.Context, afterID int64, limit int) ([]EventMessage, error)
	LatestEventID(ctx context.Context) (int64, error)
}

// StateFingerprinter summarizes state so callers can detect writes made by
// another process.
type StateFingerprinter interface {
	StateFingerprint(ctx context.Context) (string, error)
}

// SearchRepository provides cross-entity search.
type SearchRepository interface {
	Search(ctx context.Context, query string, projectSlug string, limit int) ([]SearchResult, error)
}

// ContextPacketBuilder assembles the prompt-ready packet for a task.
type ContextPacketBuilder interface {
	Build(ctx context.Context, task Task) (*ContextPacket, error)
}
