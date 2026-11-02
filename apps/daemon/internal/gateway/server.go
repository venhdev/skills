package gateway

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
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
	// Plan APIs
	s.mux.HandleFunc("POST /api/v1/plans", s.handleCreatePlan)
	s.mux.HandleFunc("GET /api/v1/plans/{slug}", s.handleGetPlan)

	// Task APIs
	s.mux.HandleFunc("GET /api/v1/tasks", s.handleListTasks)

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

	var err error
	if s.scheduler != nil {
		err = s.scheduler.SubmitPlan(r.Context(), plan, req.Tasks)
	} else if s.storage != nil {
		err = s.storage.SavePlanWithTasks(r.Context(), plan, req.Tasks)
	} else {
		err = errors.New("no scheduler or storage configured")
	}

	if err != nil {
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

	var plan *domain.Plan
	var tasks []domain.Task
	var err error

	if s.scheduler != nil {
		plan, tasks, err = s.scheduler.GetPlan(r.Context(), slug)
	} else if s.storage != nil {
		plan, err = s.storage.GetPlan(r.Context(), slug)
		if err == nil {
			tasks, err = s.storage.GetTasksByPlan(r.Context(), slug)
		}
	} else {
		http.Error(w, "no scheduler or storage configured", http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("plan not found: %v", err), http.StatusNotFound)
		return
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
