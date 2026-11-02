package parser

import (
	"strings"
	"testing"

	"zenfd/internal/domain"
)

func TestParseMarkdownPlan_Synthetic(t *testing.T) {
	mdContent := `
# Plan Dossier: feature-payment · ACTIVE

Target: payment-service

## Master Tasks

| Task ID | Description | Dependencies | Status | Harness |
| :--- | :--- | :--- | :--- | :--- |
| **T-01** | Create payment schema | - | **VERIFIED** | postgres |
| **T-02** | Stripe webhook | T-01 | **IN PROGRESS** | antigravity |
| **T-03** | Email receipt | T-02 | **PLANNED** | antigravity |
`

	plan, tasks, err := ParseMarkdownPlan(strings.NewReader(mdContent), "default", "/test/repo")
	if err != nil {
		t.Fatalf("ParseMarkdownPlan failed: %v", err)
	}

	if plan.Slug != "feature-payment" {
		t.Fatalf("expected slug feature-payment, got %s", plan.Slug)
	}
	if plan.Status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %s", plan.Status)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// T-01
	if tasks[0].ID != "T-01" || tasks[0].Status != domain.StatusCompleted {
		t.Errorf("task 0 mismatch: %+v", tasks[0])
	}
	// T-02
	if tasks[1].ID != "T-02" || tasks[1].Status != domain.StatusRunning || len(tasks[1].Dependencies) != 1 || tasks[1].Dependencies[0] != "T-01" {
		t.Errorf("task 1 mismatch: %+v", tasks[1])
	}
	// T-03
	if tasks[2].ID != "T-03" || tasks[2].Status != domain.StatusPending {
		t.Errorf("task 2 mismatch: %+v", tasks[2])
	}
}

func TestParseMarkdownPlanFile_ZenfdPlan(t *testing.T) {
	planPath := "../../../../.agents/scratch/plans/zenfd/README.md"
	plan, tasks, err := ParseMarkdownPlanFile(planPath, "/home/rirong/src/owner/my-skills")
	if err != nil {
		t.Fatalf("ParseMarkdownPlanFile on zenfd README failed: %v", err)
	}

	if plan.Slug != "zenfd" {
		t.Errorf("expected plan slug zenfd, got %s", plan.Slug)
	}

	if len(tasks) < 5 {
		t.Fatalf("expected at least 5 phases parsed, got %d", len(tasks))
	}

	t.Logf("Successfully parsed %d tasks from %s (Plan: %s)", len(tasks), planPath, plan.Slug)
	for _, task := range tasks {
		t.Logf("  Task: %s | Status: %s | Title: %s", task.ID, task.Status, task.Title)
	}
}
