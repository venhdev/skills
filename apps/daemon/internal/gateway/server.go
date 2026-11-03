package gateway

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"zenfd/internal/domain"

	"github.com/coder/websocket"
)

//go:embed web_dist/*
var defaultWebDist embed.FS

// CreatePlanRequest specifies the payload for plan ingestion via REST.
type CreatePlanRequest struct {
	Slug     string        `json:"slug"`
	Title    string        `json:"title"`
	RepoPath string        `json:"repo_path,omitempty"`
	Tasks    []domain.Task `json:"tasks"`
}

// Server provides the HTTP/SSE/WebSocket gateway for zenfd control plane.
type Server struct {
	storage    domain.StorageEngine
	supervisor domain.Supervisor
	scheduler  domain.Scheduler

	sseMu      sync.RWMutex
	sseClients map[chan domain.EventMessage]struct{}

	mux        *http.ServeMux
	httpServer *http.Server
	listener   net.Listener
	webFS      fs.FS
}

// NewServer constructs a new Gateway Server.
func NewServer(storage domain.StorageEngine, supervisor domain.Supervisor, scheduler domain.Scheduler) *Server {
	sub, err := fs.Sub(defaultWebDist, "web_dist")
	var webFS fs.FS
	if err == nil {
		webFS = sub
	}

	s := &Server{
		storage:    storage,
		supervisor: supervisor,
		scheduler:  scheduler,
		sseClients: make(map[chan domain.EventMessage]struct{}),
		mux:        http.NewServeMux(),
		webFS:      webFS,
	}

	s.setupRoutes()
	return s
}

// SetWebFS allows overriding the embedded web file system (e.g. for testing).
func (s *Server) SetWebFS(f fs.FS) {
	s.webFS = f
}

// setupRoutes registers all HTTP, SSE, and WebSocket endpoints using Go 1.22+ routing syntax.
func (s *Server) setupRoutes() {
	// Projects API
	s.mux.HandleFunc("GET /api/v1/projects", s.handleListProjects)

	// Plan APIs
	s.mux.HandleFunc("GET /api/v1/plans", s.handleListPlans)
	s.mux.HandleFunc("POST /api/v1/plans", s.handleCreatePlan)
	s.mux.HandleFunc("DELETE /api/v1/plans/{slug}", s.handleDeletePlan)
	s.mux.HandleFunc("GET /api/v1/plans/{slug}", s.handleGetPlan)

	// Task APIs
	s.mux.HandleFunc("GET /api/v1/tasks", s.handleListTasks)
	s.mux.HandleFunc("POST /api/v1/tasks", s.handleCreateTask)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/status", s.handleUpdateTaskStatus)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/dependencies", s.handleUpdateTaskDependencies)
	s.mux.HandleFunc("PATCH /api/v1/tasks/{id}/context", s.handleUpdateTaskContext)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/claim", s.handleClaimTask)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/reclaim", s.handleReclaimTask)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/evict", s.handleEvictTask)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/review", s.handleRequestReview)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/resolve", s.handleResolveReview)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/resolve-review", s.handleResolveReview)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/merge", s.handleMergeTask)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/attachments", s.handleUploadTaskAttachment)
	s.mux.HandleFunc("GET /api/v1/tasks/{id}/artifacts", s.handleListTaskArtifacts)
	s.mux.HandleFunc("GET /api/v1/tasks/{id}/logs", s.handleGetTaskLogs)

	// Agent Sessions API
	s.mux.HandleFunc("GET /api/v1/sessions", s.handleListSessions)
	s.mux.HandleFunc("POST /api/v1/sessions", s.handleRegisterSession)

	// Living Architecture & Project Docs APIs
	s.mux.HandleFunc("GET /api/v1/docs", s.handleListDocs)
	s.mux.HandleFunc("POST /api/v1/docs", s.handleUpsertDoc)
	s.mux.HandleFunc("POST /api/v1/docs/export", s.handleExportDocs)
	s.mux.HandleFunc("GET /api/v1/docs/{slug}", s.handleGetDoc)

	// Blueprint & Technical Specs APIs
	s.mux.HandleFunc("GET /api/v1/specs", s.handleListSpecs)
	s.mux.HandleFunc("POST /api/v1/specs", s.handleUpsertSpec)
	s.mux.HandleFunc("GET /api/v1/specs/search", s.handleSearchSpecs)
	s.mux.HandleFunc("GET /api/v1/specs/{slug}", s.handleGetSpec)

	// Plan Dossier Snapshots & Revisions APIs
	s.mux.HandleFunc("POST /api/v1/plans/{slug}/snapshot", s.handleSavePlanSnapshot)
	s.mux.HandleFunc("GET /api/v1/plans/{slug}/snapshot", s.handleGetPlanSnapshot)
	s.mux.HandleFunc("GET /api/v1/plans/{slug}/revisions", s.handleListPlanRevisions)
	s.mux.HandleFunc("GET /api/v1/plans/{slug}/diff", s.handleGetPlanDiff)

	// Global Environment & Toolchains API
	s.mux.HandleFunc("GET /api/v1/env/global", s.handleGetGlobalEnv)

	// Run Supervisor APIs
	s.mux.HandleFunc("POST /api/v1/runs/{id}/kill", s.handleKillRun)

	// Real-time Event Stream (SSE)
	s.mux.HandleFunc("GET /api/v1/events/stream", s.handleEventStream)

	// Interactive Terminal WebSocket
	s.mux.HandleFunc("GET /api/v1/terminal/ws", s.handleTerminalWS)
	s.mux.HandleFunc("GET /api/v1/terminal/ws/{run_id}", s.handleTerminalWS)

	// Artifact Retrieval API
	s.mux.HandleFunc("GET /api/v1/artifacts/{id}", s.handleGetArtifact)

	// Web UI SPA & Static Files
	s.mux.HandleFunc("GET /", s.handleStaticSPA)
}

