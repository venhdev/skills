package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"zenfd/internal/attachment"
	"zenfd/internal/domain"
)

// Storage is the subset of persistence the MCP tool surface needs. It composes
// per-aggregate interfaces so each repository stays independently replaceable.
type Storage interface {
	domain.PlanRepository
	domain.TaskRepository
	domain.TaskLifecycle
	domain.ArtifactRepository
}

// Server implements a Model Context Protocol (MCP) server for zenfd.
type Server struct {
	mcpServer *server.MCPServer
	store     Storage
}

// NewServer initializes the MCP server and registers all tools.
func NewServer(store Storage) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer("zenfd", "0.1.3", server.WithToolCapabilities(true)),
		store:     store,
	}
	s.registerTools()
	return s
}

// ServeStdio launches the stdio transport. CRITICAL: Redirects all standard logging to stderr
// so that stdout is purely preserved for JSON-RPC 2.0 messages.
func (s *Server) ServeStdio() error {
	log.SetOutput(os.Stderr)
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) registerTools() {
	s.registerPlanTools()
	s.registerTaskTools()
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

func (s *Server) registerPlanTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("plans",
			mcp.WithDescription("Manage plans: create (or retitle), archive, unarchive, or list. create accepts an optional tasks[] to build the whole DAG in one transaction."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | archive | unarchive | list")),
			mcp.WithString("slug", mcp.Description("Plan slug (required for create/archive/unarchive)")),
			mcp.WithString("title", mcp.Description("Human-readable plan title (create only; defaults to slug)")),
			mcp.WithBoolean("include_archived", mcp.Description("Include archived plans in list (default false)")),
			mcp.WithArray("tasks", mcp.Description("Optional task DAG to create with the plan. Each item follows the tasks create shape, including media[]."), mcp.Items(taskItemSchema())),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			action := request.GetString("action", "")
			slug := request.GetString("slug", "")

			switch action {
			case "create":
				if slug == "" {
					return mcp.NewToolResultError("slug is required for create"), nil
				}
				title := request.GetString("title", slug)
				now := time.Now().UTC()
				plan := domain.Plan{Slug: slug, Title: title, Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}

				var tasks []domain.Task
				mediaByTask := map[string][]mediaInput{}
				if raw, ok := args["tasks"]; ok {
					items, ok := raw.([]any)
					if !ok {
						return mcp.NewToolResultError("tasks must be an array"), nil
					}
					parsed, media, err := parseTaskInputs(slug, items)
					if err != nil {
						return mcp.NewToolResultError(err.Error()), nil
					}
					tasks, mediaByTask = parsed, media
				}

				if err := s.store.SavePlanWithTasks(ctx, plan, tasks); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create plan %s: %v", slug, err)), nil
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

			case "archive", "unarchive":
				if slug == "" {
					return mcp.NewToolResultError("slug is required for " + action), nil
				}
				archived := action == "archive"
				if err := s.store.ArchivePlan(ctx, slug, archived); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to %s plan %s: %v", action, slug, err)), nil
				}
				return marshalToolResult(map[string]any{"slug": slug, "archived": archived}), nil

			case "list":
				plans, err := s.store.ListPlans(ctx, request.GetBool("include_archived", false))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list plans: %v", err)), nil
				}
				if plans == nil {
					plans = []domain.Plan{}
				}
				return marshalToolResult(plans), nil

			default:
				return mcp.NewToolResultError("unknown action (expected create, archive, unarchive, or list)"), nil
			}
		},
	)
}

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
	AcceptanceCriteria   []string     `json:"acceptance_criteria"`
	VerificationCommands []string     `json:"verification_commands"`
	Priority             string       `json:"priority"`
	Harness              string       `json:"harness"`
	WorktreePath         string       `json:"worktree_path"`
	Media                []mediaInput `json:"media"`
}

func toDomainTask(planID string, in taskInput) domain.Task {
	priority := domain.TaskPriority(in.Priority)
	if priority == "" {
		priority = domain.PriorityP2
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
		PlanID:               planID,
		Title:                in.Title,
		Description:          in.Description,
		Dependencies:         in.Dependencies,
		TargetFiles:          in.TargetFiles,
		RequiredSkills:       in.Skills,
		AcceptanceCriteria:   in.AcceptanceCriteria,
		VerificationCommands: in.VerificationCommands,
		Priority:             priority,
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
			"acceptance_criteria":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"verification_commands": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"priority":              map[string]any{"type": "string", "enum": []string{"P0", "P1", "P2", "P3"}},
			"harness":               map[string]any{"type": "string"},
			"worktree_path":         map[string]any{"type": "string"},
			"media":                 map[string]any{"type": "array", "items": mediaItemSchema()},
		},
		"required": []string{"id", "title"},
	}
}

