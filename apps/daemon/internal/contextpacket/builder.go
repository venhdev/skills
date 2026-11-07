// Package contextpacket assembles prompt-ready context packets for agent
// tasks. It owns entity resolution, skill path resolution, artifact loading, and
// prompt compilation so the storage layer stays persistence-only.
package contextpacket

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"zenf/internal/domain"
	"zenf/internal/domain/prompt"
)

// Builder compiles context packets for tasks.
type Builder struct {
	artifacts domain.ArtifactReader
	projects  domain.ProjectReader
	plans     domain.PlanReader
	agents    domain.AgentReader
	documents domain.DocumentReader
	compiler  prompt.Compiler
}

var _ domain.ContextPacketBuilder = (*Builder)(nil)

// Option configures optional entity readers.
type Option func(*Builder)

// WithProjectReader enables project context in the packet.
func WithProjectReader(reader domain.ProjectReader) Option {
	return func(b *Builder) { b.projects = reader }
}

// WithPlanReader enables plan context in the packet.
func WithPlanReader(reader domain.PlanReader) Option {
	return func(b *Builder) { b.plans = reader }
}

// WithAgentReader enables agent persona context in the packet.
func WithAgentReader(reader domain.AgentReader) Option {
	return func(b *Builder) { b.agents = reader }
}

// WithDocumentReader enables linked SSOT document resolution.
func WithDocumentReader(reader domain.DocumentReader) Option {
	return func(b *Builder) { b.documents = reader }
}

// New returns a Builder that loads input artifacts from artifacts and compiles
// prompts with compiler. A nil compiler falls back to the Markdown compiler.
func New(artifacts domain.ArtifactReader, compiler prompt.Compiler, opts ...Option) *Builder {
	if compiler == nil {
		compiler = prompt.NewCompiler()
	}
	builder := &Builder{artifacts: artifacts, compiler: compiler}
	for _, opt := range opts {
		opt(builder)
	}
	return builder
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

	var project *domain.Project
	if task.ProjectSlug != "" && b.projects != nil {
		loaded, err := b.projects.GetProject(ctx, task.ProjectSlug)
		if err != nil {
			return nil, fmt.Errorf("failed to load project %s for task %s: %w", task.ProjectSlug, task.ID, err)
		}
		project = loaded
	}

	var plan *domain.Plan
	if task.PlanSlug != "" && b.plans != nil {
		loaded, err := b.plans.GetPlan(ctx, task.PlanSlug)
		if err != nil {
			return nil, fmt.Errorf("failed to load plan %s for task %s: %w", task.PlanSlug, task.ID, err)
		}
		plan = loaded
	}

	var agent *domain.Agent
	if task.AgentSlug != "" && b.agents != nil {
		loaded, err := b.agents.GetAgent(ctx, task.AgentSlug)
		if err != nil {
			return nil, fmt.Errorf("failed to load agent %s for task %s: %w", task.AgentSlug, task.ID, err)
		}
		agent = loaded
	}

	linked, err := b.loadLinkedDocuments(ctx, task)
	if err != nil {
		return nil, err
	}

	packet, err := b.compiler.Compile(ctx, prompt.Input{
		Task:             task,
		Project:          project,
		Plan:             plan,
		Agent:            agent,
		InputAttachments: attachments,
		LinkedDocuments:  linked,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to compile prompt for task %s: %w", task.ID, err)
	}
	if packet == nil {
		return nil, fmt.Errorf("prompt compiler returned a nil packet for task %s", task.ID)
	}

	packet.SkillPaths = resolveSkillPaths(task)
	return packet, nil
}

// loadLinkedDocuments resolves required document keys inside the task's
// project, then falls back to globally scoped documents.
func (b *Builder) loadLinkedDocuments(ctx context.Context, task domain.Task) ([]domain.DocumentBrief, error) {
	linked := make([]domain.DocumentBrief, 0, len(task.RequiredDocs))
	if b.documents == nil {
		return linked, nil
	}
	for _, key := range task.RequiredDocs {
		document, err := b.documents.GetDocumentByKey(ctx, task.ProjectSlug, key)
		if errors.Is(err, domain.ErrNotFound) && task.ProjectSlug != "" {
			document, err = b.documents.GetDocumentByKey(ctx, "", key)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load required document %s for task %s: %w", key, task.ID, err)
		}
		linked = append(linked, document.Brief())
	}
	return linked, nil
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
