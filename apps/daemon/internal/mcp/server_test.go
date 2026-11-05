package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func newTestServer(t *testing.T) (*Server, *storage.Store) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_mcp.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := storage.NewStore(db)
	return NewServer(store), store
}

func callTool(t *testing.T, srv *Server, name string, args map[string]any) *mcpgo.CallToolResult {
	t.Helper()
	tool := srv.mcpServer.GetTool(name)
	if tool == nil {
		t.Fatalf("Tool %s not registered", name)
	}
	res, err := tool.Handler(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: name, Arguments: args},
	})
	if err != nil {
		t.Fatalf("%s handler failed: %v", name, err)
	}
	if res == nil || res.IsError {
		t.Fatalf("%s returned error result: %+v", name, res)
	}
	return res
}

func resultText(t *testing.T, res *mcpgo.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("expected tool result content")
	}
	text, ok := mcpgo.AsTextContent(res.Content[0])
	if !ok {
		t.Fatalf("expected text content, got %T", res.Content[0])
	}
	return text.Text
}

// createTaskTool wraps the consolidated zenfd_tasks create action with a
// single-item batch, which is how the tests exercise per-task semantics.
func createTaskTool(t *testing.T, srv *Server, planID string, task map[string]any) {
	t.Helper()
	callTool(t, srv, "zenfd_tasks", map[string]any{
		"action":  "create",
		"plan_id": planID,
		"tasks":   []any{task},
	})
}

func TestMCPServer_Surface(t *testing.T) {
	srv, _ := newTestServer(t)

	expectedTools := []string{
		"zenfd_plans",
		"zenfd_tasks",
		"zenfd_claim_task",
		"zenfd_review",
		"zenfd_transition_task",
		"zenfd_attach_media",
	}
	for _, name := range expectedTools {
		if srv.mcpServer.GetTool(name) == nil {
			t.Errorf("expected tool %s to be registered", name)
		}
	}

	removedTools := []string{
		"zenfd_create_plan",
		"zenfd_create_task",
		"zenfd_request_review",
		"zenfd_resolve_review",
		"zenfd_get_task",
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
		if srv.mcpServer.GetTool(name) != nil {
			t.Errorf("tool %s should not be registered", name)
		}
	}

	for uri := range srv.mcpServer.ListResources() {
		t.Errorf("unexpected direct resource registered: %s", uri)
	}
}

// TestMCPServer_RequiredParams locks the documented parameter contract; the
// skill card and MCP.md must not drift from the registered schemas.
func TestMCPServer_RequiredParams(t *testing.T) {
	srv, _ := newTestServer(t)

	expected := map[string][]string{
		"zenfd_plans":           {"action"},
		"zenfd_tasks":           {"action"},
		"zenfd_claim_task":      {"task_id"},
		"zenfd_review":          {"action", "task_id"},
		"zenfd_transition_task": {"task_ids", "action"},
		"zenfd_attach_media":    {"task_id", "file_path"},
	}

	for name, required := range expected {
		tool := srv.mcpServer.GetTool(name)
		if tool == nil {
			t.Fatalf("tool %s is not registered", name)
		}
		got := make(map[string]bool, len(tool.Tool.InputSchema.Required))
		for _, field := range tool.Tool.InputSchema.Required {
			got[field] = true
		}
		for _, field := range required {
			if !got[field] {
				t.Errorf("%s: expected required parameter %q, schema has %v", name, field, tool.Tool.InputSchema.Required)
			}
		}
	}
}

