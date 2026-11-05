package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"zenfd/internal/attachment"
	"zenfd/internal/domain"
)

// Storage is the subset of persistence the MCP tool surface needs. It composes
// per-aggregate interfaces so each repository stays independently replaceable.
type Storage interface {
	domain.ProjectRepository
	domain.PlanRepository
	domain.TaskRepository
	domain.TaskLifecycle
	domain.DocumentRepository
	domain.AgentRepository
	domain.SessionRepository
	domain.ArtifactRepository
	domain.SearchRepository
}

// Server implements a Model Context Protocol (MCP) server for zenfd.
type Server struct {
	mcpServer *server.MCPServer
	store     Storage
}

// NewServer initializes the MCP server and registers all tools.
func NewServer(store Storage) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer("zenfd", "0.2.0", server.WithToolCapabilities(true)),
		store:     store,
	}
	s.registerTools()
	return s
}

// ServeStdio launches the stdio transport. CRITICAL: Redirects all standard
// logging to stderr so stdout is reserved for JSON-RPC 2.0 messages.
func (s *Server) ServeStdio() error {
	log.SetOutput(os.Stderr)
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) registerTools() {
	s.registerProjectTools()
	s.registerPlanTools()
	s.registerTaskTools()
	s.registerDocumentTools()
	s.registerAgentTools()
	s.registerSessionTools()
	s.registerSearchTool()
	s.registerWorkflowTools()
}

// marshalToolResult renders a tool response as indented JSON so agents get a
// stable, parseable payload.
func marshalToolResult(value any) *mcp.CallToolResult {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to encode result: %v", err))
	}
	return mcp.NewToolResultText(string(encoded))
}

func toolError(format string, args ...any) *mcp.CallToolResult {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...))
}

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

