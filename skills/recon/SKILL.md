---
name: recon
description: "Map codebase topology, module seams, and execution call chains."
disable-model-invocation: true
---

# recon — Codebase Topology & Architecture Cartography Engine

Map repository topology, module seams, and execution call chains through progressive exploration.

## Delegation & Synthesis Protocol

1. **Dispatch (Fan-out)**:
   - Launch 1 `research` subagent for a single module, or 2–3 concurrent subagents partitioned by layer for broad cross-service flows.
   - In each subagent prompt, supply:
     - The target scope and any known entrypoints, files, or symbols from the session.
     - The **Canonical Codebase Cartography Format** below as the required output contract.

2. **Synthesis & Halt (Fan-in)**:
   - Reconcile subagent findings into a single unified cartography artifact.
   - Emit the final cartography into the stream and halt turn immediately. Never propose diffs or edit files.

## Canonical Codebase Cartography Format

```markdown
# Codebase Cartography: <Subsystem / Target Scope>

## 1. Topography & Architectural Seams
- **Entrypoints**: <Primary routes, CLI commands, handlers, or event consumers with file pointers>
- **Core Contracts & Schemas**: <Key interfaces, database models, or governing doc specifications>
- **Module Boundaries**: <Key directories/packages and their distinct responsibilities>

## 2. Key Execution Flows
### Flow: <Primary Flow Name, e.g., Request Handling / Ingestion Pipeline>
1. `<Symbol>` ([<file>#L<N>](file:///path/to/file#L10)) ── <Trigger or input intake>
2. `<Symbol>` ([<file>#L<N>](file:///path/to/file#L50)) ── <Intermediate validation, dispatch, or middleware>
3. `<Symbol>` ([<file>#L<N>](file:///path/to/file#L90)) ── <Core transformation, state transition, or persistence>
*(Add secondary flow if distinct execution path exists, e.g., background worker or event consumer)*

## 3. Invariants & Gotchas
- **Invariants (Must Preserve)**: <Core rules, transaction boundaries, or state lifecycles that cannot be broken>
- **Gotchas (Watch Out)**: <Hidden side-effects, spec drift, or surprising couplings; state 'None observed' if clean>

## 4. Pipeline Routing
- `/changeset` ── Execution paths and mutation boundaries are clear; proceed to change planning.
- `/clarify` ── Discovered architectural tensions, trade-offs, or multiple viable implementation paths.
```
