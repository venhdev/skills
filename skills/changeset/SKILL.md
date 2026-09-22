---
name: changeset
description: "Map exact filesystem modifications and trace dependency blast radiuses for approved features or architectural decisions before code mutation."
disable-model-invocation: true
---

# changeset — Codebase Blast Radius & Change Planning Engine

Map exact filesystem modifications, trace dependency blast radius, and establish atomic change sets before code mutation begins.

## Operating Invariants

- **Read-Only Discipline**: Maintain zero filesystem mutations and inspect existing contracts; forbid creating, editing, or deleting files on disk.
- **Scope Discipline**: Confine planned changes strictly to approved scope and mandatory dependency cascades (imports, signatures, tests); forbid unsolicited refactoring, formatting churn, or adjacent cleanup.
- **Pre-Mutation Gate**: Deliver the Changeset Tree and halt turn immediately; forbid executing file edits, writing diffs, or invoking mutating tools within this turn.

## Domain Rubric

### 1. Four Atomic Actions

Classify every planned file operation exclusively into one of 4 discrete actions:

- `[CREATE]`: New files (modules, tests, schemas, configurations).
- `[MOVE]`: Relocating or renaming existing files.
- `[UPDATE]`: Modifying existing logic, signatures, or cascade dependencies.
- `[DELETE]`: Removing deprecated, obsolete, or superseded files.

### 2. Blast Radius & Guardrails

- **Cascade & Ripple Tracing**: Trace direct and indirect callers, interface implementations, and tests affected by contract changes. Reconcile against authoritative specifications (via `/ssot` when available).
- **Guardrails (When NOT to Touch)**:
  - Preserve untouched callers if interface changes remain fully backward-compatible.
  - Forbid bundling incidental formatting, opportunistic lint fixes, or speculative files.
  - Forbid phantom or speculative file operations lacking approved requirements.

## Canonical Output Contract

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

1. Ingest governing requirements, Decision Matrix, or architectural specifications. Reconcile with authoritative documentation (via `/ssot` when available).
2. Inspect relevant codebase files, type contracts, schemas, and tests using non-mutating file tools.
3. Trace dependency ripple: identify all callers, broken imports, and affected test suites.
4. If target scope contains unresolved architectural tensions or ambiguous requirements: halt turn immediately and deliberate (via `/clarify` when available).

### Phase 2: Changeset Construction

1. Map each identified file strictly to one of the 4 Atomic Actions (`[CREATE]`, `[MOVE]`, `[UPDATE]`, `[DELETE]`).
2. Enforce Blast Radius Guardrails: exclude backward-compatible callers and cosmetic churn.
3. State concise technical summaries per file, using indented bullet points for multi-part contract modifications.
4. Append the mandatory blast radius summary line (`Summary: <N> files affected (...)`).

### Phase 3: Authorization Gate

1. Present the completed Changeset Tree into the conversation stream and halt turn immediately. Never modify workspace files, write code diffs, or execute mutating commands without affirmative user authorization.
2. Upon affirmative approval:
   - For direct implementation: route to `/forge`.
   - For multi-task decomposition: route to `/to-tasks`.