func (s *Server) registerProjectTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("projects",
			mcp.WithDescription("Manage projects (top-level working environments): create, get, list, update, archive, unarchive, or delete."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | get | list | update | archive | unarchive | delete")),
			mcp.WithString("slug", mcp.Description("Project slug")),
			mcp.WithString("name", mcp.Description("Human-readable project name")),
			mcp.WithString("root_path", mcp.Description("Absolute path of the project working tree")),
			mcp.WithString("description", mcp.Description("Project purpose and boundaries")),
			mcp.WithString("status", mcp.Description("ACTIVE or ARCHIVED")),
			mcp.WithBoolean("include_archived", mcp.Description("Include archived projects in list (default false)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			action := request.GetString("action", "")
			slug := request.GetString("slug", "")

			switch action {
			case "create":
				if slug == "" || request.GetString("name", "") == "" {
					return toolError("slug and name are required for create"), nil
				}
				project := domain.Project{
					Slug:        slug,
					Name:        request.GetString("name", ""),
					RootPath:    request.GetString("root_path", ""),
					Description: request.GetString("description", ""),
					Status:      domain.ProjectStatus(request.GetString("status", string(domain.ProjectStatusActive))),
				}
				if err := s.store.CreateProject(ctx, project); err != nil {
					return toolError("failed to create project %s: %v", slug, err), nil
				}
				created, err := s.store.GetProject(ctx, slug)
				if err != nil {
					return toolError("failed to load project %s: %v", slug, err), nil
				}
				return marshalToolResult(created), nil

			case "get":
				if slug == "" {
					return toolError("slug is required for get"), nil
				}
				project, err := s.store.GetProject(ctx, slug)
				if err != nil {
					return toolError("failed to load project %s: %v", slug, err), nil
				}
				return marshalToolResult(project), nil

			case "list":
				projects, err := s.store.ListProjects(ctx, request.GetBool("include_archived", false))
				if err != nil {
					return toolError("failed to list projects: %v", err), nil
				}
				if projects == nil {
					projects = []domain.Project{}
				}
				return marshalToolResult(projects), nil

			case "update":
				if slug == "" {
					return toolError("slug is required for update"), nil
				}
				patch, err := parseProjectPatch(request.GetArguments())
				if err != nil {
					return toolError("invalid update payload: %v", err), nil
				}
				if err := s.store.UpdateProject(ctx, slug, patch); err != nil {
					return toolError("failed to update project %s: %v", slug, err), nil
				}
				updated, err := s.store.GetProject(ctx, slug)
				if err != nil {
					return toolError("failed to load project %s: %v", slug, err), nil
				}
				return marshalToolResult(updated), nil

			case "archive", "unarchive":
				if slug == "" {
					return toolError("slug is required for %s", action), nil
				}
				archived := action == "archive"
				if err := s.store.ArchiveProject(ctx, slug, archived); err != nil {
					return toolError("failed to %s project %s: %v", action, slug, err), nil
				}
				return marshalToolResult(map[string]any{"slug": slug, "archived": archived}), nil

			case "delete":
				if slug == "" {
					return toolError("slug is required for delete"), nil
				}
				if err := s.store.DeleteProject(ctx, slug); err != nil {
					return toolError("failed to delete project %s: %v", slug, err), nil
				}
				return marshalToolResult(map[string]any{"slug": slug, "deleted": true}), nil

			default:
				return toolError("unknown action (expected create, get, list, update, archive, unarchive, or delete)"), nil
			}
		},
	)
}

func parseProjectPatch(args map[string]any) (domain.ProjectPatch, error) {
	encoded, err := json.Marshal(args)
	if err != nil {
		return domain.ProjectPatch{}, err
	}
	var payload struct {
		Name        *string `json:"name"`
		RootPath    *string `json:"root_path"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return domain.ProjectPatch{}, err
	}
	patch := domain.ProjectPatch{
		Name:        payload.Name,
		RootPath:    payload.RootPath,
		Description: payload.Description,
	}
	if payload.Status != nil {
		status := domain.ProjectStatus(*payload.Status)
		patch.Status = &status
	}
	return patch, nil
}

// ---------------------------------------------------------------------------
// Plans
// ---------------------------------------------------------------------------

func (s *Server) registerPlanTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("plans",
			mcp.WithDescription("Manage plans inside a project: create (with an optional tasks[] DAG in one transaction), get, list, update, archive, unarchive, or delete."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | get | list | update | archive | unarchive | delete")),
			mcp.WithString("slug", mcp.Description("Plan slug")),
			mcp.WithString("project_slug", mcp.Description("Owning project slug (required for create; optional list filter)")),
			mcp.WithString("title", mcp.Description("Human-readable plan title (defaults to slug)")),
			mcp.WithString("status", mcp.Description("ACTIVE, COMPLETED, DRIFTED, or ARCHIVED")),
			mcp.WithBoolean("include_archived", mcp.Description("Include archived plans in list (default false)")),
			mcp.WithArray("tasks", mcp.Description("Optional task DAG to create with the plan. Each item follows the tasks create shape, including media[]."), mcp.Items(taskItemSchema())),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			action := request.GetString("action", "")
			slug := request.GetString("slug", "")
			projectSlug := request.GetString("project_slug", "")

			switch action {
			case "create":
				if slug == "" || projectSlug == "" {
					return toolError("slug and project_slug are required for create"), nil
				}
				plan := domain.Plan{
					Slug:        slug,
					ProjectSlug: projectSlug,
					Title:       request.GetString("title", slug),
					Status:      planStatusOrDefault(request.GetString("status", "")),
				}

				var tasks []domain.Task
				mediaByTask := map[string][]mediaInput{}
				if raw, ok := args["tasks"]; ok {
					items, ok := raw.([]any)
					if !ok {
						return toolError("tasks must be an array"), nil
					}
					parsed, media, err := parseTaskInputs(projectSlug, slug, items)
					if err != nil {
						return toolError("%v", err), nil
					}
					tasks, mediaByTask = parsed, media
				}

				if err := s.store.CreatePlanWithTasks(ctx, plan, tasks); err != nil {
					return toolError("failed to create plan %s: %v", slug, err), nil
				}

				created := make([]string, len(tasks))
				for i, task := range tasks {
					created[i] = task.ID
				}
				return marshalToolResult(map[string]any{
					"plan":     plan,
					"task_ids": created,
					"media":    s.attachMediaForTasks(ctx, mediaByTask),
				}), nil

			case "get":
				if slug == "" {
					return toolError("slug is required for get"), nil
				}
				plan, err := s.store.GetPlan(ctx, slug)
				if err != nil {
					return toolError("failed to load plan %s: %v", slug, err), nil
				}
				return marshalToolResult(plan), nil

			case "list":
				plans, err := s.store.ListPlans(ctx, projectSlug, request.GetBool("include_archived", false))
				if err != nil {
					return toolError("failed to list plans: %v", err), nil
				}
				if plans == nil {
					plans = []domain.Plan{}
				}
				return marshalToolResult(plans), nil

			case "update":
				if slug == "" {
					return toolError("slug is required for update"), nil
				}
				patch, err := parsePlanPatch(request.GetArguments())
				if err != nil {
					return toolError("invalid update payload: %v", err), nil
				}
				if err := s.store.UpdatePlan(ctx, slug, patch); err != nil {
					return toolError("failed to update plan %s: %v", slug, err), nil
				}
				updated, err := s.store.GetPlan(ctx, slug)
				if err != nil {
					return toolError("failed to load plan %s: %v", slug, err), nil
				}
				return marshalToolResult(updated), nil

			case "archive", "unarchive":
				if slug == "" {
					return toolError("slug is required for %s", action), nil
				}
				archived := action == "archive"
				if err := s.store.ArchivePlan(ctx, slug, archived); err != nil {
					return toolError("failed to %s plan %s: %v", action, slug, err), nil
				}
				return marshalToolResult(map[string]any{"slug": slug, "archived": archived}), nil

			case "delete":
				if slug == "" {
					return toolError("slug is required for delete"), nil
				}
				if err := s.store.DeletePlan(ctx, slug); err != nil {
					return toolError("failed to delete plan %s: %v", slug, err), nil
				}
				return marshalToolResult(map[string]any{"slug": slug, "deleted": true}), nil

			default:
				return toolError("unknown action (expected create, get, list, update, archive, unarchive, or delete)"), nil
			}
		},
	)
}

func planStatusOrDefault(raw string) domain.PlanStatus {
	status := domain.PlanStatus(strings.ToUpper(strings.TrimSpace(raw)))
	if status.IsValid() {
		return status
	}
	return domain.PlanStatusActive
}

func parsePlanPatch(args map[string]any) (domain.PlanPatch, error) {
	encoded, err := json.Marshal(args)
	if err != nil {
		return domain.PlanPatch{}, err
	}
	var payload struct {
		Title  *string `json:"title"`
		Status *string `json:"status"`
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return domain.PlanPatch{}, err
	}
	patch := domain.PlanPatch{Title: payload.Title}
	if payload.Status != nil {
		status := domain.PlanStatus(*payload.Status)
		patch.Status = &status
	}
	return patch, nil
}

// ---------------------------------------------------------------------------
// Tasks
// ---------------------------------------------------------------------------

// mediaInput is a local file to copy into the attachment store.
type mediaInput struct {
	FilePath string `json:"file_path"`
	Name     string `json:"name"`
}

// mediaResult reports one attach attempt.
type mediaResult struct {
	TaskID   string           `json:"task_id"`
	FilePath string           `json:"file_path,omitempty"`
	Artifact *domain.Artifact `json:"artifact,omitempty"`
	Error    string           `json:"error,omitempty"`
}

// mediaRemoveResult reports one detach attempt.
type mediaRemoveResult struct {
	TaskID     string `json:"task_id"`
	ArtifactID string `json:"artifact_id"`
	Removed    bool   `json:"removed"`
	Error      string `json:"error,omitempty"`
}

// taskInput is the wire shape accepted by tasks create and plans create.
type taskInput struct {
	ID                   string       `json:"id"`
	Title                string       `json:"title"`
	Description          string       `json:"description"`
	Dependencies         []string     `json:"dependencies"`
	TargetFiles          []string     `json:"target_files"`
	Skills               []string     `json:"skills"`
	Tools                []string     `json:"tools"`
	Docs                 []string     `json:"docs"`
	AcceptanceCriteria   []string     `json:"acceptance_criteria"`
	VerificationCommands []string     `json:"verification_commands"`
	Priority             string       `json:"priority"`
	Complexity           string       `json:"complexity"`
	RiskLevel            string       `json:"risk_level"`
	AgentSlug            string       `json:"agent_slug"`
	Harness              string       `json:"harness"`
	WorktreePath         string       `json:"worktree_path"`
	Media                []mediaInput `json:"media"`
}

func toDomainTask(projectSlug, planSlug string, in taskInput) domain.Task {
	priority := domain.TaskPriority(in.Priority)
	if priority == "" {
		priority = domain.PriorityP2
	}
	complexity := domain.TaskComplexity(in.Complexity)
	if complexity == "" {
		complexity = domain.ComplexityMedium
	}
	risk := domain.TaskRisk(in.RiskLevel)
	if risk == "" {
		risk = domain.RiskLow
	}
	harness := in.Harness
	if harness == "" {
		harness = "orchestrator"
	}
	status := domain.StatusPending
	if len(in.Dependencies) == 0 {
		status = domain.StatusReady
	}
	return domain.Task{
		ID:                   in.ID,
		ProjectSlug:          projectSlug,
		PlanSlug:             planSlug,
		Title:                in.Title,
		Description:          in.Description,
		Dependencies:         in.Dependencies,
		TargetFiles:          in.TargetFiles,
		RequiredSkills:       in.Skills,
		RequiredTools:        in.Tools,
		RequiredDocs:         in.Docs,
		AcceptanceCriteria:   in.AcceptanceCriteria,
		VerificationCommands: in.VerificationCommands,
		Priority:             priority,
		Complexity:           complexity,
		RiskLevel:            risk,
		AgentSlug:            in.AgentSlug,
		Harness:              harness,
		WorktreePath:         in.WorktreePath,
		Status:               status,
	}
}

func mediaItemSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{"type": "string"},
			"name":      map[string]any{"type": "string"},
		},
		"required": []string{"file_path"},
	}
}

// taskItemSchema is shared by plans create and tasks create.
func taskItemSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":                    map[string]any{"type": "string"},
			"title":                 map[string]any{"type": "string"},
			"description":           map[string]any{"type": "string"},
			"dependencies":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"target_files":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"skills":                map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"tools":                 map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"docs":                  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"acceptance_criteria":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"verification_commands": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"priority":              map[string]any{"type": "string", "enum": []string{"P0", "P1", "P2", "P3"}},
			"complexity":            map[string]any{"type": "string", "enum": []string{"LOW", "MEDIUM", "HIGH", "SPIKE"}},
			"risk_level":            map[string]any{"type": "string", "enum": []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}},
			"agent_slug":            map[string]any{"type": "string"},
			"harness":               map[string]any{"type": "string"},
			"worktree_path":         map[string]any{"type": "string"},
			"media":                 map[string]any{"type": "array", "items": mediaItemSchema()},
		},
		"required": []string{"id", "title"},
	}
}

func parseTaskInputs(projectSlug, planSlug string, raw []any) ([]domain.Task, map[string][]mediaInput, error) {
	tasks := make([]domain.Task, 0, len(raw))
	media := make(map[string][]mediaInput)

	for i, item := range raw {
		encoded, err := json.Marshal(item)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid task at index %d: %v", i, err)
		}
		var in taskInput
		if err := json.Unmarshal(encoded, &in); err != nil {
			return nil, nil, fmt.Errorf("invalid task at index %d: %v", i, err)
		}
		if in.ID == "" || in.Title == "" {
			return nil, nil, fmt.Errorf("task at index %d requires id and title", i)
		}
		tasks = append(tasks, toDomainTask(projectSlug, planSlug, in))
		if len(in.Media) > 0 {
			media[in.ID] = in.Media
		}
	}
	return tasks, media, nil
}

func (s *Server) attachMedia(ctx context.Context, taskID string, items []mediaInput) []mediaResult {
	results := make([]mediaResult, 0, len(items))
	for _, item := range items {
		if item.FilePath == "" {
			results = append(results, mediaResult{TaskID: taskID, Error: "file_path is required"})
			continue
		}

		artifact, err := attachment.Ingest(taskID, item.FilePath, item.Name)
		if err != nil {
			results = append(results, mediaResult{TaskID: taskID, FilePath: item.FilePath, Error: err.Error()})
			continue
		}
		if err := s.store.RecordArtifact(ctx, artifact); err != nil {
			_ = os.Remove(artifact.FilePath)
			results = append(results, mediaResult{TaskID: taskID, FilePath: item.FilePath, Error: err.Error()})
			continue
		}
		results = append(results, mediaResult{TaskID: taskID, FilePath: item.FilePath, Artifact: &artifact})
	}
	return results
}

func (s *Server) attachMediaForTasks(ctx context.Context, media map[string][]mediaInput) []mediaResult {
	results := []mediaResult{}
	if len(media) == 0 {
		return results
	}

	ids := make([]string, 0, len(media))
	for id := range media {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		results = append(results, s.attachMedia(ctx, id, media[id])...)
	}
	return results
}

func (s *Server) removeMedia(ctx context.Context, taskID string, ids []string) []mediaRemoveResult {
	results := make([]mediaRemoveResult, 0, len(ids))
	for _, id := range ids {
		artifact, err := s.store.DeleteArtifact(ctx, taskID, id)
		if err != nil {
			results = append(results, mediaRemoveResult{TaskID: taskID, ArtifactID: id, Error: err.Error()})
			continue
		}
		if err := attachment.Remove(artifact.FilePath); err != nil {
			results = append(results, mediaRemoveResult{TaskID: taskID, ArtifactID: id, Error: err.Error()})
			continue
		}
		results = append(results, mediaRemoveResult{TaskID: taskID, ArtifactID: id, Removed: true})
	}
	return results
}

type taskUpdatePayload struct {
	Title                *string      `json:"title"`
	Description          *string      `json:"description"`
	Dependencies         *[]string    `json:"dependencies"`
	TargetFiles          *[]string    `json:"target_files"`
	Skills               *[]string    `json:"skills"`
	Tools                *[]string    `json:"tools"`
	Docs                 *[]string    `json:"docs"`
	AcceptanceCriteria   *[]string    `json:"acceptance_criteria"`
	VerificationCommands *[]string    `json:"verification_commands"`
	Priority             *string      `json:"priority"`
	Complexity           *string      `json:"complexity"`
	RiskLevel            *string      `json:"risk_level"`
	AgentSlug            *string      `json:"agent_slug"`
	Harness              *string      `json:"harness"`
	WorktreePath         *string      `json:"worktree_path"`
	MediaAdd             []mediaInput `json:"media_add"`
	MediaRemove          []string     `json:"media_remove"`
}

func parseTaskPatch(args map[string]any) (domain.TaskPatch, []mediaInput, []string, error) {
	encoded, err := json.Marshal(args)
	if err != nil {
		return domain.TaskPatch{}, nil, nil, err
	}
	var payload taskUpdatePayload
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return domain.TaskPatch{}, nil, nil, err
	}

	patch := domain.TaskPatch{
		Title:                payload.Title,
		Description:          payload.Description,
		Dependencies:         payload.Dependencies,
		RequiredSkills:       payload.Skills,
		RequiredTools:        payload.Tools,
		RequiredDocs:         payload.Docs,
		TargetFiles:          payload.TargetFiles,
		AcceptanceCriteria:   payload.AcceptanceCriteria,
		VerificationCommands: payload.VerificationCommands,
		AgentSlug:            payload.AgentSlug,
		Harness:              payload.Harness,
		WorktreePath:         payload.WorktreePath,
	}
	if payload.Priority != nil {
		value := domain.TaskPriority(*payload.Priority)
		patch.Priority = &value
	}
	if payload.Complexity != nil {
		value := domain.TaskComplexity(*payload.Complexity)
		patch.Complexity = &value
	}
	if payload.RiskLevel != nil {
		value := domain.TaskRisk(*payload.RiskLevel)
		patch.RiskLevel = &value
	}
	return patch, payload.MediaAdd, payload.MediaRemove, nil
}

func taskPatchEmpty(patch domain.TaskPatch) bool {
	return patch.Title == nil && patch.Description == nil && patch.Dependencies == nil &&
		patch.Priority == nil && patch.Complexity == nil && patch.RiskLevel == nil &&
		patch.RequiredSkills == nil && patch.RequiredTools == nil && patch.RequiredDocs == nil &&
		patch.TargetFiles == nil && patch.AcceptanceCriteria == nil && patch.VerificationCommands == nil &&
		patch.AgentSlug == nil && patch.Harness == nil && patch.WorktreePath == nil
}

func (s *Server) registerTaskTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("tasks",
			mcp.WithDescription("Manage tasks: create a batch with optional media, get metadata or the ContextPacket, list by project/plan/status/agent, or update authoring metadata and media. Prefer one batch create over many calls."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | get | list | update")),
			mcp.WithString("project_slug", mcp.Description("Project scope. For create: project task when plan_slug is empty, or the plan's project when plan_slug is set. Optional list filter.")),
			mcp.WithString("plan_slug", mcp.Description("Plan scope for create or list. Omit for a project task or orphan task.")),
			mcp.WithArray("tasks", mcp.Description("Batch of tasks to create. Each item accepts id, title, and optional metadata plus media[]."), mcp.Items(taskItemSchema())),
			mcp.WithString("task_id", mcp.Description("Task ID (required for get and update)")),
			mcp.WithString("status", mcp.Description("Optional status filter for list")),
			mcp.WithString("agent_slug", mcp.Description("Optional agent filter for list")),
			mcp.WithBoolean("orphan", mcp.Description("List only orphan tasks (no project and no plan)")),
			mcp.WithBoolean("include_packet", mcp.Description("For get: return the full ContextPacket instead of metadata (default false)")),
			mcp.WithString("title", mcp.Description("update: new title")),
			mcp.WithString("description", mcp.Description("update: new description")),
			mcp.WithArray("dependencies", mcp.Description("update: replacement dependency list"), mcp.WithStringItems()),
			mcp.WithArray("target_files", mcp.Description("update: replacement target files"), mcp.WithStringItems()),
			mcp.WithArray("skills", mcp.Description("update: replacement required skills"), mcp.WithStringItems()),
			mcp.WithArray("tools", mcp.Description("update: replacement required tools"), mcp.WithStringItems()),
			mcp.WithArray("docs", mcp.Description("update: replacement required document keys"), mcp.WithStringItems()),
			mcp.WithArray("acceptance_criteria", mcp.Description("update: replacement acceptance criteria"), mcp.WithStringItems()),
			mcp.WithArray("verification_commands", mcp.Description("update: replacement verification commands"), mcp.WithStringItems()),
			mcp.WithString("priority", mcp.Description("update: P0, P1, P2, or P3")),
			mcp.WithString("complexity", mcp.Description("update: LOW, MEDIUM, HIGH, or SPIKE")),
			mcp.WithString("risk_level", mcp.Description("update: LOW, MEDIUM, HIGH, or CRITICAL")),
			mcp.WithString("agent_slug", mcp.Description("update: agent definition slug")),
			mcp.WithString("harness", mcp.Description("update: intended executor label")),
			mcp.WithString("worktree_path", mcp.Description("update: worktree path")),
			mcp.WithArray("media_add", mcp.Description("update: media to attach; each item has file_path and optional name"), mcp.Items(mediaItemSchema())),
			mcp.WithArray("media_remove", mcp.Description("update: artifact IDs to detach"), mcp.WithStringItems()),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()

			switch request.GetString("action", "") {
			case "create":
				projectSlug := request.GetString("project_slug", "")
				planSlug := request.GetString("plan_slug", "")
				rawTasks, ok := args["tasks"].([]any)
				if !ok || len(rawTasks) == 0 {
					return toolError("tasks must be a non-empty array"), nil
				}

				tasks, media, err := parseTaskInputs(projectSlug, planSlug, rawTasks)
				if err != nil {
					return toolError("%v", err), nil
				}
				if err := s.store.CreateTasks(ctx, tasks); err != nil {
					return toolError("failed to create %d task(s): %v", len(tasks), err), nil
				}

				ids := make([]string, len(tasks))
				for i, task := range tasks {
					ids[i] = task.ID
				}
				return marshalToolResult(map[string]any{
					"project_slug": projectSlug,
					"plan_slug":    planSlug,
					"created":      len(tasks),
					"task_ids":     ids,
					"media":        s.attachMediaForTasks(ctx, media),
				}), nil

			case "get":
				taskID := request.GetString("task_id", "")
				if taskID == "" {
					return toolError("task_id is required for get"), nil
				}
				if request.GetBool("include_packet", false) {
					packet, err := s.store.GetTaskContextPacket(ctx, taskID)
					if err != nil {
						return toolError("failed to load task %s: %v", taskID, err), nil
					}
					return marshalToolResult(packet), nil
				}
				task, err := s.store.GetTask(ctx, taskID)
				if err != nil {
					return toolError("failed to load task %s: %v", taskID, err), nil
				}
				return marshalToolResult(task), nil

			case "list":
				filter := domain.TaskFilter{
					ProjectSlug: request.GetString("project_slug", ""),
					PlanSlug:    request.GetString("plan_slug", ""),
					Status:      domain.TaskStatus(request.GetString("status", "")),
					AgentSlug:   request.GetString("agent_slug", ""),
					OrphanOnly:  request.GetBool("orphan", false),
				}
				tasks, err := s.store.ListTasks(ctx, filter)
				if err != nil {
					return toolError("failed to list tasks: %v", err), nil
				}
				if tasks == nil {
					tasks = []domain.Task{}
				}
				return marshalToolResult(tasks), nil

			case "update":
				taskID := request.GetString("task_id", "")
				if taskID == "" {
					return toolError("task_id is required for update"), nil
				}
				patch, mediaAdd, mediaRemove, err := parseTaskPatch(args)
				if err != nil {
					return toolError("invalid update payload: %v", err), nil
				}
				if taskPatchEmpty(patch) && len(mediaAdd) == 0 && len(mediaRemove) == 0 {
					return toolError("nothing to update; provide metadata, media_add, or media_remove"), nil
				}

				if !taskPatchEmpty(patch) {
					if err := s.store.UpdateTaskMetadata(ctx, taskID, patch); err != nil {
						return toolError("failed to update task %s: %v", taskID, err), nil
					}
				}
				updated, err := s.store.GetTask(ctx, taskID)
				if err != nil {
					return toolError("failed to load task %s: %v", taskID, err), nil
				}

				added := s.attachMedia(ctx, taskID, mediaAdd)
				removed := s.removeMedia(ctx, taskID, mediaRemove)
				for _, result := range added {
					if result.Error != "" {
						return toolError("failed to attach media to task %s: %s", taskID, result.Error), nil
					}
				}
				for _, result := range removed {
					if result.Error != "" {
						return toolError("failed to detach media from task %s: %s", taskID, result.Error), nil
					}
				}

				return marshalToolResult(map[string]any{
					"task":          updated,
					"media_added":   added,
					"media_removed": removed,
				}), nil

			default:
				return toolError("unknown action (expected create, get, list, or update)"), nil
			}
		},
	)
}

// ---------------------------------------------------------------------------
// Documents
// ---------------------------------------------------------------------------

func (s *Server) registerDocumentTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("docs",
			mcp.WithDescription("Manage SSOT documents. Files stay canonical; the registry stores identity, metadata, tags, hash, and a searchable snapshot. Also supports FTS search."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | get | list | update | refresh | archive | unarchive | delete | search")),
			mcp.WithString("id", mcp.Description("Document id")),
			mcp.WithString("project_slug", mcp.Description("Project scope. Omit for a global document or an unscoped list.")),
			mcp.WithString("key", mcp.Description("Stable document key, unique per project")),
			mcp.WithString("slug", mcp.Description("Human-readable slug, unique per project (defaults to key)")),
			mcp.WithString("type", mcp.Description("ARCHITECTURE | ADR | SPEC | RULE | GUIDE | REFERENCE | MEMORY | OTHER")),
			mcp.WithString("scope", mcp.Description("GLOBAL | PROJECT | PLAN | TASK")),
			mcp.WithString("scope_ref", mcp.Description("Plan slug or task id when scope is PLAN/TASK")),
			mcp.WithString("title", mcp.Description("Document title")),
			mcp.WithString("path", mcp.Description("Absolute or repository-relative path of the canonical file")),
			mcp.WithString("summary", mcp.Description("Short summary used by search results")),
			mcp.WithString("content", mcp.Description("Content snapshot. When omitted on create/refresh, the file at path is read.")),
			mcp.WithString("content_hash", mcp.Description("SHA-256 of the content. Computed when omitted.")),
			mcp.WithNumber("byte_size", mcp.Description("Content size in bytes. Computed when omitted.")),
			mcp.WithString("mime_type", mcp.Description("MIME type (default text/markdown)")),
			mcp.WithString("status", mcp.Description("DRAFT | ACTIVE | SUPERSEDED | ARCHIVED")),
			mcp.WithArray("tags", mcp.Description("Normalized tags"), mcp.WithStringItems()),
			mcp.WithBoolean("include_global", mcp.Description("For project lists: include globally scoped documents")),
			mcp.WithString("tag", mcp.Description("Filter by one tag")),
			mcp.WithNumber("limit", mcp.Description("Result limit (default 50, max 100)")),
			mcp.WithString("query", mcp.Description("Full-text query for action=search")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			action := request.GetString("action", "")
			projectSlug := request.GetString("project_slug", "")

			switch action {
			case "create":
				document, err := s.documentFromRequest(ctx, request)
				if err != nil {
					return toolError("%v", err), nil
				}
				if err := s.store.CreateDocument(ctx, document); err != nil {
					return toolError("failed to create document: %v", err), nil
				}
				created, err := s.store.GetDocumentByKey(ctx, document.ProjectSlug, document.Key)
				if err != nil {
					return toolError("failed to load document %s: %v", document.ID, err), nil
				}
				return marshalToolResult(created), nil

			case "get":
				document, err := s.lookupDocument(ctx, request)
				if err != nil {
					return toolError("%v", err), nil
				}
				return marshalToolResult(document), nil

			case "list":
				filter := domain.DocumentFilter{
					ProjectSlug:   projectSlug,
					IncludeGlobal: request.GetBool("include_global", false),
					Type:          domain.DocumentType(request.GetString("type", "")),
					Scope:         domain.DocumentScope(request.GetString("scope", "")),
					Status:        domain.DocumentStatus(request.GetString("status", "")),
					Tag:           request.GetString("tag", ""),
					Limit:         request.GetInt("limit", 50),
				}
				documents, err := s.store.ListDocuments(ctx, filter)
				if err != nil {
					return toolError("failed to list documents: %v", err), nil
				}
				if documents == nil {
					documents = []domain.Document{}
				}
				return marshalToolResult(documents), nil

			case "update":
				id := request.GetString("id", "")
				if id == "" {
					return toolError("id is required for update"), nil
				}
				patch, err := parseDocumentPatch(args)
				if err != nil {
					return toolError("invalid update payload: %v", err), nil
				}
				if err := s.store.UpdateDocument(ctx, id, patch); err != nil {
					return toolError("failed to update document %s: %v", id, err), nil
				}
				updated, err := s.store.GetDocument(ctx, id)
				if err != nil {
					return toolError("failed to load document %s: %v", id, err), nil
				}
				return marshalToolResult(updated), nil

			case "refresh":
				id := request.GetString("id", "")
				if id == "" {
					return toolError("id is required for refresh"), nil
				}
				content := request.GetString("content", "")
				if content == "" {
					document, err := s.store.GetDocument(ctx, id)
					if err != nil {
						return toolError("failed to load document %s: %v", id, err), nil
					}
					content, err = readDocumentContent(document.Path)
					if err != nil {
						return toolError("failed to read %s: %v", document.Path, err), nil
					}
				}
				hash := request.GetString("content_hash", "")
				size := int64(request.GetInt("byte_size", len(content)))
				if content != "" {
					if hash == "" {
						hash = contentHash(content)
					}
				} else if hash == "" {
					size = 0
				}
				if err := s.store.RefreshDocument(ctx, id, content, hash, size); err != nil {
					return toolError("failed to refresh document %s: %v", id, err), nil
				}
				refreshed, err := s.store.GetDocument(ctx, id)
				if err != nil {
					return toolError("failed to load document %s: %v", id, err), nil
				}
				return marshalToolResult(refreshed), nil

			case "archive", "unarchive":
				id := request.GetString("id", "")
				if id == "" {
					return toolError("id is required for %s", action), nil
				}
				archived := action == "archive"
				if err := s.store.ArchiveDocument(ctx, id, archived); err != nil {
					return toolError("failed to %s document %s: %v", action, id, err), nil
				}
				return marshalToolResult(map[string]any{"id": id, "archived": archived}), nil

			case "delete":
				id := request.GetString("id", "")
				if id == "" {
					return toolError("id is required for delete"), nil
				}
				if err := s.store.DeleteDocument(ctx, id); err != nil {
					return toolError("failed to delete document %s: %v", id, err), nil
				}
				return marshalToolResult(map[string]any{"id": id, "deleted": true}), nil

			case "search":
				query := request.GetString("query", "")
				if query == "" {
					return toolError("query is required for search"), nil
				}
				filter := domain.DocumentFilter{
					ProjectSlug:   projectSlug,
					IncludeGlobal: request.GetBool("include_global", true),
					Type:          domain.DocumentType(request.GetString("type", "")),
					Scope:         domain.DocumentScope(request.GetString("scope", "")),
					Status:        domain.DocumentStatus(request.GetString("status", "")),
					Limit:         request.GetInt("limit", 25),
				}
				results, err := s.store.SearchDocuments(ctx, query, filter)
				if err != nil {
					return toolError("failed to search documents: %v", err), nil
				}
				if results == nil {
					results = []domain.DocumentSearchResult{}
				}
				return marshalToolResult(results), nil

			default:
				return toolError("unknown action (expected create, get, list, update, refresh, archive, unarchive, delete, or search)"), nil
			}
		},
	)
}

func (s *Server) documentFromRequest(ctx context.Context, request mcp.CallToolRequest) (domain.Document, error) {
	projectSlug := request.GetString("project_slug", "")
	key := request.GetString("key", "")
	slug := request.GetString("slug", "")
	if key == "" && slug == "" {
		return domain.Document{}, fmt.Errorf("key or slug is required")
	}
	if key == "" {
		key = slug
	}
	if slug == "" {
		slug = key
	}
	scope := domain.DocumentScope(request.GetString("scope", ""))
	if scope == "" {
		if projectSlug == "" {
			scope = domain.DocScopeGlobal
		} else {
			scope = domain.DocScopeProject
		}
	}
	if scope == domain.DocScopeGlobal {
		projectSlug = ""
	}

	docType := domain.DocumentType(strings.ToUpper(request.GetString("type", "")))
	if docType == "" {
		docType = domain.DocTypeOther
	}
	status := domain.DocumentStatus(strings.ToUpper(request.GetString("status", "")))
	if status == "" {
		status = domain.DocStatusActive
	}
	path := request.GetString("path", "")
	if path == "" {
		return domain.Document{}, fmt.Errorf("path is required")
	}

	content := request.GetString("content", "")
	if content == "" {
		read, err := readDocumentContent(path)
		if err == nil {
			content = read
		}
	}
	hash := request.GetString("content_hash", "")
	if hash == "" && content != "" {
		hash = contentHash(content)
	}
	size := int64(request.GetInt("byte_size", len(content)))
	mimeType := request.GetString("mime_type", "text/markdown")

	return domain.Document{
		ProjectSlug: projectSlug,
		Key:         key,
		Slug:        slug,
		Type:        docType,
		Scope:       scope,
		ScopeRef:    request.GetString("scope_ref", ""),
		Title:       request.GetString("title", slug),
		Path:        path,
		ContentHash: hash,
		ByteSize:    size,
		MimeType:    mimeType,
		Status:      status,
		Summary:     request.GetString("summary", ""),
		Content:     content,
		Version:     1,
		Tags:        request.GetStringSlice("tags", nil),
	}, nil
}

func (s *Server) lookupDocument(ctx context.Context, request mcp.CallToolRequest) (*domain.Document, error) {
	if id := request.GetString("id", ""); id != "" {
		return s.store.GetDocument(ctx, id)
	}
	projectSlug := request.GetString("project_slug", "")
	if key := request.GetString("key", ""); key != "" {
		return s.store.GetDocumentByKey(ctx, projectSlug, key)
	}
	if slug := request.GetString("slug", ""); slug != "" {
		return s.store.GetDocumentBySlug(ctx, projectSlug, slug)
	}
	return nil, fmt.Errorf("id, key, or slug is required")
}

func parseDocumentPatch(args map[string]any) (domain.DocumentPatch, error) {
	encoded, err := json.Marshal(args)
	if err != nil {
		return domain.DocumentPatch{}, err
	}
	var payload struct {
		Title   *string   `json:"title"`
		Summary *string   `json:"summary"`
		Status  *string   `json:"status"`
		Path    *string   `json:"path"`
		Tags    *[]string `json:"tags"`
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return domain.DocumentPatch{}, err
	}
	patch := domain.DocumentPatch{
		Title:   payload.Title,
		Summary: payload.Summary,
		Path:    payload.Path,
		Tags:    payload.Tags,
	}
	if payload.Status != nil {
		status := domain.DocumentStatus(*payload.Status)
		patch.Status = &status
	}
	return patch, nil
}

func readDocumentContent(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", path)
	}
	if info.Size() > 4<<20 {
		return "", fmt.Errorf("%s is larger than 4 MiB; register it without a content snapshot", path)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".txt", ".json", ".yaml", ".yml", ".xml", ".csv",
		".toml", ".sql", ".go", ".ts", ".tsx", ".js", ".jsx", ".css", ".html":
	default:
		// Binary/media documents stay path-only; never index their bytes.
		return "", nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// Agents
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

func (s *Server) registerSearchTool() {
	s.mcpServer.AddTool(
		mcp.NewTool("search",
			mcp.WithDescription("Search projects, plans, tasks, and SSOT documents in one call."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search text")),
			mcp.WithString("project_slug", mcp.Description("Optional project filter")),
			mcp.WithNumber("limit", mcp.Description("Result limit (default 25, max 100)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, err := request.RequireString("query")
			if err != nil {
				return toolError("%v", err), nil
			}
			results, err := s.store.Search(ctx, query, request.GetString("project_slug", ""), request.GetInt("limit", 25))
			if err != nil {
				return toolError("failed to search: %v", err), nil
			}
			if results == nil {
				results = []domain.SearchResult{}
			}
			return marshalToolResult(results), nil
		},
	)
}

// ---------------------------------------------------------------------------
// Workflow
// ---------------------------------------------------------------------------

func (s *Server) registerWorkflowTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("claim_task",
			mcp.WithDescription("Atomically claims a READY task with mutual exclusion, transitions it to RUNNING, and returns its ContextPacket."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Unique task ID to claim")),
			mcp.WithString("worker_id", mcp.Description("Optional worker or subagent identifier")),
			mcp.WithString("orchestrator_id", mcp.Description("Optional orchestrator session ID")),
			mcp.WithString("worker_role", mcp.Description("Optional worker role (e.g. 'CODER', 'TESTER', 'INSPECTOR')")),
			mcp.WithString("session_id", mcp.Description("Optional observed session id to link the claim")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return toolError("%v", err), nil
			}
			opts := domain.ClaimOptions{
				WorkerID:       request.GetString("worker_id", ""),
				OrchestratorID: request.GetString("orchestrator_id", ""),
				WorkerRole:     request.GetString("worker_role", "CODER"),
				SessionID:      request.GetString("session_id", ""),
			}
			packet, err := s.store.ClaimTask(ctx, taskID, opts)
			if err != nil {
				return toolError("failed to claim task %s: %v", taskID, err), nil
			}
			return marshalToolResult(packet), nil
		},
	)

	s.mcpServer.AddTool(
		mcp.NewTool("review",
			mcp.WithDescription("Review handshake. action=request pauses a RUNNING task at AWAITING_REVIEW (worker exits); action=resolve records the human decision and returns the task to READY (main agent only)."),
			mcp.WithString("action", mcp.Required(), mcp.Description("request | resolve")),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID")),
			mcp.WithString("reason", mcp.Description("Short summary of architectural or requirement drift (request)")),
			mcp.WithString("details", mcp.Description("Detailed explanation and trade-offs (request)")),
			mcp.WithArray("options", mcp.Description("Actionable choice strings for the developer (request)")),
			mcp.WithString("response", mcp.Description("Human decision relayed by the main agent (resolve)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return toolError("%v", err), nil
			}

			switch request.GetString("action", "") {
			case "request":
				reason, err := request.RequireString("reason")
				if err != nil {
					return toolError("%v", err), nil
				}
				details, err := request.RequireString("details")
				if err != nil {
					return toolError("%v", err), nil
				}
				options := request.GetStringSlice("options", []string{})
				if err := s.store.RequestReview(ctx, taskID, reason, details, options); err != nil {
					return toolError("failed to request review: %v", err), nil
				}
				return marshalToolResult(map[string]any{"task_id": taskID, "status": domain.StatusAwaitingReview}), nil

			case "resolve":
				response, err := request.RequireString("response")
				if err != nil {
					return toolError("%v", err), nil
				}
				if err := s.store.ResolveReview(ctx, taskID, response); err != nil {
					return toolError("failed to resolve review: %v", err), nil
				}
				return marshalToolResult(map[string]any{"task_id": taskID, "status": domain.StatusReady}), nil

			default:
				return toolError("unknown action (expected request or resolve)"), nil
			}
		},
	)

	s.mcpServer.AddTool(
		mcp.NewTool("transition_task",
			mcp.WithDescription("Apply COMPLETE, FAIL, RESET, or CANCEL. Use task_ids + action for one shared action, or items[] for mixed per-task actions. Each task is processed independently and returns results[]."),
			mcp.WithArray("task_ids", mcp.Description("Task IDs for a shared action")),
			mcp.WithString("action", mcp.Description("COMPLETE, FAIL, RESET, or CANCEL (with task_ids)")),
			mcp.WithArray("items", mcp.Description("Mixed per-task transitions"), mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id":       map[string]any{"type": "string"},
					"action":        map[string]any{"type": "string", "enum": []string{"COMPLETE", "FAIL", "RESET", "CANCEL"}},
					"reason":        map[string]any{"type": "string"},
					"summary":       map[string]any{"type": "string"},
					"merge_outcome": map[string]any{"type": "string", "enum": []string{"MERGED", "CONFLICT"}},
				},
				"required": []string{"task_id", "action"},
			})),
			mcp.WithString("reason", mcp.Description("Failure, reset, or cancellation reason (appended to Notes)")),
			mcp.WithString("summary", mcp.Description("Completion summary (appended to Notes)")),
			mcp.WithString("merge_outcome", mcp.Description("MERGED or CONFLICT. Required when resolving AWAITING_MERGE/CONFLICT tasks.")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			taskIDs := request.GetStringSlice("task_ids", nil)
			action := request.GetString("action", "")
			reason := request.GetString("reason", "")
			summary := request.GetString("summary", "")
			mergeOutcome := request.GetString("merge_outcome", "")

			type taskResult struct {
				TaskID string `json:"task_id"`
				Status string `json:"status,omitempty"`
				Error  string `json:"error,omitempty"`
			}

			results := []taskResult{}

			if rawItems, ok := args["items"]; ok {
				if len(taskIDs) > 0 || action != "" {
					return toolError("provide either items or task_ids + action, not both"), nil
				}
				items, ok := rawItems.([]any)
				if !ok || len(items) == 0 {
					return toolError("items must be a non-empty array"), nil
				}
				for i, raw := range items {
					encoded, err := json.Marshal(raw)
					if err != nil {
						results = append(results, taskResult{Error: fmt.Sprintf("invalid item %d: %v", i, err)})
						continue
					}
					var item struct {
						TaskID       string `json:"task_id"`
						Action       string `json:"action"`
						Reason       string `json:"reason"`
						Summary      string `json:"summary"`
						MergeOutcome string `json:"merge_outcome"`
					}
					if err := json.Unmarshal(encoded, &item); err != nil {
						results = append(results, taskResult{Error: fmt.Sprintf("invalid item %d: %v", i, err)})
						continue
					}
					if item.TaskID == "" || item.Action == "" {
						results = append(results, taskResult{TaskID: item.TaskID, Error: fmt.Sprintf("item %d requires task_id and action", i)})
						continue
					}
					status, err := s.transitionTask(ctx, item.TaskID, item.Action, item.Reason, item.Summary, item.MergeOutcome)
					if err != nil {
						results = append(results, taskResult{TaskID: item.TaskID, Error: err.Error()})
						continue
					}
					results = append(results, taskResult{TaskID: item.TaskID, Status: string(status)})
				}
				return marshalToolResult(map[string]any{"results": results}), nil
			}

			if len(taskIDs) == 0 {
				return toolError("task_ids must contain at least one task ID"), nil
			}
			if action == "" {
				return toolError("action is required with task_ids"), nil
			}
			for _, taskID := range taskIDs {
				status, err := s.transitionTask(ctx, taskID, action, reason, summary, mergeOutcome)
				if err != nil {
					results = append(results, taskResult{TaskID: taskID, Error: err.Error()})
					continue
				}
				results = append(results, taskResult{TaskID: taskID, Status: string(status)})
			}

			return marshalToolResult(map[string]any{"results": results}), nil
		},
	)
}

// transitionTask applies a single task transition and returns the resulting status.
func (s *Server) transitionTask(ctx context.Context, taskID string, action string, reason string, summary string, mergeOutcome string) (domain.TaskStatus, error) {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("task not found: %w", err)
	}

	note := ""
	if summary != "" {
		note = "Summary: " + summary
	}
	if reason != "" && action != "RESET" && action != "CANCEL" {
		if note != "" {
			note += "\n"
		}
		note += "Reason: " + reason
	}

	switch action {
	case "COMPLETE":
		switch task.Status {
		case domain.StatusRunning:
			target := domain.StatusCompleted
			if task.WorktreePath != "" {
				target = domain.StatusAwaitingMerge
			}
			if err := s.store.UpdateTaskStatusWithNote(ctx, taskID, domain.StatusRunning, target, note); err != nil {
				return "", err
			}
			return target, nil
		case domain.StatusAwaitingMerge:
			switch mergeOutcome {
			case "MERGED":
				if err := s.store.UpdateTaskStatusWithNote(ctx, taskID, domain.StatusAwaitingMerge, domain.StatusCompleted, note); err != nil {
					return "", err
				}
				return domain.StatusCompleted, nil
			case "CONFLICT":
				if err := s.store.UpdateTaskStatusWithNote(ctx, taskID, domain.StatusAwaitingMerge, domain.StatusConflict, note); err != nil {
					return "", err
				}
				return domain.StatusConflict, nil
			default:
				return "", fmt.Errorf("merge_outcome MERGED or CONFLICT is required when task is AWAITING_MERGE")
			}
		case domain.StatusConflict:
			if mergeOutcome != "MERGED" {
				return "", fmt.Errorf("merge_outcome MERGED is required when task is CONFLICT")
			}
			if err := s.store.UpdateTaskStatusWithNote(ctx, taskID, domain.StatusConflict, domain.StatusCompleted, note); err != nil {
				return "", err
			}
			return domain.StatusCompleted, nil
		default:
			return "", fmt.Errorf("cannot COMPLETE task in %s status", task.Status)
		}
	case "FAIL":
		if task.Status != domain.StatusRunning {
			return "", fmt.Errorf("cannot FAIL task in %s status", task.Status)
		}
		if err := s.store.UpdateTaskStatusWithNote(ctx, taskID, domain.StatusRunning, domain.StatusFailed, note); err != nil {
			return "", err
		}
		return domain.StatusFailed, nil
	case "RESET":
		if task.Status != domain.StatusRunning {
			return "", fmt.Errorf("cannot RESET task in %s status", task.Status)
		}
		resetReason := reason
		if resetReason == "" {
			resetReason = "Reset by orchestrator"
		}
		status, err := s.store.EvictTask(ctx, taskID, resetReason, "RESET_TO_READY")
		if err != nil {
			return "", err
		}
		if note != "" {
			if err := s.store.AppendTaskNotes(ctx, taskID, note); err != nil {
				return "", err
			}
		}
		return status, nil
	case "CANCEL":
		if task.Status != domain.StatusRunning {
			return "", fmt.Errorf("cannot CANCEL task in %s status", task.Status)
		}
		cancelReason := reason
		if cancelReason == "" {
			cancelReason = "Cancelled by orchestrator"
		}
		status, err := s.store.EvictTask(ctx, taskID, cancelReason, "CANCEL")
		if err != nil {
			return "", err
		}
		if note != "" {
			if err := s.store.AppendTaskNotes(ctx, taskID, note); err != nil {
				return "", err
			}
		}
		return status, nil
	default:
		return "", fmt.Errorf("unknown action %q (expected COMPLETE, FAIL, RESET, or CANCEL)", action)
	}
}