func parseTaskInputs(planID string, raw []any) ([]domain.Task, map[string][]mediaInput, error) {
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
		tasks = append(tasks, toDomainTask(planID, in))
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
	AcceptanceCriteria   *[]string    `json:"acceptance_criteria"`
	VerificationCommands *[]string    `json:"verification_commands"`
	Priority             *string      `json:"priority"`
	Complexity           *string      `json:"complexity"`
	RiskLevel            *string      `json:"risk_level"`
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
		RequiredTools:        nil,
		TargetFiles:          payload.TargetFiles,
		AcceptanceCriteria:   payload.AcceptanceCriteria,
		VerificationCommands: payload.VerificationCommands,
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
		patch.RequiredSkills == nil && patch.RequiredTools == nil && patch.TargetFiles == nil &&
		patch.AcceptanceCriteria == nil && patch.VerificationCommands == nil &&
		patch.Harness == nil && patch.WorktreePath == nil
}

func (s *Server) registerTaskTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("tasks",
			mcp.WithDescription("Manage tasks: create a batch with optional media, get metadata or the ContextPacket, list, or update authoring metadata and media. Prefer one batch create over many calls."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | get | list | update")),
			mcp.WithString("plan_id", mcp.Description("Parent plan slug (required for create)")),
			mcp.WithArray("tasks", mcp.Description("Batch of tasks to create. Each item accepts id, title, and optional metadata plus media[]."), mcp.Items(taskItemSchema())),
			mcp.WithString("task_id", mcp.Description("Task ID (required for get and update)")),
			mcp.WithString("status", mcp.Description("Optional status filter for list")),
			mcp.WithBoolean("include_packet", mcp.Description("For get: return the full ContextPacket instead of metadata (default false)")),
			mcp.WithString("title", mcp.Description("update: new title")),
			mcp.WithString("description", mcp.Description("update: new description")),
			mcp.WithArray("dependencies", mcp.Description("update: replacement dependency list"), mcp.WithStringItems()),
			mcp.WithArray("target_files", mcp.Description("update: replacement target files"), mcp.WithStringItems()),
			mcp.WithArray("skills", mcp.Description("update: replacement required skills"), mcp.WithStringItems()),
			mcp.WithArray("acceptance_criteria", mcp.Description("update: replacement acceptance criteria"), mcp.WithStringItems()),
			mcp.WithArray("verification_commands", mcp.Description("update: replacement verification commands"), mcp.WithStringItems()),
			mcp.WithString("priority", mcp.Description("update: P0, P1, P2, or P3")),
			mcp.WithString("complexity", mcp.Description("update: LOW, MEDIUM, HIGH, or SPIKE")),
			mcp.WithString("risk_level", mcp.Description("update: LOW, MEDIUM, HIGH, or CRITICAL")),
			mcp.WithString("harness", mcp.Description("update: intended executor label")),
			mcp.WithString("worktree_path", mcp.Description("update: worktree path")),
			mcp.WithArray("media_add", mcp.Description("update: media to attach; each item has file_path and optional name"), mcp.Items(mediaItemSchema())),
			mcp.WithArray("media_remove", mcp.Description("update: artifact IDs to detach"), mcp.WithStringItems()),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()

			switch request.GetString("action", "") {
			case "create":
				planID := request.GetString("plan_id", "")
				if planID == "" {
					return mcp.NewToolResultError("plan_id is required for create"), nil
				}
				rawTasks, ok := args["tasks"].([]any)
				if !ok || len(rawTasks) == 0 {
					return mcp.NewToolResultError("tasks must be a non-empty array"), nil
				}

				tasks, media, err := parseTaskInputs(planID, rawTasks)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := s.store.CreateTasks(ctx, tasks); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create %d task(s): %v", len(tasks), err)), nil
				}

				ids := make([]string, len(tasks))
				for i, task := range tasks {
					ids[i] = task.ID
				}
				return marshalToolResult(map[string]any{
					"plan_id":  planID,
					"created":  len(tasks),
					"task_ids": ids,
					"media":    s.attachMediaForTasks(ctx, media),
				}), nil

			case "get":
				taskID := request.GetString("task_id", "")
				if taskID == "" {
					return mcp.NewToolResultError("task_id is required for get"), nil
				}
				if request.GetBool("include_packet", false) {
					packet, err := s.store.GetTaskContextPacket(ctx, taskID)
					if err != nil {
						return mcp.NewToolResultError(fmt.Sprintf("failed to load task %s: %v", taskID, err)), nil
					}
					return marshalToolResult(packet), nil
				}
				task, err := s.store.GetTask(ctx, taskID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to load task %s: %v", taskID, err)), nil
				}
				return marshalToolResult(task), nil

			case "list":
				planID := request.GetString("plan_id", "")
				if planID != "" {
					tasks, err := s.store.GetTasksByPlan(ctx, planID)
					if err != nil {
						return mcp.NewToolResultError(fmt.Sprintf("failed to list tasks: %v", err)), nil
					}
					if tasks == nil {
						tasks = []domain.Task{}
					}
					return marshalToolResult(tasks), nil
				}

				status := domain.TaskStatus(request.GetString("status", ""))
				tasks, err := s.store.ListTasks(ctx, status)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list tasks: %v", err)), nil
				}
				if tasks == nil {
					tasks = []domain.Task{}
				}
				return marshalToolResult(tasks), nil

			case "update":
				taskID := request.GetString("task_id", "")
				if taskID == "" {
					return mcp.NewToolResultError("task_id is required for update"), nil
				}
				patch, mediaAdd, mediaRemove, err := parseTaskPatch(args)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("invalid update payload: %v", err)), nil
				}
				if taskPatchEmpty(patch) && len(mediaAdd) == 0 && len(mediaRemove) == 0 {
					return mcp.NewToolResultError("nothing to update; provide metadata, media_add, or media_remove"), nil
				}

				if err := s.store.UpdateTaskMetadata(ctx, taskID, patch); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to update task %s: %v", taskID, err)), nil
				}
				updated, err := s.store.GetTask(ctx, taskID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to load task %s: %v", taskID, err)), nil
				}

				return marshalToolResult(map[string]any{
					"task":          updated,
					"media_added":   s.attachMedia(ctx, taskID, mediaAdd),
					"media_removed": s.removeMedia(ctx, taskID, mediaRemove),
				}), nil

			default:
				return mcp.NewToolResultError("unknown action (expected create, get, list, or update)"), nil
			}
		},
	)
}

