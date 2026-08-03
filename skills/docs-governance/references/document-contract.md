# Document Contract

Preferred convention for documentation governance. Use it to audit and propose
normalization; do not silently override repository conventions.

## Contents

- [Precedence](#precedence)
- [Frontmatter](#frontmatter)
- [Status Taxonomy](#status-taxonomy)
- [Type Taxonomy](#type-taxonomy)
- [Common Authority Candidates](#common-authority-candidates)
- [Governance Rules](#governance-rules)

## Precedence

For current work, follow:

1. Applicable repository and directory instructions.
2. The effective repository documentation contract.
3. Existing same-type conventions for gaps in that contract.

Use this preferred contract only to identify gaps and propose normalization. An
approved migration replaces an effective convention only within its confirmed
scope; all other repository conventions remain effective.

## Frontmatter

Frontmatter is optional. Report its absence as a defect only when repository
rules require it. Otherwise, recommend it only when metadata would materially
improve authority, lifecycle, or discoverability.

```yaml
---
title: API v1
description: Legacy API contract. Use for existing API v1 integrations.
status: deprecated; prefer ./api-v2.md
type: ssot, spec
created: 2026-07-01
updated: 2026-08-03
---
```

- Use flat fields in any order.
- Use lowercase field names and ISO dates when practical.
- Resolve paths in `status` relative to the current file.

| Field | Meaning |
| --- | --- |
| `title` | Short display name. |
| `description` | What the document contains and when agents should use it. |
| `status` | Current lifecycle state and optional replacement guidance. |
| `type` | One or more comma-separated document roles. |
| `created` | Date the file was first created. |
| `updated` | Latest meaningful content or lifecycle change. |

## Status Taxonomy

| Status | Use when |
| --- | --- |
| `draft` | Being written or reviewed; not yet effective. |
| `active` | Currently applicable and should be used. |
| `completed` | Finite purpose completed; retain only if still useful. |
| `deprecated` | Valid for legacy use; discouraged for new work. |
| `deprecated; prefer <path>` | Legacy use remains; prefer another document. |
| `superseded by <path>` | Fully replaced and no longer current. |

## Type Taxonomy

Write types inline, separated by a comma and one space:

```yaml
type: ssot, spec
```

Or, for a single role:

```yaml
type: adr
```

Use lowercase unique values. Order does not indicate priority.

| Type | Use when |
| --- | --- |
| `plan` | Plan, roadmap, migration, task, or milestone. |
| `spec` | Durable behavior, requirements, constraints, contract, or policy. |
| `adr` | Architectural decision record. |
| `ssot` | Canonical truth; other documents link instead of copying. |
| `research` | Evidence, experiments, alternatives, or investigation. |
| `runbook` | Operational procedure, recovery, or incident response. |
| `til` | Short practical note from a real issue or task. |

Preserve values established by the effective repository contract. Propose and
obtain user confirmation before introducing or mapping to a custom value. Omit
`type` when none fits.

## Common Authority Candidates

| Category | File | Scope |
| --- | --- | --- |
| Authority candidate | `ARCHITECTURE.md` | Boundaries, components, integrations, constraints. |
| Instruction candidate | `AGENTS.md` | Agent instructions; nested files narrow directory scope. |
| Authority candidate | `BUSINESS.md` | Business rules, actors, terms, invariants, exceptions. |
| Authority candidate | `DESIGN.md` | UI system, components, and design patterns. |
| Authority candidate | `SECURITY.md` | Security policies, constraints, and boundaries. |
| Long-term supporting | `README.md` | Project entry point and documentation map. |
| Long-term supporting | `CHANGELOG.md` | Curated notable change history. |
| Short-term | `MEMORY.md` | Findings, lessons, or handoff notes. |
| Short-term | `CONTEXT.md` | Current task, migration, or initiative context. |

Filenames, folders, categories, and `type: ssot` identify candidates only.
Repository evidence or an approved assignment establishes authority.

## Governance Rules

- Keep one authoritative owner per rule; link instead of copying.
- Treat a draft SSOT as non-authoritative and a superseded document as no longer
  current. Limit a deprecated SSOT to its declared legacy scope.
- Use code, configuration, and tests as evidence of implemented behavior. Use
  authoritative documents for durable intent, rules, constraints, accepted
  future state, and rationale.
- Treat divergence between implementation and accepted documentation as an
  unresolved conflict, not a new truth.
