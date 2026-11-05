package contextpacket_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"zenfd/internal/contextpacket"
	"zenfd/internal/domain"
	"zenfd/internal/domain/prompt"
)

type fakeArtifacts struct {
	artifacts []domain.Artifact
	err       error
	role      string
	calls     int
}

func (f *fakeArtifacts) GetArtifactsByTask(_ context.Context, _ string, role string) ([]domain.Artifact, error) {
	f.calls++
	f.role = role
	if f.err != nil {
		return nil, f.err
	}
	return f.artifacts, nil
}

type fakeCompiler struct {
	packet *domain.ContextPacket
	err    error
	called bool
	input  prompt.Input
}

func (f *fakeCompiler) Compile(_ context.Context, input prompt.Input) (*domain.ContextPacket, error) {
	f.called = true
	f.input = input
	if f.packet != nil {
		f.packet.Task = input.Task
		f.packet.LinkedDocuments = input.LinkedDocuments
	}
	return f.packet, f.err
}

type fakeProjects struct {
	project *domain.Project
	err     error
}

func (f *fakeProjects) GetProject(context.Context, string) (*domain.Project, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.project, nil
}

type fakePlans struct {
	plan *domain.Plan
	err  error
}

func (f *fakePlans) GetPlan(context.Context, string) (*domain.Plan, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.plan, nil
}

type fakeAgents struct {
	agent *domain.Agent
	err   error
}

func (f *fakeAgents) GetAgent(context.Context, string) (*domain.Agent, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.agent, nil
}

type fakeDocuments struct {
	byKey map[string]*domain.Document
	err   error
}

