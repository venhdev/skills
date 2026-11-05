package gateway

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"zenfd/internal/attachment"
	"zenfd/internal/domain"
)

//go:embed all:web_dist
var defaultWebDist embed.FS

// CreatePlanRequest specifies the payload for plan ingestion via REST.
type CreatePlanRequest struct {
	Slug  string        `json:"slug"`
	Title string        `json:"title"`
	Tasks []domain.Task `json:"tasks"`
}

// Storage is the subset of persistence the HTTP gateway needs. It composes
// per-aggregate interfaces so each repository stays independently replaceable.
type Storage interface {
	domain.PlanRepository
	domain.TaskRepository
	domain.TaskLifecycle
	domain.ArtifactRepository
	domain.EventRepository
}

// Server provides the HTTP/SSE gateway for the zenfd control plane.
type Server struct {
	storage Storage

	sseMu      sync.RWMutex
	sseClients map[chan domain.EventMessage]struct{}

	watchMu     sync.Mutex
	watchCancel context.CancelFunc

	done     chan struct{}
	doneOnce sync.Once

	envOnce sync.Once
	envInfo map[string]any

	mux        *http.ServeMux
	httpServer *http.Server
	webFS      fs.FS
}

// NewServer constructs a new Gateway Server.
func NewServer(storage Storage) *Server {
	sub, err := fs.Sub(defaultWebDist, "web_dist")
	var webFS fs.FS
	if err == nil {
		webFS = sub
	}

	s := &Server{
		storage:    storage,
		sseClients: make(map[chan domain.EventMessage]struct{}),
		done:       make(chan struct{}),
		mux:        http.NewServeMux(),
		webFS:      webFS,
	}

	s.setupRoutes()
	return s
}

// setupRoutes registers all HTTP and SSE endpoints using Go 1.22+ routing syntax.
func (s *Server) setupRoutes() {
	// Projects API

	// Plan APIs
	s.mux.HandleFunc("GET /api/v1/plans", s.handleListPlans)
	s.mux.HandleFunc("POST /api/v1/plans", s.handleCreatePlan)
	s.mux.HandleFunc("PATCH /api/v1/plans/{slug}", s.handleArchivePlan)
	s.mux.HandleFunc("DELETE /api/v1/plans/{slug}", s.handleDeletePlan)
	s.mux.HandleFunc("GET /api/v1/plans/{slug}", s.handleGetPlan)

	// Task APIs
	s.mux.HandleFunc("GET /api/v1/tasks", s.handleListTasks)
	s.mux.HandleFunc("POST /api/v1/tasks", s.handleCreateTask)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/status", s.handleUpdateTaskStatus)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/dependencies", s.handleUpdateTaskDependencies)
	s.mux.HandleFunc("PATCH /api/v1/tasks/{id}/context", s.handleUpdateTaskContext)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/claim", s.handleClaimTask)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/evict", s.handleEvictTask)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/attachments", s.handleUploadTaskAttachment)
	s.mux.HandleFunc("GET /api/v1/tasks/{id}/artifacts", s.handleListTaskArtifacts)

	// Global Environment & Toolchains API
	s.mux.HandleFunc("GET /api/v1/env/global", s.handleGetGlobalEnv)

	// Activity log + real-time stream
	s.mux.HandleFunc("GET /api/v1/events", s.handleListEvents)
	s.mux.HandleFunc("GET /api/v1/events/stream", s.handleEventStream)

	// Artifact Retrieval API
	s.mux.HandleFunc("GET /api/v1/artifacts/{id}", s.handleGetArtifact)

	// Web UI SPA & Static Files
	s.mux.HandleFunc("GET /", s.handleStaticSPA)
}

// Handler returns the HTTP handler with origin guards applied.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowRequest(r) {
			http.Error(w, "cross-origin request rejected", http.StatusForbidden)
			return
		}

		// The SPA is served same-origin (directly or through the Vite proxy), so
		// CORS is only echoed back for loopback origins.
		if origin := r.Header.Get("Origin"); origin != "" && isLoopbackOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		s.mux.ServeHTTP(w, r)
	})
}

