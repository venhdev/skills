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

// mockPromptCompiler records the task passed through the packet builder.
type mockPromptCompiler struct {
	called           bool
	task             domain.Task
	inputAttachments []domain.Artifact
}

var _ prompt.Compiler = (*mockPromptCompiler)(nil)

func (m *mockPromptCompiler) Compile(ctx context.Context, task domain.Task, inputAttachments []domain.Artifact) (*domain.ContextPacket, error) {
	m.called = true
	m.task = task
	m.inputAttachments = inputAttachments
	return &domain.ContextPacket{
		Task:           task,
		TargetFiles:    task.TargetFiles,
		PromptMarkdown: fmt.Sprintf("# MOCK PROMPT: %s", task.Title),
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
	store.TaskLifecycle.packets = contextpacket.New(store.ArtifactRepository, mock)

	plan := domain.Plan{
		Slug:   "plan-compiler-test",
		Title:  "Compiler Delegation Plan",
		Status: "ACTIVE",
	}
	task := domain.Task{
		ID:          "T-DELEGATE-01",
		PlanID:      plan.Slug,
		Title:       "Test Compiler Delegation",
		TargetFiles: []string{"a.go"},
		Status:      domain.StatusReady,
	}

	if err := store.SavePlanWithTasks(ctx, plan, []domain.Task{task}); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	packet, err := store.GetTaskContextPacket(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskContextPacket failed: %v", err)
	}

	if !mock.called {
		t.Fatal("expected mock prompt compiler to be called")
	}
	if mock.task.ID != "T-DELEGATE-01" {
		t.Errorf("expected task ID 'T-DELEGATE-01', got: %s", mock.task.ID)
	}
	if packet.PromptMarkdown != "# MOCK PROMPT: Test Compiler Delegation" {
		t.Errorf("unexpected PromptMarkdown: %s", packet.PromptMarkdown)
	}
}