func (f *fakeDocuments) GetDocumentByKey(_ context.Context, projectSlug string, key string) (*domain.Document, error) {
	if f.err != nil {
		return nil, f.err
	}
	document, ok := f.byKey[projectSlug+"/"+key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return document, nil
}

func TestBuilder_ResolvesWorktreeSkillPaths(t *testing.T) {
	worktree := t.TempDir()
	skillFile := filepath.Join(worktree, ".agents", "skills", "tdd", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillFile), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(skillFile, []byte("# tdd"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	compiler := &fakeCompiler{packet: &domain.ContextPacket{}}
	builder := contextpacket.New(&fakeArtifacts{}, compiler)

	packet, err := builder.Build(context.Background(), domain.Task{
		ID:             "T-1",
		WorktreePath:   worktree,
		RequiredSkills: []string{"tdd", "recon"},
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(packet.SkillPaths) != 2 {
		t.Fatalf("expected 2 skill paths, got %d", len(packet.SkillPaths))
	}
	if packet.SkillPaths[0] != skillFile {
		t.Errorf("expected resolved skill file %q, got %q", skillFile, packet.SkillPaths[0])
	}
	if packet.SkillPaths[1] != "recon" {
		t.Errorf("expected bare skill name for missing file, got %q", packet.SkillPaths[1])
	}
}

func TestBuilder_WithoutWorktreeKeepsBareSkillNames(t *testing.T) {
	compiler := &fakeCompiler{packet: &domain.ContextPacket{}}
	builder := contextpacket.New(&fakeArtifacts{}, compiler)

	packet, err := builder.Build(context.Background(), domain.Task{
		ID:             "T-2",
		RequiredSkills: []string{"tdd"},
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(packet.SkillPaths) != 1 || packet.SkillPaths[0] != "tdd" {
		t.Fatalf("expected bare skill name, got %v", packet.SkillPaths)
	}
}

func TestBuilder_LoadsInputAttachments(t *testing.T) {
	artifacts := &fakeArtifacts{artifacts: []domain.Artifact{{ID: "art-1", TaskID: "T-3", Role: domain.ArtifactRoleInput}}}
	compiler := &fakeCompiler{packet: &domain.ContextPacket{}}
	builder := contextpacket.New(artifacts, compiler)

	if _, err := builder.Build(context.Background(), domain.Task{ID: "T-3"}); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if artifacts.role != domain.ArtifactRoleInput {
		t.Errorf("expected INPUT role query, got %q", artifacts.role)
	}
	if len(compiler.input.InputAttachments) != 1 || compiler.input.InputAttachments[0].ID != "art-1" {
		t.Fatalf("expected attachment to reach the compiler, got %v", compiler.input.InputAttachments)
	}
}

func TestBuilder_ResolvesEntityContextAndLinkedDocuments(t *testing.T) {
	compiler := &fakeCompiler{packet: &domain.ContextPacket{}}
	builder := contextpacket.New(&fakeArtifacts{}, compiler,
		contextpacket.WithProjectReader(&fakeProjects{project: &domain.Project{Slug: "alpha", Name: "Alpha"}}),
		contextpacket.WithPlanReader(&fakePlans{plan: &domain.Plan{Slug: "p1", ProjectSlug: "alpha", Title: "Plan One"}}),
		contextpacket.WithAgentReader(&fakeAgents{agent: &domain.Agent{Slug: "coder", Name: "Coder"}}),
		contextpacket.WithDocumentReader(&fakeDocuments{byKey: map[string]*domain.Document{
			"alpha/ARCH": {ID: "doc-1", ProjectSlug: "alpha", Key: "ARCH", Slug: "arch", Title: "Architecture", Path: "/tmp/arch.md"},
		}}),
	)

	task := domain.Task{
		ID: "T-CTX", ProjectSlug: "alpha", PlanSlug: "p1", AgentSlug: "coder",
		RequiredDocs: []string{"ARCH"},
	}
	packet, err := builder.Build(context.Background(), task)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if compiler.input.Project == nil || compiler.input.Project.Slug != "alpha" {
		t.Errorf("expected project to be resolved, got %+v", compiler.input.Project)
	}
	if compiler.input.Plan == nil || compiler.input.Plan.Slug != "p1" {
		t.Errorf("expected plan to be resolved, got %+v", compiler.input.Plan)
	}
	if compiler.input.Agent == nil || compiler.input.Agent.Slug != "coder" {
		t.Errorf("expected agent to be resolved, got %+v", compiler.input.Agent)
	}
	if len(compiler.input.LinkedDocuments) != 1 || compiler.input.LinkedDocuments[0].Key != "ARCH" {
		t.Fatalf("expected linked document, got %+v", compiler.input.LinkedDocuments)
	}
	if len(packet.LinkedDocuments) != 1 {
		t.Fatalf("expected packet to expose linked documents, got %+v", packet.LinkedDocuments)
	}
}

func TestBuilder_FallsBackToGlobalDocument(t *testing.T) {
	compiler := &fakeCompiler{packet: &domain.ContextPacket{}}
	builder := contextpacket.New(&fakeArtifacts{}, compiler,
		contextpacket.WithDocumentReader(&fakeDocuments{byKey: map[string]*domain.Document{
			"/RULES": {ID: "doc-global", Key: "RULES", Slug: "rules", Title: "Rules", Path: "/tmp/rules.md"},
		}}),
	)

	_, err := builder.Build(context.Background(), domain.Task{
		ID: "T-GLOBAL", ProjectSlug: "alpha", RequiredDocs: []string{"RULES"},
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(compiler.input.LinkedDocuments) != 1 || compiler.input.LinkedDocuments[0].ID != "doc-global" {
		t.Fatalf("expected global fallback, got %+v", compiler.input.LinkedDocuments)
	}
}

func TestBuilder_PropagatesArtifactError(t *testing.T) {
	artifacts := &fakeArtifacts{err: errors.New("db down")}
	compiler := &fakeCompiler{packet: &domain.ContextPacket{}}
	builder := contextpacket.New(artifacts, compiler)

	if _, err := builder.Build(context.Background(), domain.Task{ID: "T-4"}); err == nil {
		t.Fatal("expected artifact read error to propagate")
	}
	if compiler.called {
		t.Fatal("compiler must not run when artifact loading fails")
	}
}

func TestBuilder_PropagatesEntityErrors(t *testing.T) {
	compiler := &fakeCompiler{packet: &domain.ContextPacket{}}
	cases := []struct {
		name    string
		options []contextpacket.Option
		task    domain.Task
	}{
		{
			name:    "project",
			options: []contextpacket.Option{contextpacket.WithProjectReader(&fakeProjects{err: errors.New("project read failed")})},
			task:    domain.Task{ID: "T-P", ProjectSlug: "alpha"},
		},
		{
			name:    "plan",
			options: []contextpacket.Option{contextpacket.WithPlanReader(&fakePlans{err: errors.New("plan read failed")})},
			task:    domain.Task{ID: "T-PL", ProjectSlug: "alpha", PlanSlug: "p1"},
		},
		{
			name:    "agent",
			options: []contextpacket.Option{contextpacket.WithAgentReader(&fakeAgents{err: errors.New("agent read failed")})},
			task:    domain.Task{ID: "T-A", AgentSlug: "coder"},
		},
		{
			name: "document",
			options: []contextpacket.Option{contextpacket.WithDocumentReader(&fakeDocuments{
				byKey: map[string]*domain.Document{}, err: errors.New("document read failed"),
			})},
			task: domain.Task{ID: "T-D", RequiredDocs: []string{"ARCH"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			builder := contextpacket.New(&fakeArtifacts{}, compiler, tc.options...)
			if _, err := builder.Build(context.Background(), tc.task); err == nil {
				t.Fatal("expected entity read error to propagate")
			}
		})
	}
}

func TestBuilder_PropagatesCompilerError(t *testing.T) {
	compiler := &fakeCompiler{err: errors.New("compile failed")}
	builder := contextpacket.New(&fakeArtifacts{}, compiler)

	if _, err := builder.Build(context.Background(), domain.Task{ID: "T-5"}); err == nil {
		t.Fatal("expected compiler error to propagate")
	}
}

func TestBuilder_NilPacketReturnsError(t *testing.T) {
	compiler := &fakeCompiler{}
	builder := contextpacket.New(&fakeArtifacts{}, compiler)

	if _, err := builder.Build(context.Background(), domain.Task{ID: "T-6"}); err == nil {
		t.Fatal("expected nil packet to become an error instead of a panic")
	}
}
