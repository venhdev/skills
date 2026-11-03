package prompt_test

import (
	"context"
	"strings"
	"testing"

	"zenfd/internal/domain"
	"zenfd/internal/domain/prompt"
)

func TestMarkdownPromptCompiler_Compile_CompletePacket(t *testing.T) {
	compiler := prompt.NewCompiler()
	ctx := context.Background()

	task := domain.Task{
		ID:                   "P-03",
		PlanID:               "plan-engine-v1",
		RepoPath:             "/home/dev/src/owner/my-skills",
		Title:                "Prompt Context Compiler Domain Seam",
		Description:          "Extract prompt markdown assembly into domain service.",
		Priority:             "P2",
		Complexity:           "MEDIUM",
		RiskLevel:            "LOW",
		WorktreePath:         "/home/dev/src/owner/my-skills",
		TargetFiles:          []string{"apps/daemon/internal/domain/prompt/compiler.go", "apps/daemon/internal/domain/prompt/compiler_test.go"},
		AcceptanceCriteria:   []string{"prompt.Compiler encapsulates formatting", "Store delegates compilation"},
		VerificationCommands: []string{"go test -v ./internal/domain/prompt/..."},
	}

	upstreamDiffs := []string{
		"diff --git a/file1.go b/file1.go\n+func Hello() {}\n",
	}

	inputAttachments := []domain.Artifact{
		{
			ID:       "art-01",
			TaskID:   "P-03",
			Type:     domain.ArtifactTypeDiff,
			FilePath: "/tmp/diff1.patch",
		},
	}

	specSnippets := []string{
		"### Spec `spec-flow`: Flow Diagram (SEQUENCE)\n```mermaid\nsequenceDiagram\nAgent->>Engine: Claim\n```\n",
	}

	packet, err := compiler.Compile(ctx, task, upstreamDiffs, inputAttachments, specSnippets)
	if err != nil {
		t.Fatalf("unexpected error compiling context packet: %v", err)
	}

	if packet == nil {
		t.Fatal("expected non-nil context packet")
	}

	// Verify ContextPacket properties
	if packet.Task.ID != "P-03" {
		t.Errorf("expected task ID 'P-03', got '%s'", packet.Task.ID)
	}
	if len(packet.TargetFiles) != 2 {
		t.Errorf("expected 2 target files, got %d", len(packet.TargetFiles))
	}
	if len(packet.VerificationCommands) != 1 {
		t.Errorf("expected 1 verification command, got %d", len(packet.VerificationCommands))
	}
	if len(packet.UpstreamDiffs) != 1 {
		t.Errorf("expected 1 upstream diff, got %d", len(packet.UpstreamDiffs))
	}
	if len(packet.InputAttachments) != 1 {
		t.Errorf("expected 1 input attachment, got %d", len(packet.InputAttachments))
	}

	md := packet.PromptMarkdown

	// 1. Task header and metadata
	if !strings.Contains(md, "# Task Assignment: Prompt Context Compiler Domain Seam (P-03)") {
		t.Error("prompt markdown missing task assignment header")
	}
	if !strings.Contains(md, "- **Plan**: `plan-engine-v1`") {
		t.Error("prompt markdown missing plan metadata")
	}
	if !strings.Contains(md, "- **Priority**: `P2` | **Complexity**: `MEDIUM` | **Risk**: `LOW`") {
		t.Error("prompt markdown missing priority/complexity/risk metadata")
	}
	if !strings.Contains(md, "- **Worktree Path**: `/home/dev/src/owner/my-skills`") {
		t.Error("prompt markdown missing worktree path")
	}

	// 2. Description
	if !strings.Contains(md, "## 1. Description & Context") {
		t.Error("prompt markdown missing description header")
	}
	if !strings.Contains(md, "Extract prompt markdown assembly into domain service.") {
		t.Error("prompt markdown missing description body")
	}

	// 3. Spec snippets
	if !strings.Contains(md, "## 1.5. Linked Architectural Blueprints & Contracts (SSOT)") {
		t.Error("prompt markdown missing spec snippets section")
	}
	if !strings.Contains(md, "```mermaid\nsequenceDiagram\nAgent->>Engine: Claim\n```") {
		t.Error("prompt markdown missing mermaid fence from spec snippet")
	}

	// 4. Acceptance Criteria
	if !strings.Contains(md, "## 2. Acceptance Criteria") {
		t.Error("prompt markdown missing acceptance criteria section")
	}
	if !strings.Contains(md, "- [ ] prompt.Compiler encapsulates formatting") {
		t.Error("prompt markdown missing acceptance criterion 1")
	}
	if !strings.Contains(md, "- [ ] Store delegates compilation") {
		t.Error("prompt markdown missing acceptance criterion 2")
	}

	// 5. Target Files
	if !strings.Contains(md, "## 3. Target Files") {
		t.Error("prompt markdown missing target files section")
	}
	if !strings.Contains(md, "- `apps/daemon/internal/domain/prompt/compiler.go`") {
		t.Error("prompt markdown missing target file 1")
	}

	// 6. Verification Commands
	if !strings.Contains(md, "## 4. Verification Commands") {
		t.Error("prompt markdown missing verification commands section")
	}
	if !strings.Contains(md, "```bash\ngo test -v ./internal/domain/prompt/...\n```") {
		t.Error("prompt markdown missing verification commands bash code fence")
	}

	// 7. Upstream Diffs
	if !strings.Contains(md, "## Upstream Diffs") {
		t.Error("prompt markdown missing upstream diffs section")
	}
	if !strings.Contains(md, "```diff\ndiff --git a/file1.go b/file1.go\n+func Hello() {}\n```") {
		t.Error("prompt markdown missing upstream diff code fence")
	}

	// 8. MCP Tool Handshake Protocol
	if !strings.Contains(md, "## 5. MCP Tool Handshake Protocol") {
		t.Error("prompt markdown missing tool handshake protocol section")
	}
	if !strings.Contains(md, "zenfd_claim_task(task_id=\"P-03\"") {
		t.Error("prompt markdown missing claim task tool instruction")
	}
	if !strings.Contains(md, "zenfd_yield_review(task_id=\"P-03\"") {
		t.Error("prompt markdown missing yield review tool instruction")
	}
	if !strings.Contains(md, "zenfd_stage_merge(task_id=\"P-03\"") {
		t.Error("prompt markdown missing stage merge tool instruction")
	}
}

