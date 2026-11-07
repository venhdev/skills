package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"zenf/internal/contextpacket"
	"zenf/internal/domain"
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

// Server implements a Model Context Protocol (MCP) server for zenf.
type Server struct {
	mcpServer *server.MCPServer
	store     Storage
}

// NewServer initializes the MCP server and registers all tools.
func NewServer(store Storage) *Server {
	if ps, ok := store.(interface{ SetPacketBuilder(domain.ContextPacketBuilder) }); ok {
		builder := contextpacket.New(store, nil,
			contextpacket.WithProjectReader(store),
			contextpacket.WithPlanReader(store),
			contextpacket.WithAgentReader(store),
			contextpacket.WithDocumentReader(store),
		)
		ps.SetPacketBuilder(builder)
	}

	s := &Server{
		mcpServer: server.NewMCPServer("zenf", "0.3.2", server.WithToolCapabilities(true)),
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
