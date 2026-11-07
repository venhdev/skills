package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"zenfd/internal/domain"
)

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