func TestMarkdownPromptCompiler_Compile_MinimalAndDefaults(t *testing.T) {
	compiler := prompt.NewCompiler()
	ctx := context.Background()

	task := domain.Task{
		ID:     "T-01",
		PlanID: "minimal-plan",
		Title:  "Minimal Task",
	}

	packet, err := compiler.Compile(ctx, task, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if packet == nil {
		t.Fatal("expected non-nil packet")
	}
	if len(packet.TargetFiles) != 0 {
		t.Errorf("expected empty target files, got %d", len(packet.TargetFiles))
	}
	if len(packet.VerificationCommands) != 0 {
		t.Errorf("expected empty verification commands, got %d", len(packet.VerificationCommands))
	}
	if len(packet.UpstreamDiffs) != 0 {
		t.Errorf("expected empty upstream diffs, got %d", len(packet.UpstreamDiffs))
	}
	if len(packet.InputAttachments) != 0 {
		t.Errorf("expected empty input attachments, got %d", len(packet.InputAttachments))
	}

	// Fallback description
	if !strings.Contains(packet.PromptMarkdown, "Execute the implementation according to plan specifications.") {
		t.Error("expected fallback description when task description is empty")
	}

	// When empty, optional sections shouldn't be present
	if strings.Contains(packet.PromptMarkdown, "## 1.5. Linked Architectural Blueprints") {
		t.Error("did not expect spec section when specSnippets is nil")
	}
	if strings.Contains(packet.PromptMarkdown, "## 2. Acceptance Criteria") {
		t.Error("did not expect acceptance criteria section when empty")
	}
	if strings.Contains(packet.PromptMarkdown, "## 3. Target Files") {
		t.Error("did not expect target files section when empty")
	}
	if strings.Contains(packet.PromptMarkdown, "## 4. Verification Commands") {
		t.Error("did not expect verification commands section when empty")
	}
	if strings.Contains(packet.PromptMarkdown, "## Upstream Diffs") {
		t.Error("did not expect upstream diffs section when nil")
	}
}
