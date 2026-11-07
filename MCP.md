# ZenForge MCP

The `zenfd` MCP server exposes the ZenForge tracking plane over JSON-RPC 2.0 via
stdio. It owns projects, plans, tasks, SSOT document metadata, agent
definitions, observed sessions, cross-entity search, and the task workflow. It
does not run git operations; worktree and merge lifecycle stays in the task
domain.

## Build

```bash
cd apps/daemon && make build
```

## Client setup

Replace `/abs/path/to/apps/daemon/bin/zenfd` with the built binary path and
`/home/<user>` with your home directory. Restart the AI client after setup so it
discovers the tools.

### Automated (agy, claude, cursor, opencode)

View status and available actions:
```bash
zenfd mcp
```

Setup or audit individual AI clients:
```bash
zenfd mcp check --client opencode
zenfd mcp setup --client opencode --dry-run
zenfd mcp setup --client opencode
```

Or view overall system and MCP status:
```bash
zenfd status
```

| Client | `--client` | Config file |
| :--- | :--- | :--- |
| Antigravity | `agy` (default) | `~/.gemini/config/mcp_config.json` |
| Claude Code | `claude` | `~/.claude.json` |
| Cursor | `cursor` | `~/.cursor/mcp.json` |
| OpenCode | `opencode` | `~/.config/opencode/opencode.json` |

Flags: `check` takes `--client/-c`, `--config`, `--json`; `setup` takes those plus
`--db/-d`, `--binary/-b`, `--force/-f`, `--dry-run`; `mcp` takes `--db/-d`
(default `~/.zenforge/state.db`). Existing config is backed up to `<config>.bak`
and other MCP servers are preserved.

### Manual

Most clients read an `mcpServers` object:

```json
{
  "mcpServers": {
    "zenfd": {
      "command": "/abs/path/to/apps/daemon/bin/zenfd",
      "args": ["mcp", "--db", "/home/<user>/.zenforge/state.db"]
    }
  }
}
```

| Client | File |
| :--- | :--- |
| Claude Code | `~/.claude.json` (user) or `.mcp.json` (project) |
| Cursor | `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (project) |
| Antigravity | `~/.gemini/config/mcp_config.json` |
| Other `mcpServers` clients | that client's MCP config |

**OpenCode** (`opencode.json` at the repo root, or `~/.config/opencode/opencode.json`):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "zenfd": {
      "type": "local",
      "command": ["/abs/path/to/apps/daemon/bin/zenfd", "mcp", "--db", "/home/<user>/.zenforge/state.db"],
      "enabled": true
    }
  }
}
```

**VS Code** (`.vscode/mcp.json`):

```json
{
  "servers": {
    "zenfd": {
      "type": "stdio",
      "command": "/abs/path/to/apps/daemon/bin/zenfd",
      "args": ["mcp", "--db", "/home/<user>/.zenforge/state.db"]
    }
  }
}
```

**Codex CLI** (`~/.codex/config.toml`):

```bash
codex mcp add zenfd -- /abs/path/to/apps/daemon/bin/zenfd mcp --db /home/<user>/.zenforge/state.db
```

```toml
[mcp_servers.zenfd]
command = "/abs/path/to/apps/daemon/bin/zenfd"
args = ["mcp", "--db", "/home/<user>/.zenforge/state.db"]
```

## Hierarchy and scope

```text
project <- plans <- tasks
```

Every plan belongs to exactly one project. A task has exactly one scope:

| Scope | Identifying fields | Dependency container |
| :--- | :--- | :--- |
| Plan task | `plan_slug` + `project_slug` | `plan:<slug>` |
| Project task | `project_slug`, no `plan_slug` | `project:<slug>` |
| Orphan task | neither field | `orphan` |

Dependencies are validated inside the task's container. A plan task cannot
depend on a task from another plan. Archive hides a plan without deleting rows;
delete removes the plan and its tasks/artifacts.

## Roles

- **Main agent**: registers the project, authors plans and the task DAG,
  dispatches workers, relays human decisions, resolves reviews, records merge
  outcomes, archives finished plans. It also registers MAIN/SUBAGENT sessions.
- **Worker**: claims one task, executes it, reports completion/failure/review,
  then exits. An external runner registers its WORKER session.
- **Web Cockpit**: read-only for execution state. It may CRUD projects, document
  registry entries, and agent definitions. It never creates, claims, or
  transitions tasks, and never drives a session.

## Lifecycle

`projects(action=create)` → `plans(action=create, project_slug=...)` →
`tasks(action=create, plan_slug=...)` → `claim_task` → execute →
`transition_task(COMPLETE|FAIL)` **or** `review(action=request)` → main agent
calls `review(action=resolve)` → fresh claim. Recovery is `RESET` followed by a
fresh claim.

## Tool reference

All ten tools require `action` except `claim_task`, `search`, and
`transition_task`.

