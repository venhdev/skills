---
name: zenfd-tools
description: "Use when working with ZenForge daemon MCP tools: plans (create/archive/list), batched tasks with inline media, task update/get/list, claim, review, and mixed transitions."
---

# zenfd-tools — ZenForge Tracking Plane

Use this card when work is governed by `zenfd`. The daemon is a **tracking
plane**, not a process runtime: it stores plans/tasks in SQLite WAL, serves a
read-only web cockpit, and exposes this MCP surface. Agents do the executing;
`zenfd` records what happened.

## Boundary

- Writes go through MCP (or the `zenfd` CLI / public HTTP API). The Web Cockpit
  is read-only for everyone.
- Context media attaches to a task while it is `READY` or `PENDING`; the next
  claim includes it in the `ContextPacket`.

## Roles

- **Main agent**: creates the plan and its task DAG, dispatches workers, relays
  human decisions, resolves reviews, records merge outcomes, archives finished
  plans.
- **Worker**: claims one task, executes it, reports completion/failure/review,
  then exits.

## Lifecycle

`plans(action=create)` (optional `tasks[]`) → `claim_task` → execute →
`transition_task(COMPLETE|FAIL)` **or** `review(action=request)` → main agent
calls `review(action=resolve)` → fresh claim.

```text
PENDING ──deps done──▶ READY ──claim──▶ RUNNING ──complete──▶ COMPLETED
                                   │        └──complete (worktree)──▶ AWAITING_MERGE ──MERGED──▶ COMPLETED
                                   │                                  └──CONFLICT──▶ CONFLICT ──MERGED──▶ COMPLETED
                                   ├──fail──▶ FAILED  (blocks downstream dependents)
                                   ├──review request──▶ AWAITING_REVIEW ──resolve──▶ READY
                                   └──reset──▶ READY | BLOCKED        cancel──▶ CANCELLED
```

Recovery is `RESET` followed by a fresh claim.

## Tools

| Tool | Function | Parameters |
| :--- | :--- | :--- |
| `plans` | `create` a plan (optionally with the whole task DAG), `archive`/`unarchive`, or `list`. Archived plans are hidden unless `include_archived=true`. | `action` (required); optional `slug`, `title`, `include_archived`, `tasks[]` |
| `tasks` | `create` an atomic batch with inline media, `get` metadata or the `ContextPacket`, `list`, or `update` authoring metadata and media. | `action` (required); create: `plan_id` + `tasks[]`; get: `task_id`, optional `include_packet`; list: optional `plan_id` or `status`; update: `task_id`, metadata fields, `media_add[]`, `media_remove[]` |
| `claim_task` | Atomically claim a `READY` task, transition to `RUNNING`, return its `ContextPacket`. | `task_id` (required); optional `worker_id`, `orchestrator_id`, `worker_role` (default `CODER`) |
| `review` | `request` pauses a `RUNNING` task at `AWAITING_REVIEW` with a drift question (the worker exits); `resolve` records the human decision and returns the task to `READY`. Resolve is main-agent only. | `action`, `task_id` (required); request: `reason`, `details`, optional `options[]`; resolve: `response` |
| `transition_task` | Apply `COMPLETE`, `FAIL`, `RESET`, or `CANCEL`. Use `task_ids` + `action` for one shared action, or `items[]` for mixed actions. | `task_ids[]` + `action`, or `items[]`; optional `reason`, `summary`, `merge_outcome` (`MERGED`/`CONFLICT`) |

Clients namespace tools by server: OpenCode shows `zenfd.plans`, `zenfd.tasks`,
and so on; Claude Code shows `mcp__zenfd__plans`.

### `tasks[]` item shape

Only `id` and `title` are required per item. Everything else is optional:

```json
{
  "id": "P-02",
  "title": "Implement claim preflight",
  "description": "Move packet assembly outside the write transaction.",
  "dependencies": ["P-01"],
  "target_files": ["apps/daemon/internal/storage/task_lifecycle.go"],
  "skills": ["tdd"],
  "acceptance_criteria": ["claim no longer holds the write lock"],
  "verification_commands": ["go test ./..."],
  "priority": "P1",
  "harness": "orchestrator",
  "worktree_path": "/path/to/worktree",
  "media": [{ "file_path": "/abs/design.png", "name": "Checkout redesign" }]
}
```

Tasks with no `dependencies` start `READY`. A task with dependencies starts
`PENDING`, unless its prerequisites are already complete (→ `READY`) or already
failed/blocked (→ `BLOCKED`). Later completions unlock `PENDING` dependents
automatically.

## Create a plan and DAG in one call

