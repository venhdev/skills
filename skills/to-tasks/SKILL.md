---
name: to-tasks
description: "Record settled changesets as trackable tasks, split work when useful, and publish to the configured tracker for later implementation."
disable-model-invocation: true
---

# to-tasks — Task Recording & Conditional Decomposition

## Operating Invariants

- **Recording Scope**: Publish task definitions and tracking metadata only; keep production code untouched and retain the supplied direction.
- **Tracker Selection**: Use the configured tracker exclusively; default to local Markdown when setup is absent, without publishing to multiple trackers.
- **Authorization Boundary**: Stage concrete tasks and destinations before writing files or remote issues. Honor existing publication authorization; task publication never authorizes implementation.

## Domain Rubric

### Input & Task Size

Receive a settled changeset, outcome, acceptance criteria, constraints, and supporting proof/references. Record small work as one task. Split medium/large work only when independently verifiable slices, dependencies, or separate execution sessions make tracking useful; task count is not a goal.

Each split task delivers a complete vertical capability and owns its corresponding changeset. Keep dependencies acyclic. Use prerequisite refactoring only when needed; for wide migrations, preserve expand/migrate/contract ordering rather than arbitrary horizontal layers.

### Tracker & Persistence

Read `.agents/task-tracker.md` before staging to identify the destination and its native operations. If absent, stage local tasks under `.agents/tasks/<feature-slug>/` and creation of `.agents/task-tracker.md` by copying [references/issue-tracker-local.md](references/issue-tracker-local.md); write that setup only within publication authorization. Existing remote setup wins over the default; unavailable remote access is a blocker, not permission to switch trackers silently.

Use repo-relative POSIX paths in stored tasks. Local tasks use `<NN>-<slug>.md` in dependency order; remote tasks use native dependencies or explicit `Blocked by` IDs. Re-read the published tasks to verify content, identifiers, and dependencies before completion.

### State & Task Lifecycle

- `STAGED`: Task definitions, count, dependencies, and destination are reviewable; publication is pending.
- `DONE`: Tasks have been published and their existence/content verified; implementation remains pending.
- `UNRESOLVED`: Input, tracker access, or publication prevents reliable recording; report any partial publication and its identifiers.

Stored implementation tasks retain `ready`, `in-progress`, and `done`. Eligibility requires `ready` and all predecessors `done`; blocking is derived, not a persisted lifecycle state. Publishing a task leaves it `ready`, even when this skill reports `DONE`.

## Canonical Output Contract

```markdown
# Task Recording: <Feature> · <STAGED | DONE | UNRESOLVED>

Tracker: <Configured destination or local default.>

| Task | Scope / Acceptance | Blocked by | Destination |
| :--- | :--- | :--- | :--- |
| <ID> | <Complete deliverable and criterion> | <IDs or None> | <Path or issue link> |

<Why splitting is useful; omit when one task suffices.>
Gap: <Blocker or partial publication; omit if none.>
```

Read [references/task-template.md](references/task-template.md) when preparing task bodies; embed each task's changeset and preserve evidence/constraints for later execution.

### Routing

- The input changeset has unresolved requirements → (via /clarify).
- The input direction is stale or needs redesign → (via /architect-decision).
- The caller requests implementation of an eligible, separately authorized task → (via /forge).

## Execution Protocol

1. Reconcile the input changeset with current context. If direction is stale or unresolved, report the gap rather than inventing tasks.
2. Resolve the tracker, retain one task or partition useful slices, and present `STAGED`. Halt before publication unless the concrete task scope and destination are already authorized.
3. Publish through the configured operations, verify the stored records, and report `DONE` or `UNRESOLVED`. Publication completes the recording request; keep implementation separately authorized.