| Tool | Behavior | Key parameters |
| :--- | :--- | :--- |
| `projects` | Create, get, list, update, archive/unarchive, or delete a project. | `action`; `slug`, `name`, `root_path`, `description`, `status`, `include_archived` |
| `plans` | Create a plan (optionally with its whole task DAG), get, list, update, archive/unarchive, or delete. Create requires `project_slug`. | `action`; `slug`, `project_slug`, `title`, `status`, `include_archived`, `tasks[]` |
| `tasks` | Create an atomic batch with optional inline media; get metadata or the `ContextPacket`; list by project/plan/status/agent/orphan; update authoring metadata and media. | `action`; create: `plan_slug` or `project_slug`, `tasks[]`; get/update: `task_id`; list: `project_slug`, `plan_slug`, `status`, `agent_slug`, `orphan`; update: metadata, `media_add[]`, `media_remove[]` |
| `docs` | Register and manage SSOT documents: create, get, list, update, refresh, archive/unarchive, delete, and FTS search. Files remain canonical. | `action`; `id` or `key`/`slug` + `project_slug`; `type`, `scope`, `scope_ref`, `title`, `path`, `summary`, `content`, `tags[]`, `status`, `query`, `tag`, `include_global`, `limit` |
| `agents` | Create, get, list, update, or delete reusable worker definitions (persona, mode, goal, conditions, schedule, skills, model, runner). | `action`; `slug`, `name`, `persona`, `mode`, `goal`, `start_conditions[]`, `stop_conditions[]`, `schedule`, `skills[]`, `model`, `runner`, `enabled`, `include_disabled` |
| `sessions` | Register, get, list, heartbeat, or close observed sessions. | `action`; `id`, `agent_slug`, `session_type`, `parent_id`, `external_ref`, `status`, `active_only`, `limit` |
| `search` | Search projects, plans, tasks, and document snapshots in one call. | `query`; `project_slug`, `limit` |
| `claim_task` | Atomically claim a `READY` task, transition to `RUNNING`, return its `ContextPacket`. | `task_id`; `worker_id`, `orchestrator_id`, `worker_role` |
| `review` | `request` pauses at `AWAITING_REVIEW` with a drift question; `resolve` records the human decision and returns to `READY`. | `action`, `task_id`; `reason`, `details`, `options[]`; `response` |
| `transition_task` | Apply `COMPLETE`, `FAIL`, `RESET`, or `CANCEL` to many tasks; returns `results[]`. | `task_ids[]` + `action`, or `items[]`; `reason`, `summary`, `merge_outcome` |

Clients namespace tools by server name. OpenCode shows them as `zenfd.projects`,
`zenfd.plans`, and so on; Claude Code uses `mcp__zenfd__projects`.

### Task item shape

Only `id` and `title` are required per item. Everything else is optional:

```json
{
  "id": "P-02",
  "title": "Implement claim preflight",
  "description": "Move packet assembly outside the write transaction.",
  "dependencies": ["P-01"],
  "target_files": ["apps/daemon/internal/storage/task_lifecycle.go"],
  "skills": ["tdd"],
  "docs": ["ARCH"],
  "acceptance_criteria": ["claim no longer holds the write lock"],
  "verification_commands": ["go test ./..."],
  "priority": "P1",
  "complexity": "MEDIUM",
  "risk_level": "LOW",
  "agent_slug": "coder-alpha",
  "harness": "orchestrator",
  "worktree_path": "/path/to/worktree",
  "media": [{ "file_path": "/abs/design.png", "name": "Checkout redesign" }]
}
```

Tasks with no dependencies start `READY`. A task with dependencies starts
`PENDING`, unless its prerequisites are already complete (→ `READY`) or already
failed/blocked (→ `BLOCKED`). Later completions unlock `PENDING` dependents
automatically.

## Create a project, plan, and DAG

```json
{
  "action": "create",
  "slug": "zenforge",
  "name": "ZenForge",
  "root_path": "/home/dev/src/owner/zenforge"
}
```

```json
{
  "action": "create",
  "project_slug": "zenforge",
  "slug": "auth-refactor",
  "title": "Auth refactor",
  "tasks": [
    { "id": "P-01", "title": "Freeze session schema", "media": [{ "file_path": "/abs/spec.pdf" }] },
    { "id": "P-02", "title": "Migrate call sites", "dependencies": ["P-01"] },
    { "id": "P-03", "title": "Delete legacy path", "dependencies": ["P-02"] }
  ]
}
```

The plan and DAG commit in one transaction. To add tasks later:
`tasks(action="create", plan_slug="auth-refactor", tasks=[...])`. Create is
create-only; duplicate IDs, cycles, and cross-container dependencies abort the
batch.

## Documents (SSOT registry)

Content lives in the file at `path`. The registry stores identity (`id`, `key`,
`slug`), classification (`type`, `scope`, `scope_ref`, `status`), `tags`,
content hash, byte size, and an FTS-searchable snapshot. `content` is optional:
when omitted on create or refresh, the daemon reads the file. Files larger than
4 MiB are registered without a snapshot.

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

