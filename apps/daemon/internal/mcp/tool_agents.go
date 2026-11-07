package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"zenf/internal/domain"
)

func (s *Server) registerAgentTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("agents",
			mcp.WithDescription("Manage reusable worker agent definitions: create, get, list, update, or delete. The MVP stores definitions and schedules; an external runner executes them."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | get | list | update | delete")),
			mcp.WithString("slug", mcp.Description("Agent slug")),
			mcp.WithString("name", mcp.Description("Human-readable agent name")),
			mcp.WithString("persona", mcp.Description("EXPLORER | CODER | REVIEWER | CUSTOM")),
			mcp.WithString("mode", mcp.Description("HEADLESS | INTERACTIVE")),
			mcp.WithString("goal", mcp.Description("What the agent must accomplish")),
			mcp.WithArray("start_conditions", mcp.Description("Conditions that allow a run to start"), mcp.WithStringItems()),
			mcp.WithArray("stop_conditions", mcp.Description("Conditions that end a run"), mcp.WithStringItems()),
			mcp.WithString("schedule", mcp.Description("Beat or cron expression")),
			mcp.WithArray("skills", mcp.Description("Skills assigned to the agent"), mcp.WithStringItems()),
			mcp.WithString("model", mcp.Description("Preferred model label")),
			mcp.WithString("runner", mcp.Description("External runner label")),
			mcp.WithBoolean("enabled", mcp.Description("Whether the definition is active (default true)")),
			mcp.WithBoolean("include_disabled", mcp.Description("Include disabled agents in list")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			action := request.GetString("action", "")
			slug := request.GetString("slug", "")

			switch action {
			case "create":
				if slug == "" || request.GetString("name", "") == "" {
					return toolError("slug and name are required for create"), nil
				}
				agent := domain.Agent{
					Slug:            slug,
					Name:            request.GetString("name", ""),
					Persona:         domain.AgentPersona(strings.ToUpper(request.GetString("persona", string(domain.PersonaCustom)))),
					Mode:            domain.AgentMode(strings.ToUpper(request.GetString("mode", string(domain.AgentModeHeadless)))),
					Goal:            request.GetString("goal", ""),
					StartConditions: request.GetStringSlice("start_conditions", nil),
					StopConditions:  request.GetStringSlice("stop_conditions", nil),
					Schedule:        request.GetString("schedule", ""),
					Skills:          request.GetStringSlice("skills", nil),
					Model:           request.GetString("model", ""),
					Runner:          request.GetString("runner", ""),
					Enabled:         request.GetBool("enabled", true),
				}
				if err := s.store.CreateAgent(ctx, agent); err != nil {
					return toolError("failed to create agent %s: %v", slug, err), nil
				}
				created, err := s.store.GetAgent(ctx, slug)
				if err != nil {
					return toolError("failed to load agent %s: %v", slug, err), nil
				}
				return marshalToolResult(created), nil

			case "get":
				if slug == "" {
					return toolError("slug is required for get"), nil
				}
				agent, err := s.store.GetAgent(ctx, slug)
				if err != nil {
					return toolError("failed to load agent %s: %v", slug, err), nil
				}
				return marshalToolResult(agent), nil

			case "list":
				agents, err := s.store.ListAgents(ctx, request.GetBool("include_disabled", false))
				if err != nil {
					return toolError("failed to list agents: %v", err), nil
				}
				if agents == nil {
					agents = []domain.Agent{}
				}
				return marshalToolResult(agents), nil

			case "update":
				if slug == "" {
					return toolError("slug is required for update"), nil
				}
				patch, err := parseAgentPatch(args)
				if err != nil {
					return toolError("invalid update payload: %v", err), nil
				}
				if err := s.store.UpdateAgent(ctx, slug, patch); err != nil {
					return toolError("failed to update agent %s: %v", slug, err), nil
				}
				updated, err := s.store.GetAgent(ctx, slug)
				if err != nil {
					return toolError("failed to load agent %s: %v", slug, err), nil
				}
				return marshalToolResult(updated), nil

			case "delete":
				if slug == "" {
					return toolError("slug is required for delete"), nil
				}
				if err := s.store.DeleteAgent(ctx, slug); err != nil {
					return toolError("failed to delete agent %s: %v", slug, err), nil
				}
				return marshalToolResult(map[string]any{"slug": slug, "deleted": true}), nil

			default:
				return toolError("unknown action (expected create, get, list, update, or delete)"), nil
			}
		},
	)
}

