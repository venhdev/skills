---
name: zenf-tools
description: "Use when working with ZenForge daemon MCP tools: projects, plans, batched tasks with inline media, SSOT documents, agent definitions, observed sessions, cross-entity search, claim, review, and mixed transitions."
---

# zenf-tools — ZenForge Tracking Plane

Use this card when work is governed by `zenf`. The daemon is a **tracking
plane**, not a process runtime: it stores projects, plans, tasks, document
metadata, and agent definitions in SQLite WAL, serves a web cockpit, and exposes
this MCP surface. Agents do the executing; `zenf` records what happened.

This is the only skill that assumes the MCP server is connected. Every other
skill must remain executable from local files and standard tooling.

## Boundary

- Writes go through MCP (or the `zenf` CLI / public HTTP API). The Web Cockpit
  is read-only for plans, tasks, and sessions; it may CRUD project, document,
  and agent definitions.
- Context media attaches to a task while it is `READY` or `PENDING`; the next
  claim includes it in the `ContextPacket`.
- Document content stays file-canonical. The registry stores identity, metadata,
  tags, hash, and an FTS snapshot; never treat the snapshot as a second source.

## Hierarchy

```text
project <- plans <- tasks
```

A task has exactly one scope: plan task (`plan_slug` + `project_slug`), project
task (`project_slug`, no plan), or orphan (neither). Dependencies are validated
inside the same container; a plan task cannot depend on another plan.

## Roles

- **Main agent**: registers the project, authors the plan and task DAG,
  dispatches workers, relays human decisions, resolves reviews, records merge
  outcomes, archives finished plans. It registers MAIN/SUBAGENT sessions.
- **Worker**: claims one task, executes it, reports completion/failure/review,
  then exits. An external runner registers its WORKER session.

## Lifecycle

`projects(action=create)` → `plans(action=create, project_slug=...)` →
`tasks(action=create, plan_slug=...)` → `claim_task` → execute →
`transition_task(COMPLETE|FAIL)` **or** `review(action=request)` → main agent
calls `review(action=resolve)` → fresh claim.

