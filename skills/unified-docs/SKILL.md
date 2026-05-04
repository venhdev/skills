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

## Args (Quick Triggers)

Pass an arg directly after the skill name to skip mode selection. Args pre-select both mode and action scope. Load only the files needed for that arg.

| Arg | Effect |
|---|---|
| `--create-plan` | Create mode, plan doc type. Skip to plan discovery immediately. |
| `--audit-codebase` | Audit mode, full corpus + organization checks. |
| `--audit-org` | Audit mode, organization-only. Loads `contracts/organization.md`. Skips frontmatter/lifecycle checks. |
| `--maintain-plan` | Maintain mode, plans only. Scans for all plans with status draft or in-progress. No archive. |

If no arg is given, use normal mode routing.

## Lazy-load routing

- Metadata/schema: `contracts/frontmatter.md`.
- Type/kind choice: `contracts/classification.md`.
- ADR, plan, spec lifecycle: `contracts/lifecycle.md`.
- Dependency and reciprocal update graph: `contracts/cascade.md`.
- Folder structure patterns and reorganization: `contracts/organization.md` — load only when triggered (see triggers in the contract).
- Create doc artifact: `modes/create.md` + matching `templates/authoring/*.md`.
- Two-tier plan workflow: all files in `workflows/create-plan/` if creating a plan doc.
- Read current status: `modes/read.md` + `templates/reports/read-status.md`.
- Maintain existing docs: `modes/maintain.md` + `templates/reports/mutation-report.md`.
- Audit docs health: `modes/audit.md` + `templates/reports/health-report.md`.
- **Drift check**: When context indicates a code change or newly shipped feature (not an explicit doc request), use drift-check flow in `modes/maintain.md` to surface docs that may need updating.

## Always preserve these invariants

- `kind` is always a YAML list, even for one value.
- `stale` is never persisted as `kind`.

## Default response budget

Use the report template for the chosen mode. Keep output compact: changed files, key findings, followups, and validation only.
