---
name: unified-docs
icon: icon.svg
description: >
  Documentation lifecycle hub for reading, creating, maintaining, and auditing
  project docs with consistent frontmatter, type/kind classification, ADR and plan
  lifecycle handling, stale-doc checks, SSOT/spec behavior, and dependency cascade
  metadata. Use whenever the user asks to create docs, check whether docs are
  current, supersede ADRs, update plans, normalize frontmatter, audit docs health,
  or repair depends-on/updates chains.
---

# unified-docs

Choose the narrowest mode that satisfies the request. Load only the files named for that mode.

## Modes

- **Read**: answer status/current-truth questions without mutating files. Use `modes/read.md`.
- **Create**: create a new doc from an authoring skeleton. Use `modes/create.md`.
- **Maintain**: update existing docs, lifecycle metadata, or cascade links. Use `modes/maintain.md`.
- **Audit**: report docs health across a target set. Use `modes/audit.md`.

If intent is unclear, start with Read or targeted Audit. Do not run full-corpus audit unless the user asks for full/complete/entire corpus.

## Lazy-load routing

- Metadata/schema: `contracts/frontmatter.md`.
- Type/kind choice: `contracts/classification.md`.
- ADR, plan, spec, stale lifecycle: `contracts/lifecycle.md`.
- Dependency and reciprocal update graph: `contracts/cascade.md`.
- Create doc artifact: `modes/create.md` + matching `templates/authoring/*.md`.
- Read current status: `modes/read.md` + `templates/reports/read-status.md`.
- Maintain existing docs: `modes/maintain.md` + `templates/reports/mutation-report.md`.
- Audit docs health: `modes/audit.md` + `templates/reports/health-report.md`.

## Always preserve these invariants

- `kind` is always a YAML list, even for one value.
- `reviewCadence` is optional; stale status is computed from `lastReviewed` plus effective cadence.
- `stale` is never persisted as `kind`.
- `depends-on` targets current prerequisite docs; ADR lineage belongs in `supersedes` / `supersededBy`.
- ADR lineage values are ADR IDs like `ADR-002`, not paths.
- `ssot` is a canonical role, not a folder mandate.
- Assess-only and Read mode never mutate files.
- Maintain mode may repair cascade inverse links only when the relationship is current and unambiguous.

## Default response budget

Use the report template for the chosen mode. Keep output compact: changed files, key findings, followups, and validation only.
