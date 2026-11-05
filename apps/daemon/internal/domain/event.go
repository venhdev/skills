package domain

import "time"

// EventMessage is one entry of the append-only activity log. Rows are produced
// by database triggers, so every writer (MCP, CLI, HTTP, raw SQL) is recorded
// and the daemon can stream the same log it shows in the UI.
type EventMessage struct {
	ID          int64     `json:"id"`
	ProjectSlug string    `json:"project_slug,omitempty"`
	PlanSlug    string    `json:"plan_slug,omitempty"`
	TaskID      string    `json:"task_id,omitempty"`
	EntityType  string    `json:"entity_type"`
	EntityID    string    `json:"entity_id"`
	Type        string    `json:"type"` // e.g. "task.created", "document.updated"
	Actor       string    `json:"actor,omitempty"`
	Payload     string    `json:"payload"`
	Timestamp   time.Time `json:"timestamp"`
}
