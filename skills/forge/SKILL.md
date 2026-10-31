---
name: forge
description: "Implement authorized changesets backed by proof or official references, verify acceptance criteria, and report completion with a concise commit message."
disable-model-invocation: true
---

# forge — Changeset Implementation & Verification

## Operating Invariants

- **Authorized Scope & Clean Replacement**: Implement the authorized changeset and necessary cascades while actively removing superseded logic, dead helpers, and obsolete tests directly replaced by the solution; purge related transient scratch artifacts (`.agents/scratch/`) and temporary worktrees once implementation and verification reach `DONE`; preserve unrelated work and exclude opportunistic refactoring.
- **Direction Fidelity**: Correct implementation errors within the agreed direction. Stop mutations when success requires significant changes to direction, contracts, risk, or blast radius; report the redesign gap rather than choosing a replacement design.
- **Verification Integrity**: Require every acceptance criterion and required check to pass before `DONE`; preserve failing evidence and existing edits rather than silently reverting or weakening assertions.
- **Safe Rollback**: Use `git reset --soft HEAD~1` when undoing local commits to preserve uncommitted deltas in the staging index.
- **Environment & Secret Isolation**: Never inspect or modify secret files (`.env*`, `*.key`, `*.pem`); ground test execution and linter invocations in `.agents/environment.md`.
- **Delivery Boundary**: Report implementation and verification with a proposed commit message; keep committing, merging, and publishing outside this skill's completion scope.

## Domain Rubric

### Input & Readiness

Receive an authorized Changeset Tree, acceptance criteria, constraints, and supporting solution proof or applicable official references. An embedded changeset from a task or audit is valid input; a prior `/changeset` invocation is not required.

Inspect the current checkout, callers, contracts, baseline checks, and supporting evidence. Official references must support the load-bearing mechanics in this environment; a generic documentation link is not proof of integration. Reconcile planned files without broadening the accepted direction.

Determine required verification from project rules and acceptance criteria before edits. Disclose unrelated baseline failures; if a required check cannot pass without out-of-scope repair, report `UNRESOLVED`. Keep the required set intact rather than dropping a failing check to reach `DONE`.

### Construction & Verification

- Order `[CREATE]`, `[MOVE]`, `[UPDATE]`, and `[DELETE]` by dependencies, keeping callers and interfaces consistent.
- Establish the exact failing baseline for a bugfix; for a feature, cover the agreed acceptance criteria before implementation. Use meaningful existing checks for behavior-preserving changes rather than manufacturing shallow tests.
- Apply minimal implementation, then run applicable typecheck, lint, targeted tests, and affected subsystem checks.
- Continue fixing ordinary compiler, linter, and test failures within scope. Distinguish implementation mistakes from invalid design assumptions and unrelated baseline failures.
- Treat import, signature, and caller updates as small cascades only when they preserve agreed behavior outside the target. New ownership, schema semantics, or substantial scope expansion requires redesign.
- For tracked work, follow the configured task lifecycle: claim eligible work before edits and mark it complete only after required verification. Preserve dependency gates; task tracking is not a substitute for execution authorization.
- Purge related scratch artifacts (`.agents/scratch/`) and temporary worktrees upon verified completion.

### State

- `STAGED`: An existing changeset is ready, but execution authorization is still missing.
- `IN_PROGRESS`: Implementation is underway with no blocker requiring redesign; this is a progress state, not a reason to stop ordinary correction.
- `DONE`: Implementation, acceptance criteria, and all required verification pass for the authorized scope.
- `UNRESOLVED`: A blocker prevents the expected result, including stale design or evidence, unavailable required verification, or a change that would exceed the agreed direction.

## Canonical Output Contract

```markdown
# Implementation Report: <Scope> · <STAGED | IN_PROGRESS | DONE | UNRESOLVED>

<Short-to-medium conclusion and material delta; repo-relative file pointers.>

Verification: <Executed checks and results; identify blocked or baseline failures.>
Remaining: <Blocker, redesign decision, or unfinished work; omit if none.>
Checkout: <Partial edits retained when unresolved; omit if no partial edits.>

Commit message: <Include only when DONE.>
<type>(<scope>): <observable change>
<Brief implementation detail and why it matters.>
```

### Routing

- Unresolved requirements prevent the agreed result → (via /clarify).
- The direction must change to achieve the outcome → (via /architect-decision).
- The direction remains settled but file impact needs restaging → (via /changeset).
- Missing toolchain or failing test datastores prevent verification → (via /env-doctor).

## Execution Protocol

1. Inspect readiness. If a direction or required changeset is missing, report `UNRESOLVED` and hand off to design or staging. If only authorization is missing, report `STAGED` and halt without edits. Honor authorization already granted for the concrete scope.
2. Implement and verify according to the rubric. If a material blocker appears, stop mutations, retain evidence and partial edits, and report `UNRESOLVED`; otherwise continue to `DONE`.
3. Deliver the report and commit message. Keep redesign outside the original implementation scope.