// isLoopbackOrigin accepts http(s) origins whose host is a loopback address on
// any port, which covers the Vite dev server and preview builds.
func isLoopbackOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// allowRequest blocks cross-site browser writes. Requests without an Origin
// header come from the CLI, curl, or an MCP client and are allowed. A browser
// always attaches Origin to a cross-origin mutating request, so rejecting a
// non-loopback Origin closes the CSRF hole without requiring tokens.
func allowRequest(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	return isLoopbackOrigin(origin)
}

// writeStorageError maps domain sentinels onto HTTP status codes so clients can
// distinguish a conflict from a server failure.
func writeStorageError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		http.Error(w, fmt.Sprintf("%s: %v", op, err), http.StatusNotFound)
	case errors.Is(err, domain.ErrAlreadyExists),
		errors.Is(err, domain.ErrConflictStatus),
		errors.Is(err, domain.ErrInvalidState):
		http.Error(w, fmt.Sprintf("%s: %v", op, err), http.StatusConflict)
	case errors.Is(err, domain.ErrInvalidGraph):
		http.Error(w, fmt.Sprintf("%s: %v", op, err), http.StatusUnprocessableEntity)
	default:
		http.Error(w, fmt.Sprintf("%s: %v", op, err), http.StatusInternalServerError)
	}
}

// Broadcast distributes an event message to all connected SSE clients. Audit
// persistence is handled by database triggers, so the client sees the same
// activity log regardless of which process performed the write.
func (s *Server) Broadcast(evt domain.EventMessage) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
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

