---
name: to-tasks
description: Use when breaking down specifications, plans, changesets, or feature requests into sequential execution tasks.
---

# to-tasks — Task Decomposition & Slicing Engine

Decompose architectural plans, specifications, and changesets into dependency-sequenced, tracer-bullet vertical slice tasks with embedded file changesets.

## Operating Invariants

- **Scope Discipline**: Generate and publish strictly task definitions and tracking metadata. Forbid modifying codebase implementation files or tests.
- **Tracker Authority Grounding**: Resolve target configuration strictly from `.agents/task-tracker.md`. Forbid guessing storage paths or publishing to unconfigured remotes.
- **Vertical Slicing Discipline**: Mandate end-to-end vertical capability per task; forbid horizontal layer-only separation except wide refactors.
- **Pre-Mutation Gate**: Stage proposed tasks in high-density summary format. Forbid writing task files to disk or remote trackers without affirmative human approval.

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
  - `ready`: Blocker is `None`, or all blockers are `done`.
  - `blocked`: One or more predecessor tasks remain incomplete.
  - `done`: Implementation complete, all acceptance criteria verified `[x]`.

### 3. Canonical Task Template

```markdown
# <NN>: <Task Title>

**Status**: ready | blocked | done
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
```

## Execution Protocol

### Step 1: Context & Tracker Discovery
1. Inspect available input context (discussion, specs in `docs/specs/`, or staged changesets).
2. Read `.agents/task-tracker.md` to resolve tracker type (`local-markdown`, `github`, `gitlab`) and target destination. If absent, halt and offer options:
   - Run `zenforge-init` for full repository onboarding.
   - Fast-path setup: initialize local tracker at `.agents/tasks/` immediately.

### Step 2: Slice Decomposition & Changeset Partitioning
1. Decompose requirements into sequential tracer-bullet vertical slices.
2. Establish acyclic blocking edges (`Blocked by`).
3. Partition the global Changeset into per-task Embedded Changesets with initial states (`ready` for unblocked tasks, `blocked` otherwise).

### Step 3: Staging & Authorization Gate
1. Present the task breakdown in Lean Delivery summary format:
   ```text
   | Task ID | Title | Blocked by | Initial Status | Key Deliverable |
   | :--- | :--- | :--- | :--- | :--- |
   | **01** | <Title 1> | None | `ready` | <One-line capability> |
   | **02** | <Title 2> | 01 | `blocked` | <One-line capability> |
   ```
2. Offer review options: approve publication, adjust granularity, or inspect individual changesets.
3. Forbid writing files or calling tracker APIs within this turn.
4. Halt turn immediately and wait for affirmative human authorization.

### Step 4: Atomic Publication & Frontier Handoff
1. Upon receiving approval, publish tasks according to `.agents/task-tracker.md`:
   - **Local Markdown**: Write one file per task as `<task_directory>/<NN>-<slug>.md` in dependency order.
   - **Remote Tracker**: Publish issues to the target repository configured in `.agents/task-tracker.md`.
2. Report the active frontier (first `ready` task) and hand off to `/forge`:
   - Local: `/forge .agents/tasks/01-<slug>.md`
   - Remote: `/forge #<ID>`
