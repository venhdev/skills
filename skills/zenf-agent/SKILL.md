---
name: zenf-agent
description: "Define ZenForge WORKER personas, goals, start/stop conditions, schedules, and observe MAIN/SUBAGENT/WORKER sessions without driving them from the web."
disable-model-invocation: true
---

# zenf-agent — Agent Definition and Session Front Door

Use this card when the caller wants to create or tune a worker persona, bind a
persona to tasks, inspect active sessions, or decide which session type a runner
should register. The MVP stores definitions and observes execution; an external
runner performs the work.

## Session Separation

| Type | Owner | Web access |
| :--- | :--- | :--- |
| `MAIN` | The primary agent session | Read-only |
| `SUBAGENT` | A child session spawned by the main agent | Read-only |
| `WORKER` | An external runner driven by beat/schedule/manual dispatch | Definition CRUD; session observed read-only |

No web action may start, stop, claim, or transition a session. The web only lists
sessions and manages WORKER definitions.

## Prerequisites

- Read `zenf-tools` for exact `agents` and `sessions` payloads.
- Decide the persona: `EXPLORER` (investigate), `CODER` (implement), `REVIEWER`
  (verify), or `CUSTOM`.
- Decide the mode: `HEADLESS` (runner executes without a human at the terminal)
  or `INTERACTIVE` (human participates).
- Choose a write surface:
  - **MCP** (preferred when connected): `agents`, `sessions`, `tasks`.
  - **HTTP** (MCP unavailable): `/api/v1/agents`, `GET /api/v1/sessions`.
  - **Files only** (daemon unreachable): produce a definition JSON block and a
    handoff note; do not claim the agent is registered.

## Step Protocol

1. **State the goal.** One outcome-oriented sentence. Avoid naming tools or
   implementation steps in the goal.
2. **Write start conditions.** Observable predicates only: task status, claim
   state, plan window, schedule, dependency completion.
3. **Write stop conditions.** Explicit terminal predicates: `AWAITING_REVIEW`,
   verification failure, budget/time limit, blocking dependency.
4. **Choose schedule and runner.** `schedule` is a cron/beat expression or empty
   for manual dispatch. `runner` is a label the external supervisor understands.
5. **Select skills and model.** List only skills the runner can resolve. Leave
   `model` empty to use the runner default.
6. **Create or update.**
   - MCP: `agents(action=create, slug=..., name=..., persona=..., mode=..., ...)`.
   - HTTP: `POST /api/v1/agents`; update with `PATCH /api/v1/agents/{slug}`.
7. **Bind to tasks when needed.** Set `agent_slug` on task create or update while
   the task is `READY`/`PENDING`. Binding is advisory for the runner, not a
   claim.
8. **Register the session.** The runner calls
   `sessions(action=register, id=..., session_type=WORKER, agent_slug=..., parent_id=...)`.
   The main agent registers `MAIN`/`SUBAGENT` sessions.
9. **Heartbeat and close.** `sessions(action=heartbeat)` updates liveness;
   `sessions(action=close)` marks `TERMINATED`. Stale `ACTIVE` sessions indicate
   a crashed runner, not a web-actionable state.
10. **Verify.** Read the definition back; list sessions filtered by
    `agent_slug` and `active_only=true`; confirm counts in the cockpit Agents
    view.

## Definition Rubric

| Field | Required | Rule |
| :--- | :---: | :--- |
| `slug` | yes | Stable, lowercase, unique |
| `name` | yes | Human-readable |
| `persona` | yes | `EXPLORER`, `CODER`, `REVIEWER`, or `CUSTOM` |
| `mode` | yes | `HEADLESS` or `INTERACTIVE` |
| `goal` | recommended | One outcome, no implementation prescription |
| `start_conditions` | recommended | Observable and testable |
| `stop_conditions` | recommended | Includes failure and review exits |
| `schedule` | optional | Empty means manual/beat dispatch |
| `skills` | optional | Resolvable skill names only |
| `runner` | optional | Label matched by the external supervisor |
| `enabled` | yes | Disabled definitions are hidden from dispatch |

## MCP-Optional Matrix

| Need | MCP | HTTP fallback |
| :--- | :--- | :--- |
| Create definition | `agents(action=create)` | `POST /api/v1/agents` |
| Read/list | `agents(action=get/list)` | `GET /api/v1/agents[/{slug}]` |
| Update | `agents(action=update)` | `PATCH /api/v1/agents/{slug}` |
| Delete | `agents(action=delete)` | `DELETE /api/v1/agents/{slug}` |
| Sessions | `sessions(action=list)` | `GET /api/v1/sessions?agent_slug=&active_only=` |
| Bind to task | `tasks(action=update, agent_slug=...)` | task context update |

## Guardrails

- **Definition is not execution.** Creating an agent does not start anything.
- **No web-driven sessions.** MAIN/SUBAGENT are read-only; WORKER sessions are
  registered by the runner.
- **Do not reuse a slug for a different persona.** Create a new definition.
- **No secrets in fields.** Model credentials belong to the runner environment,
  never to `goal`, `runner`, or `skills`.
- **Delete is safe for definitions but does not remove historical sessions.**
  Disable first when unsure.
- **One persona per agent.** If a workflow needs coder and reviewer, define two
  agents and let the DAG sequence them.

## Output

```markdown
## Agent
- slug / name / persona / mode / enabled
- goal
- start: <conditions>
- stop: <conditions>
- schedule / runner / model
- skills

## Sessions
| id | type | status | last seen |
| :--- | :--- | :--- | :--- |

## Verification
- definition read-back: pass/fail
- active session count: <n>
```

## Handoffs

- Need exact definition or session payloads → (via /zenf-tools).
- Need to place the agent on a plan/task DAG → (via /zenf-project).
- Need to attach a persona charter or operating guide → (via /zenf-docs).
- Need to design the runner or dispatch loop → (via /architect-decision).
