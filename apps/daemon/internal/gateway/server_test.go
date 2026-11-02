package gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/scheduler"
	"zenfd/internal/storage"
	"zenfd/internal/supervisor"

	"github.com/coder/websocket"
)

func setupTestGateway(t *testing.T) (*Server, string, *storage.Store, *supervisor.ProcessSupervisor, func()) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_gateway.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	store := storage.NewStore(db)
	super := supervisor.NewProcessSupervisor(tempDir)
	cfg := scheduler.DefaultConfig()
	sched := scheduler.NewDAGScheduler(cfg, store, super)

	srv := NewServer(store, super, sched)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	baseURL := fmt.Sprintf("http://%s", ln.Addr().String())

	go func() {
		_ = srv.Serve(ln)
	}()

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = store.Close()
		_ = db.Close()
	}

	return srv, baseURL, store, super, cleanup
}

func TestGateway_PlansAndTasksLifecycle(t *testing.T) {
	_, baseURL, _, _, cleanup := setupTestGateway(t)
	defer cleanup()

	// 1. POST /api/v1/plans
	planPayload := CreatePlanRequest{
		Slug:     "plan-alpha",
		Title:    "Alpha Engineering Plan",
		RepoPath: "/test/repo",
		Tasks: []domain.Task{
			{ID: "T-01", Title: "Scaffold Schema", Dependencies: []string{}},
			{ID: "T-02", Title: "Compile Engine", Dependencies: []string{"T-01"}},
		},
	}

	bodyBytes, _ := json.Marshal(planPayload)
	postResp, err := http.Post(baseURL+"/api/v1/plans", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to post plan: %v", err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d", postResp.StatusCode)
	}

	// 2. GET /api/v1/plans/plan-alpha
	getResp, err := http.Get(baseURL + "/api/v1/plans/plan-alpha")
	if err != nil {
		t.Fatalf("failed to get plan: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", getResp.StatusCode)
	}

	var planData struct {
		Plan  domain.Plan   `json:"plan"`
		Tasks []domain.Task `json:"tasks"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&planData); err != nil {
		t.Fatalf("failed to decode plan response: %v", err)
	}

	if planData.Plan.Slug != "plan-alpha" || len(planData.Tasks) != 2 {
		t.Fatalf("unexpected plan payload: %+v", planData)
	}

	// 3. GET /api/v1/tasks?status=READY
	listResp, err := http.Get(baseURL + "/api/v1/tasks?status=READY")
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	defer listResp.Body.Close()

	var tasks []domain.Task
	if err := json.NewDecoder(listResp.Body).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode tasks list: %v", err)
	}

	if len(tasks) != 1 || tasks[0].ID != "T-01" {
		t.Fatalf("expected 1 READY task (T-01), got: %+v", tasks)
	}
}

func TestGateway_SSEEventStreaming(t *testing.T) {
	srv, baseURL, _, _, cleanup := setupTestGateway(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/events/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for SSE, got %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)

	// First line should be initial ping
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read initial SSE line: %v", err)
	}
	if !strings.Contains(line, "event: ping") {
		t.Fatalf("expected event: ping, got %s", line)
	}

	// Trigger broadcast
	go func() {
		time.Sleep(50 * time.Millisecond)
		srv.Broadcast(domain.EventMessage{
			Type:      "task:status",
			TaskID:    "T-01",
			Payload:   `{"status":"RUNNING"}`,
			Timestamp: time.Now().UTC(),
		})
	}()

	// Read until our broadcast event appears
	received := false
	for i := 0; i < 10; i++ {
		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, "task:status") {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive broadcasted task:status event via SSE")
	}
}

func TestGateway_WebSocketTerminal(t *testing.T) {
	_, baseURL, _, _, cleanup := setupTestGateway(t)
	defer cleanup()

	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/api/v1/terminal/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send TERMINAL_PING
	if err := conn.Write(ctx, websocket.MessageText, []byte("TERMINAL_PING")); err != nil {
		t.Fatalf("failed to write websocket ping: %v", err)
	}

	typ, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read websocket pong: %v", err)
	}

	if typ != websocket.MessageText || string(msg) != "TERMINAL_PONG_ACK" {
		t.Fatalf("expected TERMINAL_PONG_ACK, got: %s", string(msg))
	}

	// Send Resize Command
	resizeJSON := `{"type":"resize","rows":40,"cols":120}`
	if err := conn.Write(ctx, websocket.MessageText, []byte(resizeJSON)); err != nil {
		t.Fatalf("failed to write resize command: %v", err)
	}
}

func TestGateway_ArtifactServing(t *testing.T) {
	_, baseURL, store, _, cleanup := setupTestGateway(t)
	defer cleanup()

	tempDir := t.TempDir()
	artPath := filepath.Join(tempDir, "patch.diff")
	diffContent := "--- a/main.go\n+++ b/main.go\n@@ -1,2 +1,3 @@\n+package main\n"
	if err := os.WriteFile(artPath, []byte(diffContent), 0644); err != nil {
		t.Fatalf("failed to create artifact file: %v", err)
	}

	plan := domain.Plan{Slug: "plan-art", Title: "Art Plan", RepoPath: "/repo", Status: "ACTIVE"}
	task := domain.Task{ID: "task-100", PlanID: "plan-art", RepoPath: "/repo", Title: "Art Task", Status: domain.StatusReady}
	if err := store.SavePlanWithTasks(context.Background(), plan, []domain.Task{task}); err != nil {
		t.Fatalf("failed to save plan and task: %v", err)
	}

	art := domain.Artifact{
		ID:        "art-100",
		RunID:     "run-100",
		TaskID:    "task-100",
		Type:      domain.ArtifactTypeDiff,
		FilePath:  artPath,
		MimeType:  "text/x-diff",
		ByteSize:  int64(len(diffContent)),
		Sha256:    "abc1234",
		CreatedAt: time.Now().UTC(),
	}

	if err := store.RecordArtifact(context.Background(), art); err != nil {
		t.Fatalf("failed to record artifact in storage: %v", err)
	}

	// GET /api/v1/artifacts/art-100
	resp, err := http.Get(baseURL + "/api/v1/artifacts/art-100")
	if err != nil {
		t.Fatalf("failed to retrieve artifact: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	if buf.String() != diffContent {
		t.Fatalf("expected diff content, got %s", buf.String())
	}
}

func TestGateway_StaticSPA(t *testing.T) {
	_, baseURL, _, _, cleanup := setupTestGateway(t)
	defer cleanup()

	// GET /
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("failed to get root: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// GET /unknown-client-route (SPA fallback)
	resp2, err := http.Get(baseURL + "/dag/visualizer")
	if err != nil {
		t.Fatalf("failed to get client route: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on client route fallback, got %d", resp2.StatusCode)
	}
}
