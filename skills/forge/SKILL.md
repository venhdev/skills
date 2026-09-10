---
name: forge
description: "Implement approved specifications, vertical slice tasks, or changesets through test-driven verification, delivering clean Git commits."
disable-model-invocation: true
---

# forge — Implementation & Verification Engine

Implement approved specifications, tickets, or changesets through test-driven verification, and deliver clean commits.

## Operating Invariants

- **Scope Discipline**: Confine modifications strictly to target acceptance criteria, approved changeset, and direct mechanical cascades (imports, signatures, tests); forbid adjacent refactoring, cosmetic churn, or scope creep.
- **Pre-Mutation Gate**: Verify target scope, acceptance criteria, and baseline test status before mutating files; forbid creating or editing files on ambiguous or unapproved requirements.
- **Specialist Boundary**: Keep implementation focused strictly on the target task; forbid unsolicited mid-stream execution of on-demand specialist skills (`simplify`, `ssot`, `to-tasks`, `diagnose`).
- **Verification Integrity**: Deliver commit handoffs exclusively after 100% clean typecheck, lint, and test runs; forbid committing or reporting completion on failing verification.
- **Verification Circuit Breaker**: Stop execution immediately upon post-mutation verification failure, report stderr with file citations, and prompt user whether to revert or keep debugging; forbid silent lossy reversions or unguided retry loops.

## Domain Engine & Standards

### 1. Construction Sequencing (Topological Ordering)

Execute filesystem modifications in strict dependency order:

1. `[CREATE]`: New types, interfaces, schemas, and foundational modules.
2. `[MOVE]`: Relocations and renames.
3. `[UPDATE]`: Implementation logic, callers, and cascade updates.
4. `[DELETE]`: Deprecated files and obsolete tests.

### 2. Test-Driven Verification Standards

- **Red Baseline**: For bugfixes, confirm or author a failing test verifying the issue before modifying code. For features, confirm test harness covers target acceptance criteria.
- **Green Implementation**: Apply minimal code necessary to pass the failing baseline.
- **Verification Cascade**: Run verification in order: typecheck → linter → targeted tests → affected subsystem tests.

## Execution Protocol

**SUB-SKILL:** changeset, clarify, ssot

### Phase 1: Readiness Audit & Scope Staging

1. Ingest input (ticket, spec, changeset, or user prompt).
2. Evaluate readiness before mutating workspace:
   - If requirements contain unresolved trade-offs: halt turn and deliberate (via `/clarify` when available).
   - If scope lacks an approved changeset: stage target files and halt turn for approval (via `/changeset` when available).
   - If acceptance criteria and changeset are pre-authorized: proceed directly to Phase 2.

### Phase 2: Construction & Test-Driven Verification

1. Execute file modifications in topological dependency order (`[CREATE]` → `[MOVE]` → `[UPDATE]` → `[DELETE]`).
2. Follow Test-Driven Verification Standards: establish Red baseline, apply minimal Green code, and execute Verification Cascade.
3. Apply Verification Circuit Breaker: If tests fail, stop execution immediately, report stderr with file citations, and prompt whether to debug or revert.
4. Verify all ticket acceptance criteria are satisfied `[x]`.

### Phase 3: Delivery & Git Commit Handoff

1. Present a high-density Delta Summary of modified files and passing test proof. Forbid dumping raw diffs into chat by default.
2. Deliver the ready-to-run Conventional Commit command linking the issue:

   ```bash
   git add <approved_files>
   git commit -m "<type>(<scope>): <subject> (<issue-ref>)" -m "<body>"
   ```

3. Offer execution options: run the commit directly upon approval or provide for manual terminal execution.
