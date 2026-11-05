package mcp_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"zenfd/internal/domain"
	zenfdmcp "zenfd/internal/mcp"
	"zenfd/internal/storage"
)

func newTestServer(t *testing.T) (*zenfdmcp.Server, *storage.Store) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_mcp.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := storage.NewStore(db)
	return zenfdmcp.NewServer(store), store
}

func callTool(t *testing.T, srv *zenfdmcp.Server, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	tool := srv.MCPServer().GetTool(name)
	if tool == nil {
		t.Fatalf("Tool %s not registered", name)
	}
	res, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	if err != nil {
		t.Fatalf("%s handler failed: %v", name, err)
	}
	if res == nil || res.IsError {
		t.Fatalf("%s returned error result: %+v", name, res)
	}
	return res
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("expected tool result content")
	}
	text, ok := mcp.AsTextContent(res.Content[0])
	if !ok {
		t.Fatalf("expected text content, got %T", res.Content[0])
	}
	return text.Text
}

func TestMCPServer_Surface(t *testing.T) {
	srv, _ := newTestServer(t)

	expectedTools := []string{
		"zenfd_create_plan",
		"zenfd_create_task",
		"zenfd_claim_task",
		"zenfd_request_review",
		"zenfd_resolve_review",
		"zenfd_transition_task",
		"zenfd_get_task",
	}
	for _, name := range expectedTools {
		if srv.MCPServer().GetTool(name) == nil {
			t.Errorf("expected tool %s to be registered", name)
		}
	}

	removedTools := []string{
		"zenfd_push_plan",
		"zenfd_reclaim_task",
		"zenfd_evict_task",
		"zenfd_complete_task",
		"zenfd_fail_task",
		"zenfd_merge_task",
		"zenfd_register_session",
		"zenfd_read_doc",
		"zenfd_write_doc",
		"zenfd_export_docs",
		"zenfd_upsert_spec",
		"zenfd_get_spec",
		"zenfd_search_specs",
		"zenfd_snapshot_plan",
	}
	for _, name := range removedTools {
		if srv.MCPServer().GetTool(name) != nil {
			t.Errorf("tool %s should not be registered", name)
		}
	}

	for uri := range srv.MCPServer().ListResources() {
		t.Errorf("unexpected direct resource registered: %s", uri)
	}
}

func TestMCPServer_LifecycleAndTransition(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "zenfd_create_plan", map[string]any{
		"slug":  "test-mcp-plan",
		"title": "MCP Server Verification",
	})
	callTool(t, srv, "zenfd_create_task", map[string]any{
		"plan_id":               "test-mcp-plan",
		"id":                    "task-mcp-01",
		"title":                 "Test MCP Tool Lifecycle",
		"description":           "Verify claim, review, and transition tools via MCP",
		"priority":              "P1",
		"acceptance_criteria":   []string{"MCP claim returns context packet", "Transition completes task"},
		"verification_commands": []string{"go test ./..."},
	})

	// Claim.
	callTool(t, srv, "zenfd_claim_task", map[string]any{
		"task_id":   "task-mcp-01",
		"worker_id": "test-agent-worker",
	})
	task, _ := store.GetTask(ctx, "task-mcp-01")
	if task.Status != domain.StatusRunning {
		t.Fatalf("expected RUNNING after claim, got %s", task.Status)
	}
	if len(task.AcceptanceCriteria) != 2 || len(task.VerificationCommands) != 1 {
		t.Errorf("create_task did not persist criteria/verification: %+v", task)
	}

	// Worker requests review; the main agent relays the human decision and resolves.
	callTool(t, srv, "zenfd_request_review", map[string]any{
		"task_id": "task-mcp-01",
		"reason":  "Schema drift",
		"details": "Need developer decision on migration vs recreate",
		"options": []string{"Option A: Migrate", "Option B: Recreate"},
	})
	task, _ = store.GetTask(ctx, "task-mcp-01")
	if task.Status != domain.StatusAwaitingReview {
		t.Fatalf("expected AWAITING_REVIEW, got %s", task.Status)
	}
	if task.ReviewReason != "Schema drift" || len(task.ReviewOptions) != 2 {
		t.Errorf("expected review question persisted, got %+v", task)
	}

	callTool(t, srv, "zenfd_resolve_review", map[string]any{
		"task_id":  "task-mcp-01",
		"response": "Option A chosen",
	})
	task, _ = store.GetTask(ctx, "task-mcp-01")
	if task.Status != domain.StatusReady {
		t.Fatalf("expected READY after review resolution, got %s", task.Status)
	}

	// Re-claim and complete without a worktree.
	callTool(t, srv, "zenfd_claim_task", map[string]any{
		"task_id":   "task-mcp-01",
		"worker_id": "subagent-coder-2",
	})
	res := callTool(t, srv, "zenfd_transition_task", map[string]any{
		"task_ids": []string{"task-mcp-01"},
		"action":   "COMPLETE",
		"summary":  "Implementation verified",
	})
	var payload struct {
		Results []struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &payload); err != nil {
		t.Fatalf("failed to decode transition result: %v", err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Status != "COMPLETED" {
		t.Fatalf("unexpected transition result: %+v", payload)
	}

	task, _ = store.GetTask(ctx, "task-mcp-01")
	if task.Status != domain.StatusCompleted {
		t.Fatalf("expected COMPLETED, got %s", task.Status)
	}
	if task.Notes == "" {
		t.Error("expected completion summary appended to Notes")
	}
}

func TestMCPServer_WorktreeMergeOutcome(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "zenfd_create_plan", map[string]any{"slug": "wt-plan", "title": "Worktree Plan"})
	callTool(t, srv, "zenfd_create_task", map[string]any{
		"plan_id":       "wt-plan",
		"id":            "task-wt-01",
		"title":         "Worktree Task",
		"worktree_path": "/tmp/fake-worktree",
	})
	callTool(t, srv, "zenfd_claim_task", map[string]any{"task_id": "task-wt-01"})

	callTool(t, srv, "zenfd_transition_task", map[string]any{
		"task_ids": []string{"task-wt-01"},
		"action":   "COMPLETE",
	})
	task, _ := store.GetTask(ctx, "task-wt-01")
	if task.Status != domain.StatusAwaitingMerge {
		t.Fatalf("expected AWAITING_MERGE, got %s", task.Status)
	}

	callTool(t, srv, "zenfd_transition_task", map[string]any{
		"task_ids":      []string{"task-wt-01"},
		"action":        "COMPLETE",
		"merge_outcome": "CONFLICT",
	})
	task, _ = store.GetTask(ctx, "task-wt-01")
	if task.Status != domain.StatusConflict {
		t.Fatalf("expected CONFLICT, got %s", task.Status)
	}

	callTool(t, srv, "zenfd_transition_task", map[string]any{
		"task_ids":      []string{"task-wt-01"},
		"action":        "COMPLETE",
		"merge_outcome": "MERGED",
	})
	task, _ = store.GetTask(ctx, "task-wt-01")
	if task.Status != domain.StatusCompleted {
		t.Fatalf("expected COMPLETED after CONFLICT -> MERGED, got %s", task.Status)
	}
}