// Handler returns the HTTP handler with CORS middleware applied.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Enable standard CORS for local dev & dashboard access
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		s.mux.ServeHTTP(w, r)
	})
}

// Broadcast distributes an event message to all connected SSE clients and persists to SQLite.
func (s *Server) Broadcast(evt domain.EventMessage) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	// Persist event to storage
	if s.storage != nil {
		_ = s.storage.AppendEvent(context.Background(), evt)
	}

	// Distribute non-blocking to active SSE clients
	s.sseMu.RLock()
	defer s.sseMu.RUnlock()

	for ch := range s.sseClients {
		select {
		case ch <- evt:
		default:
			// Slow consumer, skip to prevent stalling control plane
		}
	}
}

// handleCreatePlan handles POST /api/v1/plans
func (s *Server) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}

	if req.Slug == "" {
		http.Error(w, "missing required field: slug", http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		req.Title = req.Slug
	}

	plan := domain.Plan{
		Slug:      req.Slug,
		Title:     req.Title,
		RepoPath:  req.RepoPath,
		Status:    "ACTIVE",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	for i := range req.Tasks {
		if req.Tasks[i].PlanID == "" {
			req.Tasks[i].PlanID = req.Slug
		}
		if req.Tasks[i].RepoPath == "" {
			req.Tasks[i].RepoPath = req.RepoPath
		}
		if req.Tasks[i].Status == "" {
			if len(req.Tasks[i].Dependencies) == 0 {
				req.Tasks[i].Status = domain.StatusReady
			} else {
				req.Tasks[i].Status = domain.StatusPending
			}
		}
	}

	if s.storage == nil {
		http.Error(w, "no storage configured", http.StatusInternalServerError)
		return
	}

	if err := s.storage.SavePlanWithTasks(r.Context(), plan, req.Tasks); err != nil {
		http.Error(w, fmt.Sprintf("failed to submit plan: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		Type:      "plan:created",
		Payload:   fmt.Sprintf(`{"slug":"%s","tasks_count":%d}`, plan.Slug, len(req.Tasks)),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "INGESTED",
		"slug":   plan.Slug,
		"tasks":  len(req.Tasks),
	})
}

// handleGetPlan handles GET /api/v1/plans/{slug}
func (s *Server) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "missing slug path parameter", http.StatusBadRequest)
		return
	}

	if s.storage == nil {
		http.Error(w, "no storage configured", http.StatusInternalServerError)
		return
	}

	plan, err := s.storage.GetPlan(r.Context(), slug)
	if err != nil {
		http.Error(w, fmt.Sprintf("plan not found: %v", err), http.StatusNotFound)
		return
	}

	tasks, err := s.storage.GetTasksByPlan(r.Context(), slug)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get plan tasks: %v", err), http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"plan":  plan,
		"tasks": tasks,
	})
}

