---
name: zenfd-tools
description: "Use when working with ZenForge daemon MCP tools: plans (create/archive/list), batch task creation, task claim, review handshake, state transitions, media attach, and task inspection."
---

# zenfd-tools — ZenForge Tracking Plane

Use this card when work is governed by `zenfd`. The daemon is a **tracking
plane**, not a process runtime: it stores plans/tasks in SQLite WAL, serves a
read-only web cockpit, and exposes this MCP surface. Agents do the executing;
`zenfd` records what happened.

## Boundary

- Writes go through MCP (or the `zenfd` CLI / public HTTP API). The Web Cockpit
  is read-only for everyone.
- Context media can only be attached while a task is `READY` or `PENDING`; the
  next claim includes it in the `ContextPacket`.

## Roles

- **Main agent**: creates the plan and its task DAG, dispatches workers, relays
  human decisions, resolves reviews, records merge outcomes, archives finished
  plans.
- **Worker**: claims one task, executes it, reports completion/failure/review,
  then exits.

## Lifecycle

`plans(action=create)` → one batched `tasks(action=create)` →
`claim_task` → execute → `transition_task(COMPLETE|FAIL)` **or**
`review(action=request)` → main agent calls `review(action=resolve)`
→ fresh claim.

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
| `plans` | `create` a plan (or retitle an existing one), `archive`/`unarchive` it, or `list` plans. Archived plans are hidden from list and the cockpit unless `include_archived=true`. | `action` (required); optional `slug`, `title`, `include_archived` |
| `tasks` | `create` a batch **atomically** in one call, `get` one task with its `ContextPacket`, or `list` tasks. Create fails if any ID already exists; the whole batch rolls back. | `action` (required); create: `plan_id` + `tasks[]`; get: `task_id`; list: optional `plan_id` or `status` |
| `claim_task` | Atomically claim a `READY` task, transition to `RUNNING`, return its `ContextPacket`. | `task_id` (required); optional `worker_id`, `orchestrator_id`, `worker_role` (default `CODER`) |
| `review` | `request` pauses a `RUNNING` task at `AWAITING_REVIEW` with a drift question (the worker exits); `resolve` records the human decision and returns the task to `READY`. Resolve is main-agent only. | `action`, `task_id` (required); request: `reason`, `details`, optional `options[]`; resolve: `response` |
| `transition_task` | Apply `COMPLETE`, `FAIL`, `RESET`, or `CANCEL` to one or more tasks. Each task is processed independently and returns `results[]`. | `task_ids[]`, `action` (required); optional `reason`, `summary`, `merge_outcome` (`MERGED`/`CONFLICT`) |
| `attach_media` | Copy a local image/PDF/document into the attachment store and register it as an `INPUT` artifact. | `task_id`, `file_path` (required); optional `name` |

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
  "worktree_path": "/path/to/worktree"
}
```

Tasks with no `dependencies` start `READY`. A task with dependencies starts
`PENDING`, unless its prerequisites are already complete (→ `READY`) or already
failed/blocked (→ `BLOCKED`). Later completions unlock `PENDING` dependents
automatically.

## Batch creation

Create the whole DAG in **one** call so it validates and commits atomically.

```json
{
  "action": "create",
  "plan_id": "auth-refactor",
  "tasks": [
    { "id": "P-01", "title": "Freeze session schema" },
    { "id": "P-02", "title": "Migrate call sites", "dependencies": ["P-01"] },
    { "id": "P-03", "title": "Delete legacy path", "dependencies": ["P-02"] }
  ]
}
```

Forward references inside the batch are allowed. Cycles, self-dependencies, and
duplicate IDs abort the whole call.

## Media attach

Attach an image, PDF, or document before a worker starts:

```json
{
  "task_id": "P-02",
  "file_path": "/abs/path/design.png",
  "name": "Checkout redesign"
}
```

Call `attach_media` with those arguments. The task must still be `READY` or
`PENDING`; once claimed, the packet has been handed to a worker. Files are copied
under `~/.zenforge/attachments/<task>/` and capped at 25 MB.

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

## Guardrails

- **One batch, one plan.** Create never overwrites an existing task; re-ingest
  the plan dossier to change authoring metadata.
- **Claim before touching files.** A worker that skips the claim can collide with
  another worker on the same task.
- **Watch the RUNNING badge.** A task stuck in `RUNNING` for an unusually long
  time means the orchestrator should check its worker and then `RESET` or `FAIL`
  it.
- **Review happens in the agent session.** The worker calls
  `review(action=request)` and exits; the main agent surfaces the question
  to the human, then calls `review(action=resolve)`.
- **Attach media before dispatch.** Input media is part of the claim packet.
- **Read state only when you need the payload.** The cockpit shows live state;
  use `tasks(action=list|get)` for the data itself.
- **Prefer archive over delete.** `DELETE` on the HTTP API removes the plan and
  its tasks/artifacts; archive keeps them readable and reversible.

## Observability

Every plan/task/artifact mutation is recorded by database triggers, so MCP, CLI,
HTTP, and raw SQLite writes all appear in the cockpit's **Live Log** (bottom-left
panel) and at `GET /api/v1/events?after_id=&limit=`. Use it to confirm a batch
landed or to see who claimed what.
