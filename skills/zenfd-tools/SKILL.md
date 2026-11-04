---
name: zenfd-tools
description: "Use when working with ZenForge daemon MCP tools: plan and task creation, task claim, review handshake, state transitions, and task inspection."
---

# zenfd-tools — ZenForge Daemon MCP Surface

Use this card when a task is governed by `zenfd`. The MCP server manages plans and tasks only and is not bound to a repository.

## Roles

- **Main agent**: creates plans/tasks, dispatches workers, relays human decisions, resolves reviews, records merge outcomes.
- **Worker**: claims one task, executes it, reports completion/failure/review, then terminates.

## Lifecycle

`create_plan` → `create_task` → `claim_task` → execute → `transition_task(COMPLETE|FAIL)` or `request_review` → main agent resolves review → fresh claim. Worktree tasks stop at `AWAITING_MERGE`; merge outside MCP, then `transition_task(merge_outcome=MERGED|CONFLICT)`.

## Tools

| Tool | Function | Parameters |
| :--- | :--- | :--- |
| `zenfd_create_plan` | Create/update a plan. | `slug`, `title` |
| `zenfd_create_task` | Create/update a task in a plan DAG. | `plan_id`, `id`, `title`; optional: `description`, `dependencies`, `target_files`, `skills`, `acceptance_criteria`, `verification_commands`, `priority`, `harness`, `worktree_path` |
| `zenfd_claim_task` | Atomically claim a READY task and return its ContextPacket. | `task_id`; optional: `worker_id`, `orchestrator_id`, `worker_role` |
| `zenfd_request_review` | Pause at AWAITING_REVIEW and record a drift question. Worker exits. | `task_id`, `reason`, `details`; optional: `options` |
| `zenfd_resolve_review` | Record the human decision and return to READY. Main agent only. | `task_id`, `response` |
| `zenfd_transition_task` | Batch transition tasks. | `task_ids`, `action` (`COMPLETE\|FAIL\|RESET\|CANCEL`); optional: `reason`, `summary`, `merge_outcome` (`MERGED\|CONFLICT`) |
| `zenfd_get_task` | Read task metadata and ContextPacket without mutation. | `task_id` |

## Guardrails

- `COMPLETE`: worktree → `AWAITING_MERGE`; no worktree → `COMPLETED`.
- `FAIL`: `RUNNING`; `RESET`/`CANCEL`: `RUNNING` or `SUSPENDED`. Recovery is `RESET` then a fresh claim.
- Review decisions never happen in the Web Cockpit. The main agent surfaces the question to the human, then calls `zenfd_resolve_review`.
