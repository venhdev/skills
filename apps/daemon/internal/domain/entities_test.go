package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"zenfd/internal/domain"
)

func TestTaskStatus_ValidationAndTerminal(t *testing.T) {
	validStatuses := []domain.TaskStatus{
		domain.StatusPending,
		domain.StatusReady,
		domain.StatusRunning,
		domain.StatusAwaitingReview,
		domain.StatusAwaitingMerge,
		domain.StatusConflict,
		domain.StatusBlocked,
		domain.StatusCompleted,
		domain.StatusFailed,
		domain.StatusCancelled,
	}

	for _, s := range validStatuses {
		if !s.IsValid() {
			t.Errorf("Expected status %s to be valid", s)
		}
	}

	invalidStatus := domain.TaskStatus("UNKNOWN")
	if invalidStatus.IsValid() {
		t.Errorf("Expected status %s to be invalid", invalidStatus)
	}

}

func TestTaskStatus_StateTransitions(t *testing.T) {
	tests := []struct {
		from    domain.TaskStatus
		to      domain.TaskStatus
		allowed bool
	}{
		// PENDING transitions
		{domain.StatusPending, domain.StatusReady, true},
		{domain.StatusPending, domain.StatusBlocked, true},
		{domain.StatusPending, domain.StatusCancelled, true},
		{domain.StatusPending, domain.StatusRunning, false},
		{domain.StatusPending, domain.StatusCompleted, false},

		// READY transitions
		{domain.StatusReady, domain.StatusRunning, true},
		{domain.StatusReady, domain.StatusBlocked, true},
		{domain.StatusReady, domain.StatusCancelled, true},
		{domain.StatusReady, domain.StatusCompleted, false},

		// RUNNING transitions
		{domain.StatusRunning, domain.StatusCompleted, true},
		{domain.StatusRunning, domain.StatusAwaitingReview, true},
		{domain.StatusRunning, domain.StatusAwaitingMerge, true},
		{domain.StatusRunning, domain.StatusFailed, true},
		{domain.StatusRunning, domain.StatusCancelled, true},
		{domain.StatusRunning, domain.StatusBlocked, true},
		{domain.StatusRunning, domain.StatusPending, false},

		// AWAITING_REVIEW transitions
		{domain.StatusAwaitingReview, domain.StatusRunning, true},
		{domain.StatusAwaitingReview, domain.StatusFailed, true},
		{domain.StatusAwaitingReview, domain.StatusCancelled, true},
		{domain.StatusAwaitingReview, domain.StatusCompleted, false},

		// AWAITING_MERGE transitions
		{domain.StatusAwaitingMerge, domain.StatusCompleted, true},
		{domain.StatusAwaitingMerge, domain.StatusConflict, true},
		{domain.StatusAwaitingMerge, domain.StatusRunning, true},
		{domain.StatusAwaitingMerge, domain.StatusCancelled, true},

		// CONFLICT transitions
		{domain.StatusConflict, domain.StatusRunning, true},
		{domain.StatusConflict, domain.StatusFailed, true},
		{domain.StatusConflict, domain.StatusCancelled, true},

		// BLOCKED transitions
		{domain.StatusBlocked, domain.StatusReady, true},
		{domain.StatusBlocked, domain.StatusPending, true},
		{domain.StatusBlocked, domain.StatusCancelled, true},
		{domain.StatusBlocked, domain.StatusRunning, false},

		// Re-open and Retry transitions
		{domain.StatusCompleted, domain.StatusReady, true}, // Re-open by operator
		{domain.StatusCompleted, domain.StatusRunning, false},
		{domain.StatusFailed, domain.StatusReady, true},   // Retry
		{domain.StatusFailed, domain.StatusRunning, true}, // Immediate re-run
		{domain.StatusCancelled, domain.StatusReady, false},

		// Idempotent self-transition
		{domain.StatusRunning, domain.StatusRunning, true},
		{domain.StatusPending, domain.StatusPending, true},
	}

	for _, tt := range tests {
		got := tt.from.CanTransitionTo(tt.to)
		if got != tt.allowed {
			t.Errorf("CanTransitionTo(%s -> %s) = %v; expected %v", tt.from, tt.to, got, tt.allowed)
		}
	}
}

func TestTask_Validate(t *testing.T) {
	validTask := domain.Task{
		ID:     "task-01",
		PlanID: "plan-01",
		Status: domain.StatusReady,
	}
	if err := validTask.Validate(); err != nil {
		t.Errorf("Expected valid task, got error: %v", err)
	}

	missingID := domain.Task{PlanID: "plan-01", Status: domain.StatusReady}
	if err := missingID.Validate(); err == nil {
		t.Error("Expected error for missing task ID, got nil")
	}

	missingPlanID := domain.Task{ID: "task-01", Status: domain.StatusReady}
	if err := missingPlanID.Validate(); err == nil {
		t.Error("Expected error for missing plan ID, got nil")
	}

	invalidStatus := domain.Task{ID: "task-01", PlanID: "plan-01", Status: "BAD"}
	if err := invalidStatus.Validate(); err == nil {
		t.Error("Expected error for invalid status, got nil")
	}
}

func TestEntities_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)

	// Test Plan
	plan := domain.Plan{
		Slug:      "test-plan",
		Title:     "Test Plan",
		Status:    "ACTIVE",
		CreatedAt: now,
		UpdatedAt: now,
	}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal Plan: %v", err)
	}
	var unmarshaledPlan domain.Plan
	if err := json.Unmarshal(data, &unmarshaledPlan); err != nil {
		t.Fatalf("Failed to unmarshal Plan: %v", err)
	}
	if unmarshaledPlan.Slug != plan.Slug || unmarshaledPlan.Title != plan.Title {
		t.Errorf("Plan round-trip mismatch: got %+v, want %+v", unmarshaledPlan, plan)
	}

	// Test Task
	task := domain.Task{
		ID:           "task-01",
		PlanID:       "test-plan",
		Title:        "Implement Feature",
		Dependencies: []string{"task-00"},
		Harness:      "claude-code",
		WorktreePath: "/tmp/worktree-01",
		Status:       domain.StatusReady,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	taskBytes, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal Task: %v", err)
	}
	var unmarshaledTask domain.Task
	if err := json.Unmarshal(taskBytes, &unmarshaledTask); err != nil {
		t.Fatalf("Failed to unmarshal Task: %v", err)
	}
	if unmarshaledTask.ID != task.ID || len(unmarshaledTask.Dependencies) != 1 || unmarshaledTask.Dependencies[0] != "task-00" {
		t.Errorf("Task round-trip mismatch: got %+v, want %+v", unmarshaledTask, task)
	}

	// Test Artifact
	artifact := domain.Artifact{
		ID:        "art-01",
		TaskID:    "task-01",
		Type:      domain.ArtifactTypeImage,
		FilePath:  "/tmp/screenshot.png",
		MimeType:  "image/png",
		ByteSize:  1024,
		Sha256:    "abc123",
		CreatedAt: now,
	}
	artBytes, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("Failed to marshal Artifact: %v", err)
	}
	var unmarshaledArtifact domain.Artifact
	if err := json.Unmarshal(artBytes, &unmarshaledArtifact); err != nil {
		t.Fatalf("Failed to unmarshal Artifact: %v", err)
	}
	if unmarshaledArtifact.Type != domain.ArtifactTypeImage || unmarshaledArtifact.MimeType != "image/png" {
		t.Errorf("Artifact round-trip mismatch: got %+v, want %+v", unmarshaledArtifact, artifact)
	}
}
