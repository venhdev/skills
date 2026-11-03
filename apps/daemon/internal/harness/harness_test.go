package harness_test

import (
	"strings"
	"testing"

	"zenfd/internal/domain"
	"zenfd/internal/harness"
)

func TestHarnessRegistry_ResolveShell(t *testing.T) {
	reg := harness.DefaultRegistry()

	task := domain.Task{
		ID:           "T-01",
		Harness:      "shell",
		SkillCommand: "make test",
	}

	adapter, err := reg.Resolve(task)
	if err != nil {
		t.Fatalf("failed to resolve shell adapter: %v", err)
	}

	if adapter.Name() != "shell" {
		t.Errorf("expected adapter name 'shell', got '%s'", adapter.Name())
	}
	if !adapter.IsAvailable() {
		t.Errorf("expected shell adapter to be available")
	}

	spec, err := adapter.BuildExecution(task)
	if err != nil {
		t.Fatalf("BuildExecution failed: %v", err)
	}
	if spec.Binary != "bash" || len(spec.Args) != 2 || spec.Args[1] != "make test" {
		t.Errorf("unexpected spec: %+v", spec)
	}
	if spec.IsExternal {
		t.Errorf("expected IsExternal to be false for shell")
	}
}

func TestHarnessRegistry_ResolveAntigravity(t *testing.T) {
	reg := harness.DefaultRegistry()

	task := domain.Task{
		ID:           "T-02",
		Harness:      "antigravity",
		SkillCommand: "/forge T-02",
	}

	adapter, err := reg.Resolve(task)
	if err != nil {
		t.Fatalf("failed to resolve antigravity adapter: %v", err)
	}

	if adapter.Name() != "antigravity" {
		t.Errorf("expected adapter name 'antigravity', got '%s'", adapter.Name())
	}

	if adapter.IsAvailable() {
		spec, err := adapter.BuildExecution(task)
		if err != nil {
			t.Fatalf("BuildExecution failed: %v", err)
		}
		if !strings.Contains(spec.Binary, "agy") {
			t.Errorf("expected binary to contain 'agy', got %s", spec.Binary)
		}
		if len(spec.Args) != 2 || spec.Args[0] != "-p" || spec.Args[1] != "/forge T-02" {
			t.Errorf("unexpected args: %+v", spec.Args)
		}
	}
}

func TestHarnessRegistry_ResolveClaudeCode(t *testing.T) {
	reg := harness.DefaultRegistry()

	task := domain.Task{
		ID:           "T-03",
		Harness:      "claude-code",
		SkillCommand: "implement feature X",
	}

	adapter, err := reg.Resolve(task)
	if err != nil {
		t.Fatalf("failed to resolve claude-code adapter: %v", err)
	}

	if adapter.Name() != "claude-code" {
		t.Errorf("expected adapter name 'claude-code', got '%s'", adapter.Name())
	}

	if adapter.IsAvailable() {
		spec, err := adapter.BuildExecution(task)
		if err != nil {
			t.Fatalf("BuildExecution failed: %v", err)
		}
		if !strings.Contains(spec.Binary, "claude") {
			t.Errorf("expected binary to contain 'claude', got %s", spec.Binary)
		}
		if len(spec.Args) != 2 || spec.Args[0] != "-p" {
			t.Errorf("unexpected args: %+v", spec.Args)
		}
	}
}

func TestHarnessRegistry_ResolveOrchestrator(t *testing.T) {
	reg := harness.DefaultRegistry()

	task := domain.Task{
		ID:           "T-04",
		Harness:      "orchestrator",
		SkillCommand: "/manual review",
	}

	adapter, err := reg.Resolve(task)
	if err != nil {
		t.Fatalf("failed to resolve orchestrator adapter: %v", err)
	}

	if adapter.Name() != "orchestrator" {
		t.Errorf("expected adapter name 'orchestrator', got '%s'", adapter.Name())
	}
	if adapter.IsAvailable() {
		t.Errorf("expected orchestrator IsAvailable to be false for local OS execution")
	}

	spec, err := adapter.BuildExecution(task)
	if err != nil {
		t.Fatalf("BuildExecution failed: %v", err)
	}
	if !spec.IsExternal {
		t.Errorf("expected IsExternal to be true for orchestrator")
	}
}
