package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/mark3labs/mcp-go/mcp"

	"zenfd/internal/attachment"
	"zenfd/internal/domain"
)

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
