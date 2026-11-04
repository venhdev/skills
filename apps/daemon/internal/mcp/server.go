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

	"zenfd/internal/domain"
)

// Server implements a Model Context Protocol (MCP) server for zenfd.
type Server struct {
	mcpServer *server.MCPServer
	store     domain.StorageEngine
}

// NewServer initializes the MCP server and registers all tools.
func NewServer(store domain.StorageEngine) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer("zenfd", "0.1.0", server.WithToolCapabilities(true)),
		store:     store,
	}
	s.registerTools()
	return s
}

// MCPServer returns the underlying mark3labs MCPServer instance.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// ServeStdio launches the stdio transport. CRITICAL: Redirects all standard logging to stderr
// so that stdout is purely preserved for JSON-RPC 2.0 messages.
func (s *Server) ServeStdio() error {
	log.SetOutput(os.Stderr)
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) registerTools() {
	// 1. zenfd_create_plan
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_create_plan",
			mcp.WithDescription("Creates or updates a plan directly in SQLite WAL (SSOT)."),
			mcp.WithString("slug", mcp.Required(), mcp.Description("Unique plan slug (e.g. 'zenfd' or 'auth-refactor')")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Human-readable title of the plan")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			slug, err := request.RequireString("slug")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			title, err := request.RequireString("title")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			now := time.Now().UTC()
			plan := domain.Plan{
				Slug:      slug,
				Title:     title,
				Status:    "ACTIVE",
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := s.store.CreatePlan(ctx, plan); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to create plan: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Plan '%s' (%s) created successfully in SQLite WAL.", slug, title)), nil
		},
	)

	// 2. zenfd_create_task
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_create_task",
			mcp.WithDescription("Creates or updates a vertical slice task in a plan DAG directly in SQLite WAL."),
			mcp.WithString("plan_id", mcp.Required(), mcp.Description("Parent plan slug")),
			mcp.WithString("id", mcp.Required(), mcp.Description("Unique task ID (e.g. 'P-01')")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
			mcp.WithString("description", mcp.Description("Detailed task description and context")),
			mcp.WithArray("dependencies", mcp.Description("List of prerequisite task IDs")),
			mcp.WithArray("target_files", mcp.Description("Target files for this task")),
			mcp.WithArray("skills", mcp.Description("Required skill names")),
			mcp.WithArray("acceptance_criteria", mcp.Description("Acceptance criteria that must be satisfied")),
			mcp.WithArray("verification_commands", mcp.Description("Commands that verify implementation correctness")),
			mcp.WithString("priority", mcp.Description("P0, P1, P2, P3")),
			mcp.WithString("harness", mcp.Description("Execution harness (default 'orchestrator' for agent-driven tasks, or 'shell')")),
			mcp.WithString("worktree_path", mcp.Description("Optional worktree path. When set, completion stops at AWAITING_MERGE.")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			planID, err := request.RequireString("plan_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			id, err := request.RequireString("id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			title, err := request.RequireString("title")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			desc := request.GetString("description", "")
			deps := request.GetStringSlice("dependencies", []string{})
			files := request.GetStringSlice("target_files", []string{})
			skills := request.GetStringSlice("skills", []string{})
			criteria := request.GetStringSlice("acceptance_criteria", []string{})
			verification := request.GetStringSlice("verification_commands", []string{})
			priority := domain.TaskPriority(request.GetString("priority", "P2"))
			harness := request.GetString("harness", "orchestrator")
			worktreePath := request.GetString("worktree_path", "")

			task := domain.Task{
				ID:                   id,
				PlanID:               planID,
				Title:                title,
				Description:          desc,
				Dependencies:         deps,
				TargetFiles:          files,
				RequiredSkills:       skills,
				AcceptanceCriteria:   criteria,
				VerificationCommands: verification,
				Priority:             priority,
				WorktreePath:         worktreePath,
				Harness:              harness,
				SkillCommand:         fmt.Sprintf("zenfd task claim %s", id),
			}
			if len(deps) == 0 {
				task.Status = domain.StatusReady
			} else {
				task.Status = domain.StatusPending
			}
			if err := s.store.CreateTask(ctx, task); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to create task: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Task '%s' created successfully in plan '%s'.", id, planID)), nil
		},
	)

	// 3. zenfd_claim_task
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_claim_task",
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

			packet, err := s.store.ClaimTaskWithMeta(ctx, taskID, workerID, orchestratorID, workerRole, 0, 0)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to claim task %s: %v", taskID, err)), nil
			}
			bytes, _ := json.MarshalIndent(packet, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// 4. zenfd_request_review
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_request_review",
			mcp.WithDescription("Pauses task execution and records a scope-drift question in AWAITING_REVIEW. The main orchestrator agent surfaces the question to the human; the worker must exit after calling this."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID currently in RUNNING status")),
			mcp.WithString("reason", mcp.Required(), mcp.Description("Short summary of architectural or requirement drift")),
			mcp.WithString("details", mcp.Required(), mcp.Description("Detailed explanation of the issue or trade-offs")),
			mcp.WithArray("options", mcp.Description("List of actionable choice strings for developer to decide")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
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
			return mcp.NewToolResultText(fmt.Sprintf("Task %s paused at AWAITING_REVIEW. The main agent must surface the decision request to the human.", taskID)), nil
		},
	)

	// 5. zenfd_resolve_review
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_resolve_review",
			mcp.WithDescription("Resolves an AWAITING_REVIEW task with guidance supplied by the human through the main orchestrator agent, then transitions back to READY. Only the main agent should call this tool."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID currently in AWAITING_REVIEW")),
			mcp.WithString("response", mcp.Required(), mcp.Description("Human decision or guidance relayed by the main agent")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			response, err := request.RequireString("response")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := s.store.ResolveReview(ctx, taskID, response); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to resolve review: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Task %s resolved and transitioned to READY with human guidance recorded.", taskID)), nil
		},
	)

	// 6. zenfd_transition_task
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_transition_task",
			mcp.WithDescription("Applies a state transition to one or more tasks. COMPLETE stages worktree tasks to AWAITING_MERGE or completes them; merge_outcome resolves AWAITING_MERGE/CONFLICT. FAIL marks RUNNING tasks FAILED. RESET returns RUNNING/SUSPENDED tasks to READY. CANCEL cancels RUNNING/SUSPENDED tasks. Each task is processed independently."),
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

			bytes, _ := json.MarshalIndent(map[string]any{"results": results}, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// 7. zenfd_get_task
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_get_task",
			mcp.WithDescription("Retrieves metadata and prompt ContextPacket for a task without mutating state."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID to retrieve")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			packet, err := s.store.GetTaskContextPacket(ctx, taskID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			bytes, _ := json.MarshalIndent(packet, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
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
			if err := s.store.UpdateTaskStatus(ctx, taskID, domain.StatusRunning, target); err != nil {
				return "", err
			}
			_ = s.store.AppendTaskNotes(ctx, taskID, note)
			return target, nil
		case domain.StatusAwaitingMerge:
			switch mergeOutcome {
			case "MERGED":
				if err := s.store.UpdateTaskStatus(ctx, taskID, domain.StatusAwaitingMerge, domain.StatusCompleted); err != nil {
					return "", err
				}
				_ = s.store.AppendTaskNotes(ctx, taskID, note)
				return domain.StatusCompleted, nil
			case "CONFLICT":
				if err := s.store.UpdateTaskStatus(ctx, taskID, domain.StatusAwaitingMerge, domain.StatusConflict); err != nil {
					return "", err
				}
				_ = s.store.AppendTaskNotes(ctx, taskID, note)
				return domain.StatusConflict, nil
			default:
				return "", fmt.Errorf("merge_outcome MERGED or CONFLICT is required when task is AWAITING_MERGE")
			}
		case domain.StatusConflict:
			if mergeOutcome != "MERGED" {
				return "", fmt.Errorf("merge_outcome MERGED is required when task is CONFLICT")
			}
			if err := s.store.UpdateTaskStatus(ctx, taskID, domain.StatusConflict, domain.StatusCompleted); err != nil {
				return "", err
			}
			_ = s.store.AppendTaskNotes(ctx, taskID, note)
			return domain.StatusCompleted, nil
		default:
			return "", fmt.Errorf("cannot COMPLETE task in %s status", task.Status)
		}
	case "FAIL":
		if task.Status != domain.StatusRunning {
			return "", fmt.Errorf("cannot FAIL task in %s status", task.Status)
		}
		if err := s.store.UpdateTaskStatus(ctx, taskID, domain.StatusRunning, domain.StatusFailed); err != nil {
			return "", err
		}
		_ = s.store.AppendTaskNotes(ctx, taskID, note)
		return domain.StatusFailed, nil
	case "RESET":
		if task.Status != domain.StatusRunning && task.Status != domain.StatusSuspended {
			return "", fmt.Errorf("cannot RESET task in %s status", task.Status)
		}
		resetReason := reason
		if resetReason == "" {
			resetReason = "Reset by orchestrator"
		}
		if err := s.store.EvictTask(ctx, taskID, resetReason, "RESET_TO_READY"); err != nil {
			return "", err
		}
		_ = s.store.AppendTaskNotes(ctx, taskID, note)
		return domain.StatusReady, nil
	case "CANCEL":
		if task.Status != domain.StatusRunning && task.Status != domain.StatusSuspended {
			return "", fmt.Errorf("cannot CANCEL task in %s status", task.Status)
		}
		cancelReason := reason
		if cancelReason == "" {
			cancelReason = "Cancelled by orchestrator"
		}
		if err := s.store.EvictTask(ctx, taskID, cancelReason, "CANCEL"); err != nil {
			return "", err
		}
		_ = s.store.AppendTaskNotes(ctx, taskID, note)
		return domain.StatusCancelled, nil
	default:
		return "", fmt.Errorf("unknown action %q (expected COMPLETE, FAIL, RESET, or CANCEL)", action)
	}
}