func (s *Server) registerWorkflowTools() {
	// claim_task
	s.mcpServer.AddTool(
		mcp.NewTool("claim_task",
			mcp.WithDescription("Atomically claims a READY task with mutual exclusion lock, transitions to RUNNING, and returns its ContextPacket."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Unique task ID to claim")),
			mcp.WithString("worker_id", mcp.Description("Optional worker or subagent identifier")),
			mcp.WithString("orchestrator_id", mcp.Description("Optional orchestrator session ID")),
			mcp.WithString("worker_role", mcp.Description("Optional worker role (e.g. 'CODER', 'TESTER', 'INSPECTOR')")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			workerID := request.GetString("worker_id", fmt.Sprintf("worker-%d", time.Now().UnixNano()))
			orchestratorID := request.GetString("orchestrator_id", "")
			workerRole := request.GetString("worker_role", "CODER")

			packet, err := s.store.ClaimTaskWithMeta(ctx, taskID, workerID, orchestratorID, workerRole)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to claim task %s: %v", taskID, err)), nil
			}
			return marshalToolResult(packet), nil
		},
	)

	// review
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
				return mcp.NewToolResultError(err.Error()), nil
			}

			switch request.GetString("action", "") {
			case "request":
				reason, err := request.RequireString("reason")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				details, err := request.RequireString("details")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				options := request.GetStringSlice("options", []string{})
				if err := s.store.RequestReview(ctx, taskID, reason, details, options); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to request review: %v", err)), nil
				}
				return marshalToolResult(map[string]any{"task_id": taskID, "status": domain.StatusAwaitingReview}), nil

			case "resolve":
				response, err := request.RequireString("response")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := s.store.ResolveReview(ctx, taskID, response); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to resolve review: %v", err)), nil
				}
				return marshalToolResult(map[string]any{"task_id": taskID, "status": domain.StatusReady}), nil

			default:
				return mcp.NewToolResultError("unknown action (expected request or resolve)"), nil
			}
		},
	)

	// transition_task
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
					return mcp.NewToolResultError("provide either items or task_ids + action, not both"), nil
				}
				items, ok := rawItems.([]any)
				if !ok || len(items) == 0 {
					return mcp.NewToolResultError("items must be a non-empty array"), nil
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
				return mcp.NewToolResultError("task_ids must contain at least one task ID"), nil
			}
			if action == "" {
				return mcp.NewToolResultError("action is required with task_ids"), nil
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
