package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zenfd/internal/attachment"
	"zenfd/internal/domain"
)
// handleListTasks handles GET /api/v1/tasks?status=
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := domain.TaskFilter{
		ProjectSlug: query.Get("project_slug"),
		PlanSlug:    query.Get("plan_slug"),
		AgentSlug:   query.Get("agent_slug"),
		OrphanOnly:  query.Get("orphan") == "true",
	}
	if statusStr := query.Get("status"); statusStr != "" {
		filter.Status = domain.TaskStatus(strings.ToUpper(statusStr))
	}

	tasks, err := s.storage.ListTasks(r.Context(), filter)
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to query tasks: %v", err), http.StatusInternalServerError)
		return
	}

	if tasks == nil {
		tasks = []domain.Task{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"tasks": tasks})
}

// handleGetTask handles GET /api/v1/tasks/{id}?include_packet=true
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeHttpError(w, "missing task id", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("include_packet") == "true" {
		packet, err := s.storage.GetTaskContextPacket(r.Context(), taskID)
		if err != nil {
			writeStorageError(w, "load task context packet", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(packet)
		return
	}
	task, err := s.storage.GetTask(r.Context(), taskID)
	if err != nil {
		writeStorageError(w, "load task", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(task)
}

// handleUpdateTaskStatus handles POST /api/v1/tasks/{id}/status
func (s *Server) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeHttpError(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		From   domain.TaskStatus `json:"from"`
		Status domain.TaskStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHttpError(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	if s.storage == nil {
		writeHttpError(w, "no storage configured", http.StatusInternalServerError)
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


// handleGetArtifact handles GET /api/v1/artifacts/{id}
func (s *Server) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeHttpError(w, "missing artifact id", http.StatusBadRequest)
		return
	}

	art, err := s.storage.GetArtifact(r.Context(), id)
	if err != nil {
		writeHttpError(w, "artifact not found", http.StatusNotFound)
		return
	}

	fileInfo, err := os.Stat(art.FilePath)
	if err != nil {
		writeHttpError(w, "artifact file not accessible on disk", http.StatusNotFound)
		return
	}
	if fileInfo.IsDir() {
		writeHttpError(w, "artifact cannot be a directory", http.StatusBadRequest)
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
		writeHttpError(w, "missing task id", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(attachment.MaxBytes); err != nil {
		writeHttpError(w, fmt.Sprintf("file too large or invalid multipart form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeHttpError(w, fmt.Sprintf("missing file in form field 'file': %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	displayName := r.FormValue("name")
	if displayName == "" {
		displayName = header.Filename
	}

	if err := attachment.ValidateTaskID(taskID); err != nil {
		writeHttpError(w, err.Error(), http.StatusBadRequest)
		return
	}

	home, _ := os.UserHomeDir()
	storageDir := filepath.Join(home, ".zenforge", "attachments", taskID)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		writeHttpError(w, fmt.Sprintf("failed to create attachment directory: %v", err), http.StatusInternalServerError)
		return
	}

	artID := fmt.Sprintf("art-in-%s-%d", taskID, time.Now().UnixNano())
	targetPath := filepath.Join(storageDir, fmt.Sprintf("%s_%s", artID, header.Filename))

	targetFile, err := os.Create(targetPath)
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to save attachment: %v", err), http.StatusInternalServerError)
		return
	}
	defer targetFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(targetFile, hasher)

	byteSize, err := io.Copy(multiWriter, file)
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to write attachment bytes: %v", err), http.StatusInternalServerError)
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
		writeHttpError(w, "missing task id", http.StatusBadRequest)
		return
	}

	role := r.URL.Query().Get("role")
	artifacts, err := s.storage.GetArtifactsByTask(r.Context(), taskID, role)
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to query artifacts: %v", err), http.StatusInternalServerError)
		return
	}
	if artifacts == nil {
		artifacts = []domain.Artifact{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"artifacts": artifacts})
}


func (s *Server) handleClaimTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeHttpError(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		WorkerID       string `json:"worker_id"`
		OrchestratorID string `json:"orchestrator_id"`
		WorkerRole     string `json:"worker_role"`
		SessionID      string `json:"session_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	packet, err := s.storage.ClaimTask(r.Context(), taskID, domain.ClaimOptions{
		WorkerID:       req.WorkerID,
		OrchestratorID: req.OrchestratorID,
		WorkerRole:     req.WorkerRole,
		SessionID:      req.SessionID,
	})
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

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeHttpError(w, fmt.Sprintf("invalid task payload: %v", err), http.StatusBadRequest)
		return
	}

	if task.ID == "" || task.Title == "" {
		writeHttpError(w, "task ID and title are required", http.StatusBadRequest)
		return
	}

	if err := s.storage.CreateTasks(r.Context(), []domain.Task{task}); err != nil {
		writeStorageError(w, "create task", err)
		return
	}
	created, err := s.storage.GetTask(r.Context(), task.ID)
	if err != nil {
		writeStorageError(w, "load task", err)
		return
	}

	s.Broadcast(domain.EventMessage{
		ProjectSlug: created.ProjectSlug,
		PlanSlug:    created.PlanSlug,
		TaskID:      created.ID,
		EntityType:  "task",
		EntityID:    created.ID,
		Type:        "task.created",
		Payload:     eventPayload(map[string]string{"id": created.ID, "plan_slug": created.PlanSlug}),
		Timestamp:   time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// POST /api/v1/tasks/{id}/dependencies
func (s *Server) handleUpdateTaskDependencies(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeHttpError(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		Dependencies []string `json:"dependencies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHttpError(w, "invalid request body", http.StatusBadRequest)
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
		writeHttpError(w, "missing task id", http.StatusBadRequest)
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHttpError(w, "invalid request body", http.StatusBadRequest)
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
		writeHttpError(w, "missing task id", http.StatusBadRequest)
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

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

// GET /api/v1/projects?include_archived=true
