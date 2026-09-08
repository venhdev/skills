---
name: worktree
description: "Create and delete isolated Git worktrees under .agents/worktree/ for concurrent agent sessions."
---

# worktree

Isolate concurrent agent sessions in dedicated Git worktrees to prevent workspace collisions.

## Conventions
- `<branch>`: `<type>/<slug>` (`feat/`, `fix/`, `refactor/`, `chore/`, `test/`).
- `<path>`: `.agents/worktree/<branch>`.
- Root `.gitignore` must contain `.agents/worktree/`.

## Rules
- Confine all edits, commands, and tests strictly within `<path>`. Never touch root files.
- Always confirm with the user before creating or deleting a worktree.
- Never delete `<path>` if `git status --porcelain` shows uncommitted changes.

## Workflow

### 1. Create
1. Check `git worktree list` for collisions.
2. Propose to user:
   ```text
   Worktree: <path> (<branch> from <base_ref>)
   ```
3. On approval, execute:
   ```bash
   git worktree add <path> -b <branch> <base_ref>
   ```

### 2. List
- Run `git worktree list` to show active worktrees, branches, and commits.

### 3. Delete
1. Check `git status --porcelain` inside `<path>`. If dirty, halt.
2. On approval, execute:
   ```bash
   git worktree remove <path>
   git worktree prune
   ```