// eventPayload encodes a broadcast payload as JSON. Building payloads with
// fmt.Sprintf produced invalid JSON for identifiers containing quotes.
func eventPayload(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
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
		Status:    "ACTIVE",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	for i := range req.Tasks {
		if req.Tasks[i].PlanID == "" {
			req.Tasks[i].PlanID = req.Slug
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
		Type:      "plan.created",
		Payload:   eventPayload(map[string]any{"slug": plan.Slug, "tasks_count": len(req.Tasks)}),
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
		writeStorageError(w, "load plan", err)
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

// handleListTasks handles GET /api/v1/tasks?status=
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	statusStr := r.URL.Query().Get("status")

	var status domain.TaskStatus
	if statusStr != "" {
		status = domain.TaskStatus(strings.ToUpper(statusStr))
	}

	tasks, err := s.storage.ListTasks(r.Context(), status)
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
	if fromStatus == "" {
		current, err := s.storage.GetTask(r.Context(), taskID)
		if err != nil {
			writeStorageError(w, "load task status", err)
			return
		}
		fromStatus = current.Status
	}

	err := s.storage.UpdateTaskStatus(r.Context(), taskID, fromStatus, req.Status)
	if err != nil && errors.Is(err, domain.ErrConflictStatus) && req.From == "" {
		// Retry once with the latest status in case of a concurrent transition.
		if current, getErr := s.storage.GetTask(r.Context(), taskID); getErr == nil {
			fromStatus = current.Status
			err = s.storage.UpdateTaskStatus(r.Context(), taskID, fromStatus, req.Status)
		}
	}

	if err != nil {
		writeStorageError(w, "transition task status", err)
		return
	}

	s.Broadcast(domain.EventMessage{
		Type:      "task.status",
		TaskID:    taskID,
		Payload:   eventPayload(map[string]string{"task_id": taskID, "status": string(req.Status)}),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"task_id": taskID,
		"status":  string(req.Status),
	})
}

// handleListEvents handles GET /api/v1/events?after_id=&limit=
func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	afterID := int64(0)
	if raw := r.URL.Query().Get("after_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			http.Error(w, "invalid after_id", http.StatusBadRequest)
			return
		}
		afterID = parsed
	}

	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	events, err := s.storage.ListEvents(r.Context(), afterID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(events)
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

	// Idle SSE connections are closed by proxies and browsers; a periodic
	// comment keeps the stream alive without inventing domain events.
	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.done:
			return
		case <-keepAlive.C:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
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

	if err := r.ParseMultipartForm(attachment.MaxBytes); err != nil {
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

	if err := attachment.ValidateTaskID(taskID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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

	mimeType := attachment.ResolveMime(targetPath, header.Filename, header.Header.Get("Content-Type"))
	artType := attachment.Classify(mimeType, header.Filename)

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
		// Do not leave an orphaned file when the task is not READY/PENDING or
		// the row cannot be written.
		_ = os.Remove(targetPath)
		writeStorageError(w, "record artifact", err)
		return
	}

	// Broadcast SSE
	s.Broadcast(domain.EventMessage{
		Type:      "artifact.added",
		TaskID:    taskID,
		Payload:   eventPayload(map[string]string{"artifact_id": art.ID, "role": "INPUT", "name": art.Name, "type": string(art.Type)}),
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

// handleStaticSPA serves static files from the embedded web filesystem or falls back to index.html.
func (s *Server) handleStaticSPA(w http.ResponseWriter, r *http.Request) {
	// Unknown API paths must not be answered with the SPA shell.
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	if s.webFS == nil {
		serveControlPlaneBanner(w)
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
			// The bundle was never built into this binary (only the committed
			// placeholder is embedded).
			serveControlPlaneBanner(w)
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
		serveControlPlaneBanner(w)
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

// serveControlPlaneBanner is the response when no web bundle is embedded, so a
// clean checkout still builds and runs the daemon headlessly.
func serveControlPlaneBanner(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, "ZenForge Daemon (zenfd) - Control Plane active\n")
	fmt.Fprint(w, "Web cockpit bundle is not embedded in this build.\n")
	fmt.Fprint(w, "Run: make -C apps/daemon build-web && make -C apps/daemon build\n")
}

// Serve accepts incoming connections on the listener ln.
func (s *Server) Serve(ln net.Listener) error {
	s.httpServer = &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	watchCtx, cancel := context.WithCancel(context.Background())
	s.watchMu.Lock()
	s.watchCancel = cancel
	s.watchMu.Unlock()
	go s.watchStateChanges(watchCtx, stateWatchInterval)

	err := s.httpServer.Serve(ln)
	cancel()
	return err
}

// stateWatchInterval controls how often the gateway fingerprints the SQLite
// state to detect writes from processes outside the daemon event hub. The
// fingerprint is an O(1) revision read, so a sub-second interval is cheap and
// keeps MCP/CLI writes visible almost immediately.
const stateWatchInterval = 1 * time.Second

// watchStateChanges polls the storage fingerprint and broadcasts state.changed
// when an external writer (MCP stdio server, CLI, raw SQLite) commits. This
// keeps SSE as the single update channel for the web cockpit without requiring
// every writer to publish events.
func (s *Server) watchStateChanges(ctx context.Context, interval time.Duration) {
	fp, ok := s.storage.(domain.StateFingerprinter)
	if !ok {
		return
	}

	last, err := fp.StateFingerprint(ctx)
	if err != nil {
		log.Printf("zenfd: state watcher disabled, fingerprint failed: %v", err)
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastErr string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current, err := fp.StateFingerprint(ctx)
			if err != nil {
				// Log transitions only: a persistent failure would otherwise
				// spam the log every interval.
				if msg := err.Error(); msg != lastErr {
					log.Printf("zenfd: state fingerprint failed: %v", err)
					lastErr = msg
				}
				continue
			}
			lastErr = ""
			if current == last {
				continue
			}
			last = current
			s.Broadcast(domain.EventMessage{
				Type:      "state.changed",
				Payload:   "{}",
				Timestamp: time.Now().UTC(),
			})
		}
	}
}

// Start begins listening on the specified address (e.g. "127.0.0.1:1123" or ":1123").
func (s *Server) Start(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind address %s: %w", addr, err)
	}
	return s.Serve(ln)
}

// Shutdown gracefully stops the HTTP server. SSE handlers are released through
// the done channel so graceful shutdown does not wait for clients to disconnect.
func (s *Server) Shutdown(ctx context.Context) error {
	s.doneOnce.Do(func() { close(s.done) })

	s.watchMu.Lock()
	if s.watchCancel != nil {
		s.watchCancel()
		s.watchCancel = nil
	}
	s.watchMu.Unlock()

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
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	packet, err := s.storage.ClaimTask(r.Context(), taskID, req.WorkerID)
	if err != nil {
		writeStorageError(w, "claim task", err)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task.status",
		Payload:   eventPayload(map[string]string{"task_id": taskID, "status": string(domain.StatusRunning)}),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(packet)
}

// GET /api/v1/env/global
func (s *Server) handleGetGlobalEnv(w http.ResponseWriter, r *http.Request) {
	s.envOnce.Do(func() {
		gitVer := "unknown"
		if out, err := exec.Command("git", "version").Output(); err == nil {
			gitVer = strings.TrimSpace(string(out))
		}
		nodeVer := "unknown"
		if out, err := exec.Command("node", "-v").Output(); err == nil {
			nodeVer = strings.TrimSpace(string(out))
		}

		s.envInfo = map[string]any{
			"version":     "0.1.2",
			"os":          runtime.GOOS,
			"arch":        runtime.GOARCH,
			"num_cpu":     runtime.NumCPU(),
			"go_version":  runtime.Version(),
			"git_version": gitVer,
			"node":        nodeVer,
		}
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.envInfo)
}

// GET /api/v1/plans?include_archived=true
func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	plans, err := s.storage.ListPlans(r.Context(), includeArchived)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list plans: %v", err), http.StatusInternalServerError)
		return
	}
	if plans == nil {
		plans = []domain.Plan{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"plans": plans})
}

// PATCH /api/v1/plans/{slug} with {"archived": true|false}
func (s *Server) handleArchivePlan(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "missing plan slug", http.StatusBadRequest)
		return
	}

	var req struct {
		Archived *bool `json:"archived"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}
	if req.Archived == nil {
		http.Error(w, "missing required field: archived", http.StatusBadRequest)
		return
	}

	if err := s.storage.ArchivePlan(r.Context(), slug, *req.Archived); err != nil {
		writeStorageError(w, "archive plan", err)
		return
	}

	event := "plan.deleted"
	if *req.Archived {
		event = "plan.archived"
	} else {
		event = "plan.unarchived"
	}
	s.Broadcast(domain.EventMessage{
		Type:      event,
		Payload:   eventPayload(map[string]any{"slug": slug, "archived": *req.Archived}),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"slug": slug, "archived": *req.Archived})
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
		Type:      "plan.deleted",
		Payload:   eventPayload(map[string]string{"slug": slug}),
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
		writeStorageError(w, "create task", err)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    task.ID,
		Type:      "task.created",
		Payload:   eventPayload(map[string]string{"id": task.ID, "plan_id": task.PlanID}),
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
		writeStorageError(w, "update dependencies", err)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task.dependencies_updated",
		Payload:   eventPayload(map[string]string{"id": taskID}),
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
		writeStorageError(w, "update notes", err)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task.notes_updated",
		Payload:   eventPayload(map[string]string{"id": taskID}),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID, "notes": req.Notes})
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

	status, err := s.storage.EvictTask(r.Context(), taskID, req.Reason, req.Action)
	if err != nil {
		writeStorageError(w, "evict task", err)
		return
	}

	s.Broadcast(domain.EventMessage{
		TaskID:    taskID,
		Type:      "task.status",
		Payload:   eventPayload(map[string]string{"task_id": taskID, "status": string(status)}),
		Timestamp: time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID, "status": status})
}
