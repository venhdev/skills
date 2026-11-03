package mcp_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"zenfd/internal/domain"
	zenfdmcp "zenfd/internal/mcp"
	"zenfd/internal/storage"
)

func TestMCPServer_ToolsAndResources(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_mcp.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	plan := domain.Plan{
		Slug:     "test-mcp-plan",
		Title:    "MCP Server Verification",
		RepoPath: tempDir,
		Status:   "ACTIVE",
	}

	task := domain.Task{
		ID:                   "task-mcp-01",
		PlanID:               plan.Slug,
		RepoPath:             tempDir,
		Title:                "Test MCP Tool Lifecycle",
		Description:          "Verify claim, review, and complete tools via MCP",
		Priority:             domain.PriorityP1,
		Complexity:           domain.ComplexityMedium,
		RiskLevel:            domain.RiskLow,
		Dependencies:         []string{},
		AcceptanceCriteria:   []string{"MCP claim returns context packet", "Review pause functions"},
		VerificationCommands: []string{"go test -v ./..."},
		Status:               domain.StatusReady,
	}

	if err := store.SavePlanWithTasks(ctx, plan, []domain.Task{task}); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	srv := zenfdmcp.NewServer(store, tempDir)
	mcpServer := srv.MCPServer()

	// 1. Test zenfd_claim_task
	claimTool := mcpServer.GetTool("zenfd_claim_task")
	if claimTool == nil {
		t.Fatal("Tool zenfd_claim_task not registered")
	}
	claimReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_claim_task",
			Arguments: map[string]any{
				"task_id":   "task-mcp-01",
				"worker_id": "test-agent-worker",
			},
		},
	}
	claimRes, err := claimTool.Handler(ctx, claimReq)
	if err != nil {
		t.Fatalf("claim handler failed: %v", err)
	}
	if claimRes == nil || claimRes.IsError {
		t.Fatalf("Claim returned error result: %+v", claimRes)
	}

	// 2. Test zenfd_request_review
	reviewTool := mcpServer.GetTool("zenfd_request_review")
	reviewReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_request_review",
			Arguments: map[string]any{
				"task_id": "task-mcp-01",
				"reason":  "Schema drift",
				"details": "Need developer decision on migration vs recreate",
				"options": []string{"Option A: Migrate", "Option B: Recreate"},
			},
		},
	}
	reviewRes, err := reviewTool.Handler(ctx, reviewReq)
	if err != nil || (reviewRes != nil && reviewRes.IsError) {
		t.Fatalf("Request review failed: err=%v, res=%+v", err, reviewRes)
	}

	t1, _ := store.GetTask(ctx, "task-mcp-01")
	if t1.Status != domain.StatusAwaitingReview {
		t.Fatalf("Expected status AWAITING_REVIEW, got: %s", t1.Status)
	}

	// 3. Test zenfd_resolve_review
	resolveTool := mcpServer.GetTool("zenfd_resolve_review")
	resolveReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_resolve_review",
			Arguments: map[string]any{
				"task_id":  "task-mcp-01",
				"response": "Option A chosen",
			},
		},
	}
	resolveRes, err := resolveTool.Handler(ctx, resolveReq)
	if err != nil || (resolveRes != nil && resolveRes.IsError) {
		t.Fatalf("Resolve review failed: err=%v, res=%+v", err, resolveRes)
	}

	t2, _ := store.GetTask(ctx, "task-mcp-01")
	if t2.Status != domain.StatusReady {
		t.Fatalf("Expected status READY after review resolution, got: %s", t2.Status)
	}

	// Re-claim task to resume execution
	claimReq2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_claim_task",
			Arguments: map[string]any{
				"task_id":   "task-mcp-01",
				"worker_id": "subagent-coder-2",
			},
		},
	}
	claimRes2, err := claimTool.Handler(ctx, claimReq2)
	if err != nil || (claimRes2 != nil && claimRes2.IsError) {
		t.Fatalf("Re-claim task failed: %v", err)
	}

	// 4. Test zenfd_complete_task
	compTool := mcpServer.GetTool("zenfd_complete_task")
	compReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_complete_task",
			Arguments: map[string]any{
				"task_id": "task-mcp-01",
				"summary": "Implementation verified",
			},
		},
	}
	compRes, err := compTool.Handler(ctx, compReq)
	if err != nil || (compRes != nil && compRes.IsError) {
		t.Fatalf("Complete task failed: err=%v, res=%+v", err, compRes)
	}

	t3, _ := store.GetTask(ctx, "task-mcp-01")
	if t3.Status != domain.StatusCompleted {
		t.Fatalf("Expected status COMPLETED, got: %s", t3.Status)
	}

	// 5. Test zenfd_write_doc and zenfd_read_doc
	writeTool := mcpServer.GetTool("zenfd_write_doc")
	writeDocReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_write_doc",
			Arguments: map[string]any{
				"slug":     "arch-doc",
				"title":    "Living Architecture",
				"category": "ARCHITECTURE",
				"content":  "# Living Architecture Spec",
			},
		},
	}
	writeRes, err := writeTool.Handler(ctx, writeDocReq)
	if err != nil || (writeRes != nil && writeRes.IsError) {
		t.Fatalf("Write doc failed: err=%v, res=%+v", err, writeRes)
	}

	readTool := mcpServer.GetTool("zenfd_read_doc")
	readDocReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_read_doc",
			Arguments: map[string]any{
				"slug": "arch-doc",
			},
		},
	}
	readRes, err := readTool.Handler(ctx, readDocReq)
	if err != nil || (readRes != nil && readRes.IsError) {
		t.Fatalf("Read doc failed: err=%v, res=%+v", err, readRes)
	}

	// 6. Test Resource Read: zenfd://global/env
	resMap := mcpServer.ListResources()
	envResource := resMap["zenfd://global/env"]
	if envResource == nil {
		t.Fatal("Resource zenfd://global/env not registered")
	}
	envContents, err := envResource.Handler(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "zenfd://global/env",
		},
	})
	if err != nil || len(envContents) == 0 {
		t.Fatalf("Read resource env failed: err=%v, contents=%+v", err, envContents)
	}
	textContents, ok := mcp.AsTextResourceContents(envContents[0])
	if !ok || textContents.URI != "zenfd://global/env" {
		t.Fatalf("Unexpected env content: %+v", envContents[0])
	}
	var envData map[string]any
	if err := json.Unmarshal([]byte(textContents.Text), &envData); err != nil {
		t.Fatalf("Failed to parse env json: %v", err)
	}
	if envData["os"] == "" {
		t.Errorf("Expected os in env data, got: %+v", envData)
	}

	// 7. Test zenfd_upsert_spec
	upsertTool := mcpServer.GetTool("zenfd_upsert_spec")
	if upsertTool == nil {
		t.Fatal("Tool zenfd_upsert_spec not registered")
	}
	upsertRes, err := upsertTool.Handler(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_upsert_spec",
			Arguments: map[string]any{
				"slug":         "topology-mcp-test",
				"title":        "Topology MCP Test",
				"tier":         1.0,
				"kind":         "CONTEXT",
				"diagram_type": "mermaid:flowchart",
				"diagram_code": "flowchart TD\nA-->B",
				"notes":        "Test spec via MCP",
				"bound_paths":  []string{"internal/mcp/"},
			},
		},
	})
	if err != nil || (upsertRes != nil && upsertRes.IsError) {
		t.Fatalf("zenfd_upsert_spec failed: err=%v, res=%+v", err, upsertRes)
	}

	// 8. Test zenfd_get_spec
	getSpecTool := mcpServer.GetTool("zenfd_get_spec")
	if getSpecTool == nil {
		t.Fatal("Tool zenfd_get_spec not registered")
	}
	getSpecRes, err := getSpecTool.Handler(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_get_spec",
			Arguments: map[string]any{
				"slug": "topology-mcp-test",
			},
		},
	})
	if err != nil || (getSpecRes != nil && getSpecRes.IsError) {
		t.Fatalf("zenfd_get_spec failed: err=%v, res=%+v", err, getSpecRes)
	}

	// 9. Test zenfd_search_specs
	searchTool := mcpServer.GetTool("zenfd_search_specs")
	if searchTool == nil {
		t.Fatal("Tool zenfd_search_specs not registered")
	}
	searchRes, err := searchTool.Handler(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_search_specs",
			Arguments: map[string]any{
				"query": "Topology",
			},
		},
	})
	if err != nil || (searchRes != nil && searchRes.IsError) {
		t.Fatalf("zenfd_search_specs failed: err=%v, res=%+v", err, searchRes)
	}

	// 10. Test zenfd_snapshot_plan
	snapshotTool := mcpServer.GetTool("zenfd_snapshot_plan")
	if snapshotTool == nil {
		t.Fatal("Tool zenfd_snapshot_plan not registered")
	}
	snapshotRes, err := snapshotTool.Handler(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "zenfd_snapshot_plan",
			Arguments: map[string]any{
				"plan_slug":   plan.Slug,
				"folder_path": tempDir,
				"revision":    "v1.0",
				"author":      "agent-test",
				"summary":     "Initial test snapshot",
			},
		},
	})
	if err != nil || (snapshotRes != nil && snapshotRes.IsError) {
		t.Fatalf("zenfd_snapshot_plan failed: err=%v, res=%+v", err, snapshotRes)
	}
}
