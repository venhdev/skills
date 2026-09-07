---
name: recon
description: "Survey codebase topology, map module boundaries, and trace data-flow execution paths through progressive read-only disclosure before planning modifications."
disable-model-invocation: true
---

# recon — Codebase Topology & Architecture Cartography Engine

Map repository topology, module seams, and data-flow paths through progressive read-only exploration before architectural planning.

## Domain Engine & Standards

### 1. Progressive Disclosure Strategy
Explore unfamiliar codebases through layered depth expansion:
- **Topography & Entrypoints**: Scan directory layouts, project manifests, and documentation indices (`docs/README.md`, specs, ADRs) to locate boundary entrypoints.
- **Architectural Seams**: Inspect public interfaces, exported contracts, and module boundaries before reading internal implementation logic.
- **Targeted Flow Tracing**: Trace the execution path through active call sites, mapping how data transforms and where state persists.

### 2. Reconnaissance Guardrails
- **Confined Search**: Exclude build caches, package vendor directories, and lockfiles (`build/`, `dist/`, `node_modules/`, `vendor/`, `*.lock`). Focus queries on source roots.
- **Call-Site Focus**: Inspect source files in targeted spans around active call chains; avoid bulk reading large, unrelated implementation files.
- **Upstream First**: Understand caller invariants and state lifecycles before evaluating leaf helper utilities.

### 3. Canonical Codebase Cartography Format
Present findings in the structured, high-density format:

````markdown
# Codebase Cartography: <Subsystem / Target Scope>

## 1. Topography & Architectural Seams
- **Boundary Entrypoints**: <Primary entrypoints: routes, CLI commands, handlers, or UI roots>
- **Core Abstractions & SSOTs**: <Governing interfaces, schemas, or authoritative specifications in docs/>
- **Module Boundaries**: <Division of responsibilities across packages/directories>

## 2. Execution Path & Data Flow
```mermaid
<flowchart TD or LR mapping the observed execution path, decision branches, and persistence using NodeID["[Role] Symbol (path/to/file#L10)"]>
```

## 3. Critical Invariants & Gotchas
- **State & Concurrency Invariants**: <Observed lifecycle rules, mutexes, or transaction boundaries>
- **Specification Drift & Gotchas**: <Discrepancies against authoritative specifications, implicit side-effects, or hidden couplings>

## 4. Pipeline Routing
- `/changeset` ── Execution paths and mutation boundaries are clear; proceed to change planning.
- `/clarify` ── Deliberate architectural trade-offs or component boundaries before planning.
- `/distill <inferred_slug>` ── Reframe problem boundaries if observed architecture reveals XY tensions.
````

## Workflow

**SUB-SKILL:** ssot

### Phase 1: Topography Scan
1. Survey directory structure, project manifests, and canonical specifications, using `discover-docs.sh` in sub-skill `/ssot` when available.
2. Identify primary entrypoints and module boundaries without reading internal implementation logic.

### Phase 2: Seam & Flow Tracing
1. Trace execution paths across identified seams using targeted symbol search and code inspection.
2. Uncover caller relationships, data transformations, and architectural invariants.

### Phase 3: Cartography Delivery & Turn Halt
1. Synthesize findings into the Canonical Codebase Cartography format and deliver strictly in the conversation stream in a single turn. Never create, modify, or delete repository files or propose code diffs.
2. Halt turn immediately to await user review before any downstream architectural commitment.
