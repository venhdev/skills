// Package contextpacket assembles prompt-ready context packets for agent
// tasks. It owns skill path resolution, artifact loading, and prompt
// compilation so the storage layer stays persistence-only.
package contextpacket

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"zenfd/internal/domain"
	"zenfd/internal/domain/prompt"
)

// Builder compiles context packets for tasks.
type Builder struct {
	artifacts domain.ArtifactReader
	compiler  prompt.Compiler
}

var _ domain.ContextPacketBuilder = (*Builder)(nil)

// New returns a Builder that loads input artifacts from artifacts and compiles
// prompts with compiler. A nil compiler falls back to the Markdown compiler.
func New(artifacts domain.ArtifactReader, compiler prompt.Compiler) *Builder {
	if compiler == nil {
		compiler = prompt.NewCompiler()
	}
	return &Builder{artifacts: artifacts, compiler: compiler}
}

// Build compiles the full context packet for task.
func (b *Builder) Build(ctx context.Context, task domain.Task) (*domain.ContextPacket, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	attachments := []domain.Artifact{}
	if b.artifacts != nil {
		loaded, err := b.artifacts.GetArtifactsByTask(ctx, task.ID, domain.ArtifactRoleInput)
		if err != nil {
			return nil, fmt.Errorf("failed to load input artifacts for task %s: %w", task.ID, err)
		}
		attachments = loaded
	}

	packet, err := b.compiler.Compile(ctx, task, attachments)
	if err != nil {
		return nil, fmt.Errorf("failed to compile prompt for task %s: %w", task.ID, err)
	}
	if packet == nil {
		return nil, fmt.Errorf("prompt compiler returned a nil packet for task %s", task.ID)
	}

	packet.SkillPaths = resolveSkillPaths(task)
	return packet, nil
}

// resolveSkillPaths maps required skill names to SKILL.md files when the task
// has a worktree binding. Without a repo binding the daemon cannot discover a
// repository root, so agents resolve bare skill names through their own
// registry.
func resolveSkillPaths(task domain.Task) []string {
	paths := make([]string, 0, len(task.RequiredSkills))
	if task.WorktreePath == "" {
		return append(paths, task.RequiredSkills...)
	}

	for _, skill := range task.RequiredSkills {
		localPath := filepath.Join(task.WorktreePath, ".agents", "skills", skill, "SKILL.md")
		if _, err := os.Stat(localPath); err == nil {
			paths = append(paths, localPath)
			continue
		}
		paths = append(paths, skill)
	}
	return paths
}