// handleListTasks handles GET /api/v1/tasks?repo_path=&status=
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repo_path")
	statusStr := r.URL.Query().Get("status")

	var status domain.TaskStatus
	if statusStr != "" {
		status = domain.TaskStatus(strings.ToUpper(statusStr))
	}

	tasks, err := s.storage.ListTasks(r.Context(), repoPath, status)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to query tasks: %v", err), http.StatusInternalServerError)
		return
	}

	if tasks == nil {
		tasks = []domain.Task{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

// handleUpdateTaskStatus handles POST /api/v1/tasks/{id}/status
func (s *Server) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		From   domain.TaskStatus `json:"from"`
		Status domain.TaskStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	if s.storage == nil {
		http.Error(w, "no storage configured", http.StatusInternalServerError)
		return
	}

	fromStatus := req.From
	getCurrentStatus := func() domain.TaskStatus {
		tasks, err := s.storage.ListTasks(r.Context(), "", "")
		if err == nil {
			for _, t := range tasks {
				if t.ID == taskID {
					return t.Status
				}
			}
		}
		return domain.StatusReady
	}

	if fromStatus == "" {
		fromStatus = getCurrentStatus()
	}

	err := s.storage.UpdateTaskStatus(r.Context(), taskID, fromStatus, req.Status)
	if err != nil && req.From == "" {
		// Retry once with the latest status in case of concurrent scheduler dispatch
		fromStatus = getCurrentStatus()
		err = s.storage.UpdateTaskStatus(r.Context(), taskID, fromStatus, req.Status)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to transition task status: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		Type:      "task:status",
		TaskID:    taskID,
		Payload:   fmt.Sprintf(`{"task_id":"%s","status":"%s"}`, taskID, req.Status),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"task_id": taskID,
		"status":  string(req.Status),
	})
}

// handleKillRun handles POST /api/v1/runs/{id}/kill
func (s *Server) handleKillRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		http.Error(w, "missing run id path parameter", http.StatusBadRequest)
		return
	}

	if s.supervisor == nil {
		http.Error(w, "supervisor not available", http.StatusInternalServerError)
		return
	}

	gracePeriod := 2 * time.Second
	if err := s.supervisor.Kill(runID, gracePeriod); err != nil {
		http.Error(w, fmt.Sprintf("failed to kill run: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		RunID:     runID,
		Type:      "run:killed",
		Payload:   fmt.Sprintf(`{"run_id":"%s"}`, runID),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "KILLED",
		"run_id": runID,
	})
}

// handleEventStream handles GET /api/v1/events/stream
func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported by client", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientCh := make(chan domain.EventMessage, 32)

	s.sseMu.Lock()
	s.sseClients[clientCh] = struct{}{}
	s.sseMu.Unlock()

	defer func() {
		s.sseMu.Lock()
		delete(s.sseClients, clientCh)
		s.sseMu.Unlock()
	}()

	// Send initial connect acknowledgement
	fmt.Fprintf(w, "event: ping\ndata: connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-clientCh:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, string(data))
			flusher.Flush()
		}
	}
}

// handleTerminalWS handles GET /api/v1/terminal/ws and GET /api/v1/terminal/ws/{run_id}
func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run_id")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "session closed")

	ctx := r.Context()

	// Send initial terminal connection banner if attached to a specific run
	if runID != "" {
		welcomeMsg := fmt.Sprintf("\x1b[38;5;39m[zenfd@control-plane]\x1b[0m Connected to PTY session for: \x1b[1;37m%s\x1b[0m\r\n\x1b[38;5;242mStreaming Unix PTY output (type to interact)...\x1b[0m\r\n\r\n", runID)
		_ = conn.Write(ctx, websocket.MessageText, []byte(welcomeMsg))
	}

	// Bidirectional communication loop
	for {
		typ, msg, err := conn.Read(ctx)
		if err != nil {
			return
		}

		strMsg := string(msg)

		// 1. Healthcheck / Ping support
		if strMsg == "TERMINAL_PING" {
			_ = conn.Write(ctx, typ, []byte("TERMINAL_PONG_ACK"))
			continue
		}

		// 2. Terminal resize command
		if strings.Contains(strMsg, `"type":"resize"`) || strings.Contains(strMsg, `"type": "resize"`) {
			var cmd struct {
				Type string `json:"type"`
				Rows uint16 `json:"rows"`
				Cols uint16 `json:"cols"`
			}
			if err := json.Unmarshal(msg, &cmd); err == nil && s.supervisor != nil && runID != "" {
				_ = s.supervisor.ResizePTY(runID, domain.WindowSize{Rows: cmd.Rows, Cols: cmd.Cols})
			}
			continue
		}

		// 3. Echo / Ack data back for terminal input
		_ = conn.Write(ctx, typ, msg)
	}
}

