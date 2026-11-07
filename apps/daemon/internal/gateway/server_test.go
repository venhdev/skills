package gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func setupTestGateway(t *testing.T) (*Server, string, *storage.Store, func()) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_gateway.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	store := storage.NewStore(db)
	if err := store.CreateProject(context.Background(), domain.Project{
		Slug: "test-project", Name: "Test Project", RootPath: tempDir,
	}); err != nil {
		t.Fatalf("failed to seed project: %v", err)
	}

	srv := NewServer(store)

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
		_ = db.Close()
	}

	return srv, baseURL, store, cleanup
}

func TestGateway_PlansAndTasksLifecycle(t *testing.T) {
	_, baseURL, _, cleanup := setupTestGateway(t)
	defer cleanup()

	// 1. POST /api/v1/plans
	planPayload := CreatePlanRequest{
		Slug:        "plan-alpha",
		ProjectSlug: "test-project",
		Title:       "Alpha Engineering Plan",
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

	var taskPayload struct {
		Tasks []domain.Task `json:"tasks"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&taskPayload); err != nil {
		t.Fatalf("failed to decode tasks list: %v", err)
	}
	tasks := taskPayload.Tasks

	if len(tasks) != 1 || tasks[0].ID != "T-01" {
		t.Fatalf("expected 1 READY task (T-01), got: %+v", tasks)
	}
}

func TestGateway_ListPlansIncludesPlanWithoutTasks(t *testing.T) {
	_, baseURL, _, cleanup := setupTestGateway(t)
	defer cleanup()

	bodyBytes, _ := json.Marshal(CreatePlanRequest{Slug: "empty-plan", ProjectSlug: "test-project", Title: "Empty Plan"})
	resp, err := http.Post(baseURL+"/api/v1/plans", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to post plan: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d", resp.StatusCode)
	}

	listResp, err := http.Get(baseURL + "/api/v1/plans")
	if err != nil {
		t.Fatalf("failed to list plans: %v", err)
	}
	defer listResp.Body.Close()

	var payload struct {
		Plans []domain.Plan `json:"plans"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode plans response: %v", err)
	}

	for _, plan := range payload.Plans {
		if plan.Slug == "empty-plan" {
			if plan.Title != "Empty Plan" {
				t.Errorf("expected plan title to be returned, got %q", plan.Title)
			}
			return
		}
	}
	t.Fatalf("expected empty plan in /api/v1/plans, got: %+v", payload.Plans)
}

func TestGateway_SSEEventStreaming(t *testing.T) {
	srv, baseURL, _, cleanup := setupTestGateway(t)
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
			Type:      "task.status",
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
		if strings.Contains(line, "task.status") {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive broadcasted task.status event via SSE")
	}
}

func TestGateway_ArtifactServing(t *testing.T) {
	_, baseURL, store, cleanup := setupTestGateway(t)
	defer cleanup()

	tempDir := t.TempDir()
	artPath := filepath.Join(tempDir, "notes.md")
	artContent := "--- a/main.go\n+++ b/main.go\n@@ -1,2 +1,3 @@\n+package main\n"
	if err := os.WriteFile(artPath, []byte(artContent), 0644); err != nil {
		t.Fatalf("failed to create artifact file: %v", err)
	}

	plan := domain.Plan{Slug: "plan-art", ProjectSlug: "test-project", Title: "Art Plan", Status: domain.PlanStatusActive}
	task := domain.Task{ID: "task-100", ProjectSlug: "test-project", PlanSlug: "plan-art", Title: "Art Task", Status: domain.StatusReady}
	if err := store.CreatePlanWithTasks(context.Background(), plan, []domain.Task{task}); err != nil {
		t.Fatalf("failed to save plan and task: %v", err)
	}

	art := domain.Artifact{
		ID:        "art-100",
		TaskID:    "task-100",
		Type:      domain.ArtifactTypeMarkdown,
		FilePath:  artPath,
		MimeType:  "text/markdown",
		ByteSize:  int64(len(artContent)),
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
	if buf.String() != artContent {
		t.Fatalf("expected artifact content, got %s", buf.String())
	}
}

func TestGateway_StaticSPA(t *testing.T) {
	_, baseURL, _, cleanup := setupTestGateway(t)
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

func TestGateway_TaskAttachmentsAndLogs(t *testing.T) {
	_, baseURL, store, cleanup := setupTestGateway(t)
	defer cleanup()

	plan := domain.Plan{Slug: "plan-attach", ProjectSlug: "test-project", Title: "Attach Plan", Status: domain.PlanStatusActive}
	task := domain.Task{ID: "task-attach-1", ProjectSlug: "test-project", PlanSlug: "plan-attach", Title: "Attach Task", Status: domain.StatusReady}
	if err := store.CreatePlanWithTasks(context.Background(), plan, []domain.Task{task}); err != nil {
		t.Fatalf("failed to save plan and task: %v", err)
	}

	// 1. Test POST /api/v1/tasks/{id}/attachments (multipart upload)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "mockup.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write([]byte("fake-png-binary-content"))
	_ = writer.WriteField("name", "Design Mockup")
	_ = writer.Close()

	uploadReq, _ := http.NewRequest("POST", baseURL+"/api/v1/tasks/task-attach-1/attachments", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

	uploadResp, err := http.DefaultClient.Do(uploadReq)
	if err != nil {
		t.Fatalf("failed to upload attachment: %v", err)
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", uploadResp.StatusCode)
	}

	var createdArt domain.Artifact
	if err := json.NewDecoder(uploadResp.Body).Decode(&createdArt); err != nil {
		t.Fatalf("failed to decode created artifact: %v", err)
	}
	if createdArt.Role != domain.ArtifactRoleInput || createdArt.Name != "Design Mockup" {
		t.Errorf("unexpected artifact payload: %+v", createdArt)
	}

	// 2. Test GET /api/v1/tasks/{id}/artifacts
	listResp, err := http.Get(baseURL + "/api/v1/tasks/task-attach-1/artifacts")
	if err != nil {
		t.Fatalf("failed to list artifacts: %v", err)
	}
	defer listResp.Body.Close()

	var artPayload struct {
		Artifacts []domain.Artifact `json:"artifacts"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&artPayload); err != nil {
		t.Fatalf("failed to decode artifact list: %v", err)
	}
	artList := artPayload.Artifacts
	if len(artList) != 1 || artList[0].ID != createdArt.ID {
		t.Errorf("expected 1 artifact in list, got %+v", artList)
	}

}

func TestGateway_DocumentRegistry(t *testing.T) {
	_, baseURL, _, cleanup := setupTestGateway(t)
	defer cleanup()

	docPath := filepath.Join(t.TempDir(), "architecture.md")
	content := "# Architecture\n\nClaim preflight keeps packet assembly outside the transaction.\n"
	if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write document: %v", err)
	}

	payload := map[string]any{
		"project_slug": "test-project",
		"key":          "ARCH",
		"slug":         "architecture",
		"type":         "ARCHITECTURE",
		"scope":        "PROJECT",
		"title":        "Architecture",
		"path":         docPath,
		"summary":      "Registry probe",
		"tags":         []string{"architecture", "core"},
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/api/v1/docs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create document failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var created domain.Document
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode document: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated document id in response")
	}
	if created.ContentHash == "" || created.ByteSize == 0 {
		t.Fatalf("expected hash and size, got %+v", created)
	}

	listResp, err := http.Get(baseURL + "/api/v1/docs?project_slug=test-project&include_global=true")
	if err != nil {
		t.Fatalf("list documents failed: %v", err)
	}
	defer listResp.Body.Close()
	var list struct {
		Documents []domain.Document `json:"documents"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode document list: %v", err)
	}
	if len(list.Documents) != 1 || list.Documents[0].ID != created.ID {
		t.Fatalf("unexpected document list: %+v", list.Documents)
	}

	searchResp, err := http.Get(baseURL + "/api/v1/docs/search?query=preflight&project_slug=test-project&include_global=true")
	if err != nil {
		t.Fatalf("search documents failed: %v", err)
	}
	defer searchResp.Body.Close()
	var searchPayload struct {
		Results []domain.DocumentSearchResult `json:"results"`
	}
	if err := json.NewDecoder(searchResp.Body).Decode(&searchPayload); err != nil {
		t.Fatalf("failed to decode search hits: %v", err)
	}
	hits := searchPayload.Results
	if len(hits) != 1 || hits[0].Document.ID != created.ID {
		t.Fatalf("unexpected search hits: %+v", hits)
	}

	refreshBody := strings.NewReader(`{"content":"# Architecture v2\n\nUpdated claim preflight.\n"}`)
	refreshResp, err := http.Post(baseURL+"/api/v1/docs/"+created.ID+"/refresh", "application/json", refreshBody)
	if err != nil {
		t.Fatalf("refresh document failed: %v", err)
	}
	defer refreshResp.Body.Close()
	var refreshed domain.Document
	if err := json.NewDecoder(refreshResp.Body).Decode(&refreshed); err != nil {
		t.Fatalf("failed to decode refreshed document: %v", err)
	}
	if refreshed.Version != 2 || refreshed.ContentHash == created.ContentHash {
		t.Fatalf("expected version 2 with new hash, got %+v", refreshed)
	}
}

func TestGateway_JSONErrorFormat(t *testing.T) {
	_, baseURL, _, cleanup := setupTestGateway(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/api/v1/tasks/non-existent-task-id")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON error: %v", err)
	}
	if errResp.Error.Code != "RESOURCE_NOT_FOUND" {
		t.Errorf("expected RESOURCE_NOT_FOUND code, got %q", errResp.Error.Code)
	}
}

func TestGateway_AgentRegistry(t *testing.T) {
	_, baseURL, _, cleanup := setupTestGateway(t)
	defer cleanup()

	body := strings.NewReader(`{"slug":"coder-a","name":"Coder A","persona":"CODER","mode":"HEADLESS","goal":"implement","start_conditions":["task READY"],"stop_conditions":["task AWAITING_REVIEW"],"enabled":true}`)
	resp, err := http.Post(baseURL+"/api/v1/agents", "application/json", body)
	if err != nil {
		t.Fatalf("create agent failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var created domain.Agent
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode agent: %v", err)
	}
	if created.Persona != domain.PersonaCoder || !created.Enabled {
		t.Fatalf("unexpected agent: %+v", created)
	}

	patchBody := strings.NewReader(`{"enabled":false}`)
	req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/agents/coder-a", patchBody)
	req.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update agent failed: %v", err)
	}
	defer patchResp.Body.Close()
	var updated domain.Agent
	if err := json.NewDecoder(patchResp.Body).Decode(&updated); err != nil {
		t.Fatalf("failed to decode updated agent: %v", err)
	}
	if updated.Enabled {
		t.Fatal("expected agent to be disabled")
	}

	sessionResp, err := http.Get(baseURL + "/api/v1/sessions?active_only=true")
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	defer sessionResp.Body.Close()
	if sessionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing sessions, got %d", sessionResp.StatusCode)
	}
}
