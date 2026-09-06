---
name: changeset
description: Use when architectural decisions or feature requirements need decomposing into exact file modifications, mapping the codebase blast radius, or planning atomic changes before execution.
status: active
---

# changeset — Codebase Blast Radius & Change Planning Engine

Map exact filesystem modifications, trace dependency blast radius, and establish atomic change sets before code mutation begins.

## Operating Principles

- **Read-Only Discipline**: Maintain zero filesystem mutations. Only inspect existing contracts, trace dependencies, and plan file operations.
- **Scope Discipline**: Confine planned changes strictly to the approved scope and mandatory dependency ripples (imports, signatures, tests). Forbid unsolicited refactoring.
- **Four Atomic Actions**: Classify every planned file operation exclusively as `[CREATE]`, `[UPDATE]`, `[MOVE]`, or `[DELETE]`.
- **Hierarchical Visualization**: Render changesets as a clean directory tree grouped by subsystem. State a concise summary per file, appending indented bullet points for complex multi-part modifications.

## Process

**RECOMMENDED PREREQUISITE:** clarify
**OPTIONAL SUB-SKILL:** docs-governance

### Phase 1: Ingestion & Blast Radius Tracing
1. Ingest the governing architectural decisions (such as a `Decision Matrix`), feature requirements, or target scope.
2. Inspect relevant codebase files, type contracts, schema definitions, and tests using available non-mutating capabilities.
3. Trace all files affected by the change (callers, broken imports, unit tests, and documentation). Leverage sub-skill `docs-governance` when available to locate authoritative docs.
4. If the target scope contains unresolved architectural dilemmas or ambiguous requirements, halt immediately and request explicit clarification (recommending sub-skill `clarify` when available) before mapping the changeset.

### Phase 2: Changeset Construction
1. Map each identified file to exactly one of the 4 atomic actions:
   - `[CREATE]`: New files (modules, tests, configurations).
   - `[UPDATE]`: Existing files requiring modifications.
   - `[MOVE]`: Files requiring relocation or renaming.
   - `[DELETE]`: Deprecated or obsolete files slated for removal.
2. State a concise technical summary for each file. Detail complex multi-part contract or invariant modifications using indented bullet points.
3. Construct the hierarchical tree using the canonical Changeset template:

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

### Phase 3: Authorization Gate
1. Present the completed Changeset Tree and halt turn immediately.
2. Forbid executing file edits, writing diffs, or invoking mutating tools within this turn.
3. Wait for explicit affirmative human authorization (e.g., 'proceed', 'approved', 'go') before mutating any workspace files.
4. If the user rejects items, requests scope adjustments, or asks questions, refine the Changeset Tree and halt again.
