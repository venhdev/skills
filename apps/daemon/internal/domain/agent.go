package domain

import (
	"fmt"
	"time"
)

type AgentPersona string

const (
	PersonaExplorer AgentPersona = "EXPLORER"
	PersonaCoder    AgentPersona = "CODER"
	PersonaReviewer AgentPersona = "REVIEWER"
	PersonaCustom   AgentPersona = "CUSTOM"
)

func (p AgentPersona) IsValid() bool {
	switch p {
	case PersonaExplorer, PersonaCoder, PersonaReviewer, PersonaCustom:
		return true
	default:
		return false
	}
}

type AgentMode string

const (
	AgentModeHeadless    AgentMode = "HEADLESS"
	AgentModeInteractive AgentMode = "INTERACTIVE"
)

func (m AgentMode) IsValid() bool {
	return m == AgentModeHeadless || m == AgentModeInteractive
}

// Agent is a reusable worker definition. The MVP stores the definition and its
// schedule; an external runner executes it.
type Agent struct {
	Slug            string       `json:"slug"`
	Name            string       `json:"name"`
	Persona         AgentPersona `json:"persona"`
	Mode            AgentMode    `json:"mode"`
	Goal            string       `json:"goal,omitempty"`
	StartConditions []string     `json:"start_conditions,omitempty"`
	StopConditions  []string     `json:"stop_conditions,omitempty"`
	Schedule        string       `json:"schedule,omitempty"`
	Skills          []string     `json:"skills,omitempty"`
	Model           string       `json:"model,omitempty"`
	Runner          string       `json:"runner,omitempty"`
	Enabled         bool         `json:"enabled"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

func (a *Agent) Validate() error {
	if a.Slug == "" {
		return fmt.Errorf("agent slug cannot be empty")
	}
	if a.Name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	if !a.Persona.IsValid() {
		return fmt.Errorf("invalid agent persona: %s", a.Persona)
	}
	if !a.Mode.IsValid() {
		return fmt.Errorf("invalid agent mode: %s", a.Mode)
	}
	return nil
}

// AgentPatch carries partial updates. Nil fields are left untouched.
type AgentPatch struct {
	Name            *string
	Persona         *AgentPersona
	Mode            *AgentMode
	Goal            *string
	StartConditions *[]string
	StopConditions  *[]string
	Schedule        *string
	Skills          *[]string
	Model           *string
	Runner          *string
	Enabled         *bool
}

type SessionType string

const (
	SessionMain     SessionType = "MAIN"
	SessionSubagent SessionType = "SUBAGENT"
	SessionWorker   SessionType = "WORKER"
)

func (t SessionType) IsValid() bool {
	return t == SessionMain || t == SessionSubagent || t == SessionWorker
}

type SessionStatus string

const (
	SessionActive       SessionStatus = "ACTIVE"
	SessionIdle         SessionStatus = "IDLE"
	SessionDisconnected SessionStatus = "DISCONNECTED"
	SessionTerminated   SessionStatus = "TERMINATED"
)

func (s SessionStatus) IsValid() bool {
	switch s {
	case SessionActive, SessionIdle, SessionDisconnected, SessionTerminated:
		return true
	default:
		return false
	}
}

// AgentSession records an observed or scheduled agent session. MAIN/SUBAGENT
// sessions are registered by the main agent through MCP; WORKER sessions are
// registered by an external runner. The web UI only reads these rows.
type AgentSession struct {
	ID          string        `json:"id"`
	AgentSlug   string        `json:"agent_slug,omitempty"`
	SessionType SessionType   `json:"session_type"`
	ParentID    string        `json:"parent_id,omitempty"`
	ExternalRef string        `json:"external_ref,omitempty"`
	Status      SessionStatus `json:"status"`
	StartedAt   time.Time     `json:"started_at"`
	LastSeenAt  time.Time     `json:"last_seen_at"`
	EndedAt     *time.Time    `json:"ended_at,omitempty"`
}

func (s *AgentSession) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("session id cannot be empty")
	}
	if !s.SessionType.IsValid() {
		return fmt.Errorf("invalid session type: %s", s.SessionType)
	}
	if !s.Status.IsValid() {
		return fmt.Errorf("invalid session status: %s", s.Status)
	}
	if s.SessionType == SessionWorker && s.AgentSlug == "" {
		return fmt.Errorf("worker session %s requires agent_slug", s.ID)
	}
	return nil
}

// SessionFilter narrows a session query. Empty fields are not applied.
type SessionFilter struct {
	Type      SessionType
	AgentSlug string
	Status    SessionStatus
	ActiveOnly bool
	Limit     int
}
