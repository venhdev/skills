package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// WorktreeManager abstracts Git worktree and branch merging operations.
type WorktreeManager interface {
	MergeWorktree(ctx context.Context, repoRoot, worktreePath string) (merged bool, conflicts []string, err error)
	ResolveBranchName(ctx context.Context, worktreePath string) (string, error)
}

// LocalWorktreeManager implements WorktreeManager by executing local git CLI commands.
type LocalWorktreeManager struct{}

// NewLocalWorktreeManager constructs a new LocalWorktreeManager.
func NewLocalWorktreeManager() *LocalWorktreeManager {
	return &LocalWorktreeManager{}
}

// ResolveBranchName resolves the current branch name checked out at worktreePath.
func (m *LocalWorktreeManager) ResolveBranchName(ctx context.Context, worktreePath string) (string, error) {
	if strings.TrimSpace(worktreePath) == "" {
		return "", errors.New("worktreePath cannot be empty")
	}

	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve branch name in %s: %w", worktreePath, err)
	}

	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("detached HEAD or empty branch name in %s", worktreePath)
	}

	return branch, nil
}

// MergeWorktree merges the worktree branch into repoRoot.
// On clean merge, it creates a merge commit and returns (true, nil, nil).
// On merge conflict, it aborts the merge and returns (false, conflicts, nil).
func (m *LocalWorktreeManager) MergeWorktree(ctx context.Context, repoRoot, worktreePath string) (bool, []string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return false, nil, errors.New("repoRoot cannot be empty")
	}

	branch, err := m.ResolveBranchName(ctx, worktreePath)
	if err != nil {
		return false, nil, err
	}

	msg := fmt.Sprintf("zenfd: merge branch %s", branch)
	mergeCmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "merge", "--no-ff", "-m", msg, branch)
	mergeOut, err := mergeCmd.CombinedOutput()
	if err != nil {
		// Merge conflict or error encountered
		conflictCmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "diff", "--name-only", "--diff-filter=U")
		confOut, diffErr := conflictCmd.Output()

		// Abort merge to ensure repoRoot is left in a clean state
		abortCmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "merge", "--abort")
		_ = abortCmd.Run()

		var conflictFiles []string
		if diffErr == nil {
			for _, line := range strings.Split(string(confOut), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					conflictFiles = append(conflictFiles, line)
				}
			}
		}

		if len(conflictFiles) == 0 {
			conflictFiles = []string{strings.TrimSpace(string(mergeOut))}
		}

		return false, conflictFiles, nil
	}

	return true, nil, nil
}
