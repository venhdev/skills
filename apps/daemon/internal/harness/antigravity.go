package harness

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"zenfd/internal/domain"
)

// AntigravityAdapter executes tasks via the Google Antigravity CLI (agy).
type AntigravityAdapter struct {
	binaryPath string
}

// NewAntigravityAdapter constructs an AntigravityAdapter, resolving binary location.
func NewAntigravityAdapter() *AntigravityAdapter {
	path := resolveAgyBinary()
	return &AntigravityAdapter{binaryPath: path}
}

func resolveAgyBinary() string {
	// 1. Check PATH
	if p, err := exec.LookPath("agy"); err == nil {
		return p
	}

	// 2. Check standard user local bin
	if home, err := os.UserHomeDir(); err == nil {
		localBin := filepath.Join(home, ".local", "bin", "agy")
		if info, err := os.Stat(localBin); err == nil && !info.IsDir() {
			return localBin
		}
	}

	return ""
}

func (a *AntigravityAdapter) Name() string {
	return "antigravity"
}

func (a *AntigravityAdapter) IsAvailable() bool {
	return a.binaryPath != ""
}

func (a *AntigravityAdapter) BuildExecution(task domain.Task) (ExecutionSpec, error) {
	if !a.IsAvailable() {
		return ExecutionSpec{}, errors.New("antigravity CLI (agy) is not available on host")
	}

	cmdStr := strings.TrimSpace(task.SkillCommand)
	if cmdStr == "" {
		return ExecutionSpec{}, errors.New("empty command for antigravity harness")
	}

	// If command is already full 'agy ...' invocation, pass arguments through
	if strings.HasPrefix(cmdStr, "agy ") {
		fields := strings.Fields(cmdStr)
		return ExecutionSpec{
			Binary:     a.binaryPath,
			Args:       fields[1:],
			IsExternal: false,
		}, nil
	}

	// For slash commands (e.g. /forge task-01) or prompt brief
	return ExecutionSpec{
		Binary:     a.binaryPath,
		Args:       []string{"-p", cmdStr},
		IsExternal: false,
	}, nil
}
