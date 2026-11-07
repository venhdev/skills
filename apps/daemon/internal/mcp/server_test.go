package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"zenf/internal/domain"
	"zenf/internal/storage"
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
	if err := store.CreateProject(context.Background(), domain.Project{
		Slug: "test-project", Name: "Test Project", RootPath: tempDir,
	}); err != nil {
		t.Fatalf("Failed to seed project: %v", err)
	}
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

// createTaskTool wraps the consolidated tasks create action with a
// single-item batch, which is how the tests exercise per-task semantics.
func createTaskTool(t *testing.T, srv *Server, planID string, task map[string]any) {
	t.Helper()
	callTool(t, srv, "tasks", map[string]any{
		"action":    "create",
		"plan_slug": planID,
		"tasks":     []any{task},
	})
}

func TestMCPServer_Surface(t *testing.T) {
	srv, _ := newTestServer(t)

	expectedTools := []string{
		"projects",
		"plans",
		"tasks",
		"docs",
		"agents",
		"sessions",
		"search",
		"claim_task",
		"review",
		"transition_task",
	}
	for _, name := range expectedTools {
		if srv.mcpServer.GetTool(name) == nil {
			t.Errorf("expected tool %s to be registered", name)
		}
	}

	removedTools := []string{
		"attach_media",
		"zenfd_plans",
		"zenfd_tasks",
		"zenfd_claim_task",
		"zenfd_review",
		"zenfd_transition_task",
		"zenfd_attach_media",
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
		"projects":   {"action"},
		"plans":      {"action"},
		"tasks":      {"action"},
		"docs":       {"action"},
		"agents":     {"action"},
		"sessions":   {"action"},
		"search":     {"query"},
		"claim_task": {"task_id"},
		"review":     {"action", "task_id"},
		// transition_task accepts either task_ids+action or items[], so the
		// schema enforces no single required field.
		"transition_task": {},
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

	callTool(t, srv, "plans", map[string]any{"action": "create",
		"project_slug": "test-project",
		"slug":         "test-mcp-plan",
		"title":        "MCP Server Verification",
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
	callTool(t, srv, "claim_task", map[string]any{
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
	callTool(t, srv, "review", map[string]any{"action": "request",
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

	callTool(t, srv, "review", map[string]any{"action": "resolve",
		"task_id":  "task-mcp-01",
		"response": "Option A chosen",
	})
	task, _ = store.GetTask(ctx, "task-mcp-01")
	if task.Status != domain.StatusReady {
		t.Fatalf("expected READY after review resolution, got %s", task.Status)
	}

	// Re-claim and complete without a worktree.
	callTool(t, srv, "claim_task", map[string]any{
		"task_id":   "task-mcp-01",
		"worker_id": "subagent-coder-2",
	})
	res := callTool(t, srv, "transition_task", map[string]any{
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

	callTool(t, srv, "plans", map[string]any{"action": "create", "project_slug": "test-project", "slug": "wt-plan", "title": "Worktree Plan"})
	createTaskTool(t, srv, "wt-plan", map[string]any{
		"id":            "task-wt-01",
		"title":         "Worktree Task",
		"worktree_path": "/tmp/fake-worktree",
	})
	callTool(t, srv, "claim_task", map[string]any{"task_id": "task-wt-01"})

	callTool(t, srv, "transition_task", map[string]any{
		"task_ids": []string{"task-wt-01"},
		"action":   "COMPLETE",
	})
	task, _ := store.GetTask(ctx, "task-wt-01")
	if task.Status != domain.StatusAwaitingMerge {
		t.Fatalf("expected AWAITING_MERGE, got %s", task.Status)
	}

	callTool(t, srv, "transition_task", map[string]any{
		"task_ids":      []string{"task-wt-01"},
		"action":        "COMPLETE",
		"merge_outcome": "CONFLICT",
	})
	task, _ = store.GetTask(ctx, "task-wt-01")
	if task.Status != domain.StatusConflict {
		t.Fatalf("expected CONFLICT, got %s", task.Status)
	}

	callTool(t, srv, "transition_task", map[string]any{
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

	callTool(t, srv, "plans", map[string]any{"action": "create", "project_slug": "test-project", "slug": "batch-plan", "title": "Batch Plan"})
	for _, id := range []string{"batch-01", "batch-02"} {
		createTaskTool(t, srv, "batch-plan", map[string]any{
			"id":    id,
			"title": id,
		})
		callTool(t, srv, "claim_task", map[string]any{"task_id": id})
	}

	// batch-01 completes, missing batch-99 fails without blocking batch-02.
	res := callTool(t, srv, "transition_task", map[string]any{
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

	callTool(t, srv, "plans", map[string]any{"action": "create", "project_slug": "test-project", "slug": "state-plan", "title": "State Plan"})
	for _, id := range []string{"state-fail", "state-reset", "state-cancel"} {
		createTaskTool(t, srv, "state-plan", map[string]any{
			"id":    id,
			"title": id,
		})
		callTool(t, srv, "claim_task", map[string]any{"task_id": id})
	}

	callTool(t, srv, "transition_task", map[string]any{
		"task_ids": []string{"state-fail"},
		"action":   "FAIL",
		"reason":   "compile error",
	})
	task, _ := store.GetTask(ctx, "state-fail")
	if task.Status != domain.StatusFailed {
		t.Fatalf("expected FAILED, got %s", task.Status)
	}

	callTool(t, srv, "transition_task", map[string]any{
		"task_ids": []string{"state-reset"},
		"action":   "RESET",
		"reason":   "worker hung",
	})
	task, _ = store.GetTask(ctx, "state-reset")
	if task.Status != domain.StatusReady {
		t.Fatalf("expected READY, got %s", task.Status)
	}

	callTool(t, srv, "transition_task", map[string]any{
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

	callTool(t, srv, "plans", map[string]any{"action": "create", "project_slug": "test-project", "slug": "read-plan", "title": "Read Plan"})
	createTaskTool(t, srv, "read-plan", map[string]any{
		"id":    "read-01",
		"title": "Read Task",
	})

	res := callTool(t, srv, "tasks", map[string]any{"action": "get", "task_id": "read-01"})
	if resultText(t, res) == "" {
		t.Fatal("expected get_task content")
	}
}

func TestMCPServer_BatchCreateAndArchive(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "plans", map[string]any{"action": "create", "project_slug": "test-project", "slug": "batch-plan", "title": "Batch Plan"})

	res := callTool(t, srv, "tasks", map[string]any{
		"action":    "create",
		"plan_slug": "batch-plan",
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

	callTool(t, srv, "plans", map[string]any{"action": "archive", "slug": "batch-plan"})

	res = callTool(t, srv, "plans", map[string]any{"action": "list"})
	if strings.TrimSpace(resultText(t, res)) != "[]" {
		t.Errorf("archived plan should be hidden from default list, got %s", resultText(t, res))
	}

	res = callTool(t, srv, "plans", map[string]any{"action": "list", "include_archived": true})
	var plans []domain.Plan
	if err := json.Unmarshal([]byte(resultText(t, res)), &plans); err != nil {
		t.Fatalf("failed to decode plan list: %v", err)
	}
	if len(plans) != 1 || plans[0].Slug != "batch-plan" || plans[0].ArchivedAt == nil {
		t.Fatalf("expected archived plan in include_archived list, got %+v", plans)
	}
}

func TestMCPServer_TaskMediaLifecycle(t *testing.T) {
	t.Setenv("ZENFD_ATTACHMENTS_DIR", t.TempDir())

	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "plans", map[string]any{"action": "create", "project_slug": "test-project", "slug": "media-plan", "title": "Media Plan"})

	src := filepath.Join(t.TempDir(), "design.png")
	if err := os.WriteFile(src, []byte("\x89PNG\r\n\x1a\nfake"), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	res := callTool(t, srv, "tasks", map[string]any{
		"action":    "create",
		"plan_slug": "media-plan",
		"tasks": []any{
			map[string]any{
				"id":    "m-01",
				"title": "media task",
				"media": []any{map[string]any{"file_path": src, "name": "Design mockup"}},
			},
		},
	})
	var created struct {
		Media []struct {
			Artifact *domain.Artifact `json:"artifact"`
			Error    string           `json:"error"`
		} `json:"media"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &created); err != nil {
		t.Fatalf("failed to decode create result: %v", err)
	}
	if len(created.Media) != 1 || created.Media[0].Artifact == nil {
		t.Fatalf("expected one attached artifact, got %+v", created.Media)
	}
	artifact := *created.Media[0].Artifact
	if artifact.Role != domain.ArtifactRoleInput || artifact.Name != "Design mockup" {
		t.Errorf("unexpected artifact: %+v", artifact)
	}
	if _, err := os.Stat(artifact.FilePath); err != nil {
		t.Errorf("attached file is missing: %v", err)
	}

	// Detach while READY: row and file both go away.
	res = callTool(t, srv, "tasks", map[string]any{
		"action":       "update",
		"task_id":      "m-01",
		"media_remove": []string{artifact.ID},
	})
	var removed struct {
		Removed []struct {
			ArtifactID string `json:"artifact_id"`
			Removed    bool   `json:"removed"`
			Error      string `json:"error"`
		} `json:"media_removed"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &removed); err != nil {
		t.Fatalf("failed to decode remove result: %v", err)
	}
	if len(removed.Removed) != 1 || !removed.Removed[0].Removed {
		t.Fatalf("expected media removal, got %+v", removed.Removed)
	}
	if _, err := os.Stat(artifact.FilePath); !os.IsNotExist(err) {
		t.Errorf("expected file removal, stat err = %v", err)
	}
	if left, err := store.GetArtifactsByTask(ctx, "m-01", string(domain.ArtifactRoleInput)); err != nil || len(left) != 0 {
		t.Errorf("expected no input artifacts, got %d (err %v)", len(left), err)
	}

	// Attach again while READY, then claim.
	callTool(t, srv, "tasks", map[string]any{
		"action":    "update",
		"task_id":   "m-01",
		"media_add": []any{map[string]any{"file_path": src}},
	})
	artifacts, err := store.GetArtifactsByTask(ctx, "m-01", string(domain.ArtifactRoleInput))
	if err != nil || len(artifacts) != 1 {
		t.Fatalf("expected one re-attached artifact, got %d (err %v)", len(artifacts), err)
	}

	if _, err := store.ClaimTask(ctx, "m-01", domain.ClaimOptions{WorkerID: "worker", WorkerRole: "CODER"}); err != nil {
		t.Fatalf("claim failed: %v", err)
	}

	tool := srv.mcpServer.GetTool("tasks")
	late, err := tool.Handler(ctx, mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{
			Name: "tasks",
			Arguments: map[string]any{
				"action":    "update",
				"task_id":   "m-01",
				"media_add": []any{map[string]any{"file_path": src}},
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

func TestMCPServer_CreatePlanWithTasksAndMedia(t *testing.T) {
	t.Setenv("ZENFD_ATTACHMENTS_DIR", t.TempDir())

	srv, store := newTestServer(t)
	ctx := context.Background()

	src := filepath.Join(t.TempDir(), "spec.pdf")
	if err := os.WriteFile(src, []byte("%PDF-1.4 fake"), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	res := callTool(t, srv, "plans", map[string]any{
		"action":       "create",
		"project_slug": "test-project",
		"slug":         "inline-plan",
		"title":        "Inline DAG",
		"tasks": []any{
			map[string]any{"id": "i-01", "title": "root"},
			map[string]any{
				"id":           "i-02",
				"title":        "child",
				"dependencies": []string{"i-01"},
				"media":        []any{map[string]any{"file_path": src, "name": "Spec"}},
			},
		},
	})
	var payload struct {
		TaskIDs []string `json:"task_ids"`
		Media   []struct {
			Artifact *domain.Artifact `json:"artifact"`
			Error    string           `json:"error"`
		} `json:"media"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &payload); err != nil {
		t.Fatalf("failed to decode create result: %v", err)
	}
	if len(payload.TaskIDs) != 2 {
		t.Fatalf("expected 2 tasks, got %v", payload.TaskIDs)
	}
	if len(payload.Media) != 1 || payload.Media[0].Artifact == nil {
		t.Fatalf("expected media on the child task, got %+v", payload.Media)
	}

	root, err := store.GetTask(ctx, "i-01")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	child, err := store.GetTask(ctx, "i-02")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if root.Status != domain.StatusReady || child.Status != domain.StatusPending {
		t.Errorf("unexpected statuses: root=%s child=%s", root.Status, child.Status)
	}
}

func TestMCPServer_TaskUpdateMetadata(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "plans", map[string]any{"action": "create", "project_slug": "test-project", "slug": "update-plan", "title": "Update"})
	createTaskTool(t, srv, "update-plan", map[string]any{"id": "u-01", "title": "old title"})
	createTaskTool(t, srv, "update-plan", map[string]any{"id": "u-02", "title": "blocker"})

	res := callTool(t, srv, "tasks", map[string]any{
		"action":       "update",
		"task_id":      "u-01",
		"title":        "new title",
		"dependencies": []string{"u-02"},
		"priority":     "P0",
	})
	var payload struct {
		Task domain.Task `json:"task"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &payload); err != nil {
		t.Fatalf("failed to decode update result: %v", err)
	}
	if payload.Task.Title != "new title" || payload.Task.Priority != domain.PriorityP0 {
		t.Errorf("metadata not applied: %+v", payload.Task)
	}
	if payload.Task.Status != domain.StatusPending {
		t.Errorf("expected PENDING after adding an unfinished dependency, got %s", payload.Task.Status)
	}

	if err := store.UpdateTaskStatus(ctx, "u-02", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("failed to start blocker: %v", err)
	}
	if err := store.UpdateTaskStatus(ctx, "u-02", domain.StatusRunning, domain.StatusCompleted); err != nil {
		t.Fatalf("failed to complete blocker: %v", err)
	}
	updated, err := store.GetTask(ctx, "u-01")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if updated.Status != domain.StatusReady {
		t.Errorf("expected READY after blocker completion, got %s", updated.Status)
	}
}

func TestMCPServer_GetTaskPacketOptIn(t *testing.T) {
	srv, _ := newTestServer(t)

	callTool(t, srv, "plans", map[string]any{"action": "create", "project_slug": "test-project", "slug": "get-plan", "title": "Get"})
	createTaskTool(t, srv, "get-plan", map[string]any{"id": "g-01", "title": "read"})

	res := callTool(t, srv, "tasks", map[string]any{"action": "get", "task_id": "g-01"})
	var metadata map[string]any
	if err := json.Unmarshal([]byte(resultText(t, res)), &metadata); err != nil {
		t.Fatalf("failed to decode metadata: %v", err)
	}
	if _, ok := metadata["prompt_markdown"]; ok {
		t.Error("default get must not compile a ContextPacket")
	}

	res = callTool(t, srv, "tasks", map[string]any{
		"action": "get", "task_id": "g-01", "include_packet": true,
	})
	var packet domain.ContextPacket
	if err := json.Unmarshal([]byte(resultText(t, res)), &packet); err != nil {
		t.Fatalf("failed to decode packet: %v", err)
	}
	if packet.PromptMarkdown == "" {
		t.Error("include_packet=true must return the prompt")
	}
}

func TestMCPServer_MixedTransitions(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()

	callTool(t, srv, "plans", map[string]any{"action": "create", "project_slug": "test-project", "slug": "mix-plan", "title": "Mix"})
	createTaskTool(t, srv, "mix-plan", map[string]any{"id": "x-01", "title": "pass"})
	createTaskTool(t, srv, "mix-plan", map[string]any{"id": "x-02", "title": "fail"})
	for _, id := range []string{"x-01", "x-02"} {
		callTool(t, srv, "claim_task", map[string]any{"task_id": id})
	}

	res := callTool(t, srv, "transition_task", map[string]any{
		"items": []any{
			map[string]any{"task_id": "x-01", "action": "COMPLETE", "summary": "ok"},
			map[string]any{"task_id": "x-02", "action": "FAIL", "reason": "bad"},
		},
	})
	var payload struct {
		Results []struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &payload); err != nil {
		t.Fatalf("failed to decode results: %v", err)
	}
	if len(payload.Results) != 2 {
		t.Fatalf("expected 2 results, got %+v", payload.Results)
	}
	first, _ := store.GetTask(ctx, "x-01")
	second, _ := store.GetTask(ctx, "x-02")
	if first.Status != domain.StatusCompleted || second.Status != domain.StatusFailed {
		t.Errorf("unexpected statuses: %s / %s", first.Status, second.Status)
	}
}

func TestMCPServer_DocumentRegistry(t *testing.T) {
	srv, _ := newTestServer(t)

	docPath := filepath.Join(t.TempDir(), "spec.md")
	if err := os.WriteFile(docPath, []byte("# Spec\n\nClaim preflight outside the transaction.\n"), 0o644); err != nil {
		t.Fatalf("failed to write document: %v", err)
	}

	res := callTool(t, srv, "docs", map[string]any{
		"action":       "create",
		"project_slug": "test-project",
		"key":          "SPEC",
		"slug":         "spec",
		"type":         "SPEC",
		"scope":        "PROJECT",
		"title":        "Spec",
		"path":         docPath,
		"tags":         []any{"core"},
	})
	var created domain.Document
	if err := json.Unmarshal([]byte(resultText(t, res)), &created); err != nil {
		t.Fatalf("failed to decode document: %v", err)
	}
	if created.ID == "" || created.ContentHash == "" || created.ByteSize == 0 {
		t.Fatalf("unexpected created document: %+v", created)
	}

	res = callTool(t, srv, "docs", map[string]any{
		"action":         "search",
		"query":          "preflight",
		"project_slug":   "test-project",
		"include_global": true,
	})
	var hits []domain.DocumentSearchResult
	if err := json.Unmarshal([]byte(resultText(t, res)), &hits); err != nil {
		t.Fatalf("failed to decode hits: %v", err)
	}
	if len(hits) != 1 || hits[0].Document.ID != created.ID {
		t.Fatalf("unexpected hits: %+v", hits)
	}

	res = callTool(t, srv, "docs", map[string]any{
		"action":  "refresh",
		"id":      created.ID,
		"content": "# Spec v2\n\nUpdated.\n",
	})
	var refreshed domain.Document
	if err := json.Unmarshal([]byte(resultText(t, res)), &refreshed); err != nil {
		t.Fatalf("failed to decode refreshed document: %v", err)
	}
	if refreshed.Version != 2 || refreshed.ContentHash == created.ContentHash {
		t.Fatalf("expected version 2 with new hash, got %+v", refreshed)
	}
}
