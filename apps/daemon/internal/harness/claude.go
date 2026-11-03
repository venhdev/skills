package harness

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"zenfd/internal/domain"
)

// ClaudeCodeAdapter executes tasks via the Claude Code CLI.
type ClaudeCodeAdapter struct {
	binaryPath string
}

// NewClaudeCodeAdapter constructs a ClaudeCodeAdapter, resolving binary location.
func NewClaudeCodeAdapter() *ClaudeCodeAdapter {
	path := resolveClaudeBinary()
	return &ClaudeCodeAdapter{binaryPath: path}
}

func resolveClaudeBinary() string {
	// 1. Check PATH
	if p, err := exec.LookPath("claude"); err == nil {
		return p
	}

	// 2. Check standard user local bin
	if home, err := os.UserHomeDir(); err == nil {
		localBin := filepath.Join(home, ".local", "bin", "claude")
		if info, err := os.Stat(localBin); err == nil && !info.IsDir() {
			return localBin
		}
	}

	return ""
}

func (c *ClaudeCodeAdapter) Name() string {
	return "claude-code"
}

func (c *ClaudeCodeAdapter) IsAvailable() bool {
	return c.binaryPath != ""
}

func (c *ClaudeCodeAdapter) BuildExecution(task domain.Task) (ExecutionSpec, error) {
	if !c.IsAvailable() {
		return ExecutionSpec{}, errors.New("claude CLI is not available on host")
	}

	cmdStr := strings.TrimSpace(task.SkillCommand)
	if cmdStr == "" {
		return ExecutionSpec{}, errors.New("empty command for claude-code harness")
	}

	// If command is already full 'claude ...' invocation
	if strings.HasPrefix(cmdStr, "claude ") {
		fields := strings.Fields(cmdStr)
		return ExecutionSpec{
			Binary:     c.binaryPath,
			Args:       fields[1:],
			IsExternal: false,
		}, nil
	}

	// Wrap in print/prompt mode for headless execution
	return ExecutionSpec{
		Binary:     c.binaryPath,
		Args:       []string{"-p", cmdStr},
		IsExternal: false,
	}, nil
}
