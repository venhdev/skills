# Audit Mode

Use this mode when the user asks for docs health, broken links, stale docs, dependency chains, duplicate/orphan docs, or cascade issues.

## Scope

Default to targeted audit:

- requested doc
- directly linked `depends-on` targets
- directly listed `updates` targets
- obvious incoming references if needed

Run full corpus audit only when the user says full, complete, entire corpus, or equivalent.

## Corpus boundaries

Audit project documentation, not tool configuration, vendored content, generated workspaces, or assistant runtime files. Exclude these paths unless the user explicitly asks to audit them:

- hidden tool/runtime directories: `.claude/**`, `.agents/**`, `.agent/**`, `.codex/**`, `.cursor/**`, `.github/**`, `.vscode/**`, `.idea/**`
- generated or temporary workspaces: `tmp/**`, `temp/**`, `.tmp/**`, `.cache/**`, `dist/**`, `build/**`, `coverage/**`, `node_modules/**`, `vendor/**`
- installed or synced skill/agent definitions unless the user is auditing the skill system itself

If a full-corpus audit sees many Markdown files outside project docs, report them as excluded scope rather than missing frontmatter. Treat likely project docs as files under conventional docs locations such as `docs/**`, top-level `README.md`, package/app READMEs, and explicitly requested documentation paths.

## Inputs to load

- `contracts/frontmatter.md`
- `contracts/classification.md`
- `contracts/lifecycle.md`
- `contracts/cascade.md`
- `templates/reports/health-report.md`

## Checks

Report:

- missing required frontmatter fields
- invalid `type` or `kind`; `kind: []` is valid for docs without a lifecycle role
- scalar `kind` instead of list syntax
- stale docs by effective cadence
- current docs depending on `docs/archive/**`
- current docs depending on superseded ADRs
- missing inverse cascade links, except documented intentional one-way relationships such as plan-to-durable lifecycle reminders
- duplicate or scattered docs with the same apparent purpose
- TIL docs being treated as canonical guidance
- unusual `ssot` placement as review signal, not format failure

## Mutation policy

Audit mode does not mutate files unless the user explicitly asks for repair. If repair is requested, switch to Maintain mode for the changed files and report the mode switch.

## Reporting

Use `templates/reports/health-report.md`. Keep findings grouped by severity and include exact replacement links when known.
