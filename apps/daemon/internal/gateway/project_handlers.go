package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"zenf/internal/domain"
)
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.storage.ListProjects(r.Context(), r.URL.Query().Get("include_archived") == "true")
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to list projects: %v", err), http.StatusInternalServerError)
		return
	}
	if projects == nil {
		projects = []domain.Project{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"projects": projects})
}

// POST /api/v1/projects
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var project domain.Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		writeHttpError(w, fmt.Sprintf("invalid project payload: %v", err), http.StatusBadRequest)
		return
	}
	if err := s.storage.CreateProject(r.Context(), project); err != nil {
		writeStorageError(w, "create project", err)
		return
	}
	created, err := s.storage.GetProject(r.Context(), project.Slug)
	if err != nil {
		writeStorageError(w, "load project", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// GET /api/v1/projects/{slug}
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.storage.GetProject(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeStorageError(w, "load project", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(project)
}

// PATCH /api/v1/projects/{slug}
func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        *string `json:"name"`
		RootPath    *string `json:"root_path"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHttpError(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}
	patch := domain.ProjectPatch{Name: req.Name, RootPath: req.RootPath, Description: req.Description}
	if req.Status != nil {
		status := domain.ProjectStatus(strings.ToUpper(*req.Status))
		patch.Status = &status
	}
	slug := r.PathValue("slug")
	if err := s.storage.UpdateProject(r.Context(), slug, patch); err != nil {
		writeStorageError(w, "update project", err)
		return
	}
	project, err := s.storage.GetProject(r.Context(), slug)
	if err != nil {
		writeStorageError(w, "load project", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(project)
}

// DELETE /api/v1/projects/{slug}
func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	if err := s.storage.DeleteProject(r.Context(), r.PathValue("slug")); err != nil {
		writeStorageError(w, "delete project", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Documents
// ---------------------------------------------------------------------------

// GET /api/v1/docs?project_slug=&include_global=true&type=&scope=&status=&tag=&limit=
