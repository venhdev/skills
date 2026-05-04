# Audit Mode

Use this mode when the user asks for docs health, broken links, dependency chains, duplicate/orphan docs, or cascade issues.

## Scope

Default to targeted audit:

- requested doc
- directly linked `depends-on` targets
- directly listed `updates` targets
- obvious incoming references if needed

Run full corpus audit only when the user says full, complete, entire corpus, or equivalent.

### Arg-triggered routing

| Arg | Workflow |
|---|---|
| `--audit-org` | `workflows/audit-org/` (organization check only) |
| `--audit-naming` | `workflows/audit-naming/` (naming check with scope prompt if needed) |
| `--audit-codebase` | `workflows/audit-codebase/` (full corpus: org + naming + frontmatter/lifecycle/cascade) |

## Corpus boundaries

Audit project documentation, not tool configuration, vendored content, generated workspaces, or assistant runtime files. Exclude these paths unless the user explicitly asks to audit them:

- hidden tool/runtime directories: `.claude/**`, `.agents/**`, `.agent/**`, `.codex/**`, `.cursor/**`, `.github/**`, `.vscode/**`, `.idea/**`
- generated or temporary workspaces: `tmp/**`, `temp/**`, `.tmp/**`, `.cache/**`, `dist/**`, `build/**`, `coverage/**`, `node_modules/**`, `vendor/**`
- installed or synced skill/agent definitions unless the user is auditing the skill system itself

If a full-corpus audit sees many Markdown files outside project docs, report them as excluded scope rather than missing frontmatter. Treat likely project docs as files under conventional docs locations such as `docs/**`, top-level `README.md`, package/app READMEs, and explicitly requested documentation paths.

## Inputs to load

For non-arg audits:

- `contracts/frontmatter.md`
- `contracts/classification.md`
- `contracts/doctypes/[type].md` (when type is determined)
- `contracts/cascade.md`
- `templates/reports/health-report.md`

## Checks

Report:

- missing required frontmatter fields
- invalid `type` or `kind`; `kind: []` is valid for docs without a lifecycle role
- scalar `kind` instead of list syntax
- current docs depending on `docs/archive/**`
- current docs depending on superseded ADRs
- missing inverse cascade links, except documented intentional one-way relationships such as plan-to-durable lifecycle reminders
- duplicate or scattered docs with the same apparent purpose
- TIL docs being treated as canonical guidance
- unusual `ssot` placement as review signal, not format failure

## Mutation policy

Audit mode does not mutate files unless the user explicitly asks for repair. Assessment-only cascade checks stay in Audit mode even when the report names safe repairs that Maintain mode could apply later. Frame those as findings and recommended fixes, not as a Maintain mutation plan. If repair is requested, switch to Maintain mode for the changed files and report the mode switch.

## Severity guidance

Severity divides into two classes: **schema/field validation** (Critical) and **lifecycle/cascade relationships** (Warning).

**Critical** — Schema and field validation failures (prevent parsing and interpretation):
- Missing any required frontmatter field: `title`, `type`, `kind`, `created`, `updated`, `depends-on`, `updates`
- Invalid field values: `type` not in allowed set, `kind` containing invalid values, `status` not matching type-specific values
- Schema format errors: scalar `kind` instead of list, `kind` containing computed values like `stale`
- Type-kind coupling violations: e.g., `type: reference` with `kind: [plan]`
- Broken required relationships: completed/archived plan missing `replacedBy`, ADR missing type-required fields

**Warning** — Lifecycle and cascade relationship issues (reduce reliability without breaking schema):
- Current docs depending on superseded ADRs or archived plans without a clear current replacement
- Duplicate SSOT claims: multiple specs/docs with `kind: [spec, ssot]` for the same purpose
- Missing inverse cascade links: `spec` lists `updates: [doc]` but `doc` has empty `depends-on` (and the pairing is intentional, not plan-to-durable)
- Lifecycle conflicts: completed plan with `replacedBy` pointing to a draft spec, not an accepted one
- ADR lineage format errors: `supersededBy` using file paths instead of ADR IDs
- TIL docs presenting themselves as canonical guidance

**Info** — Non-blocking observations:
- Clean docs with correct metadata and relationships
- Intentional one-way lifecycle reminders (plan updating a durable spec; spec not reciprocating)
- Excluded scope and tool/vendor paths
- Historical context and reference notes

## Reporting

Use `templates/reports/health-report.md`. Keep findings grouped by severity and include exact replacement links when known.
