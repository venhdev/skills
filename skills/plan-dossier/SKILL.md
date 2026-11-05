---
name: plan-dossier
description: "Initialize, track, verify, and govern multi-phase living plan packages, execution states, and drift amendments in scratch."
disable-model-invocation: true
---

# plan-dossier — Living Plan Package & Lifecycle Custodian

## Operating Invariants

- **Scratch Exclusivity & DB SSOT**: Mutate plan artifacts inside `.agents/scratch/plans/<slug>/`; never modify production source code or permanent `docs/` directly (delegate durable promotion to `/ssot`). In repositories governed by `zenfd` (ZenForge Daemon), persist the living plan with `plans(action=create)` and the whole task DAG in one batched `tasks(action=create)` call.
- **Authorization Gate on Init**: Stage the planned dossier structure and halt for affirmative human authorization before creating files on disk; honor prior explicit approval.
- **Evidence-Backed Verification**: Transition phase status to `VERIFIED` strictly when supported by concrete proof (automated test command exit code 0 and git commit SHA); never accept unverified verbal claims of completion.
- **Drift Breaker**: Transition phase status to `DRIFTED` and halt implementation immediately upon detecting architectural divergence between code reality and staged contracts; require caller re-planning before amending the plan.
- **Prompt-Embedded Handshake**: When authoring task specifications and prompts for ephemeral subagent workers, embed the explicit handshake directly in the task prompt (`claim_task` → execute → `transition_task` or `review(action=request)`). Reserve `review(action=resolve)` for the main orchestrator agent, which relays the human decision; workers request review and exit.
- **Active Plan Preservation**: Preserve plan directories containing active phases (`PLANNED`, `CLAIMED`, `IN_PROGRESS`, `DRIFTED`); mark initiative `COMPLETED` only when all phases are `VERIFIED`.

## Domain Rubric

### 1. Operations
- `init <slug>`: Bootstrap `.agents/scratch/plans/<slug>/` with `README.md` (Master State Dashboard) and embed supplied artifacts (`01-contracts.md`, `02-seam-strategy.md`, `03-changeset.md`).
- `claim <slug> <phase-id>`: Check in worker, assign branch or worktree, and transition phase to `CLAIMED`.
- `verify <slug> <phase-id>`: Record verification command output and commit SHA, transitioning phase to `VERIFIED`.
- `drift <slug> <phase-id>`: Log divergence evidence, halt downstream mutations, and transition phase to `DRIFTED`.
- `amend <slug>`: Update contracts, seams, or phase topologies to revision `v1.N` after caller re-planning.

### 2. Execution Agility & Phase Lifecycle
- *Scale-Adaptive Execution*: Localized phases (≤ 3 files, single verification turn) execute directly in-turn via `/forge`; multi-layer complex phases decompose into task DAGs via `/split-tasks`.
- *Phase States*:
  - `PLANNED`: Staged in dashboard, ready for dependency fulfillment.
  - `CLAIMED`: Assigned to active worker or branch session.
  - `IN_PROGRESS`: Actively undergoing implementation.
  - `VERIFIED`: Formally proven complete with green verification and commit SHA.
  - `DRIFTED`: Implementation blocked by architectural mismatch; mutations paused.
  - `REJECTED`: Discarded or superseded by caller.

## Canonical Output Contract

```markdown
# Plan Dossier: <Initiative Slug> · <ACTIVE | COMPLETED | DRIFTED | REJECTED>

**Target**: `.agents/scratch/plans/<slug>/` (Revision: `<v1.N>`)
**Basis**: <Frozen contracts, seam strategy, or ADR reference>

## Master State Dashboard

| Phase ID | Scope & Seam | Status | Branch / Owner | Verification Evidence |
| :--- | :--- | :--- | :--- | :--- |
| P-01 | <Deliverable or Expand seam> | <STATUS> | <branch | '-'> | <commit SHA / test proof | '-'> |
| P-02 | <Deliverable or Migrate seam>| <STATUS> | <branch | '-'> | <commit SHA / test proof | '-'> |

Summary: <N> phases (<V> verified, <I> in-progress, <P> planned, <D> drifted).
*(omit if clean)* Gap: <Divergence root cause or blocker requiring caller decision.>
```

*Autonomous Micro-Token Contract*:
```text
PLAN | slug: <slug> | rev: <v1.N> | status: <STATUS> | active: <phase-id> | verified: <V>/<N>
```

### Routing

- Complex phase scope needs vertical task decomposition → (via /split-tasks).
- Claimed phase ready for immediate in-turn implementation → (via /forge).
- Phase drifted requiring architectural redesign → (via /architect-decision) or clarification → (via /clarify).
- All phases VERIFIED; promote durable specifications to documentation authority → (via /ssot).
- Completed initiative ready for scratch cleanup → confirm with /sweep, then hide it with `plans(action=archive)`.

## Execution Protocol

1. **Ingest & Parse**: Ingest operation (`init`, `claim`, `verify`, `drift`, `amend`), initiative slug, and target phase ID or proof payload.
2. **Authorize & Mutate**:
   - For `init`: Stage dossier layout, halt for confirmation, and atomically write `.agents/scratch/plans/<slug>/README.md` and packaged artifacts upon approval. When `zenfd` is active, register the plan and the complete task DAG in SQLite SSOT with `plans(action=create)` then one batched `tasks(action=create)`. If a phase needs design references, pass them in that task's `media[]` while the task is still `READY`/`PENDING`.
   - For `claim / verify / drift / amend`: Read existing dashboard, validate state transition rules, apply atomic update to `README.md`, and reconcile overall initiative state.
3. **Emit & Report**: Render Canonical Output Contract (or micro-token for autonomous runners) and halt turn.
