---
name: to-tasks
description: "Decompose approved specifications, architectural plans, or changesets into dependency-sequenced vertical slice tasks before implementation."
disable-model-invocation: true
---

# to-tasks — Task Decomposition & Slicing Engine

Decompose architectural plans, specifications, and changesets into dependency-sequenced, tracer-bullet vertical slice tasks with embedded file changesets.

## Operating Invariants

- **Task Scope Discipline**: Generate and publish strictly task definitions and tracking metadata; forbid modifying codebase implementation files or tests.
- **Tracker Authority Grounding**: Resolve target configuration strictly from `.agents/task-tracker.md`; forbid guessing storage paths or publishing to unconfigured remotes.
- **Vertical Slicing Discipline**: Deliver end-to-end vertical capability per task; forbid horizontal layer-only separation except wide refactors.
- **Pre-Mutation Gate**: Stage proposed tasks in high-density summary format and halt turn immediately; forbid writing task files to disk or remote trackers without affirmative human approval.
- **Publication Circuit Breaker**: Stop execution immediately upon any CLI publication failure (non-zero exit code), retain already-published task IDs, report stderr with the failing task ID, and prompt user whether to retry or abort; forbid continuing silently on broken dependency chains.

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
  - `ready`: Default initial state for all tasks. Actionable once all predecessors in `Blocked by` are `done`.
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

**SUB-SKILL:** changeset, clarify, ssot

### Phase 1: Context & Tracker Discovery

1. Inspect input context (discussion, specs in `docs/specs/`, or staged changesets).
2. Derive `<feature-slug>` from context or specification.
3. Read `.agents/task-tracker.md` to resolve tracker type (`local-markdown`, `github`, `gitlab`) and bind **Execution Protocols**. If absent, halt and present an **Interactive Fork**:
   - **[1] Use Local Markdown Tracker Immediately (Fast-path)**: Automatically initialize `.agents/task-tracker.md` for local markdown, append `.agents/tasks/` to `.gitignore` to keep the repo clean, explicitly notify the user (*"Local tasks are gitignored by default to keep the repository clean. Remove from .gitignore if team tracking is desired."*), and proceed with task decomposition.
   - **[2] Configure Remote Tracker**: Direct user to run `/zenforge-init` for full repository onboarding and remote credential verification.

### Phase 2: Slice Decomposition & Changeset Partitioning

1. Decompose requirements into sequential tracer-bullet vertical slices.
2. Establish acyclic blocking edges (`Blocked by`).
3. Partition the global Changeset into per-task Embedded Changesets, sequencing by dependency edges (`Blocked by`).

### Phase 3: Staging & Authorization Gate

1. Present the task breakdown and derived `<feature-slug>` in Lean Delivery summary format:

   ```text
   # Tasks Staging: <feature-slug>

   | Task ID | Title | Blocked by | Frontier Gate | Key Deliverable |
   | :--- | :--- | :--- | :--- | :--- |
   | **01** | <Title 1> | None | `immediate` | <One-line capability> |
   | **02** | <Title 2> | 01 | `waiting on 01` | <One-line capability> |
   ```

2. Offer review options: approve publication, adjust granularity or `<feature-slug>`, or inspect individual changesets.
3. Forbid writing files or calling tracker APIs within this turn.
4. Halt turn immediately and wait for affirmative human authorization.

### Phase 4: Atomic Publication & Frontier Handoff

1. Upon receiving approval, execute task publication following `.agents/task-tracker.md`:
   - **Local Markdown**: Write one file per task as `.agents/tasks/<feature-slug>/<NN>-<slug>.md` in dependency order.
   - **Remote Tracker**: Call the execution protocol commands (e.g. `gh issue create`) and link issue dependencies.
2. **Publication Circuit Breaker**: If any CLI publication command fails (non-zero exit code):
   - Halt execution immediately.
   - Retain already-published task IDs and report the exact failure state.
   - Report `stderr` with the failing task ID, and prompt user whether to retry from the failing task or abort.
   - Forbid continuing publication on broken dependency chains.
3. Report the active frontier (first `ready` task) and hand off to `/forge`:
   - Local: `/forge .agents/tasks/<feature-slug>/01-<slug>.md`
   - Remote: `/forge #<ID>`
