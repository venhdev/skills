package domain

import (
	"fmt"
	"time"
)

type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "ACTIVE"
	ProjectStatusArchived ProjectStatus = "ARCHIVED"
)

func (s ProjectStatus) IsValid() bool {
	return s == ProjectStatusActive || s == ProjectStatusArchived
}

// Project is the top-level working environment. It isolates plans, tasks,
// documents, and agent definitions.
type Project struct {
	Slug        string        `json:"slug"`
	Name        string        `json:"name"`
	RootPath    string        `json:"root_path,omitempty"`
	Description string        `json:"description,omitempty"`
	Status      ProjectStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (p *Project) Validate() error {
	if p.Slug == "" {
		return fmt.Errorf("project slug cannot be empty")
	}
	if p.Name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if !p.Status.IsValid() {
		return fmt.Errorf("invalid project status: %s", p.Status)
	}
	return nil
}

// ProjectPatch carries partial updates. Nil fields are left untouched.
type ProjectPatch struct {
	Name        *string
	RootPath    *string
	Description *string
	Status      *ProjectStatus
}
