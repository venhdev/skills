package domain

import (
	"fmt"
	"time"
)

type PlanStatus string

const (
	PlanStatusActive    PlanStatus = "ACTIVE"
	PlanStatusCompleted PlanStatus = "COMPLETED"
	PlanStatusDrifted   PlanStatus = "DRIFTED"
	PlanStatusArchived  PlanStatus = "ARCHIVED"
)

func (s PlanStatus) IsValid() bool {
	switch s {
	case PlanStatusActive, PlanStatusCompleted, PlanStatusDrifted, PlanStatusArchived:
		return true
	default:
		return false
	}
}

// Plan is a living dossier that groups a task DAG inside one project.
type Plan struct {
	Slug        string     `json:"slug"`
	ProjectSlug string     `json:"project_slug"`
	Title       string     `json:"title"`
	Status      PlanStatus `json:"status"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (p *Plan) Validate() error {
	if p.Slug == "" {
		return fmt.Errorf("plan slug cannot be empty")
	}
	if p.ProjectSlug == "" {
		return fmt.Errorf("plan %s must belong to a project", p.Slug)
	}
	if p.Title == "" {
		return fmt.Errorf("plan title cannot be empty")
	}
	if !p.Status.IsValid() {
		return fmt.Errorf("invalid plan status: %s", p.Status)
	}
	return nil
}

// PlanPatch carries partial updates. Nil fields are left untouched.
type PlanPatch struct {
	Title  *string
	Status *PlanStatus
}

// PlanBundle is the atomic authoring shape: a plan plus its initial task DAG.
type PlanBundle struct {
	Plan  Plan
	Tasks []Task
}
