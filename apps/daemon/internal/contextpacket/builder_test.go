package contextpacket_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"zenfd/internal/contextpacket"
	"zenfd/internal/domain"
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
	packet      *domain.ContextPacket
	err         error
	called      bool
	attachments []domain.Artifact
}

func (f *fakeCompiler) Compile(_ context.Context, task domain.Task, attachments []domain.Artifact) (*domain.ContextPacket, error) {
	f.called = true
	f.attachments = attachments
	if f.packet != nil {
		f.packet.Task = task
	}
	return f.packet, f.err
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
	if len(compiler.attachments) != 1 || compiler.attachments[0].ID != "art-1" {
		t.Fatalf("expected attachment to reach the compiler, got %v", compiler.attachments)
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
