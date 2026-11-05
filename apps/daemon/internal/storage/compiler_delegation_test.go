package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"zenfd/internal/contextpacket"
	"zenfd/internal/domain"
	"zenfd/internal/domain/prompt"
)

// mockPromptCompiler records the packet input passed through the builder.
type mockPromptCompiler struct {
	called bool
	input  prompt.Input
}

var _ prompt.Compiler = (*mockPromptCompiler)(nil)

func (m *mockPromptCompiler) Compile(ctx context.Context, input prompt.Input) (*domain.ContextPacket, error) {
	m.called = true
	m.input = input
	return &domain.ContextPacket{
		Task:            input.Task,
		Project:         input.Project,
		Plan:            input.Plan,
		Agent:           input.Agent,
		TargetFiles:     input.Task.TargetFiles,
		LinkedDocuments: input.LinkedDocuments,
		PromptMarkdown:  fmt.Sprintf("# MOCK PROMPT: %s", input.Task.Title),
	}, nil
}

// TestPlanStore_PromptCompilerDelegation proves the lifecycle uses the packet
// builder wired into the store. The test lives in-package so production keeps a
// single constructor with no test-only options.
func TestPlanStore_PromptCompilerDelegation(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test_compiler_delegation.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mock := &mockPromptCompiler{}
	store := NewStore(db)
	store.TaskLifecycle.packets = contextpacket.New(store.ArtifactRepository, mock,
		contextpacket.WithProjectReader(store.ProjectRepository),
		contextpacket.WithPlanReader(store.PlanRepository),
		contextpacket.WithAgentReader(store.AgentRepository),
		contextpacket.WithDocumentReader(store.DocumentRepository),
	)

	if err := store.CreateProject(ctx, domain.Project{Slug: "compiler-project", Name: "Compiler Project"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	plan := domain.Plan{
		Slug:        "plan-compiler-test",
		ProjectSlug: "compiler-project",
		Title:       "Compiler Delegation Plan",
		Status:      domain.PlanStatusActive,
	}
	task := domain.Task{
		ID:          "T-DELEGATE-01",
		PlanSlug:    plan.Slug,
		Title:       "Test Compiler Delegation",
		TargetFiles: []string{"a.go"},
		Status:      domain.StatusReady,
	}

	if err := store.CreatePlanWithTasks(ctx, plan, []domain.Task{task}); err != nil {
		t.Fatalf("CreatePlanWithTasks failed: %v", err)
	}

	packet, err := store.GetTaskContextPacket(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskContextPacket failed: %v", err)
	}

	if !mock.called {
		t.Fatal("expected mock prompt compiler to be called")
	}
	if mock.input.Task.ID != "T-DELEGATE-01" {
		t.Errorf("expected task ID 'T-DELEGATE-01', got: %s", mock.input.Task.ID)
	}
	if mock.input.Project == nil || mock.input.Project.Slug != "compiler-project" {
		t.Errorf("expected project context, got %+v", mock.input.Project)
	}
	if mock.input.Plan == nil || mock.input.Plan.Slug != "plan-compiler-test" {
		t.Errorf("expected plan context, got %+v", mock.input.Plan)
	}
	if packet.PromptMarkdown != "# MOCK PROMPT: Test Compiler Delegation" {
		t.Errorf("unexpected PromptMarkdown: %s", packet.PromptMarkdown)
	}
}
