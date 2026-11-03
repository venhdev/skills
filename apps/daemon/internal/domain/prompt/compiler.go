package prompt

import (
	"context"
	"fmt"
	"strings"

	"zenfd/internal/domain"
)

// Compiler defines the contract for assembling structured prompt context packets.
type Compiler interface {
	Compile(ctx context.Context, task domain.Task, upstreamDiffs []string, inputAttachments []domain.Artifact, specSnippets []string) (*domain.ContextPacket, error)
}

// MarkdownPromptCompiler formats prompt packets using Markdown syntax.
type MarkdownPromptCompiler struct{}

var _ Compiler = (*MarkdownPromptCompiler)(nil)

// NewCompiler returns a new instance of MarkdownPromptCompiler.
func NewCompiler() *MarkdownPromptCompiler {
	return &MarkdownPromptCompiler{}
}

// Compile generates a prompt-ready ContextPacket for an agent.
func (c *MarkdownPromptCompiler) Compile(
	ctx context.Context,
	task domain.Task,
	upstreamDiffs []string,
	inputAttachments []domain.Artifact,
	specSnippets []string,
) (*domain.ContextPacket, error) {
	if upstreamDiffs == nil {
		upstreamDiffs = []string{}
	}
	if inputAttachments == nil {
		inputAttachments = []domain.Artifact{}
	}
	targetFiles := task.TargetFiles
	if targetFiles == nil {
		targetFiles = []string{}
	}
	verificationCommands := task.VerificationCommands
	if verificationCommands == nil {
		verificationCommands = []string{}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Task Assignment: %s (%s)\n\n", task.Title, task.ID))
	sb.WriteString(fmt.Sprintf("- **Plan**: `%s`\n", task.PlanID))
	sb.WriteString(fmt.Sprintf("- **Priority**: `%s` | **Complexity**: `%s` | **Risk**: `%s`\n", task.Priority, task.Complexity, task.RiskLevel))
	if task.WorktreePath != "" {
		sb.WriteString(fmt.Sprintf("- **Worktree Path**: `%s`\n", task.WorktreePath))
	}
	sb.WriteString("\n## 1. Description & Context\n")
	if task.Description != "" {
		sb.WriteString(task.Description + "\n\n")
	} else {
		sb.WriteString("Execute the implementation according to plan specifications.\n\n")
	}

	if len(specSnippets) > 0 {
		sb.WriteString("## 1.5. Linked Architectural Blueprints & Contracts (SSOT)\n")
		for _, snippet := range specSnippets {
			sb.WriteString(snippet)
			if !strings.HasSuffix(snippet, "\n") {
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	if len(task.AcceptanceCriteria) > 0 {
		sb.WriteString("## 2. Acceptance Criteria\n")
		for _, ac := range task.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", ac))
		}
		sb.WriteString("\n")
	}

	if len(targetFiles) > 0 {
		sb.WriteString("## 3. Target Files\n")
		for _, tf := range targetFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", tf))
		}
		sb.WriteString("\n")
	}

	if len(verificationCommands) > 0 {
		sb.WriteString("## 4. Verification Commands\n")
		sb.WriteString("Run the following commands to verify implementation correctness before completing:\n```bash\n")
		for _, vc := range verificationCommands {
			sb.WriteString(vc + "\n")
		}
		sb.WriteString("```\n\n")
	}

	if len(upstreamDiffs) > 0 {
		sb.WriteString("## Upstream Diffs\n")
		for _, diff := range upstreamDiffs {
			sb.WriteString("```diff\n")
			sb.WriteString(diff)
			if !strings.HasSuffix(diff, "\n") {
				sb.WriteString("\n")
			}
			sb.WriteString("```\n\n")
		}
	}

	sb.WriteString("## 5. MCP Tool Handshake Protocol\n")
	sb.WriteString(fmt.Sprintf("- **Claim**: Call `zenfd_claim_task(task_id=\"%s\", worker_role=\"CODER\", pid=$PID)` before editing files.\n", task.ID))
	sb.WriteString(fmt.Sprintf("- **Architectural Drift / Blocked**: Call `zenfd_yield_review(task_id=\"%s\", reason=\"...\", details=\"...\", options=[\"...\"])` and exit immediately.\n", task.ID))
	sb.WriteString(fmt.Sprintf("- **Stage Merge**: Once all verification commands pass with exit code 0, call `zenfd_stage_merge(task_id=\"%s\")`.\n", task.ID))

	packet := &domain.ContextPacket{
		Task:                 task,
		TargetFiles:          targetFiles,
		VerificationCommands: verificationCommands,
		SkillPaths:           []string{},
		UpstreamDiffs:        upstreamDiffs,
		InputAttachments:     inputAttachments,
		PromptMarkdown:       sb.String(),
	}

	return packet, nil
}