```json
{
  "action": "create",
  "slug": "auth-refactor",
  "title": "Auth refactor",
  "tasks": [
    { "id": "P-01", "title": "Freeze session schema", "media": [{ "file_path": "/abs/spec.pdf" }] },
    { "id": "P-02", "title": "Migrate call sites", "dependencies": ["P-01"] },
    { "id": "P-03", "title": "Delete legacy path", "dependencies": ["P-02"] }
  ]
}
```

The plan and DAG commit in one transaction. To add tasks later, call
`tasks(action="create", plan_id=..., tasks=[...])`. Create is create-only;
duplicate IDs, cycles, and self-dependencies abort the batch.

## Update a task

```json
{
  "action": "update",
  "task_id": "P-02",
  "title": "Checkout redesign v2",
  "dependencies": ["P-01"],
  "media_add": [{ "file_path": "/abs/design-v2.png" }],
  "media_remove": ["art-in-P-02-..."]
}
```

Only `READY`/`PENDING` tasks accept updates. Omitted fields stay unchanged; an
empty list clears dependencies or media. Dependency changes revalidate the graph
and re-derive the status. Updatable fields: `title`, `description`,
`dependencies`, `target_files`, `skills`, `acceptance_criteria`,
`verification_commands`, `priority`, `complexity`, `risk_level`, `harness`,
`worktree_path`, plus `media_add` / `media_remove`.

## Media

Attach media inline on create or update. The task must still be `READY` or
`PENDING`; once claimed, the packet has been handed to a worker. Files are copied
under `~/.zenforge/attachments/<task>/`, capped at 25 MB, and appear in
`input_attachments` on the next claim. Use `media_remove` to detach a file added
by mistake; removal uses the same `READY`/`PENDING` gate. A failed media copy
keeps the task batch and reports the error for that file.

## Archive

When every task in a plan is finished (or the plan is abandoned):

```json
{ "action": "archive", "slug": "auth-refactor" }
```

Archived plans keep all rows and remain readable; they are hidden from the
default selector and DAG until `include_archived=true` / `Show archived`.
Use `unarchive` to bring one back.

## Transition rules

- `COMPLETE` on `RUNNING`: worktree set → `AWAITING_MERGE`; no worktree →
  `COMPLETED`.
- `COMPLETE` on `AWAITING_MERGE`: requires `merge_outcome=MERGED|CONFLICT`.
- `COMPLETE` on `CONFLICT`: requires `merge_outcome=MERGED`.
- `FAIL` on `RUNNING`: `FAILED` and every transitive dependent becomes
  `BLOCKED` (including `READY` dependents).
- `RESET` / `CANCEL` on `RUNNING`: `RESET` clears the claim and returns to
  `READY`, or to `BLOCKED` when prerequisites are unfinished.
- Transitions are conditional on the current status; a concurrent change returns
  a conflict instead of overwriting the winner.

Mixed actions in one call:

```json
{
  "items": [
    { "task_id": "P-01", "action": "COMPLETE", "summary": "tests pass" },
    { "task_id": "P-02", "action": "FAIL", "reason": "compile error" },
    { "task_id": "P-03", "action": "RESET", "reason": "worker lost" }
  ]
}
```

Each item is processed independently and returns a `results[]` entry.

## Guardrails

- **One batch, one plan.** `tasks(create)` is create-only. Use
  `tasks(action=update)` for READY/PENDING metadata; `plans(create)` patches
  authoring metadata for existing tasks without resetting status or claims.
- **Claim before touching files.** A worker that skips the claim can collide with
  another worker on the same task.
- **Watch the RUNNING badge.** A task stuck in `RUNNING` for an unusually long
  time means the orchestrator should check its worker and then `RESET` or `FAIL`
  it.
- **Review happens in the agent session.** The worker calls
  `review(action=request)` and exits; the main agent surfaces the question to the
  human, then calls `review(action=resolve)`.
- **Attach media before dispatch.** Input media is part of the claim packet.
- **Read state only when you need the payload.** The cockpit shows live state;
  use `tasks(action=list)` for metadata and `include_packet=true` only when the
  prompt and attachments are needed.
- **Prefer archive over delete.** `DELETE` on the HTTP API removes the plan and
  its tasks/artifacts; archive keeps them readable and reversible.

## Observability

Every plan/task/artifact mutation is recorded by database triggers, so MCP, CLI,
HTTP, and raw SQLite writes all appear in the cockpit's **Live Log** (bottom-left
panel) and at `GET /api/v1/events?after_id=&limit=`. Use it to confirm a batch
landed or to see who claimed what.
