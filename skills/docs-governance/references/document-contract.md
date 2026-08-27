# Document Contract

Preferred convention for documentation governance. Use it to audit and propose
normalization; do not silently override repository conventions.

## Contents

- [Precedence](#precedence)
- [Frontmatter & Naming](#frontmatter--naming)
- [Status Taxonomy](#status-taxonomy)
- [Recommended Directory Structure](#recommended-directory-structure)
- [Common Authority Candidates](#common-authority-candidates)
- [Authoring Disciplines & Archetype Skeletons](#authoring-disciplines--archetype-skeletons)
- [Authority Boundaries & Governance Rules](#authority-boundaries--governance-rules)

## Precedence

For current work, follow:

1. Applicable repository and directory instructions (`AGENTS.md`).
2. The effective repository documentation contract (`docs/README.md`).
3. Existing same-type conventions for gaps in that contract.

Use this preferred contract only to identify gaps and propose normalization. An
approved migration replaces an effective convention only within its confirmed
scope; all other repository conventions remain effective.

## Frontmatter & Naming

- **File Naming**: Use `kebab-case.md` for all specification files (except standard conventions: `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `LICENSE`, and `0001-*.md`).
- **Language**: English technical documentation for consistent tool, parser, and agent interoperability.
- **Frontmatter**: Optional on standard narrative files (`README.md`); strongly recommended on durable specifications.

```yaml
---
title: Delta Sync Engine
description: Master architectural specification for offline-first two-stage delta sync.
status: active
created: 2026-07-01
updated: 2026-08-03
---
```

- Flat fields, lowercase keys, ISO dates (`YYYY-MM-DD`). Advance `updated` only on meaningful contract changes.
- Document role and scope are defined by Directory Structure and `description`.

| Field | Meaning |
| --- | --- |
| `title` | Short display name. |
| `description` | What the document contains and when agents/developers should use it. |
| `status` | Current lifecycle state and optional replacement guidance. |
| `created` | Date the file was first created (`YYYY-MM-DD`). |
| `updated` | Latest meaningful content or lifecycle change (`YYYY-MM-DD`). |

## Status Taxonomy

| Status | Use when | Body Content Rule |
| --- | --- | --- |
| `draft` | Being written or reviewed; not yet effective. | Full proposed draft. |
| `active` | Currently applicable and authoritative. | Complete living specification. |
| `completed` | Finite purpose completed; retained for historical reference. | Preserved as-is. |
| `deprecated; prefer <path>` | Valid for legacy use; prefer another document. | Retain legacy content with warning banner. |
| `superseded by <path>` | Fully replaced and no longer current. | **Tombstone**: Strip old body; leave pointer link to target. |

## Recommended Directory Structure

When bootstrapping or normalizing project documentation under `docs/`, prefer this clean, scalable layout:

```text
docs/
├── README.md                 # Navigation map, authority boundaries, placement matrix
├── architecture/             # System boundaries, component topology, master index
│   └── standards/            # Cross-cutting engineering standards (data, auth, api, test)
├── specs/ (or domain/)       # Functional domain specifications, business rules, use cases
├── adr/                      # Architectural Decision Records (0001-*.md in MADR format)
├── runbooks/                 # Operational runbooks, deployment, incident recovery
└── lessons-learned/          # Engineering postmortems and root-cause lessons (optional)
```

## Common Authority Candidates

| Category | File | Scope |
| --- | --- | --- |
| Authority candidate | `ARCHITECTURE.md` | Boundaries, components, integrations, constraints. |
| Instruction candidate | `AGENTS.md` | Agent instructions; nested files narrow directory scope. |
| Authority candidate | `BUSINESS.md` | Business rules, actors, terms, invariants, exceptions. |
| Authority candidate | `DESIGN.md` | Design system source of truth for AI-generated UI (colors, typography, spacing, component patterns). |
| Authority candidate | `SECURITY.md` | Security policies, constraints, and boundaries. |
| Authority candidate | `TEST.md` | Verification strategy, test contracts, coverage, and isolation policy. |
| Long-term supporting | `README.md` | Project entry point and documentation map. |
| Long-term supporting | `CHANGELOG.md` | Curated notable change history. |
| Short-term | `MEMORY.md` | Findings, lessons, or handoff notes. |
| Short-term | `CONTEXT.md` | Current task, migration, or initiative context. |

Filenames, folders, directory indices, and titles identify candidates only.
Repository evidence or an approved assignment establishes authority.

## Authoring Disciplines & Archetype Skeletons

### 1. Universal Authoring Disciplines
- **Invariants over Code**: Document mathematical rules, boundary limits, schema constraints, and state machines; avoid pasting volatile implementation code.
- **Tables over Dense Prose**: Use high-density Markdown matrices and ASCII wireframes/pipelines for instant scannability.
- **Deterministic Traceability Keys**: Tag requirements (`BR-*`, `UC-*`), errors (`ERR-*`), events (`EVT-*`), and decisions (`ADR-*`, `LL-*`) for 1:1 code/test mapping.
- **Define Once in SSOT, Link Elsewhere**: Maintain single ownership per fact; cross-reference with direct anchor links (`[Doc § N.N](path#anchor)`).

### 2. Archetype Skeletons
- **Archetype 1: UI & Surface Specs (`ui/`, `screens/`, `experience/`)**: ASCII Viewport Wireframe → Component Interaction Matrix (`Priority | Component | Visibility | Interaction | Route/Action`) → Guardrails & Non-Goals (`❌`).
- **Archetype 2: Domain & Feature Specs (`specs/`, `domain/`, `features/`)**: Data Schema (JSON pseudo-schema) → Domain Invariants (`BR-*`) → State Machine Decision Table (`Scenario | Trigger | Transition | Mutation`) → Standard SSOT Links.
- **Archetype 3: Architecture & Engineering Standards (`architecture/`, `standards/`)**: Context & Boundaries → Data Storage Matrix (`PostgreSQL | SQLite | DTO`) → Multi-Stage Processing Pipelines → REST Endpoints & Error Codes (`ERR-*`).
- **Archetype 4: Workflows & Use Cases (`workflows/`, `use-cases/`, `flows/`)**: Trigger & Preconditions → Step-by-Step Sequence (`Step | Actor | UI Action | System Mutation`) → Offline & Recovery States.
- **Archetype 5: Decisions & Postmortems (`adr/`, `lessons-learned/`, `postmortems/`)**: MADR (`Context → Drivers → Options → Outcome → Consequences`) & 4-Part Postmortem (`Context/Symptoms → RCA → Mitigation → Impact`).

## Authority Boundaries & Governance Rules

- **Bounded Responsibility**: Give each authority a declared scope. A correct fact in the wrong document is an ownership defect.
- **Local Consequences vs. Global Rules**: Subsystem/package READMEs may describe local consequences, but must never redefine upstream global standards.
- **Single Source of Truth**: Never duplicate canonical rules across files; link instead of copying.
- **Lifecycle Authority**: Treat a draft SSOT as non-authoritative and a superseded document as no longer current. Limit a deprecated SSOT to its declared legacy scope.
- **Durable Intent vs. Code**: Use code, configuration, and tests as evidence of implemented behavior; use authoritative documents for durable intent, constraints, and rationale.
- **Divergence as Conflict**: Treat divergence between implementation and accepted documentation as an unresolved conflict, not an automatic new truth.
- **Version Lock SSOT**: Lock runtime, framework, and package versions in exactly one master architecture index (`docs/architecture/README.md`); reference the master index rather than hardcoding versions across subsystem documents.