```text
PENDING ──deps done──▶ READY ──claim──▶ RUNNING ──checkpoint──▶ RUNNING (advances phase)
                                   │        ├──complete──▶ COMPLETED
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
| `projects` | `create`, `get`, `list`, `update`, `archive`/`unarchive`, `delete`. | `action` (required); `slug`, `name`, `root_path`, `description`, `status`, `include_archived` |
| `plans` | `create` a plan inside a project (optionally with the whole task DAG), `get`, `list`, `update`, `archive`/`unarchive`, `delete`. | `action` (required); create: `slug` + `project_slug`; `title`, `status`, `tasks[]`, `include_archived` |
| `tasks` | `create` an atomic batch with inline media, `get` metadata or the `ContextPacket`, `list`, or `update` authoring metadata and media. | `action` (required); create: `plan_slug` or `project_slug` + `tasks[]`; get/update: `task_id`; list: `project_slug`, `plan_slug`, `status`, `agent_slug`, `orphan`; update: metadata, `media_add[]`, `media_remove[]` |
| `docs` | `create`, `get`, `list`, `update`, `refresh`, `archive`/`unarchive`, `delete`, `search` SSOT registry records. | `action` (required); `id` or `key`/`slug` + `project_slug`; `type`, `scope`, `scope_ref`, `title`, `path`, `summary`, `content`, `tags[]`, `status`, `query`, `tag`, `include_global`, `limit` |
| `agents` | `create`, `get`, `list`, `update`, `delete` worker definitions. | `action` (required); `slug`, `name`, `persona`, `mode`, `goal`, `start_conditions[]`, `stop_conditions[]`, `schedule`, `skills[]`, `model`, `runner`, `enabled`, `include_disabled` |
| `sessions` | `register`, `get`, `list`, `heartbeat`, `close` observed sessions. | `action` (required); `id`, `agent_slug`, `session_type`, `parent_id`, `external_ref`, `status`, `active_only`, `limit` |
| `search` | Search projects, plans, tasks, and document snapshots in one call. | `query` (required); `project_slug`, `limit` |
| `claim_task` | Atomically claim a `READY` task, transition to `RUNNING`, return its `ContextPacket`. | `task_id` (required); `worker_id`, `orchestrator_id`, `worker_role` |
| `review` | `request` pauses a `RUNNING` task at `AWAITING_REVIEW` with a drift question; `resolve` records the human decision and returns to `READY`. | `action`, `task_id` (required); request: `reason`, `details`, `options[]`; resolve: `response` |
| `transition_task` | Apply `COMPLETE`, `FAIL`, `RESET`, `CANCEL`, or `CHECKPOINT`. Use `task_ids` + `action`, or `items[]` for mixed actions. | `task_ids[]` + `action`, or `items[]`; `reason`, `summary`, `phase_id`, `phase_status`, `merge_outcome` (`MERGED`/`CONFLICT`) |

Clients namespace tools by server: OpenCode shows `zenf.plans`, `zenf.tasks`,
and so on; Claude Code shows `mcp__zenf__plans`.

### `tasks[]` item shape & Gold Standard Context Mandate

- **Engine Minimum**: The SQLite database engine strictly requires only `id` and `title` to insert a task record.
- **Autonomous Context Mandate (Anti-Shallow Rule)**: For autonomous workers to execute without human intervention or ambiguous drift, the Main Agent **MUST** provide rich execution context for every task:
  - `description`: Scope, rationale, boundaries, and expected behavior.
  - `target_files`: Explicit relative file paths to create or modify.
  - `acceptance_criteria`: Measurable conditions of satisfaction (DoD).
  - `verification_commands`: Exact test/lint commands for the worker to prove completion.
  - `skills`: Required skills worker should activate (e.g. `["tdd"]`).
  - `docs`: SSOT documentation keys registered in `docs` registry.
  - `phases`: Intra-task micro-phases for non-trivial tasks (checkpointed via `transition_task(action="CHECKPOINT")`).
  - `media`: Visual assets, wireframes, or specs (`[{"file_path": "..."}]`).

```json
{
  "id": "P-02",
  "title": "Implement claim preflight and context packet decouple",
  "description": "Move packet assembly outside the write transaction to prevent SQLite lock contention.",
  "dependencies": ["P-01"],
  "target_files": [
    "apps/daemon/internal/storage/task_lifecycle.go",
    "apps/daemon/internal/storage/task_lifecycle_test.go"
  ],
  "skills": ["tdd"],
  "docs": ["ARCH-DECISION-01"],
  "acceptance_criteria": [
    "ClaimTask builds ContextPacket outside write lock",
    "No race conditions during concurrent worker claims",
    "All unit tests in task_lifecycle_test.go pass"
  ],
  "verification_commands": [
    "go test -v -race ./apps/daemon/internal/storage/..."
  ],
  "priority": "P1",
  "complexity": "MEDIUM",
  "risk_level": "LOW",
  "agent_slug": "coder-alpha",
  "harness": "orchestrator",
  "worktree_path": "/path/to/worktree",
  "media": [{ "file_path": "/abs/design.png", "name": "Architecture Topology" }],
  "phases": [
    { "id": "analysis", "title": "Analyze call sites and lock boundaries" },
    { "id": "red_test", "title": "Write failing concurrency tests" },
    { "id": "green_impl", "title": "Decouple builder from write transaction" },
    { "id": "verify", "title": "Run regression and race detection suite" }
  ],
  "current_phase": "analysis"
}
```
*Note*: The 4-phase sequence above is an exemplar, not an enforced dogma. Tasks may have 0 phases (atomic task), 2-3 phases, or 4+ phases as appropriate.

Tasks with no `dependencies` start `READY`. A task with dependencies starts
`PENDING`, unless its prerequisites are already complete (→ `READY`) or already
failed/blocked (→ `BLOCKED`). Later completions unlock `PENDING` dependents
automatically.

## Create a project, plan, and DAG

```json
{ "action": "create", "slug": "zenforge", "name": "ZenForge", "root_path": "/home/dev/src/owner/zenforge" }
```

```json
{
  "action": "create",
  "project_slug": "zenforge",
  "slug": "auth-refactor",
  "title": "Auth refactor",
  "tasks": [
    {
      "id": "P-01",
      "title": "Freeze session schema",
      "description": "Establish baseline session types and immutable DTO contracts.",
      "target_files": ["internal/domain/session.go"],
      "acceptance_criteria": ["Session struct defines all token fields"],
      "verification_commands": ["go test ./internal/domain/..."],
      "media": [{ "file_path": "/abs/spec.pdf", "name": "Session Spec" }]
    },
    {
      "id": "P-02",
      "title": "Migrate call sites",
      "description": "Switch all auth handlers to the new session contract.",
      "dependencies": ["P-01"],
      "target_files": ["internal/gateway/auth_handlers.go"],
      "acceptance_criteria": ["All auth handlers consume domain.Session"],
      "verification_commands": ["go test ./internal/gateway/..."]
    },
    {
      "id": "P-03",
      "title": "Delete legacy path",
      "description": "Prune obsolete authentication shims and flags.",
      "dependencies": ["P-02"],
      "target_files": ["internal/gateway/legacy_auth.go"],
      "acceptance_criteria": ["Legacy auth file removed without compile errors"],
      "verification_commands": ["go test ./..."]
    }
  ]
}
```

The plan and DAG commit in one transaction. To add tasks later:
`tasks(action="create", plan_slug=..., tasks=[...])`. Create is create-only;
duplicate IDs, cycles, cross-container dependencies, and self-dependencies abort
the batch.

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
and re-derive the status. Media add/remove failures return a tool error.

## Media

Attach media inline on create or update. The task must still be `READY` or
`PENDING`; once claimed, the packet has been handed to a worker. Files are copied
under `~/.zenforge/attachments/<task>/`, capped at 25 MB, and appear in
`input_attachments` on the next claim.

## Documents

Register a canonical Markdown (or other) file:

```json
{
  "action": "create",
  "project_slug": "zenforge",
  "key": "ARCH",
  "slug": "architecture-topology",
  "type": "ARCHITECTURE",
  "scope": "PROJECT",
  "title": "Architecture topology",
  "path": "/home/dev/src/owner/zenforge/docs/architecture-topology.md",
  "summary": "Host-daemon topology and communication seams",
  "tags": ["architecture", "core"]
}
```

- `content` is optional; when omitted, the daemon reads `path`. Files larger than
  4 MiB are registered without a snapshot.
- `key` and `slug` are unique per project; global documents omit `project_slug`.
- `refresh` re-reads the file and updates hash, size, version, and FTS snapshot.
- `search` is full-text over key, title, summary, and content. Use it before
  writing a new document to avoid duplicate authority.
- Binaries (images, PDF, DOCX, XML) are stored by absolute path only.

## Agents and sessions

`agents` defines a WORKER persona and its operating envelope. `sessions` only
observes execution; the web never drives a session.

```json
{
  "action": "create",
  "slug": "coder-alpha",
  "name": "Coder Alpha",
  "persona": "CODER",
  "mode": "HEADLESS",
  "goal": "Implement READY tasks inside the assigned plan scope.",
  "start_conditions": ["task is READY", "no active claim"],
  "stop_conditions": ["task AWAITING_REVIEW", "verification failed"],
  "schedule": "*/15 * * * *",
  "skills": ["tdd", "forge"],
  "runner": "local-worker",
  "enabled": true
}
```

`register` a session with `id`, `session_type` (`MAIN|SUBAGENT|WORKER`),
optional `agent_slug`, `parent_id`, and `external_ref`; `heartbeat` keeps it
alive; `close` marks it `TERMINATED`.

## Archive and delete

When every task in a plan is finished (or the plan is abandoned):

```json
{ "action": "archive", "slug": "auth-refactor" }
```

Archived plans keep all rows and remain readable; they are hidden from the
default selector and DAG until `include_archived=true` / `Show archived`.
Use `unarchive` to bring one back. Delete is destructive and cascades to tasks
and artifacts; prefer archive.

## Transition rules

- `COMPLETE` on `RUNNING`: worktree set → `AWAITING_MERGE`; no worktree →
  `COMPLETED`.
- `COMPLETE` on `AWAITING_MERGE`: requires `merge_outcome=MERGED|CONFLICT`.
- `COMPLETE` on `CONFLICT`: requires `merge_outcome=MERGED`.
- `FAIL` on `RUNNING`: `FAILED` and every transitive dependent becomes
  `BLOCKED`.
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

## Guardrails

- **Register the project first.** `plans(create)` rejects a missing
  `project_slug`; tasks must reference a live plan or project.
- **One batch, one container.** `tasks(create)` is create-only. Use
  `tasks(action=update)` for READY/PENDING metadata.
- **Claim before touching files.** A worker that skips the claim can collide with
  another worker on the same task.
- **Review happens in the agent session.** The worker calls
  `review(action=request)` and exits; the main agent surfaces the question to the
  human, then calls `review(action=resolve)`.
- **Attach media before dispatch.** Input media is part of the claim packet.
- **Prefer archive over delete.**
- **Keep document content in Git.** Use `docs` for identity, classification,
  tags, and search; do not fork the text into the registry.

## Observability

Every project/plan/task/document/agent/session/artifact mutation is recorded by
database triggers, so MCP, CLI, HTTP, and raw SQLite writes all appear in the
cockpit's **Activity** view, the **Live Log**, and
`GET /api/v1/events?after_id=&limit=`. Use `search` to confirm an entity landed
and `sessions(action=list)` to see active runners.