// handleGetArtifact handles GET /api/v1/artifacts/{id}
func (s *Server) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing artifact id", http.StatusBadRequest)
		return
	}

	art, err := s.storage.GetArtifact(r.Context(), id)
	if err != nil {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}

	fileInfo, err := os.Stat(art.FilePath)
	if err != nil {
		http.Error(w, "artifact file not accessible on disk", http.StatusNotFound)
		return
	}
	if fileInfo.IsDir() {
		http.Error(w, "artifact cannot be a directory", http.StatusBadRequest)
		return
	}

	if art.MimeType != "" {
		w.Header().Set("Content-Type", art.MimeType)
	}
	http.ServeFile(w, r, art.FilePath)
}

// handleUploadTaskAttachment handles POST /api/v1/tasks/{id}/attachments (multipart/form-data)
func (s *Server) handleUploadTaskAttachment(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	// Limit upload size to 25MB
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		http.Error(w, fmt.Sprintf("file too large or invalid multipart form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("missing file in form field 'file': %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	displayName := r.FormValue("name")
	if displayName == "" {
		displayName = header.Filename
	}

	home, _ := os.UserHomeDir()
	storageDir := filepath.Join(home, ".zenforge", "attachments", taskID)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("failed to create attachment directory: %v", err), http.StatusInternalServerError)
		return
	}

	artID := fmt.Sprintf("art-in-%s-%d", taskID, time.Now().UnixNano())
	targetPath := filepath.Join(storageDir, fmt.Sprintf("%s_%s", artID, header.Filename))

	targetFile, err := os.Create(targetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to save attachment: %v", err), http.StatusInternalServerError)
		return
	}
	defer targetFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(targetFile, hasher)

	byteSize, err := io.Copy(multiWriter, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to write attachment bytes: %v", err), http.StatusInternalServerError)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = mime.TypeByExtension(filepath.Ext(header.Filename))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	artType := domain.ArtifactTypeFile
	if strings.HasPrefix(mimeType, "image/") {
		artType = domain.ArtifactTypeImage
	} else if strings.Contains(mimeType, "markdown") || strings.HasSuffix(header.Filename, ".md") {
		artType = domain.ArtifactTypeMarkdown
	} else if strings.Contains(mimeType, "pdf") || strings.HasSuffix(header.Filename, ".pdf") {
		artType = domain.ArtifactTypeDoc
	}

	art := domain.Artifact{
		ID:        artID,
		TaskID:    taskID,
		Role:      domain.ArtifactRoleInput,
		Name:      displayName,
		Type:      artType,
		FilePath:  targetPath,
		MimeType:  mimeType,
		ByteSize:  byteSize,
		Sha256:    hex.EncodeToString(hasher.Sum(nil)),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.storage.RecordArtifact(r.Context(), art); err != nil {
		http.Error(w, fmt.Sprintf("failed to record artifact: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast SSE
	s.Broadcast(domain.EventMessage{
		Type:      "task:artifact_added",
		TaskID:    taskID,
		Payload:   fmt.Sprintf(`{"artifact_id":"%s","role":"INPUT","name":"%s","type":"%s"}`, art.ID, art.Name, art.Type),
		Timestamp: art.CreatedAt,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(art)
}

// handleListTaskArtifacts handles GET /api/v1/tasks/{id}/artifacts?role=
func (s *Server) handleListTaskArtifacts(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	role := r.URL.Query().Get("role")
	artifacts, err := s.storage.GetArtifactsByTask(r.Context(), taskID, role)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to query artifacts: %v", err), http.StatusInternalServerError)
		return
	}
	if artifacts == nil {
		artifacts = []domain.Artifact{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(artifacts)
}

// handleGetTaskLogs handles GET /api/v1/tasks/{id}/logs
func (s *Server) handleGetTaskLogs(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	home, _ := os.UserHomeDir()
	logDirs := []string{
		filepath.Join(home, ".zenforge", "logs"),
		filepath.Join(home, ".zenforge"),
	}

	var foundLog string
	for _, dir := range logDirs {
		matches, _ := filepath.Glob(filepath.Join(dir, fmt.Sprintf("run-%s-*.log", taskID)))
		if len(matches) > 0 {
			foundLog = matches[len(matches)-1]
			break
		}
	}

	if foundLog == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "[zenfd] No execution logs recorded yet for task %s\n", taskID)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, r, foundLog)
}

// handleStaticSPA serves static files from the embedded web filesystem or falls back to index.html.
func (s *Server) handleStaticSPA(w http.ResponseWriter, r *http.Request) {
	if s.webFS == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "ZenForge Daemon (zenfd) - Control Plane active on port 1123\n")
		return
	}

	urlPath := strings.TrimPrefix(r.URL.Path, "/")
	if urlPath == "" {
		urlPath = "index.html"
	}

	file, err := s.webFS.Open(urlPath)
	if err != nil {
		// SPA fallback: serve index.html for all frontend client-side routes
		indexFile, indexErr := s.webFS.Open("index.html")
		if indexErr != nil {
			http.NotFound(w, r)
			return
		}
		defer indexFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, indexFile)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		indexFile, indexErr := s.webFS.Open("index.html")
		if indexErr == nil {
			defer indexFile.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.Copy(w, indexFile)
			return
		}
		http.NotFound(w, r)
		return
	}

	// Determine content-type based on file extension
	switch {
	case strings.HasSuffix(urlPath, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(urlPath, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(urlPath, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(urlPath, ".json"):
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case strings.HasSuffix(urlPath, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(urlPath, ".png"):
		w.Header().Set("Content-Type", "image/png")
	}

	_, _ = io.Copy(w, file)
}

// Serve accepts incoming connections on the listener ln.
func (s *Server) Serve(ln net.Listener) error {
	s.listener = ln
	s.httpServer = &http.Server{
		Handler: s.Handler(),
	}
	return s.httpServer.Serve(ln)
}

// Start begins listening on the specified address (e.g. "127.0.0.1:1123" or ":1123").
func (s *Server) Start(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind address %s: %w", addr, err)
	}
	return s.Serve(ln)
}

// Addr returns the bound listener network address (useful when bound to :0 in tests).
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// POST /api/v1/tasks/{id}/claim
func (s *Server) handleClaimTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		WorkerID string `json:"worker_id"`
		PID      int    `json:"pid"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	packet, err := s.storage.ClaimTask(r.Context(), taskID, req.WorkerID, req.PID)
	if err != nil {
		http.Error(w, fmt.Sprintf("claim failed: %v", err), http.StatusConflict)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task:status",
		Payload:   string(domain.StatusRunning),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(packet)
}

// POST /api/v1/tasks/{id}/review
func (s *Server) handleRequestReview(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason  string   `json:"reason"`
		Details string   `json:"details"`
		Options []string `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.storage.RequestReview(r.Context(), taskID, req.Reason, req.Details, req.Options); err != nil {
		http.Error(w, fmt.Sprintf("request review failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task:status",
		Payload:   string(domain.StatusAwaitingReview),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": domain.StatusAwaitingReview})
}

// POST /api/v1/tasks/{id}/resolve
func (s *Server) handleResolveReview(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.storage.ResolveReview(r.Context(), taskID, req.Response); err != nil {
		http.Error(w, fmt.Sprintf("resolve review failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task:status",
		Payload:   string(domain.StatusReady),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": domain.StatusReady})
}

// POST /api/v1/tasks/{id}/merge
func (s *Server) handleMergeTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	merged, conflicts, err := s.storage.MergeTask(r.Context(), taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("merge failed: %v", err), http.StatusInternalServerError)
		return
	}

	status := domain.StatusCompleted
	if !merged {
		status = domain.StatusConflict
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task:status",
		Payload:   string(status),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"merged":         merged,
		"conflict_files": conflicts,
		"status":         status,
	})
}

// GET /api/v1/docs
func (s *Server) handleListDocs(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repo_path")
	category := r.URL.Query().Get("category")

	docs, err := s.storage.ListDocs(r.Context(), repoPath, category)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list docs: %v", err), http.StatusInternalServerError)
		return
	}
	if docs == nil {
		docs = []domain.ProjectDoc{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(docs)
}

// POST /api/v1/docs
func (s *Server) handleUpsertDoc(w http.ResponseWriter, r *http.Request) {
	var doc domain.ProjectDoc
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.storage.UpsertDoc(r.Context(), doc); err != nil {
		http.Error(w, fmt.Sprintf("failed to save doc: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

// POST /api/v1/docs/export
func (s *Server) handleExportDocs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetDir string `json:"target_dir"`
		RepoPath  string `json:"repo_path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.TargetDir == "" {
		req.TargetDir = "docs"
	}

	targetDir := req.TargetDir
	if !filepath.IsAbs(targetDir) && req.RepoPath != "" {
		targetDir = filepath.Join(req.RepoPath, targetDir)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}

	docs, err := s.storage.ListDocs(r.Context(), req.RepoPath, "")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list docs: %v", err), http.StatusInternalServerError)
		return
	}

	count := 0
	for _, doc := range docs {
		filename := fmt.Sprintf("%s.md", doc.Slug)
		filePath := filepath.Join(targetDir, filename)
		content := fmt.Sprintf("# %s\n\n> Category: %s | Version: %d\n\n%s\n", doc.Title, doc.Category, doc.Version, doc.ContentMarkdown)
		if err := os.WriteFile(filePath, []byte(content), 0644); err == nil {
			count++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"exported_count": count,
		"target_dir":     targetDir,
	})
}

// GET /api/v1/docs/{slug}
func (s *Server) handleGetDoc(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	repoPath := r.URL.Query().Get("repo_path")

	doc, err := s.storage.GetDoc(r.Context(), repoPath, slug)
	if err != nil {
		http.Error(w, "doc not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

// GET /api/v1/env/global
func (s *Server) handleGetGlobalEnv(w http.ResponseWriter, r *http.Request) {
	gitVer := "unknown"
	if out, err := exec.Command("git", "version").Output(); err == nil {
		gitVer = strings.TrimSpace(string(out))
	}
	nodeVer := "unknown"
	if out, err := exec.Command("node", "-v").Output(); err == nil {
		nodeVer = strings.TrimSpace(string(out))
	}

	envData := map[string]any{
		"version":     "0.1.0",
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"num_cpu":     runtime.NumCPU(),
		"go_version":  runtime.Version(),
		"git_version": gitVer,
		"node":        nodeVer,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(envData)
}

// GET /api/v1/projects
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.storage.ListProjects(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list projects: %v", err), http.StatusInternalServerError)
		return
	}
	if projects == nil {
		projects = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"projects": projects})
}

// GET /api/v1/plans
func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repo_path")
	tasks, err := s.storage.ListTasks(r.Context(), repoPath, "")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list tasks: %v", err), http.StatusInternalServerError)
		return
	}

	planMap := make(map[string]bool)
	for _, t := range tasks {
		if t.PlanID != "" {
			planMap[t.PlanID] = true
		}
	}
	var plans []string
	for p := range planMap {
		plans = append(plans, p)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"plans": plans})
}

// DELETE /api/v1/plans/{slug}
func (s *Server) handleDeletePlan(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "missing plan slug", http.StatusBadRequest)
		return
	}

	if err := s.storage.DeletePlan(r.Context(), slug); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete plan: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		Type:      "plan:deleted",
		Payload:   fmt.Sprintf(`{"slug":"%s"}`, slug),
		Timestamp: time.Now().UTC(),
	})

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/tasks
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, fmt.Sprintf("invalid task payload: %v", err), http.StatusBadRequest)
		return
	}

	if task.ID == "" || task.PlanID == "" {
		http.Error(w, "task ID and PlanID are required", http.StatusBadRequest)
		return
	}

	if err := s.storage.CreateTask(r.Context(), task); err != nil {
		http.Error(w, fmt.Sprintf("failed to create task: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    task.ID,
		Type:      "task:created",
		Payload:   fmt.Sprintf(`{"id":"%s","plan_id":"%s"}`, task.ID, task.PlanID),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(task)
}

// POST /api/v1/tasks/{id}/dependencies
func (s *Server) handleUpdateTaskDependencies(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		Dependencies []string `json:"dependencies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.storage.UpdateTaskDependencies(r.Context(), taskID, req.Dependencies); err != nil {
		http.Error(w, fmt.Sprintf("failed to update dependencies: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task:dependencies_updated",
		Payload:   fmt.Sprintf(`{"id":"%s"}`, taskID),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID, "dependencies": req.Dependencies})
}

// PATCH /api/v1/tasks/{id}/context
func (s *Server) handleUpdateTaskContext(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.storage.UpdateTaskNotes(r.Context(), taskID, req.Notes); err != nil {
		http.Error(w, fmt.Sprintf("failed to update task notes: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task:notes_updated",
		Payload:   fmt.Sprintf(`{"id":"%s"}`, taskID),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID, "notes": req.Notes})
}

// POST /api/v1/tasks/{id}/reclaim
func (s *Server) handleReclaimTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		WorkerID       string `json:"worker_id"`
		OrchestratorID string `json:"orchestrator_id"`
		PID            int    `json:"pid"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	packet, err := s.storage.ReclaimTask(r.Context(), taskID, req.WorkerID, req.OrchestratorID, req.PID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to reclaim task: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task:status",
		Payload:   string(domain.StatusRunning),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(packet)
}

// POST /api/v1/tasks/{id}/evict
func (s *Server) handleEvictTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason string `json:"reason"`
		Action string `json:"action"` // "RESET_TO_READY" | "CANCEL"
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Action == "" {
		req.Action = "RESET_TO_READY"
	}

	if err := s.storage.EvictTask(r.Context(), taskID, req.Reason, req.Action); err != nil {
		http.Error(w, fmt.Sprintf("failed to evict task: %v", err), http.StatusInternalServerError)
		return
	}

	targetStatus := domain.StatusReady
	if req.Action == "CANCEL" {
		targetStatus = domain.StatusCancelled
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task:status",
		Payload:   string(targetStatus),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID, "status": targetStatus})
}

// GET /api/v1/sessions
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.storage.ListSessions(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list sessions: %v", err), http.StatusInternalServerError)
		return
	}
	if sessions == nil {
		sessions = []domain.AgentSession{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"sessions": sessions})
}

// POST /api/v1/sessions
func (s *Server) handleRegisterSession(w http.ResponseWriter, r *http.Request) {
	var sess domain.AgentSession
	if err := json.NewDecoder(r.Body).Decode(&sess); err != nil {
		http.Error(w, fmt.Sprintf("invalid session payload: %v", err), http.StatusBadRequest)
		return
	}

	if sess.SessionID == "" {
		http.Error(w, "missing session_id", http.StatusBadRequest)
		return
	}

	if err := s.storage.RegisterSession(r.Context(), sess); err != nil {
		http.Error(w, fmt.Sprintf("failed to register session: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		Type:      "session:registered",
		Payload:   fmt.Sprintf(`{"session_id":"%s","role":"%s"}`, sess.SessionID, sess.Role),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sess)
}

// GET /api/v1/specs
func (s *Server) handleListSpecs(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repo_path")
	tierStr := r.URL.Query().Get("tier")
	tier := 0
	if tierStr != "" {
		tier, _ = strconv.Atoi(tierStr)
	}

	specs, err := s.storage.ListSpecs(r.Context(), repoPath, tier)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list specs: %v", err), http.StatusInternalServerError)
		return
	}
	if specs == nil {
		specs = []domain.TechSpec{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"specs": specs})
}

// POST /api/v1/specs
func (s *Server) handleUpsertSpec(w http.ResponseWriter, r *http.Request) {
	var spec domain.TechSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		http.Error(w, fmt.Sprintf("invalid spec payload: %v", err), http.StatusBadRequest)
		return
	}

	saved, err := s.storage.UpsertSpec(r.Context(), spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to save spec: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		Type:      "spec:upserted",
		Payload:   fmt.Sprintf(`{"slug":"%s","tier":%d,"version":%d}`, saved.Slug, saved.Tier, saved.Version),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(saved)
}

// GET /api/v1/specs/{slug}
func (s *Server) handleGetSpec(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	spec, err := s.storage.GetSpec(r.Context(), slug)
	if err != nil {
		http.Error(w, "spec not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(spec)
}

// GET /api/v1/specs/search
func (s *Server) handleSearchSpecs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := s.storage.SearchSpecs(r.Context(), q)
	if err != nil {
		http.Error(w, fmt.Sprintf("search failed: %v", err), http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []domain.TechSpec{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
}

// POST /api/v1/plans/{slug}/snapshot
func (s *Server) handleSavePlanSnapshot(w http.ResponseWriter, r *http.Request) {
	planSlug := r.PathValue("slug")
	var req struct {
		Revision   string                `json:"revision"`
		Author     string                `json:"author"`
		Summary    string                `json:"summary"`
		FolderPath string                `json:"folder_path"`
		Files      []domain.SnapshotFile `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid snapshot payload: %v", err), http.StatusBadRequest)
		return
	}

	if req.Revision == "" {
		req.Revision = "v1.0"
	}

	// If folder_path is specified, snapshot all readable text files recursively
	if req.FolderPath != "" && len(req.Files) == 0 {
		err := filepath.Walk(req.FolderPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(req.FolderPath, path)
			if relErr != nil {
				rel = filepath.Base(path)
			}
			// Skip hidden files or vendor directories
			if strings.HasPrefix(filepath.Base(path), ".") || strings.Contains(path, "node_modules") {
				return nil
			}
			contentBytes, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			h := sha256.Sum256(contentBytes)
			req.Files = append(req.Files, domain.SnapshotFile{
				PlanSlug:  planSlug,
				Revision:  req.Revision,
				FilePath:  rel,
				Content:   string(contentBytes),
				ByteSize:  info.Size(),
				SHA256:    hex.EncodeToString(h[:]),
				CreatedAt: time.Now().UTC(),
			})
			return nil
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read folder: %v", err), http.StatusInternalServerError)
			return
		}
	}

	if err := s.storage.SavePlanSnapshot(r.Context(), planSlug, req.Revision, req.Author, req.Summary, req.Files); err != nil {
		http.Error(w, fmt.Sprintf("failed to save snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	s.Broadcast(domain.EventMessage{
		Type:      "plan:snapshot_saved",
		Payload:   fmt.Sprintf(`{"plan_slug":"%s","revision":"%s","files_count":%d}`, planSlug, req.Revision, len(req.Files)),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"plan_slug":   planSlug,
		"revision":    req.Revision,
		"files_count": len(req.Files),
		"status":      "SNAPSHOT_SAVED",
	})
}

// GET /api/v1/plans/{slug}/snapshot
func (s *Server) handleGetPlanSnapshot(w http.ResponseWriter, r *http.Request) {
	planSlug := r.PathValue("slug")
	revision := r.URL.Query().Get("rev")
	if revision == "" {
		revision = "v1.0"
	}
	files, err := s.storage.GetPlanSnapshot(r.Context(), planSlug, revision)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get snapshot: %v", err), http.StatusInternalServerError)
		return
	}
	if files == nil {
		files = []domain.SnapshotFile{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"files": files, "revision": revision, "plan_slug": planSlug})
}

// GET /api/v1/plans/{slug}/revisions
func (s *Server) handleListPlanRevisions(w http.ResponseWriter, r *http.Request) {
	planSlug := r.PathValue("slug")
	revs, err := s.storage.ListPlanRevisions(r.Context(), planSlug)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list revisions: %v", err), http.StatusInternalServerError)
		return
	}
	if revs == nil {
		revs = []domain.PlanRevision{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"revisions": revs})
}

// GET /api/v1/plans/{slug}/diff
func (s *Server) handleGetPlanDiff(w http.ResponseWriter, r *http.Request) {
	planSlug := r.PathValue("slug")
	fromRev := r.URL.Query().Get("from")
	toRev := r.URL.Query().Get("to")
	if fromRev == "" || toRev == "" {
		http.Error(w, "missing 'from' or 'to' revision query params", http.StatusBadRequest)
		return
	}

	diffs, err := s.storage.GetPlanDiff(r.Context(), planSlug, fromRev, toRev)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to compute diff: %v", err), http.StatusInternalServerError)
		return
	}
	if diffs == nil {
		diffs = []domain.FileDiff{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"diffs": diffs, "from": fromRev, "to": toRev})
}
