package storage_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

// TestTaskStore_RoundTripPreservesEveryColumn writes a task where every column
// carries a distinct sentinel value and reads it back. Because taskColumns
// drives the SELECT projection while scanTask lists destinations positionally,
// a reordered column pair (for example two adjacent TEXT columns) would swap
// values here instead of silently corrupting data in production.
func TestTaskStore_RoundTripPreservesEveryColumn(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "task-roundtrip.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	claimedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	createdAt := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)

	want := domain.Task{
		ID:                   "T-ROUNDTRIP",
		PlanID:               "plan-roundtrip",
		Title:                "title-sentinel",
		Description:          "description-sentinel",
		Dependencies:         []string{"dep-a", "dep-b"},
		Priority:             domain.PriorityP0,
		Complexity:           domain.ComplexityHigh,
		RiskLevel:            domain.RiskMedium,
		RequiredSkills:       []string{"skill-a", "skill-b"},
		RequiredTools:        []string{"tool-a", "tool-b"},
		TargetFiles:          []string{"file-a.go", "file-b.go"},
		AcceptanceCriteria:   []string{"criterion-a", "criterion-b"},
		VerificationCommands: []string{"command-a", "command-b"},
		Harness:              "harness-sentinel",
		WorktreePath:         "/worktree/sentinel",
		Status:               domain.StatusRunning,
		ClaimedBy:            "worker-sentinel",
		ClaimedAt:            &claimedAt,
		OrchestratorID:       "orchestrator-sentinel",
		WorkerRole:           "CODER",
		Notes:                "notes-sentinel",
		ReviewReason:         "review-reason-sentinel",
		ReviewDetails:        "review-details-sentinel",
		ReviewOptions:        []string{"option-a", "option-b"},
		ReviewResponse:       "review-response-sentinel",
		ResolvedBy:           "resolved-by-sentinel",
		ConflictFiles:        []string{"conflict-a.go", "conflict-b.go"},
		CreatedAt:            createdAt,
	}

	plan := domain.Plan{Slug: want.PlanID, Title: "Round Trip Plan", Status: "ACTIVE"}
	if err := store.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	// CreateTask allows forward references, so the sentinel dependency IDs do
	// not need real rows; the point here is column round-trip fidelity.
	if err := store.CreateTask(ctx, want); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	got, err := store.GetTask(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if got.ID != want.ID || got.PlanID != want.PlanID {
		t.Errorf("identity mismatch: got id=%q plan_id=%q", got.ID, got.PlanID)
	}
	if got.Title != want.Title {
		t.Errorf("title = %q, want %q", got.Title, want.Title)
	}
	if got.Description != want.Description {
		t.Errorf("description = %q, want %q", got.Description, want.Description)
	}
	if !slices.Equal(got.Dependencies, want.Dependencies) {
		t.Errorf("dependencies = %v, want %v", got.Dependencies, want.Dependencies)
	}
	if got.Priority != want.Priority {
		t.Errorf("priority = %q, want %q", got.Priority, want.Priority)
	}
	if got.Complexity != want.Complexity {
		t.Errorf("complexity = %q, want %q", got.Complexity, want.Complexity)
	}
	if got.RiskLevel != want.RiskLevel {
		t.Errorf("risk_level = %q, want %q", got.RiskLevel, want.RiskLevel)
	}
	if !slices.Equal(got.RequiredSkills, want.RequiredSkills) {
		t.Errorf("required_skills = %v, want %v", got.RequiredSkills, want.RequiredSkills)
	}
	if !slices.Equal(got.RequiredTools, want.RequiredTools) {
		t.Errorf("required_tools = %v, want %v", got.RequiredTools, want.RequiredTools)
	}
	if !slices.Equal(got.TargetFiles, want.TargetFiles) {
		t.Errorf("target_files = %v, want %v", got.TargetFiles, want.TargetFiles)
	}
	if !slices.Equal(got.AcceptanceCriteria, want.AcceptanceCriteria) {
		t.Errorf("acceptance_criteria = %v, want %v", got.AcceptanceCriteria, want.AcceptanceCriteria)
	}
	if !slices.Equal(got.VerificationCommands, want.VerificationCommands) {
		t.Errorf("verification_commands = %v, want %v", got.VerificationCommands, want.VerificationCommands)
	}
	if got.Harness != want.Harness {
		t.Errorf("harness = %q, want %q", got.Harness, want.Harness)
	}
	if got.WorktreePath != want.WorktreePath {
		t.Errorf("worktree_path = %q, want %q", got.WorktreePath, want.WorktreePath)
	}
	if got.Status != want.Status {
		t.Errorf("status = %q, want %q", got.Status, want.Status)
	}
	if got.ClaimedBy != want.ClaimedBy {
		t.Errorf("claimed_by = %q, want %q", got.ClaimedBy, want.ClaimedBy)
	}
	if got.ClaimedAt == nil || !got.ClaimedAt.Equal(claimedAt) {
		t.Errorf("claimed_at = %v, want %v", got.ClaimedAt, claimedAt)
	}
	if got.OrchestratorID != want.OrchestratorID {
		t.Errorf("orchestrator_id = %q, want %q", got.OrchestratorID, want.OrchestratorID)
	}
	if got.WorkerRole != want.WorkerRole {
		t.Errorf("worker_role = %q, want %q", got.WorkerRole, want.WorkerRole)
	}
	if got.Notes != want.Notes {
		t.Errorf("notes = %q, want %q", got.Notes, want.Notes)
	}
	if got.ReviewReason != want.ReviewReason {
		t.Errorf("review_reason = %q, want %q", got.ReviewReason, want.ReviewReason)
	}
	if got.ReviewDetails != want.ReviewDetails {
		t.Errorf("review_details = %q, want %q", got.ReviewDetails, want.ReviewDetails)
	}
	if !slices.Equal(got.ReviewOptions, want.ReviewOptions) {
		t.Errorf("review_options = %v, want %v", got.ReviewOptions, want.ReviewOptions)
	}
	if got.ReviewResponse != want.ReviewResponse {
		t.Errorf("review_response = %q, want %q", got.ReviewResponse, want.ReviewResponse)
	}
	if got.ResolvedBy != want.ResolvedBy {
		t.Errorf("resolved_by = %q, want %q", got.ResolvedBy, want.ResolvedBy)
	}
	if !slices.Equal(got.ConflictFiles, want.ConflictFiles) {
		t.Errorf("conflict_files = %v, want %v", got.ConflictFiles, want.ConflictFiles)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, createdAt)
	}
	// updated_at is intentionally not compared: upsertTaskTx always overwrites it.
}