func TestMCPServer_LifecycleAndTransition(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "zenfd_plans", map[string]any{"action": "create",
		"slug":  "test-mcp-plan",
		"title": "MCP Server Verification",
	})
	createTaskTool(t, srv, "test-mcp-plan", map[string]any{
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
	callTool(t, srv, "zenfd_review", map[string]any{"action": "request",
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

	callTool(t, srv, "zenfd_review", map[string]any{"action": "resolve",
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

	callTool(t, srv, "zenfd_plans", map[string]any{"action": "create", "slug": "wt-plan", "title": "Worktree Plan"})
	createTaskTool(t, srv, "wt-plan", map[string]any{
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

	callTool(t, srv, "zenfd_plans", map[string]any{"action": "create", "slug": "batch-plan", "title": "Batch Plan"})
	for _, id := range []string{"batch-01", "batch-02"} {
		createTaskTool(t, srv, "batch-plan", map[string]any{
			"id":    id,
			"title": id,
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

	callTool(t, srv, "zenfd_plans", map[string]any{"action": "create", "slug": "state-plan", "title": "State Plan"})
	for _, id := range []string{"state-fail", "state-reset", "state-cancel"} {
		createTaskTool(t, srv, "state-plan", map[string]any{
			"id":    id,
			"title": id,
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

	callTool(t, srv, "zenfd_plans", map[string]any{"action": "create", "slug": "read-plan", "title": "Read Plan"})
	createTaskTool(t, srv, "read-plan", map[string]any{
		"id":    "read-01",
		"title": "Read Task",
	})

	res := callTool(t, srv, "zenfd_tasks", map[string]any{"action": "get", "task_id": "read-01"})
	if resultText(t, res) == "" {
		t.Fatal("expected get_task content")
	}
}

func TestMCPServer_BatchCreateAndArchive(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "zenfd_plans", map[string]any{"action": "create", "slug": "batch-plan", "title": "Batch Plan"})

	res := callTool(t, srv, "zenfd_tasks", map[string]any{
		"action":  "create",
		"plan_id": "batch-plan",
		"tasks": []any{
			map[string]any{"id": "b-01", "title": "first"},
			map[string]any{"id": "b-02", "title": "second", "dependencies": []string{"b-01"}},
		},
	})
	var created struct {
		Created int      `json:"created"`
		TaskIDs []string `json:"task_ids"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &created); err != nil {
		t.Fatalf("failed to decode create result: %v", err)
	}
	if created.Created != 2 || len(created.TaskIDs) != 2 {
		t.Fatalf("unexpected create result: %+v", created)
	}

	first, _ := store.GetTask(ctx, "b-01")
	second, _ := store.GetTask(ctx, "b-02")
	if first.Status != domain.StatusReady || second.Status != domain.StatusPending {
		t.Errorf("unexpected statuses: %s / %s", first.Status, second.Status)
	}

	callTool(t, srv, "zenfd_plans", map[string]any{"action": "archive", "slug": "batch-plan"})

	res = callTool(t, srv, "zenfd_plans", map[string]any{"action": "list"})
	if strings.TrimSpace(resultText(t, res)) != "[]" {
		t.Errorf("archived plan should be hidden from default list, got %s", resultText(t, res))
	}

	res = callTool(t, srv, "zenfd_plans", map[string]any{"action": "list", "include_archived": true})
	var plans []domain.Plan
	if err := json.Unmarshal([]byte(resultText(t, res)), &plans); err != nil {
		t.Fatalf("failed to decode plan list: %v", err)
	}
	if len(plans) != 1 || plans[0].Slug != "batch-plan" || plans[0].ArchivedAt == nil {
		t.Fatalf("expected archived plan in include_archived list, got %+v", plans)
	}
}

func TestMCPServer_AttachMediaOnlyBeforeClaim(t *testing.T) {
	t.Setenv("ZENFD_ATTACHMENTS_DIR", t.TempDir())

	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "zenfd_plans", map[string]any{"action": "create", "slug": "media-plan", "title": "Media Plan"})
	createTaskTool(t, srv, "media-plan", map[string]any{"id": "m-01", "title": "media task"})

	src := filepath.Join(t.TempDir(), "design.png")
	if err := os.WriteFile(src, []byte("\x89PNG\r\n\x1a\nfake"), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	res := callTool(t, srv, "zenfd_attach_media", map[string]any{
		"task_id":   "m-01",
		"file_path": src,
		"name":      "Design mockup",
	})
	var artifact domain.Artifact
	if err := json.Unmarshal([]byte(resultText(t, res)), &artifact); err != nil {
		t.Fatalf("failed to decode artifact: %v", err)
	}
	if artifact.Role != domain.ArtifactRoleInput || artifact.Name != "Design mockup" {
		t.Errorf("unexpected artifact: %+v", artifact)
	}

	artifacts, err := store.GetArtifactsByTask(ctx, "m-01", string(domain.ArtifactRoleInput))
	if err != nil {
		t.Fatalf("GetArtifactsByTask failed: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 input artifact, got %d", len(artifacts))
	}

	if _, err := store.ClaimTaskWithMeta(ctx, "m-01", "worker", "", "CODER"); err != nil {
		t.Fatalf("claim failed: %v", err)
	}

	tool := srv.mcpServer.GetTool("zenfd_attach_media")
	late, err := tool.Handler(ctx, mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{
			Name: "zenfd_attach_media",
			Arguments: map[string]any{
				"task_id":   "m-01",
				"file_path": src,
			},
		},
	})
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if late == nil || !late.IsError {
		t.Fatalf("expected RUNNING task to reject media, got %+v", late)
	}
}
