---
name: recon
description: "Map codebase topology, module seams, and execution call chains."
disable-model-invocation: true
---

# recon — Codebase Topology & Architecture Cartography Engine

Map repository topology, module seams, and execution call chains through progressive, grounded exploration.

## Canonical Codebase Cartography Format

```markdown
# Codebase Cartography: <Subsystem / Target Scope>

## 1. Topography & Architectural Seams
- **Entrypoints**: <Primary intake points, commands, public APIs, or event consumers with file pointers>
- **Core Contracts & Schemas**: <Key interfaces, data types, state schemas, or governing specifications>
- **Module Boundaries**: <Key directories/packages and their distinct architectural responsibilities>

## 2. Key Execution Flows
### Flow: <Primary Flow Name, e.g., Data Intake / Transformation / Render Pipeline>
1. `<Symbol>` ([<file>#L<N>](file:///path/to/file#L10)) ── <Trigger or input intake>
2. `<Symbol>` ([<file>#L<N>](file:///path/to/file#L50)) ── <Intermediate dispatch, orchestration, or validation>
3. `<Symbol>` ([<file>#L<N>](file:///path/to/file#L90)) ── <Core state transition, persistence, or terminal execution>
*(Add secondary flow if distinct execution path exists)*

## 3. Invariants & Gotchas
- **Invariants (Must Preserve)**: <Core rules, transaction boundaries, or state lifecycles that cannot be broken>
- **Gotchas (Watch Out)**: <Hidden side-effects, spec drift, or surprising couplings; state 'None observed' if clean>

## 4. Pipeline Routing
- `/changeset` ── Execution paths and mutation boundaries are clear; proceed to change planning.
- `/clarify` ── Discovered architectural tensions, trade-offs, or multiple viable implementation paths.
```

## Delegation & Synthesis Protocol

**SUB-SKILL:** changeset, clarify

1. **Scope Partitioning & Dispatch (Fan-out)**:
   - Launch 1 `research` subagent for an isolated subsystem, or 2–3 concurrent subagents partitioned by architectural layer for broad cross-boundary flows.
   - Instruct subagents to focus strictly on architectural transitions and module seams, skipping trivial leaf utilities and raw directory dumps.
   - Mandate the **Canonical Codebase Cartography Format** above as the strict output contract, requiring exact clickable file pointers (`file:///path/to/file#L<N>`) for all cited symbols.

2. **Synthesis & Turn-Halt Gate (Fan-in)**:
   - Reconcile subagent findings into a single unified cartography artifact.
   - Present the completed cartography into the conversation stream and halt turn immediately. Never propose code diffs, write mutations, or create files on disk.
