---
name: wip-tracker
description: Use this skill whenever the user asks to update task progress, mark tasks as done, track bugs, or manage the work-in-progress tracker. Trigger when starting a new task, completing a task, archiving completed items, or updating status. Also trigger proactively when a task completes or fails — update the WIP tracker IMMEDIATELY so the agent always knows what's in progress. This skill maintains the SSOT for all active work.
---

# WIP Tracker

The `.wip/context.json` is the **Single Source of Truth (SSOT)** for all active work. JSON enables O(1) field access — agents parse it natively without regex or text manipulation.

## File Structure

```
<project-root>/
└── .wip/
    ├── context.json    ← SSOT for agents (read/write here)
    └── context.md     ← Only generated when user asks ("show as markdown")
```

## Core Principle: Update First, Always

**When a task completes or issue is discovered, update `.wip/context.json` IMMEDIATELY.**

---

## JSON Schema

```json
{
  "session": {
    "id": "uuid",
    "status": "active | blocked | done",
    "updated": "ISO-8601"
  },
  "tasks": {
    "done": [
      { "id": "#47", "desc": "Rebuild Docker core", "completed": "ISO-8601" }
    ],
    "in_progress": {
      "id": "#48",
      "desc": "Rebuild backend Docker",
      "action": "pnpm docker:build:be -- --no-cache"
    },
    "blocked": [
      { "id": "#49", "desc": "...", "reason": "Waiting on external dependency" }
    ],
    "open": [
      { "id": "#50", "desc": "Run tests" },
      { "id": "#51", "desc": "Deploy" }
    ]
  },
  "bugs": {
    "fixed": [
      {
        "id": 14,
        "severity": "BLOCKER",
        "service": "Docker",
        "symptom": "redocly: not found",
        "rootCause": "shamefully-hoist missing + rm -rf node_modules",
        "fix": "Removed rm -rf, use pnpm exec -- redocly",
        "fixed": "2026-04-13"
      }
    ],
    "open": [
      {
        "id": 15,
        "severity": "HIGH",
        "service": "auth",
        "symptom": "JWT validation fails",
        "status": "in_progress"
      }
    ]
  },
  "docker_images": {
    "memozen-auth": { "status": "built | building | failed", "notes": "..." }
  },
  "files_changed": [
    { "path": "backend/infra/docker/Dockerfile.shared", "status": "updated" }
  ],
  "archived_plans": [
    { "path": "docs/plan/PNPM-DEPENDENCY-ISSUES-AUDIT.md", "archived": "2026-04-13" }
  ]
}
```

---

## Workflows

### 1. Starting a Task

Move from `open` → `in_progress`:

```json
// BEFORE
"open": [{ "id": "#48", "desc": "Rebuild backend" }]

// AFTER
"in_progress": { "id": "#48", "desc": "Rebuild backend", "action": "pnpm docker:build:be -- --no-cache" }
```

### 2. Completing a Task

Move from `in_progress` → `done`, then start next task:

```json
// Move current to done
"done": [{ "id": "#48", "desc": "Rebuild backend", "completed": "2026-04-13T10:30:00Z" }]

// Start next from open
"in_progress": { "id": "#49", "desc": "Start backend", "action": "pnpm docker:up:be" }
```

### 3. Blocking a Task

Move from `in_progress` → `blocked` with reason:

```json
"blocked": [{ "id": "#48", "desc": "Rebuild backend", "reason": "Waiting on infra to be ready" }]
```

### 4. Adding a New Bug

Append to `bugs.open`:

```json
{
  "id": 15,
  "severity": "BLOCKER",
  "service": "payment",
  "symptom": "Cannot start container",
  "rootCause": "unknown",
  "status": "discovered"
}
```

### 5. Fixing a Bug

Move from `bugs.open` → `bugs.fixed`:

```json
// Remove from open
// Add to fixed with fix + date
{
  "id": 14,
  "severity": "BLOCKER",
  "service": "Docker",
  "symptom": "...",
  "rootCause": "...",
  "fix": "Applied fix description",
  "fixed": "2026-04-13"
}
```

### 6. Tracking File Changes

```json
"files_changed": [
  { "path": "backend/infra/docker/Dockerfile.shared", "status": "updated" },
  { "path": "packages/contracts/package.json", "status": "updated" }
]
```

### 7. Archiving a Plan Doc

1. Move file to `docs/plan/in-progress/archive/`
2. Add to `archived_plans`:

```json
"archived_plans": [
  { "path": "docs/plan/FIX-MISSING-ERROR-CODES-BUILD.md", "archived": "2026-04-13" }
]
```

---

## Status Values

| Field | Values |
|-------|--------|
| `session.status` | `active`, `blocked`, `done` |
| `task status` (implicit) | `in_progress`, `done`, `blocked`, `open` |
| `bug.severity` | `BLOCKER`, `HIGH`, `MEDIUM`, `LOW`, `DOCS` |
| `bug.status` | `discovered`, `in_progress`, `fixed` |
| `docker_images.*.status` | `pending`, `building`, `built`, `failed` |
| `files_changed.*.status` | `new`, `updated`, `deleted` |

---

## Quick Reference

### Common Mutations

**Start task #49:**
```json
"in_progress": { "id": "#49", "desc": "Run tests", "action": "pnpm test" }
```

**Complete current task:**
```json
"done": [{ "id": "#48", "desc": "Rebuild backend", "completed": "ISO-8601" }]
```

**Add new bug:**
```json
"bugs.open": [{ "id": 15, "severity": "BLOCKER", "service": "...", "symptom": "...", "status": "discovered" }]
```

**Mark bug fixed:**
```json
"bugs.fixed": [{ "id": 14, ..., "fix": "...", "fixed": "2026-04-13" }]
```

---

## Markdown Export (On Demand Only)

Only generate `.wip/context.md` when user explicitly asks for it:

```
## Session
Status: active | Updated: 2026-04-13

## Tasks
### In Progress
#48: Rebuild backend → pnpm docker:build:be

### Open
#49: Run tests
#50: Deploy

## Bugs
### Fixed
#14 [BLOCKER] Docker: redocly not found → ✅ Fixed

### Open
#15 [HIGH] auth: JWT validation fails → 🚧 In Progress
```

---

## File Paths

| File | Purpose |
|------|---------|
| `.wip/context.json` | SSOT — read/write here |
| `.wip/context.md` | Human display — only on demand |
| `docs/plan/in-progress/archive/` | Archived plan docs |

---

## Example: Complete Task Flow

**Start:**
```json
"open": [{ "id": "#48" }]
→ "in_progress": { "id": "#48", "desc": "Rebuild backend", "action": "pnpm docker:build:be" }
```

**Work:**
- Agent performs the action
- If new bug found → add to `bugs.open`
- If file changed → add to `files_changed`

**Complete:**
```json
"in_progress": null
"done": [{ "id": "#48", "desc": "Rebuild backend", "completed": "2026-04-13T10:30:00Z" }]

"open": [{ "id": "#49" }]
→ "in_progress": { "id": "#49", "desc": "Start backend", "action": "pnpm docker:up:be" }
```