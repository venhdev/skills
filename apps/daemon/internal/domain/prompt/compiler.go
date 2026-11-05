package prompt

import (
	"context"
	"fmt"
	"strings"

	"zenfd/internal/domain"
)

// Input carries every entity the compiler may render into a context packet.
// Pointers stay nil when the task is an orphan or has no agent binding.
type Input struct {
	Task             domain.Task
	Project          *domain.Project
	Plan             *domain.Plan
	Agent            *domain.Agent
	InputAttachments []domain.Artifact
	LinkedDocuments  []domain.DocumentBrief
}

// Compiler defines the contract for assembling structured prompt context packets.
type Compiler interface {
	Compile(ctx context.Context, input Input) (*domain.ContextPacket, error)
}

// MarkdownPromptCompiler formats prompt packets using Markdown syntax.
type MarkdownPromptCompiler struct{}

var _ Compiler = (*MarkdownPromptCompiler)(nil)

// NewCompiler returns a new instance of MarkdownPromptCompiler.
func NewCompiler() *MarkdownPromptCompiler {
	return &MarkdownPromptCompiler{}
}

// Compile generates a prompt-ready ContextPacket for an agent.
func (c *MarkdownPromptCompiler) Compile(ctx context.Context, input Input) (*domain.ContextPacket, error) {
	task := input.Task
	if input.InputAttachments == nil {
		input.InputAttachments = []domain.Artifact{}
	}
	if input.LinkedDocuments == nil {
		input.LinkedDocuments = []domain.DocumentBrief{}
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
	if input.Project != nil {
		sb.WriteString(fmt.Sprintf("- **Project**: `%s` — %s\n", input.Project.Slug, input.Project.Name))
	}
	switch {
	case input.Plan != nil:
		sb.WriteString(fmt.Sprintf("- **Plan**: `%s` — %s\n", input.Plan.Slug, input.Plan.Title))
	case task.PlanSlug != "":
		sb.WriteString(fmt.Sprintf("- **Plan**: `%s`\n", task.PlanSlug))
	case task.ProjectSlug != "":
		sb.WriteString("- **Scope**: project task (no plan wrapper)\n")
	default:
		sb.WriteString("- **Scope**: orphan task (no project or plan)\n")
	}
	sb.WriteString(fmt.Sprintf("- **Priority**: `%s` | **Complexity**: `%s` | **Risk**: `%s`\n", task.Priority, task.Complexity, task.RiskLevel))
	if input.Agent != nil {
		sb.WriteString(fmt.Sprintf("- **Agent persona**: `%s` (%s) — %s\n", input.Agent.Slug, input.Agent.Persona, input.Agent.Name))
		if input.Agent.Goal != "" {
			sb.WriteString(fmt.Sprintf("- **Agent goal**: %s\n", input.Agent.Goal))
		}
	}
	if task.Harness != "" {
		sb.WriteString(fmt.Sprintf("- **Intended executor**: `%s`\n", task.Harness))
	}
	if task.WorktreePath != "" {
		sb.WriteString(fmt.Sprintf("- **Worktree Path**: `%s`\n", task.WorktreePath))
	}
	sb.WriteString("\n## 1. Description & Context\n")
	if task.Description != "" {
		sb.WriteString(task.Description + "\n\n")
	} else {
		sb.WriteString("Execute the implementation according to plan specifications.\n\n")
	}

	section := 2
	if len(input.LinkedDocuments) > 0 {
		sb.WriteString(fmt.Sprintf("## %d. Linked SSOT Documents\n", section))
		for _, doc := range input.LinkedDocuments {
			sb.WriteString(fmt.Sprintf("- `%s` [%s/%s] — %s (`%s`)", doc.Key, doc.Type, doc.Scope, doc.Title, doc.Path))
			if doc.Summary != "" {
				sb.WriteString(": " + doc.Summary)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		section++
	}

	if len(task.AcceptanceCriteria) > 0 {
		sb.WriteString(fmt.Sprintf("## %d. Acceptance Criteria\n", section))
		for _, ac := range task.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", ac))
		}
		sb.WriteString("\n")
		section++
	}

	if len(targetFiles) > 0 {
		sb.WriteString(fmt.Sprintf("## %d. Target Files\n", section))
		for _, tf := range targetFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", tf))
		}
		sb.WriteString("\n")
		section++
	}

	if len(verificationCommands) > 0 {
		sb.WriteString(fmt.Sprintf("## %d. Verification Commands\n", section))
		sb.WriteString("Run the following commands to verify implementation correctness before completing:\n```bash\n")
		for _, vc := range verificationCommands {
			sb.WriteString(vc + "\n")
		}
		sb.WriteString("```\n\n")
		section++
	}

	sb.WriteString(fmt.Sprintf("## %d. MCP Tool Handshake Protocol\n", section))
	sb.WriteString(fmt.Sprintf("- **Claim**: Call `claim_task(task_id=\"%s\", worker_role=\"CODER\")` before editing files.\n", task.ID))
	sb.WriteString(fmt.Sprintf("- **Architectural Drift / Blocked**: Call `review(action=\"request\", task_id=\"%s\", reason=\"...\", details=\"...\", options=[\"...\"])` and exit immediately. The main orchestrator agent relays the human decision and calls `review(action=\"resolve\", ...)`.\n", task.ID))
	sb.WriteString(fmt.Sprintf("- **Complete**: Once all verification commands pass with exit code 0, call `transition_task(task_ids=[\"%s\"], action=\"COMPLETE\", summary=\"...\")`. Tasks with a worktree stop at AWAITING_MERGE for the orchestrator to merge outside MCP.\n", task.ID))
	sb.WriteString(fmt.Sprintf("- **Failure**: If the task cannot be completed, call `transition_task(task_ids=[\"%s\"], action=\"FAIL\", reason=\"...\")`.\n", task.ID))

	return &domain.ContextPacket{
		Task:                 task,
		Project:              input.Project,
		Plan:                 input.Plan,
		Agent:                input.Agent,
		TargetFiles:          targetFiles,
		VerificationCommands: verificationCommands,
		SkillPaths:           []string{},
		InputAttachments:     input.InputAttachments,
		LinkedDocuments:      input.LinkedDocuments,
		PromptMarkdown:       sb.String(),
	}, nil
}
