package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"zenfd/internal/domain"
)

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
