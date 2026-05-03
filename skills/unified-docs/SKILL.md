---
name: unified-docs
icon: icon.svg
description: >
  Create, update, and audit project docs with strong lifecycle and current-truth rules.
  Best for plans, specs, ADRs, how-tos, and explanations when the right doc should be
  clarified before writing. Uses frontmatter to keep ownership, status,
  dependencies, and review state manageable.
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
- ADR, plan, spec lifecycle: `contracts/lifecycle.md`.
- Dependency and reciprocal update graph: `contracts/cascade.md`.
- Create doc artifact: `modes/create.md` + matching `templates/authoring/*.md`.
- Two-tier plan workflow: all files in `workflows/create-plan/` if creating a plan doc.
- Read current status: `modes/read.md` + `templates/reports/read-status.md`.
- Maintain existing docs: `modes/maintain.md` + `templates/reports/mutation-report.md`.
- Audit docs health: `modes/audit.md` + `templates/reports/health-report.md`.

## Always preserve these invariants

- `kind` is always a YAML list, even for one value.
- `stale` is never persisted as `kind`.
- `depends-on` targets current prerequisite docs; ADR lineage belongs in `supersedes` / `supersededBy`.
- ADR lineage values are ADR IDs like `ADR-002`, not paths.
- `ssot` is a canonical role, not a folder mandate.
- Assess-only and Read mode never mutate files.
- Maintain mode may repair cascade inverse links only when the relationship is current and unambiguous.

## Default response budget

Use the report template for the chosen mode. Keep output compact: changed files, key findings, followups, and validation only.