func parseAgentPatch(args map[string]any) (domain.AgentPatch, error) {
	encoded, err := json.Marshal(args)
	if err != nil {
		return domain.AgentPatch{}, err
	}
	var payload struct {
		Name            *string   `json:"name"`
		Persona         *string   `json:"persona"`
		Mode            *string   `json:"mode"`
		Goal            *string   `json:"goal"`
		StartConditions *[]string `json:"start_conditions"`
		StopConditions  *[]string `json:"stop_conditions"`
		Schedule        *string   `json:"schedule"`
		Skills          *[]string `json:"skills"`
		Model           *string   `json:"model"`
		Runner          *string   `json:"runner"`
		Enabled         *bool     `json:"enabled"`
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return domain.AgentPatch{}, err
	}
	patch := domain.AgentPatch{
		Name:            payload.Name,
		Goal:            payload.Goal,
		StartConditions: payload.StartConditions,
		StopConditions:  payload.StopConditions,
		Schedule:        payload.Schedule,
		Skills:          payload.Skills,
		Model:           payload.Model,
		Runner:          payload.Runner,
		Enabled:         payload.Enabled,
	}
	if payload.Persona != nil {
		persona := domain.AgentPersona(strings.ToUpper(*payload.Persona))
		patch.Persona = &persona
	}
	if payload.Mode != nil {
		mode := domain.AgentMode(strings.ToUpper(*payload.Mode))
		patch.Mode = &mode
	}
	return patch, nil
}

func (s *Server) registerSessionTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("sessions",
			mcp.WithDescription("Register and observe agent sessions. MAIN/SUBAGENT sessions are registered by the main agent; WORKER sessions are registered by an external runner. The web UI only reads them."),
			mcp.WithString("action", mcp.Required(), mcp.Description("register | get | list | heartbeat | close")),
			mcp.WithString("id", mcp.Description("Session id")),
			mcp.WithString("agent_slug", mcp.Description("Agent definition slug (required for WORKER sessions)")),
			mcp.WithString("session_type", mcp.Description("MAIN | SUBAGENT | WORKER")),
			mcp.WithString("parent_id", mcp.Description("Parent session id for SUBAGENT/WORKER")),
			mcp.WithString("external_ref", mcp.Description("Runner-specific reference (PID, job id, thread id)")),
			mcp.WithString("status", mcp.Description("ACTIVE | IDLE | DISCONNECTED | TERMINATED")),
			mcp.WithBoolean("active_only", mcp.Description("List only ACTIVE and IDLE sessions")),
			mcp.WithNumber("limit", mcp.Description("List limit (default 50)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			action := request.GetString("action", "")
			id := request.GetString("id", "")

			switch action {
			case "register":
				if id == "" {
					return toolError("id is required for register"), nil
				}
				session := domain.AgentSession{
					ID:          id,
					AgentSlug:   request.GetString("agent_slug", ""),
					SessionType: domain.SessionType(strings.ToUpper(request.GetString("session_type", ""))),
					ParentID:    request.GetString("parent_id", ""),
					ExternalRef: request.GetString("external_ref", ""),
					Status:      domain.SessionStatus(strings.ToUpper(request.GetString("status", string(domain.SessionActive)))),
				}
				if err := s.store.CreateSession(ctx, session); err != nil {
					return toolError("failed to register session %s: %v", id, err), nil
				}
				created, err := s.store.GetSession(ctx, id)
				if err != nil {
					return toolError("failed to load session %s: %v", id, err), nil
				}
				return marshalToolResult(created), nil

			case "get":
				if id == "" {
					return toolError("id is required for get"), nil
				}
				session, err := s.store.GetSession(ctx, id)
				if err != nil {
					return toolError("failed to load session %s: %v", id, err), nil
				}
				return marshalToolResult(session), nil

			case "list":
				filter := domain.SessionFilter{
					Type:       domain.SessionType(strings.ToUpper(request.GetString("session_type", ""))),
					AgentSlug:  request.GetString("agent_slug", ""),
					Status:     domain.SessionStatus(strings.ToUpper(request.GetString("status", ""))),
					ActiveOnly: request.GetBool("active_only", false),
					Limit:      request.GetInt("limit", 50),
				}
				sessions, err := s.store.ListSessions(ctx, filter)
				if err != nil {
					return toolError("failed to list sessions: %v", err), nil
				}
				if sessions == nil {
					sessions = []domain.AgentSession{}
				}
				return marshalToolResult(sessions), nil

			case "heartbeat":
				if id == "" {
					return toolError("id is required for heartbeat"), nil
				}
				status := domain.SessionStatus(strings.ToUpper(request.GetString("status", "")))
				if err := s.store.HeartbeatSession(ctx, id, status); err != nil {
					return toolError("failed to heartbeat session %s: %v", id, err), nil
				}
				return marshalToolResult(map[string]any{"id": id, "ok": true}), nil

			case "close":
				if id == "" {
					return toolError("id is required for close"), nil
				}
				if err := s.store.CloseSession(ctx, id); err != nil {
					return toolError("failed to close session %s: %v", id, err), nil
				}
				return marshalToolResult(map[string]any{"id": id, "status": domain.SessionTerminated}), nil

			default:
				return toolError("unknown action (expected register, get, list, heartbeat, or close)"), nil
			}
		},
	)
}
