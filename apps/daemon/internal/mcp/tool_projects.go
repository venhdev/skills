package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	"zenf/internal/domain"
)

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
