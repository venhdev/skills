# WIP Tracker

A skill for managing work-in-progress context and keeping agents aligned on active work.

## What It Does

Maintains `.wip/context.json` as the **Single Source of Truth (SSOT)** for all active work. JSON enables fast O(1) field access — agents parse it natively without regex.

## File Structure

```
<project-root>/
└── .wip/
    ├── context.json    ← SSOT for agents (read/write here)
    └── context.md     ← Human display (only on demand)
```

## When to Use

Trigger this skill when:

- **Starting a new task** → Mark it `in_progress` with current action
- **Completing a task** → Move to `done`, start next from `open`
- **Discovering a new bug** → Add to `bugs.open`
- **Fixing a bug** → Move to `bugs.fixed` with fix details
- **Blocked on dependency** → Move to `blocked` with reason
- **Archiving completed work** → Move plan docs to archive folder

## Trigger Examples

> "I'm working on bug #47"
> "This task is done, what's next?"
> "Found a new blocker in the Docker build"
> "Mark task #48 as done and start #49"
> "Archive the completed plan doc"

## Key Principle

**Update `.wip/context.json` IMMEDIATELY when state changes.** This keeps all agents knowing what they're doing.

## Quick Reference

| Field | Values |
|-------|--------|
| `session.status` | `active`, `blocked`, `done` |
| `tasks` | `in_progress`, `done`, `blocked`, `open` |
| `bugs.severity` | `BLOCKER`, `HIGH`, `MEDIUM`, `LOW`, `DOCS` |
| `bugs.status` | `discovered`, `in_progress`, `fixed` |

## Bug Severity Levels

| Severity | Meaning |
|----------|---------|
| BLOCKER | Prevents deployment/start |
| HIGH | Major functionality broken |
| MEDIUM | Minor functionality broken |
| LOW | Cosmetic issue |
| DOCS | Documentation only |
