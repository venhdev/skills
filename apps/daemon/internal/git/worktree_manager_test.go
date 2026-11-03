package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zenfd/internal/git"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %s (err: %v)", args, dir, string(out), err)
	}
	return strings.TrimSpace(string(out))
}

func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()
	repoDir := t.TempDir()

	// Initialize git repo with main as default branch
	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.name", "Zenfd Tester")
	runGit(t, repoDir, "config", "user.email", "test@zenfd.local")
	runGit(t, repoDir, "config", "commit.gpgSign", "false")

	// Create initial commit
	initialFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Repo\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "initial commit")

	cleanup := func() {
		// git worktree prune if needed
		_ = exec.Command("git", "-C", repoDir, "worktree", "prune").Run()
	}

	return repoDir, cleanup
}

func TestLocalWorktreeAdapter_ResolveBranchName(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	mgr := git.NewLocalWorktreeAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test 1: Resolve branch from main repo
	branch, err := mgr.ResolveBranchName(ctx, repoDir)
	if err != nil {
		t.Fatalf("expected successful branch resolution, got: %v", err)
	}
	if branch != "main" {
		t.Fatalf("expected branch 'main', got '%s'", branch)
	}

	// Test 2: Resolve branch from a worktree
	worktreeDir := filepath.Join(t.TempDir(), "wt-test")
	runGit(t, repoDir, "worktree", "add", "-b", "feat-resolve", worktreeDir)

	branch, err = mgr.ResolveBranchName(ctx, worktreeDir)
	if err != nil {
		t.Fatalf("expected successful branch resolution for worktree, got: %v", err)
	}
	if branch != "feat-resolve" {
		t.Fatalf("expected branch 'feat-resolve', got '%s'", branch)
	}

	// Test 3: Empty worktree path
	_, err = mgr.ResolveBranchName(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty worktree path, got nil")
	}

	// Test 4: Nonexistent directory
	_, err = mgr.ResolveBranchName(ctx, filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for nonexistent directory, got nil")
	}
}

func TestLocalWorktreeAdapter_MergeWorktree_Clean(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	mgr := git.NewLocalWorktreeAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create worktree on a feature branch
	worktreeDir := filepath.Join(t.TempDir(), "wt-clean")
	runGit(t, repoDir, "worktree", "add", "-b", "feat-clean", worktreeDir)

	// Make a commit in the worktree
	featureFile := filepath.Join(worktreeDir, "feature.txt")
	if err := os.WriteFile(featureFile, []byte("feature content\n"), 0o644); err != nil {
		t.Fatalf("failed to write feature file: %v", err)
	}
	runGit(t, worktreeDir, "add", "feature.txt")
	runGit(t, worktreeDir, "commit", "-m", "add feature.txt")

	// Merge worktree into main repo
	merged, conflicts, err := mgr.MergeWorktree(ctx, repoDir, worktreeDir)
	if err != nil {
		t.Fatalf("MergeWorktree returned unexpected error: %v", err)
	}
	if !merged {
		t.Fatalf("expected merged == true, got false with conflicts: %v", conflicts)
	}
	if len(conflicts) > 0 {
		t.Fatalf("expected no conflicts, got: %v", conflicts)
	}

	// Verify merged file exists in main repo
	mergedPath := filepath.Join(repoDir, "feature.txt")
	data, err := os.ReadFile(mergedPath)
	if err != nil {
		t.Fatalf("expected feature.txt to exist in repoDir: %v", err)
	}
	if string(data) != "feature content\n" {
		t.Fatalf("unexpected content in merged file: %s", string(data))
	}
}

func TestLocalWorktreeAdapter_MergeWorktree_Conflict(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	mgr := git.NewLocalWorktreeAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a shared file on main
	sharedFile := filepath.Join(repoDir, "shared.txt")
	if err := os.WriteFile(sharedFile, []byte("line 1\nbase content\nline 3\n"), 0o644); err != nil {
		t.Fatalf("failed to write shared file: %v", err)
	}
	runGit(t, repoDir, "add", "shared.txt")
	runGit(t, repoDir, "commit", "-m", "add shared.txt")

	// Create worktree branching off main
	worktreeDir := filepath.Join(t.TempDir(), "wt-conflict")
	runGit(t, repoDir, "worktree", "add", "-b", "feat-conflict", worktreeDir)

	// Modify shared.txt in worktree
	wtShared := filepath.Join(worktreeDir, "shared.txt")
	if err := os.WriteFile(wtShared, []byte("line 1\nworktree divergent modification\nline 3\n"), 0o644); err != nil {
		t.Fatalf("failed to write wt shared file: %v", err)
	}
	runGit(t, worktreeDir, "add", "shared.txt")
	runGit(t, worktreeDir, "commit", "-m", "worktree change to shared.txt")

	// Modify shared.txt on main concurrently
	if err := os.WriteFile(sharedFile, []byte("line 1\nmain concurrent modification\nline 3\n"), 0o644); err != nil {
		t.Fatalf("failed to write main shared file: %v", err)
	}
	runGit(t, repoDir, "add", "shared.txt")
	runGit(t, repoDir, "commit", "-m", "main change to shared.txt")

	// Merge worktree into main repo (should trigger conflict)
	merged, conflicts, err := mgr.MergeWorktree(ctx, repoDir, worktreeDir)
	if err != nil {
		t.Fatalf("expected nil error on conflict, got: %v", err)
	}
	if merged {
		t.Fatal("expected merged == false, got true")
	}
	if len(conflicts) == 0 {
		t.Fatal("expected conflict list to be non-empty")
	}

	foundShared := false
	for _, c := range conflicts {
		if strings.Contains(c, "shared.txt") {
			foundShared = true
			break
		}
	}
	if !foundShared {
		t.Fatalf("expected shared.txt in conflicts, got: %v", conflicts)
	}

	// Verify repoRoot was cleanly aborted (no MERGE_HEAD)
	mergeHeadPath := filepath.Join(repoDir, ".git", "MERGE_HEAD")
	if _, err := os.Stat(mergeHeadPath); !os.IsNotExist(err) {
		t.Fatalf("MERGE_HEAD should not exist after abort: %v", err)
	}

	// Main repo shared.txt should still have the main commit's content
	data, err := os.ReadFile(sharedFile)
	if err != nil {
		t.Fatalf("failed to read shared.txt: %v", err)
	}
	if string(data) != "line 1\nmain concurrent modification\nline 3\n" {
		t.Fatalf("unexpected content after merge abort: %s", string(data))
	}
}

func TestLocalWorktreeAdapter_MergeWorktree_Errors(t *testing.T) {
	mgr := git.NewLocalWorktreeAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Empty repoRoot
	_, _, err := mgr.MergeWorktree(ctx, "", "/some/path")
	if err == nil {
		t.Fatal("expected error for empty repoRoot, got nil")
	}

	// Empty worktreePath
	_, _, err = mgr.MergeWorktree(ctx, "/some/path", "")
	if err == nil {
		t.Fatal("expected error for empty worktreePath, got nil")
	}
}
