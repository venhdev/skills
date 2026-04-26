---
name: unified-docs
icon: icon.svg
description: >
  Fast documentation lifecycle skill for targeted create/update/read/assess tasks
  using one frontmatter contract. Use for ADR supersession, plan updates, SSOT
  checks, dependency/cascade repairs, and narrow documentation triage. Defaults
  to targeted checks unless the user explicitly asks for a full corpus audit.
---

# unified-docs

One fast, checklist-driven docs skill. Default to the narrowest correct pass.

## Execution style

- **Quick read/status**: one doc + replacement/frontmatter only. No scripts.
- **Assess-only triage**: requested target + directly linked docs. No mutation; no full scan.
- **Assess+repair**: apply requested fixes, then validate changed docs.
- **Full audit**: only when user says full/complete/entire corpus.

If intent is unclear, use Assess-only triage.

## Lazy-load references

Load only what the path needs:

- Fast paths: `workflows/fast-paths.md`
- Contract: `contracts/frontmatter.md`
- Docs structure: `contracts/docs-structure.md` for create placement or audit structure concerns
- Quality checklist: `contracts/quality-checklist.md`
- ADR/plan lifecycle only for ADR/plan changes: `workflows/adr-lifecycle.md`, `workflows/plan-lifecycle.md`
- Cascade rules only for dependency/cascade repair: `workflows/cascade-rules.md`
- Background references only when needed: `references/classification.md`, `references/writing-rules.md`

Use `workflows/fast-paths.md` instead of old read/audit workflow patterns.

## Routing table

- Create doc: `references/classification.md` + `contracts/frontmatter.md`; add `contracts/docs-structure.md` only if placement is unclear.
- Normalize or validate metadata: `contracts/frontmatter.md`; run `scripts/check_frontmatter.py` only after edits.
- ADR create/supersede/read-current: `contracts/frontmatter.md` + `workflows/adr-lifecycle.md`.
- Plan update/archive: `contracts/frontmatter.md` + `workflows/plan-lifecycle.md`.
- Dependency or cascade repair: `contracts/frontmatter.md` + `workflows/cascade-rules.md`; validate with `scripts/cascade_targets.py`.
- Style/body cleanup: `references/writing-rules.md` + `contracts/quality-checklist.md`.

## Core rules

- Stale is computed from `lastReviewed` + effective cadence; never persist `kind: stale`.
- `kind` is limited to `plan`, `spec`, `adr`, `ssot`, `draft`, `til`.
- Completed plans promote durable outcomes into a spec; archived plans leave the current dependency graph.
- ADR lineage uses `supersedes` / `supersededBy`, not `depends-on`.
- `depends-on` targets must be current and exist.
- Structure guidance helps keep docs maintainable, but never restructure a project without asking first.
- Treat unusual `ssot` placement as a review signal, not a format failure, unless the content clearly contradicts canonical metadata.
- Assess-only never mutates files.
- Repair validates only changed docs unless full audit is requested.

## Output budget

Default response shape:

- `Mode`
- `Status`
- `Changed files` or `none`
- `Key findings` up to 5
- `Followups` up to 3
- `Validation` brief

Keep answers short but complete. Do not offer next actions unless asked.
