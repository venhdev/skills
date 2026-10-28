---
name: seam-plan
description: "Formulate safe multi-phase migration strategies using seams, expand-contract patterns, and feature flags to eliminate regression cascades during large-scale refactors."
disable-model-invocation: true
---

# seam-plan — Seam-Driven Refactoring & Expand-Contract Strategy

## Domain Rubric

### 1. Seam Identification (Michael Feathers)
Locate an enabling seam where behavior can be altered without editing existing client source code:
- *Interface / Object Seam*: Introduce an abstraction wrapper around legacy components.
- *Link / Module Seam*: Intercept at the import, dependency injection, or package boundary.

### 2. Expand-Contract Triad (Three Invariant Phases)
1. **Phase 1: Expand (Additive Non-Breaking)**:
   - Introduce new interfaces, data schemas, or shadow services alongside legacy code.
   - *Invariant*: Zero modifications to existing callers. Legacy test suites remain unmodified and GREEN.
2. **Phase 2: Migrate (Progressive Cutover)**:
   - Reroute callers individually to the new abstraction via feature flags, adapters, or dual-writes.
   - Verify behavioral parity between legacy and new paths.
   - *Invariant*: Rollback to legacy path must execute in zero seconds via flag toggle without code redeployment.
3. **Phase 3: Contract (Pruning & Cleanup)**:
   - Remove legacy implementations, delete temporary flags, and purge obsolete branches and shims.
   - Run full regression suites to confirm the clean state.

### 3. State
- `STAGED`: The 3-phase migration strategy and rollback coordinates are staged for human review.
- `UNRESOLVED`: Tightly coupled code prevents seam identification without a foundational redesign.

## Canonical Output Contract

```markdown
# Seam Migration Plan: <Refactor Target> · <STAGED | UNRESOLVED>

Basis: <Frozen contract, architectural direction, or structural audit>
Seam Coordinate: <Repo-relative file and symbol representing the interception point>
Isolation Mechanism: <Interface Adapter | Feature Flag | Shadow Proxy | Branch by Abstraction>

## Phase 1: Expand (Additive Non-Breaking)
- Operations: [CREATE] <New interfaces, adapters, or shadow implementations>
- Invariant: Zero modifications to existing caller signatures; legacy tests remain unmodified and GREEN.

## Phase 2: Migrate (Progressive Cutover)
- Migration Steps:
  1. `<Caller A>`: Rerouted via `<Flag / Seam>` -> Verified via `<Parity Check>`.
  2. `<Caller B>`: Rerouted via `<Flag / Seam>` -> Verified via `<Parity Check>`.
- Fallback Gate: <Observable condition triggering instant rollback to legacy seam>

## Phase 3: Contract (Pruning & Cleanup)
- Operations:
  - [DELETE] `<Legacy implementation files or obsolete branches>`
  - [UPDATE] `<Remove temporary flags, adapters, and redundant shims>`
- Final Verification: <Full regression test suite execution without legacy paths>

Summary: 3 atomic phases planned, rollback exposure confined to Phase 2 flag.
*(omit if clean)* Gap: <Coupling blocker preventing clean seam extraction.>
```

### Routing

- Migration strategy staged, proceed to slice each phase into a DAG → (via /split-tasks).
- Single phase ready for immediate file-impact planning → (via /changeset).
- Cannot isolate a seam without a major architectural pivot → (via /architect-decision).

## Execution Protocol

1. Inspect target refactoring scope and locate available seams.
2. Formulate Phase 1 Expand artifacts (additive interfaces, wrappers, shadow pipes).
3. Plan Phase 2 progressive migration steps with explicit feature flags and rollback gates.
4. Plan Phase 3 contract cleanup operations.
5. Present the 3-phase migration plan and halt turn for human authorization before any code changes.
