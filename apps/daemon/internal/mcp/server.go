package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"zenfd/internal/domain"
	"zenfd/internal/parser"
)

// Server implements a Model Context Protocol (MCP) server for zenfd.
type Server struct {
	mcpServer *server.MCPServer
	store     domain.StorageEngine
	repoPath  string
}

// NewServer initializes the MCP server and registers all tools and resources.
func NewServer(store domain.StorageEngine, repoPath string) *Server {
	if repoPath == "" {
		repoPath, _ = os.Getwd()
	}
	s := &Server{
		mcpServer: server.NewMCPServer("zenfd", "2.3.0", server.WithToolCapabilities(true), server.WithResourceCapabilities(true, true)),
		store:     store,
		repoPath:  repoPath,
	}
	s.registerTools()
	s.registerResources()
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
	// 1. zenfd_push_plan
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_push_plan",
			mcp.WithDescription("Ingests a living plan markdown file into SQLite WAL and resolves initial task DAG."),
			mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the plan markdown file (e.g. .agents/scratch/plans/zenfd/README.md)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filePath, err := request.RequireString("file_path")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(s.repoPath, filePath)
			}
			plan, tasks, err := parser.ParseMarkdownPlanFile(filePath, s.repoPath)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to parse plan: %v", err)), nil
			}
			if err := s.store.SavePlanWithTasks(ctx, plan, tasks); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to save plan: %v", err)), nil
			}
			res := map[string]any{
				"plan_slug":   plan.Slug,
				"plan_title":  plan.Title,
				"tasks_count": len(tasks),
				"status":      "INGESTED",
			}
			bytes, _ := json.MarshalIndent(res, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// 2. zenfd_claim_task
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_claim_task",
			mcp.WithDescription("Atomically claims a READY task with mutual exclusion lock, transitions to RUNNING, and returns its ContextPacket."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Unique task ID to claim")),
			mcp.WithString("worker_id", mcp.Description("Optional worker or subagent identifier")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			workerID := request.GetString("worker_id", fmt.Sprintf("worker-%d", os.Getpid()))
			packet, err := s.store.ClaimTask(ctx, taskID, workerID, os.Getpid())
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to claim task %s: %v", taskID, err)), nil
			}
			bytes, _ := json.MarshalIndent(packet, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// 3. zenfd_request_review
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_request_review",
			mcp.WithDescription("Pauses task execution, sets state to AWAITING_REVIEW, and surfaces scope drift question on Web Cockpit."),
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
			return mcp.NewToolResultText(fmt.Sprintf("Task %s paused at AWAITING_REVIEW. Developer decision requested.", taskID)), nil
		},
	)

	// 4. zenfd_resolve_review
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_resolve_review",
			mcp.WithDescription("Resolves an AWAITING_REVIEW task by recording developer guidance and transitioning back to RUNNING."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID currently in AWAITING_REVIEW")),
			mcp.WithString("response", mcp.Required(), mcp.Description("Developer decision or guidance string")),
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
			return mcp.NewToolResultText(fmt.Sprintf("Task %s resumed back to RUNNING with developer decision.", taskID)), nil
		},
	)

	// 5. zenfd_complete_task
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_complete_task",
			mcp.WithDescription("Completes execution of a task. If task uses a worktree, stages to AWAITING_MERGE; otherwise transitions to COMPLETED."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID in RUNNING status")),
			mcp.WithString("summary", mcp.Description("Optional completion summary")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			task, err := s.store.GetTask(ctx, taskID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("task not found: %v", err)), nil
			}
			if task.WorktreePath != "" {
				if err := s.store.StageMerge(ctx, taskID); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to stage merge: %v", err)), nil
				}
				return mcp.NewToolResultText(fmt.Sprintf("Task %s staged to AWAITING_MERGE (worktree: %s).", taskID, task.WorktreePath)), nil
			}
			if err := s.store.UpdateTaskStatus(ctx, taskID, task.Status, domain.StatusCompleted); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to complete task: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Task %s transitioned to COMPLETED. Downstream DAG unlocked.", taskID)), nil
		},
	)

	// 6. zenfd_merge_task
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_merge_task",
			mcp.WithDescription("Merges an AWAITING_MERGE worktree task to the target branch. Transitions to COMPLETED or flags CONFLICT."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID to merge")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			merged, conflicts, err := s.store.MergeTask(ctx, taskID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to merge task: %v", err)), nil
			}
			if !merged {
				return mcp.NewToolResultText(fmt.Sprintf("Merge conflict encountered for task %s. Conflicts in files: %s", taskID, strings.Join(conflicts, ", "))), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Task %s cleanly merged and transitioned to COMPLETED.", taskID)), nil
		},
	)

	// 7. zenfd_fail_task
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_fail_task",
			mcp.WithDescription("Transitions a RUNNING or CLAIMED task to FAILED."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID to fail")),
			mcp.WithString("reason", mcp.Required(), mcp.Description("Reason for failure")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireString("task_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			task, err := s.store.GetTask(ctx, taskID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := s.store.UpdateTaskStatus(ctx, taskID, task.Status, domain.StatusFailed); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to fail task: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Task %s transitioned to FAILED.", taskID)), nil
		},
	)

	// 8. zenfd_get_task
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

	// 9. zenfd_read_doc
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_read_doc",
			mcp.WithDescription("Retrieves an architectural spec or project memory document from SQLite WAL."),
			mcp.WithString("slug", mcp.Required(), mcp.Description("Document slug (e.g. architecture-overview, memory)")),
			mcp.WithString("repo_path", mcp.Description("Optional repository path filter")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			slug, err := request.RequireString("slug")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			repo := request.GetString("repo_path", s.repoPath)
			doc, err := s.store.GetDoc(ctx, repo, slug)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("doc not found: %v", err)), nil
			}
			bytes, _ := json.MarshalIndent(doc, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// 10. zenfd_write_doc
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_write_doc",
			mcp.WithDescription("Upserts a living architecture, schema, or memory document in SQLite WAL."),
			mcp.WithString("slug", mcp.Required(), mcp.Description("Unique document slug")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Human-readable document title")),
			mcp.WithString("category", mcp.Description("Category: ARCHITECTURE, SCHEMA, ADR, GUIDE, MEMORY")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Markdown content of the document")),
			mcp.WithArray("mermaid_diagrams", mcp.Description("Optional list of raw mermaid code blocks")),
			mcp.WithString("repo_path", mcp.Description("Optional repository path")),
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
			content, err := request.RequireString("content")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			category := request.GetString("category", "ARCHITECTURE")
			mermaid := request.GetStringSlice("mermaid_diagrams", []string{})
			repo := request.GetString("repo_path", s.repoPath)

			doc := domain.ProjectDoc{
				Slug:            slug,
				RepoPath:        repo,
				Title:           title,
				Category:        category,
				ContentMarkdown: content,
				MermaidDiagrams: mermaid,
			}
			if err := s.store.UpsertDoc(ctx, doc); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to save doc: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Document %s (%s) saved successfully.", slug, title)), nil
		},
	)

	// 11. zenfd_export_docs
	s.mcpServer.AddTool(
		mcp.NewTool("zenfd_export_docs",
			mcp.WithDescription("Exports living project documentation from SQLite WAL into repository markdown files."),
			mcp.WithString("target_dir", mcp.Description("Target directory to export (defaults to docs/)")),
			mcp.WithString("repo_path", mcp.Description("Optional repository path")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			repo := request.GetString("repo_path", s.repoPath)
			targetDir := request.GetString("target_dir", "docs")
			if !filepath.IsAbs(targetDir) {
				targetDir = filepath.Join(repo, targetDir)
			}
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to create directory: %v", err)), nil
			}
			docs, err := s.store.ListDocs(ctx, repo, "")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list docs: %v", err)), nil
			}
			exported := []string{}
			for _, doc := range docs {
				filename := fmt.Sprintf("%s.md", doc.Slug)
				filePath := filepath.Join(targetDir, filename)
				content := fmt.Sprintf("# %s\n\n> Category: %s | Version: %d\n\n%s\n", doc.Title, doc.Category, doc.Version, doc.ContentMarkdown)
				if err := os.WriteFile(filePath, []byte(content), 0644); err == nil {
					exported = append(exported, filePath)
				}
			}
			return mcp.NewToolResultText(fmt.Sprintf("Exported %d docs to %s:\n%s", len(exported), targetDir, strings.Join(exported, "\n"))), nil
		},
	)
}

