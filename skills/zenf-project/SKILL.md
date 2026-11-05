---
name: zenf-project
description: "Create and navigate the ZenForge project-plan-task hierarchy: choose task scope, author plans and DAGs, inspect readiness, and dispatch workers."
disable-model-invocation: true
---

# zenf-project — Project, Plan, and Task Front Door

Use this card when the caller wants to register a project, open a plan, author a
task DAG, decide whether work is a plan task / project task / orphan, or inspect
the execution board. It routes execution writes to the owning surface and never
duplicates `zenfd-tools` lifecycle rules.

## Prerequisites

- Confirm the project root exists and is a Git repository when the project owns
  code. A project can also be documentation-only.
- Read `zenfd-tools` when a mutation needs the exact task lifecycle or payload.
- Choose one write surface:
  - **MCP** (preferred when connected): `projects`, `plans`, `tasks`, `search`.
  - **HTTP** (MCP unavailable): `POST /api/v1/projects`, `POST /api/v1/plans`,
    `POST /api/v1/tasks`.
  - **CLI**: `zenfd plan push <file.md> --project <slug>` for Markdown plans.
- If the daemon is unreachable and the caller only needs a plan document, write
  the Markdown file and stop; do not invent a database state.

## Step Protocol

1. **Resolve the project.**
   - Search by slug: `projects(action=get, slug=...)` or
     `GET /api/v1/projects/{slug}`.
   - If absent, capture `slug`, `name`, `root_path`, and `description`, then
     create it. Never reuse another project's slug for a different root.
2. **Resolve the owning plan.**
   - Plans always belong to a project. List with `project_slug` before creating.
   - A new plan needs `slug`, `project_slug`, and `title`.
3. **Classify each task.**
   - Same plan, shared DAG → plan task: set `plan_slug` (the daemon derives the
     project).
   - Project-wide work outside a plan → project task: set `project_slug` only.
   - Standalone experiment → orphan: set neither.
   - Do not mix containers in one dependency graph.
4. **Author the DAG.**
   - Give every task a stable `id` and a concrete `title`.
   - Add `acceptance_criteria` and `verification_commands` for executable work.
   - Add `target_files`, `skills`, `docs`, and `agent_slug` only when they change
     execution.
   - Dependencies must reference tasks in the same container and must not cycle.
5. **Commit once.** Prefer `plans(create, tasks=[...])` or one
   `tasks(create, tasks=[...])` batch over many calls. Create is create-only.
6. **Verify.** Read the plan back and confirm task count, scopes, statuses, and
   dependency edges. Use `tasks(action=list, plan_slug=...)` or
   `GET /api/v1/plans/{slug}`. Confirm `search(query=...)` returns the new
   entities.
7. **Hand off execution.** Workers claim through `claim_task`; the main agent
   resolves reviews and transitions. The web cockpit is observational only. For
   worker definitions and sessions, use `zenf-agent`.

## Scope Rubric

| Signal | Choose |
| :--- | :--- |
| Work advances a named multi-task initiative | Plan task |
| Work belongs to the repository but not a plan (chore, spike, maintenance) | Project task |
| Work is a disposable experiment with no owner | Orphan task |
| Work spans two plans | Split into two tasks or promote a new plan |
| Dependency crosses plans | Reject; restructure the container |

## MCP-Optional Matrix

| Need | MCP | HTTP fallback |
| :--- | :--- | :--- |
| Create project | `projects(action=create)` | `POST /api/v1/projects` |
| Create plan + DAG | `plans(action=create, tasks=[])` | `POST /api/v1/plans` with `project_slug` and `tasks[]` |
| Add tasks | `tasks(action=create)` | `POST /api/v1/tasks` |
| Inspect plan | `plans(action=get)` + `tasks(action=list)` | `GET /api/v1/plans/{slug}` |
| Find entity | `search(query=...)` | `GET /api/v1/search?q=` |

HTTP is a definition/ingestion surface, not an execution shortcut. Do not use it
to claim or transition a task from the web.

## Guardrails

- **Project first.** A plan without `project_slug` is rejected by design.
- **One container per task.** `plan_slug` and a project-only task are mutually
  exclusive semantics; never set both for different owners.
- **No silent status edits.** Task status follows the lifecycle; do not write
  `COMPLETED` directly through a metadata update.
- **Archive before delete.** Delete cascades to tasks and artifacts.
- **Do not clone a plan into another project** by copying slugs; recreate it
  under the target project and revalidate dependencies.

## Output

Return a compact project brief:

```markdown
## Project
- slug / name / root_path

## Plan
- slug / title / status

## Tasks
| id | title | scope | depends on | status |
| :--- | :--- | :--- | :--- | :--- |

## Verification
- plan read-back: pass/fail
- search hit: pass/fail
```

## Handoffs

- Need exact tool payloads or lifecycle transitions → (via /zenfd-tools).
- Need to register supporting documents → (via /zenf-docs).
- Need worker personas, schedules, or session tracking → (via /zenf-agent).
- Need to decompose an accepted design into slices → (via /split-tasks).
- Need to persist an external tracker instead of zenfd → (via /track-tasks).
