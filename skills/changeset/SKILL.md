---
name: changeset
description: "Map exact filesystem modifications and trace dependency blast radiuses for approved features or architectural decisions before code mutation."
disable-model-invocation: true
---

# changeset — Codebase Blast Radius & Change Planning Engine

Map exact filesystem modifications, trace dependency blast radius, and establish atomic change sets before code mutation begins.

## Operating Invariants

- **Read-Only Discipline**: Maintain zero filesystem mutations and inspect existing contracts; forbid creating, editing, or deleting files on disk.
- **Scope Discipline**: Confine planned changes strictly to approved scope and mandatory dependency ripples (imports, signatures, tests); forbid unsolicited refactoring or adjacent cleanup.
- **Pre-Mutation Gate**: Deliver the Changeset Tree and halt turn immediately; forbid executing file edits, writing diffs, or invoking mutating tools within this turn.

## Domain Engine & Standards

### 1. Four Atomic Actions

Classify every planned file operation exclusively into one of 4 categories:

- `[CREATE]`: New files (modules, tests, configurations).
- `[UPDATE]`: Existing files requiring modifications.
- `[MOVE]`: Files requiring relocation or renaming.
- `[DELETE]`: Deprecated or obsolete files slated for removal.

### 2. Blast Radius & Guardrails

- **Callers & Ripple Tracing**: Trace direct and indirect call sites, interface implementations, and tests affected by signature or contract modifications. Reconcile against authoritative specifications using sub-skill `ssot`.
- **Guardrails (When NOT to Touch)**:
  - Preserve untouched callers if interface changes remain fully backward-compatible.
  - Forbid bundling incidental formatting, opportunistic lint fixes, or unrelated refactorings.
  - Forbid speculative or phantom file operations lacking approved requirements.

### 3. Canonical Changeset Template

```text
# Changeset: <Feature / Scope Name>

📁 <subsystem_or_directory>/
├── 📄 <filename_1>
│   └── [<ACTION>] <Summary of change>.
├── 📄 <filename_2>
│   └── [<ACTION>] <Summary of change>:
│       • <Specific change detail, contract modification, or invariant>.
│       • <Specific change detail or cascade update>.
└── 📁 <subsystem_2>/
    └── 📄 <filename_3>
        └── [<ACTION>] <Summary of change>.

Summary: <N> files affected (<C> created, <U> updated, <M> moved, <D> deleted).
```

## Execution Protocol

**SUB-SKILL:** clarify, ssot

### Phase 1: Ingestion & Blast Radius Tracing

1. Ingest the governing architectural decisions (such as a `Decision Matrix`), feature requirements, or target scope.
2. Inspect relevant codebase files, type contracts, schema definitions, and tests using non-mutating capabilities (or delegated exploratory subagents for wide codebases).
3. Trace all files affected by the change (callers, broken imports, unit tests, and documentation). Leverage sub-skill `ssot` when available to locate authoritative docs.
4. If the target scope contains unresolved architectural dilemmas or ambiguous requirements, halt immediately and request explicit clarification (recommending sub-skill `clarify` when available) before mapping the changeset.

### Phase 2: Changeset Construction

1. Map each identified file strictly to one of the 4 Atomic Actions defined in the Domain Engine.
2. Enforce Blast Radius Guardrails: exclude backward-compatible callers and cosmetic churn.
3. State a concise technical summary per file; detail multi-part contract modifications with indented bullets.
4. Render the output strictly adhering to the Canonical Changeset Template.

### Phase 3: Authorization Gate

1. Present the completed Changeset Tree and halt turn immediately.
2. Forbid executing file edits, writing diffs, or invoking mutating tools within this turn.
3. Wait for explicit affirmative human authorization (e.g., 'proceed', 'approved', 'go').
4. Upon approval, hand off downstream to execution:
   - For direct implementation: route to `/forge`.
   - For complex multi-slice plans: route to `/to-tasks`.
5. If the user rejects items, requests scope adjustments, or asks questions, refine the Changeset Tree and halt again.
