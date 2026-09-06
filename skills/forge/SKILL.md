---
name: forge
description: Use when implementing features, bugfixes, or tasks from approved changesets, tickets, or specifications, and driving tests to pass.
status: active
---

# forge — Implementation & Verification Engine

Implement approved specifications, tickets, or changesets, drive code to pass tests, and deliver verified commits.

## Operating Invariants

- **Scope & Acceptance Discipline**: Confine modifications strictly to the target ticket's acceptance criteria or approved changeset.
- **Specialist Boundary**: Keep implementation focused on the target task. Forbid unsolicited mid-stream execution of on-demand specialist skills (`simplify`).
- **Verification Integrity**: Deliver commit handoff exclusively after a 100% clean test and build run.

## Execution Protocol

**RECOMMENDED PREREQUISITE:** changeset
**OPTIONAL SUB-SKILL:** clarify, ssot

### Step 1: Readiness Audit & Upstream Routing
1. Ingest input (ticket, spec, changeset, or user prompt).
2. Evaluate readiness before mutating workspace:
   - If requirements contain unresolved trade-offs: halt turn and request deliberation, recommending sub-skill `clarify`.
   - If multi-file scope lacks an atomic blast radius map: halt turn and map changes, recommending sub-skill `changeset`.
   - If acceptance criteria and scope are defined: extract criteria and target files, then proceed.

### Step 2: Construction & Iterative Verification
1. Execute file modifications in topological dependency order:
   - Complete `[CREATE]` (types, interfaces, modules) and `[MOVE]` before `[UPDATE]`.
   - Complete all updates before `[DELETE]` obsolete files.
   - If authoritative specifications or schemas require reconciliation, leverage sub-skill `ssot`.
2. Run project typechecks, linters, and relevant tests in an iterative feedback loop until all tests pass green.
3. If consecutive identical tool failures or unresolvable blockers occur, halt turn, report stderr and file/line citations, and request human guidance.

### Step 3: Git Commit & Ticket Handoff
1. Verify all ticket acceptance criteria checkboxes are satisfied.
2. Present a high-density Delta Summary of modified files and passing tests.
3. Deliver the ready-to-run Conventional Commit command linking the issue:
   ```bash
   git add <approved_files>
   git commit -m "<type>(<scope>): <subject> (<issue-ref>)" -m "<body>"
   ```
4. Offer execution options: run the commit directly or provide for manual terminal execution.
