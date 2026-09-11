---
name: to-tasks
description: "Decompose approved specifications, architectural plans, or changesets into dependency-sequenced vertical slice tasks before implementation."
disable-model-invocation: true
---

# to-tasks — Task Decomposition & Slicing Engine

Decompose architectural plans, specifications, and changesets into dependency-sequenced, tracer-bullet vertical slice tasks with embedded file changesets.

## Operating Invariants

- **Task Scope Discipline**: Generate and publish strictly task definitions and tracking metadata; forbid modifying codebase implementation files or tests.
- **Tracker Mutex**: Publish strictly to the tracker defined in `.agents/task-tracker.md`; never publish to both local and remote unless explicitly requested.
- **Vertical Slicing Discipline**: Deliver end-to-end vertical capability per task; forbid horizontal layer-only separation except wide refactors.
- **Pre-Mutation Gate**: Stage proposed tasks in high-density summary format and halt turn immediately; forbid writing task files to disk or remote trackers without affirmative human approval.

## Domain Engine & Standards

### 1. Vertical Slice Criteria

- **Narrow & Complete**: Cut end-to-end across schema, logic, and interface layers for a single capability.
- **Independently Verifiable**: Each task must pass automated verification in isolation.
- **Context-Sized**: Bound task scope so implementation and verification complete within a single fresh context.
- **Prefactor First**: When existing architecture resists new capabilities, dedicate Task 01 to preparatory refactoring.
- **Wide Refactor (Expand-Contract)**: For sweeping changes crossing many call sites:
  1. *Expand*: Add new interface alongside the old.
  2. *Migrate*: Shift callers over in dependency-bounded batches.
  3. *Contract*: Remove obsolete interface once all callers migrate.

### 2. Dependency & Task States

- **`Blocked by`**: Declare explicit predecessor task IDs (`None` designates the starting frontier).
- **Task States**:
  - `ready`: Default initial state for all tasks. Actionable when unblocked per Dynamic Frontier Rule.
  - `in-progress`: Claimed by an agent or developer during active implementation.
  - `done`: Implementation complete, all acceptance criteria verified `[x]`.
- **Dynamic Frontier Rule**: A task is unblocked and eligible for `/forge` strictly when its `Status` is `ready` AND (`Blocked by: None` OR every task listed in `Blocked by` has `Status: done`). Never write a static "blocked" status to disk.

### 3. Namespace & Storage Partitioning

- **Local Markdown Partitioning**: Tasks are partitioned per feature in `.agents/tasks/<feature-slug>/<NN>-<slug>.md`.
  - `<feature-slug>` is automatically derived by the agent from the spec name, feature title, or branch context.
  - Tasks are numbered sequentially starting from `01` in dependency order.
- **Remote Trackers**: Tasks are published as issues to the remote repository defined in `.agents/task-tracker.md`, using native issue dependencies or header `Blocked by: #<n>` links.

### 4. Canonical Task Template

````markdown
# <NN>: <Task Title>

**Status**: ready
**Blocked by**: None | <NN> (<Title>)

## What to Build
<Clear description of end-to-end behavior delivered from user/system perspective>

## Acceptance Criteria
- [ ] <Verifiable criterion 1>
- [ ] <Verifiable criterion 2>

## Embedded Changeset
> Notice: Planned blast radius at decomposition time. Reconcile against current working tree before mutating files.

```text
📁 <directory>/
└── 📄 <target_file>
    └── [<ACTION>] <File mutation summary>
```

*(Note: Restrict embedded code snippets exclusively to binding schemas, interface contracts, or state machine transitions; forbid pasting volatile implementation code).*
````

## Execution Protocol

**SUB-SKILL:** changeset, clarify, ssot, zenforge-init

### Phase 1: Ingestion & Slice Decomposition

1. Ingest requirements from input context (conversation, specifications under `docs/` or governed via `/ssot`, tickets, or staged changesets).
2. Derive `<feature-slug>` from context or specification.
3. Decompose requirements into sequential tracer-bullet vertical slices with acyclic blocking edges (`Blocked by`).
4. Partition the global Changeset into per-task Embedded Changesets, sequencing by dependency edges.

### Phase 2: Staging & Authorization Gate

1. Present the task breakdown and derived `<feature-slug>` in Lean Delivery summary format:

   ```text
   # Tasks Staging: <feature-slug>

   | Task ID | Title | Blocked by | Frontier Gate | Key Deliverable |
   | :--- | :--- | :--- | :--- | :--- |
   | **01** | <Title 1> | None | `immediate` | <One-line capability> |
   | **02** | <Title 2> | 01 | `waiting on 01` | <One-line capability> |
   ```

2. Halt turn immediately for user confirmation; forbid writing files or publishing tasks without approval.

### Phase 3: Tracker Resolution & Publication

1. Read `.agents/task-tracker.md`. If absent:
   - If `/zenforge-init` is available: prompt user to choose `[1] Full Init (run /zenforge-init)` or `[2] Quick Setup (seed local markdown tracker)`. Halt turn for response.
   - If `/zenforge-init` is unavailable: automatically bootstrap `.agents/task-tracker.md` from `references/issue-tracker-local.md` and proceed to the next step.
2. Publish tasks to the designated tracker:
   - `local-markdown`: Write `.agents/tasks/<feature-slug>/<NN>-<slug>.md` in dependency order.
   - `github`: Publish issues via `gh issue create` and link dependencies.
   - `gitlab`: Publish issues via `glab issue create` and link dependencies.
3. Report the active frontier (first `ready` task) and hand off to `/forge`:
   - `local-markdown`: `/forge .agents/tasks/<feature-slug>/01-<slug>.md`
   - `github` | `gitlab`: `/forge #<ID>`