func TestMCPServer_BatchTransitionIsPerTask(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "zenfd_create_plan", map[string]any{"slug": "batch-plan", "title": "Batch Plan"})
	for _, id := range []string{"batch-01", "batch-02"} {
		callTool(t, srv, "zenfd_create_task", map[string]any{
			"plan_id": "batch-plan",
			"id":      id,
			"title":   id,
		})
		callTool(t, srv, "zenfd_claim_task", map[string]any{"task_id": id})
	}

	// batch-01 completes, missing batch-99 fails without blocking batch-02.
	res := callTool(t, srv, "zenfd_transition_task", map[string]any{
		"task_ids": []string{"batch-01", "batch-99", "batch-02"},
		"action":   "COMPLETE",
	})
	var payload struct {
		Results []struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &payload); err != nil {
		t.Fatalf("failed to decode batch result: %v", err)
	}
	if len(payload.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(payload.Results))
	}
	if payload.Results[1].Error == "" {
		t.Error("expected error for missing batch-99")
	}

	for _, id := range []string{"batch-01", "batch-02"} {
		task, _ := store.GetTask(ctx, id)
		if task.Status != domain.StatusCompleted {
			t.Errorf("expected %s COMPLETED, got %s", id, task.Status)
		}
	}
}

func TestMCPServer_TransitionFailResetCancel(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "zenfd_create_plan", map[string]any{"slug": "state-plan", "title": "State Plan"})
	for _, id := range []string{"state-fail", "state-reset", "state-cancel"} {
		callTool(t, srv, "zenfd_create_task", map[string]any{
			"plan_id": "state-plan",
			"id":      id,
			"title":   id,
		})
		callTool(t, srv, "zenfd_claim_task", map[string]any{"task_id": id})
	}

	callTool(t, srv, "zenfd_transition_task", map[string]any{
		"task_ids": []string{"state-fail"},
		"action":   "FAIL",
		"reason":   "compile error",
	})
	task, _ := store.GetTask(ctx, "state-fail")
	if task.Status != domain.StatusFailed {
		t.Fatalf("expected FAILED, got %s", task.Status)
	}

	callTool(t, srv, "zenfd_transition_task", map[string]any{
		"task_ids": []string{"state-reset"},
		"action":   "RESET",
		"reason":   "worker hung",
	})
	task, _ = store.GetTask(ctx, "state-reset")
	if task.Status != domain.StatusReady {
		t.Fatalf("expected READY, got %s", task.Status)
	}

	callTool(t, srv, "zenfd_transition_task", map[string]any{
		"task_ids": []string{"state-cancel"},
		"action":   "CANCEL",
		"reason":   "obsolete",
	})
	task, _ = store.GetTask(ctx, "state-cancel")
	if task.Status != domain.StatusCancelled {
		t.Fatalf("expected CANCELLED, got %s", task.Status)
	}
}

func TestMCPServer_GetTask(t *testing.T) {
	srv, _ := newTestServer(t)

	callTool(t, srv, "zenfd_create_plan", map[string]any{"slug": "read-plan", "title": "Read Plan"})
	callTool(t, srv, "zenfd_create_task", map[string]any{
		"plan_id": "read-plan",
		"id":      "read-01",
		"title":   "Read Task",
	})

	res := callTool(t, srv, "zenfd_get_task", map[string]any{"task_id": "read-01"})
	if resultText(t, res) == "" {
		t.Fatal("expected get_task content")
	}
}
