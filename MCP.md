# ZenForge MCP

The `zenfd` MCP server exposes plans and tasks over JSON-RPC 2.0 via stdio. It
is not bound to a repository and does not run git operations; worktree and merge
lifecycle stays in the task domain.

## Build

```bash
cd apps/daemon && make build
```

## Client setup

Replace `/abs/path/to/apps/daemon/bin/zenfd` with the built binary path and
`/home/<user>` with your home directory. Restart the AI client after setup so it
discovers the tools.

### Automated (agy, claude, cursor, opencode)

```bash
apps/daemon/bin/zenfd mcp check --client opencode
apps/daemon/bin/zenfd mcp setup --client opencode --dry-run
apps/daemon/bin/zenfd mcp setup --client opencode
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

Two clients use a different shape.

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

## Roles

- **Main agent**: owns the plan and task DAG, dispatches workers, relays human
  decisions, resolves reviews, records merge outcomes, archives finished plans.
- **Worker**: claims one task, executes it, reports completion/failure/review,
  then exits.

## Lifecycle

`zenfd_plans(action=create)` → one batched `zenfd_tasks(action=create)` →
`zenfd_claim_task` → execute → `zenfd_transition_task(COMPLETE|FAIL)` or
`zenfd_review(action=request)` → main agent calls `zenfd_review(action=resolve)`
→ fresh claim. Recovery is `RESET` followed by a fresh claim.

Completed plans can be hidden with `zenfd_plans(action=archive)` and restored with
`action=unarchive`. Archived plans are omitted from `list` and from the cockpit
unless `include_archived` / `Show archived` is enabled.

## Tool reference

| Tool | Behavior | Required | Optional |
| :--- | :--- | :--- | :--- |
| `zenfd_plans` | `create` a plan or retitle an existing one; `archive`/`unarchive`; `list`. | `action` | `slug`, `title`, `include_archived` |
| `zenfd_tasks` | `create` a batch atomically; `get` one task with its `ContextPacket`; `list` by plan or status. | `action`; `plan_id` + `tasks[]` (create), `task_id` (get) | per task: `description`, `dependencies`, `target_files`, `skills`, `acceptance_criteria`, `verification_commands`, `priority`, `harness`, `worktree_path`; `status` (list) |
| `zenfd_claim_task` | Claim a `READY` task, transition to `RUNNING`, return its `ContextPacket`. | `task_id` | `worker_id`, `orchestrator_id`, `worker_role` |
| `zenfd_review` | `request` pauses at `AWAITING_REVIEW` with a drift question; `resolve` records the human decision and returns to `READY`. | `action`, `task_id` | `reason`, `details`, `options` (request); `response` (resolve) |
| `zenfd_transition_task` | Apply `COMPLETE`, `FAIL`, `RESET`, or `CANCEL` to many tasks; returns `results[]`. | `task_ids`, `action` | `reason`, `summary`, `merge_outcome` |
| `zenfd_attach_media` | Copy a local image/PDF/document into the attachment store as an `INPUT` artifact for the next claim. | `task_id`, `file_path` | `name` |

Create is strongly typed: duplicate IDs fail the whole batch, cycles and
self-dependencies are rejected, and initial status is derived from prerequisites
(all complete → `READY`, failed/blocked → `BLOCKED`, otherwise `PENDING`).

## Write surfaces

| Surface | Use |
| :--- | :--- |
| MCP (stdio) | Agent path: plans, batched tasks, claim, review, transitions, media. |
| `zenfd` CLI | `zenfd push`, task status, claim. |
| HTTP API | Scripted clients: `POST /api/v1/plans`, `PATCH|DELETE /api/v1/plans/{slug}`, `POST /api/v1/tasks`, `POST /api/v1/tasks/{id}/{status,claim,evict}`, `PATCH /api/v1/tasks/{id}/context`, `POST /api/v1/tasks/{id}/attachments`. Cross-origin browser writes are rejected. |
| Web Cockpit | Read-only: plans/tasks/DAG, artifacts, Live Log (`GET /api/v1/events`). |

Every mutation is recorded by database triggers, so the Live Log shows MCP, CLI,
HTTP, and raw SQLite writes alike.

## Review and merge

A worker that hits drift calls `zenfd_review(action=request)` and exits; the main
agent relays the question to the human and calls `zenfd_review(action=resolve)`.
The cockpit only displays the review state.

Merge happens outside MCP. `COMPLETE` with a `worktree_path` stops at
`AWAITING_MERGE`; record the outcome with `merge_outcome=MERGED|CONFLICT`, and
use `COMPLETE` again on `CONFLICT` once resolved. `FAIL` marks the task `FAILED`
and blocks downstream dependents; `RESET`/`CANCEL` apply to `RUNNING`.

## Live verification

1. Configure the client (`zenfd mcp check --client <name>`, or edit the manual
   file), restart the session, and confirm the 6 tools above are listed.
2. Call `zenfd_tasks(action=list)` and confirm the daemon responds.
3. Create a disposable plan and task, claim it, and complete it with
   `zenfd_transition_task`; optionally request and resolve a review.
