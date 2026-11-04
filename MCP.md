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

### Automated setup (agy, claude, cursor, opencode)

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

Flags:

- `check`: `--client/-c`, `--config`, `--json`.
- `setup`: `--client/-c`, `--config`, `--db/-d`, `--binary/-b`, `--force/-f`, `--dry-run`.
- `mcp`: `--db/-d` (default `~/.zenforge/state.db`).

Existing config is backed up to `<config>.bak`; other MCP servers are preserved.
VS Code is not covered by `zenfd mcp setup`; configure it manually with the
snippet below.

### OpenCode (auto or manual)

`zenfd mcp setup --client opencode` writes the project/global config entry
automatically. For a project-scoped config, pass `--config ./opencode.json`.
Use manual editing when your `opencode.jsonc` contains comments, since the setup
command reads and writes plain JSON.

Project config: `opencode.json` (or `opencode.jsonc`) at the repository root.
Global config: `~/.config/opencode/opencode.json`.

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "zenfd": {
      "type": "local",
      "command": [
        "/abs/path/to/apps/daemon/bin/zenfd",
        "mcp",
        "--db", "/home/<user>/.zenforge/state.db"
      ],
      "enabled": true
    }
  }
}
```

OpenCode uses a single `command` array (binary and arguments together), unlike
the `command` + `args` shape used by `mcpServers` clients. Project config
overrides global config for conflicting keys.

### Claude Code (manual or CLI)

CLI:

```bash
claude mcp add zenfd -- /abs/path/to/apps/daemon/bin/zenfd mcp --db /home/<user>/.zenforge/state.db
```

JSON (`~/.claude.json` for user scope, `.mcp.json` for project scope):

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

### Cursor (manual)

Global: `~/.cursor/mcp.json`. Project: `.cursor/mcp.json`.

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

### Antigravity (manual)

`~/.gemini/config/mcp_config.json`:

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

### VS Code (manual)

`.vscode/mcp.json`:

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

### Other clients

Clients that read the standard `mcpServers` object can reuse the Claude Code
snippet. Point `command` at the `zenfd` binary and pass `mcp --db
/home/<user>/.zenforge/state.db` as arguments.

## Roles

- **Main agent**: creates plans/tasks, dispatches workers, relays human decisions,
  resolves reviews, records merge outcomes.
- **Worker**: claims one task, executes it, reports completion/failure/review,
  then terminates.

## Lifecycle

`create_plan` → `create_task` → `claim_task` → execute →
`transition_task(COMPLETE|FAIL)` or `request_review` → main agent resolves review
→ fresh claim.

Recovery is `RESET` followed by a fresh claim; there is no `reclaim_task`.

## Tool reference

| Tool | Behavior | Required | Optional |
| :--- | :--- | :--- | :--- |
| `zenfd_create_plan` | Create or update a plan in SQLite WAL. | `slug`, `title` | — |
| `zenfd_create_task` | Create or update a task in a plan DAG. | `plan_id`, `id`, `title` | `description`, `dependencies`, `target_files`, `skills`, `acceptance_criteria`, `verification_commands`, `priority` (`P0`–`P3`, default `P2`), `harness` (default `orchestrator`, or `shell`), `worktree_path` |
| `zenfd_claim_task` | Atomically claim a `READY` task, transition to `RUNNING`, return its `ContextPacket`. | `task_id` | `worker_id`, `orchestrator_id`, `worker_role` (default `CODER`) |
| `zenfd_request_review` | Pause at `AWAITING_REVIEW` and record a drift question. The worker exits. | `task_id`, `reason`, `details` | `options` |
| `zenfd_resolve_review` | Record the human decision and return the task to `READY`. Main agent only. | `task_id`, `response` | — |
| `zenfd_transition_task` | Batch transition tasks; each task is processed independently and returns `results[]`. | `task_ids`, `action` (`COMPLETE`, `FAIL`, `RESET`, `CANCEL`) | `reason`, `summary`, `merge_outcome` (`MERGED`, `CONFLICT`) |
| `zenfd_get_task` | Read task metadata and `ContextPacket` without mutation. | `task_id` | — |

## Review policy

Review decisions never happen in the Web Cockpit. A worker calls
`zenfd_request_review` and exits; the main orchestrator agent surfaces the
question to the human, receives the decision in the agent session, and calls
`zenfd_resolve_review`. The Web Cockpit only displays `AWAITING_REVIEW` and its
reason, details, and suggested options as read-only state.

## Worktree and merge policy

Worktree lifecycle stays in the domain (`worktree_path`, `AWAITING_MERGE`,
`CONFLICT`), but MCP does not perform git merges.

- `COMPLETE` with a worktree → `AWAITING_MERGE`; without a worktree → `COMPLETED`.
- `COMPLETE` + `merge_outcome=MERGED` → `COMPLETED`.
- `COMPLETE` + `merge_outcome=CONFLICT` → `CONFLICT`.
- `FAIL` → `FAILED`; `RESET`/`CANCEL` apply to `RUNNING` or `SUSPENDED`.

Merge outside MCP, then record `merge_outcome` via `zenfd_transition_task`.

## Live verification

1. Apply the client configuration. For auto-supported clients, verify it with:

   ```bash
   apps/daemon/bin/zenfd mcp check --client opencode
   ```

   For manual clients such as VS Code, confirm the config file parses and
   contains the `zenfd` entry.

2. Restart the AI client session so it discovers the `zenfd` tools.
3. Confirm the surface exposes exactly 7 tools listed above.
4. Call a read-only tool such as `zenfd_get_task` and confirm the daemon responds.
5. Optionally create a disposable plan and task, claim it, then complete it via
   `zenfd_transition_task` to confirm SQLite WAL writes.
6. To exercise review, have the worker call `zenfd_request_review` and exit; the
   main agent relays the question to the human and calls `zenfd_resolve_review`.
   The Web Cockpit must only display the review state read-only.

Record the client, config path, tool called, and observed result.
