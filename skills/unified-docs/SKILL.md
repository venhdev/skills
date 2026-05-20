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

## First User-Gated (always)

Before any action:

1. **Determine intent** — What does the user need? If unclear → ask
2. **Validate request** — Does this request match skill capability? (e.g., "maintain API docs" without existing docs → redirect to Create mode)
3. **Detect flow count** — Single flow or multi-flow?
   - Single: proceed to appropriate mode
   - Multi: load `contracts/multi-flow.md`, present plan, confirm all flows before execution
4. **Confirm intent** before proceeding (even before reading files)

All checkpoints must be user-approved before continuing.

---

## Choose Mode

Choose the narrowest mode that satisfies the request. Load only the files named for that mode.

## Modes

- **Read**: answer status/current-truth questions without mutating files. Use `modes/read.md`.
- **Create**: create a new doc from an authoring skeleton. Use `modes/create.md`.
- **Maintain**: update existing docs, lifecycle metadata, or cascade links. Use `modes/maintain.md`.
- **Audit**: report docs health across a target set. Use `modes/audit.md`.

If intent is unclear, start with Read or targeted Audit. Do not run full-corpus audit unless the user asks for full/complete/entire corpus.

## Args (Quick Triggers)

Pass an arg directly after the skill name to skip mode selection. Args pre-select both mode and action scope. Load only the files needed for that arg.

| Arg | Mode | Action | Workflow |
|---|---|---|---|
| `--create-plan` | Create | plan | `workflows/create-plan/` |
| `--audit-codebase` | Audit | codebase | `workflows/audit-codebase/` |
| `--audit-org` | Audit | org | `workflows/audit-org/` |
| `--audit-naming` | Audit | naming | `workflows/audit-naming/` |
| `--maintain-plan` | Maintain | plan | `workflows/maintain-plan/` |

If no arg is given, use normal mode routing.

## Lazy-load routing

Load files only when needed. Each entry declares its trigger condition.

### Core routing
- **Mode selection**: `modes/[mode].md` — triggered by mode name (read/create/maintain/audit)
- **Arg shortcuts**: `SKILL.md` Args table pre-selects mode + workflow; load only the named workflow

### On-demand (lazy)
- **Frontmatter schema**: `contracts/frontmatter.md` — when reading/creating docs
- **Type/kind taxonomy**: `contracts/classification.md` — when creating or auditing
- **Doc type rules**: `contracts/doctypes/[type].md` — load one matching the target type (plan/adr/spec/how-to/explanation/til/reference)
- **Cascade graph**: `contracts/cascade.md` — when checking dependency links
- **Multi-flow detection**: `contracts/multi-flow.md` — only when first user-gated detects multiple independent flows

### Arg-triggered workflows
- `--audit-codebase` → `workflows/audit-codebase/flow.md` (chains org + naming + full audit)
- `--audit-org` → `workflows/audit-org/flow.md`
- `--audit-naming` → `workflows/audit-naming/flow.md`
- `--create-plan` → `workflows/create-plan/flow.md`
- `--maintain-plan` → `workflows/maintain-plan/flow.md`

### Report templates
- Read: `templates/reports/read-status.md`
- Create: authored doc (no report unless requested)
- Maintain: `templates/reports/mutation-report.md`
- Audit: `templates/reports/health-report.md`

## Always preserve these invariants

- `kind` is always a YAML list, even for one value.
- `stale` is never persisted as `kind`.

## Default response budget

Use the report template for the chosen mode. Keep output compact: changed files, key findings, followups, and validation only.
