package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"zenfd/internal/domain"
)

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
