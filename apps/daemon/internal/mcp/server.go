package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
		mcpServer: server.NewMCPServer("zenfd", "0.1.2", server.WithToolCapabilities(true)),
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
			mcp.WithDescription("Manage plans: create (or update title), archive, unarchive, or list. Archived plans are omitted from list unless include_archived is true."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | archive | unarchive | list")),
			mcp.WithString("slug", mcp.Description("Plan slug (required for create/archive/unarchive)")),
			mcp.WithString("title", mcp.Description("Human-readable plan title (create only; defaults to slug)")),
			mcp.WithBoolean("include_archived", mcp.Description("Include archived plans in list (default false)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			action := request.GetString("action", "")
			slug := request.GetString("slug", "")

			switch action {
			case "create":
				if slug == "" {
					return mcp.NewToolResultError("slug is required for create"), nil
				}
				title := request.GetString("title", slug)
				now := time.Now().UTC()
				plan := domain.Plan{
					Slug:      slug,
					Title:     title,
					Status:    "ACTIVE",
					CreatedAt: now,
					UpdatedAt: now,
				}
				if err := s.store.CreatePlan(ctx, plan); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create plan %s: %v", slug, err)), nil
				}
				return marshalToolResult(plan), nil

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

// taskInput is the wire shape accepted by tasks create.
type taskInput struct {
	ID                   string   `json:"id"`
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	Dependencies         []string `json:"dependencies"`
	TargetFiles          []string `json:"target_files"`
	Skills               []string `json:"skills"`
	AcceptanceCriteria   []string `json:"acceptance_criteria"`
	VerificationCommands []string `json:"verification_commands"`
	Priority             string   `json:"priority"`
	Harness              string   `json:"harness"`
	WorktreePath         string   `json:"worktree_path"`
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

func (s *Server) registerTaskTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("tasks",
			mcp.WithDescription("Manage tasks: create a batch in one transaction, get one task with its ContextPacket, or list tasks. Prefer a single batch create over many calls."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | get | list")),
			mcp.WithString("plan_id", mcp.Description("Parent plan slug (required for create)")),
			mcp.WithArray("tasks", mcp.Description("Batch of tasks to create. Each item: id, title, description, dependencies, target_files, skills, acceptance_criteria, verification_commands, priority, harness, worktree_path."),
				mcp.Items(map[string]any{
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
					},
					"required": []string{"id", "title"},
				}),
			),
			mcp.WithString("task_id", mcp.Description("Task ID (required for get)")),
			mcp.WithString("status", mcp.Description("Optional status filter for list")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			switch request.GetString("action", "") {
			case "create":
				planID := request.GetString("plan_id", "")
				if planID == "" {
					return mcp.NewToolResultError("plan_id is required for create"), nil
				}
				rawTasks, ok := request.GetArguments()["tasks"].([]any)
				if !ok || len(rawTasks) == 0 {
					return mcp.NewToolResultError("tasks must be a non-empty array"), nil
				}

				tasks := make([]domain.Task, 0, len(rawTasks))
				for i, raw := range rawTasks {
					encoded, err := json.Marshal(raw)
					if err != nil {
						return mcp.NewToolResultError(fmt.Sprintf("invalid task at index %d: %v", i, err)), nil
					}
					var in taskInput
					if err := json.Unmarshal(encoded, &in); err != nil {
						return mcp.NewToolResultError(fmt.Sprintf("invalid task at index %d: %v", i, err)), nil
					}
					if in.ID == "" || in.Title == "" {
						return mcp.NewToolResultError(fmt.Sprintf("task at index %d requires id and title", i)), nil
					}
					tasks = append(tasks, toDomainTask(planID, in))
				}

				if err := s.store.CreateTasks(ctx, tasks); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create %d task(s): %v", len(tasks), err)), nil
				}

				ids := make([]string, len(tasks))
				for i, task := range tasks {
					ids[i] = task.ID
				}
				return marshalToolResult(map[string]any{"plan_id": planID, "created": len(tasks), "task_ids": ids}), nil

			case "get":
				taskID := request.GetString("task_id", "")
				if taskID == "" {
					return mcp.NewToolResultError("task_id is required for get"), nil
				}
				packet, err := s.store.GetTaskContextPacket(ctx, taskID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to load task %s: %v", taskID, err)), nil
				}
				return marshalToolResult(packet), nil

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

			default:
				return mcp.NewToolResultError("unknown action (expected create, get, or list)"), nil
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
			mcp.WithDescription("Applies a state transition to one or more tasks. COMPLETE stages worktree tasks to AWAITING_MERGE or completes them; merge_outcome resolves AWAITING_MERGE/CONFLICT. FAIL marks RUNNING tasks FAILED and blocks downstream dependents. RESET clears the claim and returns to READY (or BLOCKED when prerequisites are unfinished). CANCEL cancels. Each task is processed independently."),
			mcp.WithArray("task_ids", mcp.Required(), mcp.Description("Task IDs to transition")),
			mcp.WithString("action", mcp.Required(), mcp.Description("COMPLETE, FAIL, RESET, or CANCEL")),
			mcp.WithString("reason", mcp.Description("Failure, reset, or cancellation reason (appended to Notes)")),
			mcp.WithString("summary", mcp.Description("Completion summary (appended to Notes)")),
			mcp.WithString("merge_outcome", mcp.Description("MERGED or CONFLICT. Required when resolving AWAITING_MERGE/CONFLICT tasks.")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskIDs := request.GetStringSlice("task_ids", nil)
			if len(taskIDs) == 0 {
				return mcp.NewToolResultError("task_ids must contain at least one task ID"), nil
			}
			action := request.GetString("action", "")
			reason := request.GetString("reason", "")
			summary := request.GetString("summary", "")
			mergeOutcome := request.GetString("merge_outcome", "")

			type taskResult struct {
				TaskID string `json:"task_id"`
				Status string `json:"status,omitempty"`
				Error  string `json:"error,omitempty"`
			}

			results := make([]taskResult, 0, len(taskIDs))
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

	// attach_media
	s.mcpServer.AddTool(
		mcp.NewTool("attach_media",
			mcp.WithDescription("Attaches input media (image, PDF, document) to a task that is READY or PENDING. The file is copied into the ZenForge attachment store and registered as an INPUT artifact, so the next claim includes it in the ContextPacket."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task in READY or PENDING status")),
			mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to a local file to copy into the attachment store")),
			mcp.WithString("name", mcp.Description("Display name (defaults to the file name)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			filePath, err := request.RequireString("file_path")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			artifact, err := attachment.Ingest(taskID, filePath, request.GetString("name", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to ingest media: %v", err)), nil
			}

			if err := s.store.RecordArtifact(ctx, artifact); err != nil {
				_ = os.Remove(artifact.FilePath)
				return mcp.NewToolResultError(fmt.Sprintf("failed to register media for task %s: %v", taskID, err)), nil
			}
			return marshalToolResult(artifact), nil
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