Search:

```json
{ "action": "search", "query": "claim preflight", "project_slug": "zenforge", "include_global": true }
```

`key` and `slug` are unique per project through `IFNULL(project_slug,'')`;
global documents have an empty `project_slug`. Omit `project_slug` for a global
document. Binary and media files must be stored by absolute path, never in the
snapshot.

## Agents and sessions

`agents` stores reusable WORKER definitions. Personas are `EXPLORER`, `CODER`,
`REVIEWER`, or `CUSTOM`; modes are `HEADLESS` or `INTERACTIVE`. Conditions and
schedules are descriptive in the MVP: an external runner reads the definition
and executes it.

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
  "model": "provider/model",
  "runner": "local-worker",
  "enabled": true
}
```

Sessions are observed, not driven:

```json
{ "action": "register", "id": "sess-01", "session_type": "WORKER", "agent_slug": "coder-alpha", "parent_id": "sess-main", "external_ref": "pid:4242" }
```

`heartbeat` updates `last_seen_at` and optionally the status. `close` marks the
session `TERMINATED`. MAIN/SUBAGENT sessions are registered by the main agent;
the web UI only lists them.

## Write surfaces

| Surface | Use |
| :--- | :--- |
| MCP (stdio) | Agent path: projects, plans, tasks, docs, agents, sessions, search, claim, review, transitions. |
| `zenfd` CLI | `zenfd project ...`, `zenfd plan push`, task status, claim. |
| HTTP API | Scripted and web-definition clients; see the endpoint list below. Cross-origin browser writes are rejected. |
| Web Cockpit | Projects/docs/agents CRUD; read-only for plans, tasks, execution state, and sessions. |

### HTTP endpoints (v0.2)

| Method | Path |
| :--- | :--- |
| `GET/POST` | `/api/v1/projects` |
| `GET/PATCH/DELETE` | `/api/v1/projects/{slug}` |
| `GET/POST` | `/api/v1/plans` (create requires `project_slug`) |
| `GET/PATCH/DELETE` | `/api/v1/plans/{slug}` |
| `GET/POST` | `/api/v1/tasks` |
| `GET` | `/api/v1/tasks/{id}` |
| `POST` | `/api/v1/tasks/{id}/status`, `/claim`, `/evict`, `/dependencies`, `/attachments` |
| `PATCH` | `/api/v1/tasks/{id}/context` |
| `GET` | `/api/v1/tasks/{id}/artifacts` |
| `GET/POST` | `/api/v1/docs` |
| `GET` | `/api/v1/docs/search?query=` |
| `GET/PATCH/DELETE` | `/api/v1/docs/{id}` |
| `POST` | `/api/v1/docs/{id}/refresh` |
| `GET/POST` | `/api/v1/agents` |
| `GET/PATCH/DELETE` | `/api/v1/agents/{slug}` |
| `GET` | `/api/v1/sessions?session_type=&agent_slug=&status=&active_only=` |
| `GET` | `/api/v1/search?q=&project_slug=` |
| `GET` | `/api/v1/events?after_id=&limit=`, `/api/v1/events/stream` (SSE) |

Every mutation is recorded by database triggers into `events`, so the Live Log
shows MCP, CLI, HTTP, and raw SQLite writes alike.

## Media

Attach input media while creating or updating a task:

```json
{
  "action": "create",
  "plan_slug": "auth-refactor",
  "tasks": [
    { "id": "P-01", "title": "Checkout redesign", "media": [{ "file_path": "/abs/design.png" }] }
  ]
}
```

`tasks(action="update", task_id, media_add=[...], media_remove=[...])` adds or
removes media on a `READY`/`PENDING` task. Files are copied under
`~/.zenforge/attachments/<task>/` and capped at 25 MB. A failed attach or detach
returns a tool error instead of a success payload with a nested error.

## Review and merge

A worker that hits drift calls `review(action=request)` and exits; the main
agent relays the question to the human and calls `review(action=resolve)`.
The cockpit only displays the review state.

Merge happens outside MCP. `COMPLETE` with a `worktree_path` stops at
`AWAITING_MERGE`; record the outcome with `merge_outcome=MERGED|CONFLICT`, and
use `COMPLETE` again on `CONFLICT` once resolved. `FAIL` marks the task `FAILED`
and blocks downstream dependents; `RESET`/`CANCEL` apply to `RUNNING`.

## Live verification

1. Configure the client (`zenfd mcp check --client <name>`, or edit the manual
   file), restart the session, and confirm the 10 tools above are listed.
2. Create a disposable project, then call `plans(action="list")` with its slug
   and confirm the daemon responds.
3. Register a document from a real file, refresh it, and find it with
   `docs(action="search")`.
4. Create a plan and task, claim it, and complete it with `transition_task`;
   optionally request and resolve a review.