func (s *Server) registerResources() {
	// 1. zenfd://global/env
	s.mcpServer.AddResource(
		mcp.NewResource("zenfd://global/env", "Host Environment Specifications", mcp.WithResourceDescription("Host machine specs, toolchains, and environment health")),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			gitVer := "unknown"
			if out, err := exec.Command("git", "version").Output(); err == nil {
				gitVer = strings.TrimSpace(string(out))
			}
			nodeVer := "unknown"
			if out, err := exec.Command("node", "-v").Output(); err == nil {
				nodeVer = strings.TrimSpace(string(out))
			}

			envData := map[string]any{
				"os":          runtime.GOOS,
				"arch":        runtime.GOARCH,
				"num_cpu":     runtime.NumCPU(),
				"go_version":  runtime.Version(),
				"git_version": gitVer,
				"node":        nodeVer,
				"repo_path":   s.repoPath,
			}
			bytes, _ := json.MarshalIndent(envData, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "zenfd://global/env",
					MIMEType: "application/json",
					Text:     string(bytes),
				},
			}, nil
		},
	)

	// 2. zenfd://global/skills
	s.mcpServer.AddResource(
		mcp.NewResource("zenfd://global/skills", "Installed Skills Catalog", mcp.WithResourceDescription("Catalog of all agent skills installed in repository and system")),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			skillsDir := filepath.Join(s.repoPath, ".agents", "skills")
			entries, _ := os.ReadDir(skillsDir)
			skills := []string{}
			for _, entry := range entries {
				if entry.IsDir() {
					skills = append(skills, entry.Name())
				}
			}
			bytes, _ := json.MarshalIndent(skills, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "zenfd://global/skills",
					MIMEType: "application/json",
					Text:     string(bytes),
				},
			}, nil
		},
	)

	// 3. zenfd://project/rules
	s.mcpServer.AddResource(
		mcp.NewResource("zenfd://project/rules", "Project Invariant Rules", mcp.WithResourceDescription("Active project rules from AGENTS.md")),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			rulesPath := filepath.Join(s.repoPath, "AGENTS.md")
			content, err := os.ReadFile(rulesPath)
			if err != nil {
				content = []byte("# No AGENTS.md found\n")
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "zenfd://project/rules",
					MIMEType: "text/markdown",
					Text:     string(content),
				},
			}, nil
		},
	)

	// 4. zenfd://project/memory
	s.mcpServer.AddResource(
		mcp.NewResource("zenfd://project/memory", "Active Project Memory", mcp.WithResourceDescription("Distilled project memory and operational decisions")),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			doc, err := s.store.GetDoc(ctx, s.repoPath, "memory")
			if err == nil {
				return []mcp.ResourceContents{
					mcp.TextResourceContents{
						URI:      "zenfd://project/memory",
						MIMEType: "text/markdown",
						Text:     doc.ContentMarkdown,
					},
				}, nil
			}
			memPath := filepath.Join(s.repoPath, "MEMORY.md")
			content, err := os.ReadFile(memPath)
			if err != nil {
				content = []byte("# No active MEMORY.md found\n")
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "zenfd://project/memory",
					MIMEType: "text/markdown",
					Text:     string(content),
				},
			}, nil
		},
	)

	// 5. zenfd://project/docs
	s.mcpServer.AddResource(
		mcp.NewResource("zenfd://project/docs", "Project Architecture Docs Index", mcp.WithResourceDescription("Index of all architectural docs and schemas")),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			docs, err := s.store.ListDocs(ctx, s.repoPath, "")
			if err != nil {
				return nil, err
			}
			bytes, _ := json.MarshalIndent(docs, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "zenfd://project/docs",
					MIMEType: "application/json",
					Text:     string(bytes),
				},
			}, nil
		},
	)

	// 6. Dynamic Resource Template: zenfd://project/docs/{slug}
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("zenfd://project/docs/{slug}", "Project Document Detail", mcp.WithTemplateDescription("Document markdown and diagrams by slug")),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			prefix := "zenfd://project/docs/"
			if !strings.HasPrefix(request.Params.URI, prefix) {
				return nil, fmt.Errorf("invalid uri format: %s", request.Params.URI)
			}
			slug := strings.TrimPrefix(request.Params.URI, prefix)
			doc, err := s.store.GetDoc(ctx, s.repoPath, slug)
			if err != nil {
				return nil, err
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "text/markdown",
					Text:     doc.ContentMarkdown,
				},
			}, nil
		},
	)

	// 7. Dynamic Resource Template: zenfd://tasks/{id}/context
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("zenfd://tasks/{id}/context", "Task Context Packet", mcp.WithTemplateDescription("Formatted prompt packet for agent execution")),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			uri := request.Params.URI
			taskID := strings.TrimPrefix(uri, "zenfd://tasks/")
			taskID = strings.TrimSuffix(taskID, "/context")
			packet, err := s.store.GetTaskContextPacket(ctx, taskID)
			if err != nil {
				return nil, err
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "text/markdown",
					Text:     packet.PromptMarkdown,
				},
			}, nil
		},
	)

	// 8. Dynamic Resource Template: zenfd://tasks/{id}/diff
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("zenfd://tasks/{id}/diff", "Task Git Diff", mcp.WithTemplateDescription("Git diff generated by task execution")),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			uri := request.Params.URI
			taskID := strings.TrimPrefix(uri, "zenfd://tasks/")
			taskID = strings.TrimSuffix(taskID, "/diff")
			task, err := s.store.GetTask(ctx, taskID)
			if err != nil {
				return nil, err
			}
			diff := ""
			workDir := task.WorktreePath
			if workDir == "" {
				workDir = task.RepoPath
			}
			if workDir != "" {
				if out, err := exec.Command("git", "-C", workDir, "diff", "HEAD").Output(); err == nil {
					diff = string(out)
				}
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "text/plain",
					Text:     diff,
				},
			}, nil
		},
	)
}
