package harness

import (
	"errors"
	"strings"

	"zenfd/internal/domain"
)

// ShellAdapter executes tasks natively via the host POSIX shell (bash).
type ShellAdapter struct{}

// NewShellAdapter constructs a new ShellAdapter.
func NewShellAdapter() *ShellAdapter {
	return &ShellAdapter{}
}

func (s *ShellAdapter) Name() string {
	return "shell"
}

func (s *ShellAdapter) IsAvailable() bool {
	return true
}

func (s *ShellAdapter) BuildExecution(task domain.Task) (ExecutionSpec, error) {
	cmdStr := strings.TrimSpace(task.SkillCommand)
	if cmdStr == "" {
		return ExecutionSpec{}, errors.New("empty shell command")
	}

	return ExecutionSpec{
		Binary:     "bash",
		Args:       []string{"-c", cmdStr},
		IsExternal: false,
	}, nil
}
