package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"zenfd/internal/domain"
)

type CreatePlanRequest struct {
	Slug        string        `json:"slug"`
	ProjectSlug string        `json:"project_slug"`
	Title       string        `json:"title"`
	Tasks       []domain.Task `json:"tasks"`
}

// handleCreatePlan handles POST /api/v1/plans
func (s *Server) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHttpError(w, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}

	if req.Slug == "" {
		writeHttpError(w, "missing required field: slug", http.StatusBadRequest)
		return
	}
	if req.ProjectSlug == "" {
		writeHttpError(w, "missing required field: project_slug", http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		req.Title = req.Slug
	}

	plan := domain.Plan{
		Slug:        req.Slug,
		ProjectSlug: req.ProjectSlug,
		Title:       req.Title,
		Status:      domain.PlanStatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	for i := range req.Tasks {
		req.Tasks[i].ProjectSlug = req.ProjectSlug
		req.Tasks[i].PlanSlug = req.Slug
		if req.Tasks[i].Status == "" {
			if len(req.Tasks[i].Dependencies) == 0 {
				req.Tasks[i].Status = domain.StatusReady
			} else {
				req.Tasks[i].Status = domain.StatusPending
			}
		}
	}

	if s.storage == nil {
		writeHttpError(w, "no storage configured", http.StatusInternalServerError)
		return
	}

	if err := s.storage.CreatePlanWithTasks(r.Context(), plan, req.Tasks); err != nil {
		writeStorageError(w, "submit plan", err)
		return
	}

	s.Broadcast(domain.EventMessage{
		ProjectSlug: plan.ProjectSlug,
		PlanSlug:    plan.Slug,
		EntityType:  "plan",
		EntityID:    plan.Slug,
		Type:        "plan.created",
		Payload:     eventPayload(map[string]any{"slug": plan.Slug, "project_slug": plan.ProjectSlug, "tasks_count": len(req.Tasks)}),
		Timestamp:   time.Now().UTC(),
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
		writeHttpError(w, "missing slug path parameter", http.StatusBadRequest)
		return
	}

	if s.storage == nil {
		writeHttpError(w, "no storage configured", http.StatusInternalServerError)
		return
	}

	plan, err := s.storage.GetPlan(r.Context(), slug)
	if err != nil {
		writeStorageError(w, "load plan", err)
		return
	}

	tasks, err := s.storage.GetTasksByPlan(r.Context(), slug)
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to get plan tasks: %v", err), http.StatusInternalServerError)
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


func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	projectSlug := r.URL.Query().Get("project_slug")
	plans, err := s.storage.ListPlans(r.Context(), projectSlug, includeArchived)
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to list plans: %v", err), http.StatusInternalServerError)
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
		writeHttpError(w, "missing plan slug", http.StatusBadRequest)
		return
	}

	var req struct {
		Archived *bool `json:"archived"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHttpError(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}
	if req.Archived == nil {
		writeHttpError(w, "missing required field: archived", http.StatusBadRequest)
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
		writeHttpError(w, "missing plan slug", http.StatusBadRequest)
		return
	}

	if err := s.storage.DeletePlan(r.Context(), slug); err != nil {
		writeHttpError(w, fmt.Sprintf("failed to delete plan: %v", err), http.StatusInternalServerError)
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
