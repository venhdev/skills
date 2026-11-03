package harness

import (
	"fmt"
	"strings"

	"zenfd/internal/domain"
)

// ExecutionSpec defines the resolved OS binary, arguments, and environment to run a task.
type ExecutionSpec struct {
	Binary       string            `json:"binary"`
	Args         []string          `json:"args"`
	EnvOverrides map[string]string `json:"env_overrides,omitempty"`
	IsExternal   bool              `json:"is_external"` // True if execution is delegated to external AI agent (no OS process)
}

// Adapter translates a domain Task into an OS-executable specification or orchestrator delegate.
type Adapter interface {
	Name() string
	IsAvailable() bool
	BuildExecution(task domain.Task) (ExecutionSpec, error)
}

// Registry manages the set of available execution harnesses.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry constructs a Registry pre-loaded with default adapters.
func NewRegistry(adapters ...Adapter) *Registry {
	r := &Registry{
		adapters: make(map[string]Adapter),
	}
	for _, a := range adapters {
		r.Register(a)
	}
	return r
}

// DefaultRegistry creates a registry with all standard ZenForge adapters.
func DefaultRegistry() *Registry {
	return NewRegistry(
		NewShellAdapter(),
		NewAntigravityAdapter(),
		NewClaudeCodeAdapter(),
		NewOrchestratorAdapter(),
	)
}

// Register adds an adapter to the registry.
func (r *Registry) Register(a Adapter) {
	if a != nil {
		r.adapters[strings.ToLower(a.Name())] = a
	}
}

// Get retrieves an adapter by harness name.
func (r *Registry) Get(name string) (Adapter, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	a, ok := r.adapters[name]
	return a, ok
}

// Resolve selects the most appropriate adapter for a task based on task.Harness and task.SkillCommand.
func (r *Registry) Resolve(task domain.Task) (Adapter, error) {
	harnessName := strings.ToLower(strings.TrimSpace(task.Harness))

	// 1. Explicit Harness matching
	if harnessName != "" {
		if a, ok := r.adapters[harnessName]; ok {
			return a, nil
		}
		// Aliases
		switch harnessName {
		case "sh", "bash", "shell-script":
			if a, ok := r.adapters["shell"]; ok {
				return a, nil
			}
		case "agy", "antigravity-cli":
			if a, ok := r.adapters["antigravity"]; ok {
				return a, nil
			}
		case "claude", "claude_code":
			if a, ok := r.adapters["claude-code"]; ok {
				return a, nil
			}
		case "manual", "external", "agent", "subagent":
			if a, ok := r.adapters["orchestrator"]; ok {
				return a, nil
			}
		}
	}

	// 2. Command-based heuristic resolution
	cmd := strings.TrimSpace(task.SkillCommand)

	// If command explicitly invokes agy or claude CLI directly
	if strings.HasPrefix(cmd, "agy ") {
		if a, ok := r.adapters["antigravity"]; ok {
			return a, nil
		}
	}
	if strings.HasPrefix(cmd, "claude ") {
		if a, ok := r.adapters["claude-code"]; ok {
			return a, nil
		}
	}

	// If command starts with an AI slash command (e.g. /forge, /spike, /clarify)
	if strings.HasPrefix(cmd, "/") {
		// Prefer antigravity adapter if available on host
		if a, ok := r.adapters["antigravity"]; ok && a.IsAvailable() {
			return a, nil
		}
		// Fallback to external orchestrator delegation (zero crash)
		if a, ok := r.adapters["orchestrator"]; ok {
			return a, nil
		}
	}

	// 3. Fallback to default shell adapter
	if a, ok := r.adapters["shell"]; ok {
		return a, nil
	}

	return nil, fmt.Errorf("no adapter registered for harness '%s'", task.Harness)
}
