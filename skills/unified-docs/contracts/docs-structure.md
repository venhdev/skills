# Docs Structure

This structure keeps docs easy to scan and maintain. It is guidance, not a mandate: follow the existing project convention unless it is clearly causing confusion.

## Preferred folders

Use familiar, repeated folder names so humans and agents can guess where things live:

```text
docs/
  specs/             # current durable behavior / requirements / contracts
  plans/             # active execution plans only
  adr/               # architectural decision records
  reference/         # canonical reference / SSOT material
  how-to/            # task guides
  explanation/       # concepts and rationale
  tutorials/         # learning paths
  archive/
    plans/           # historical completed plans, outside current graph
```

## Placement rules

- Put current product/feature truth in `docs/specs/`.
- Put active execution work in `docs/plans/`.
- Move completed archived plans to `docs/archive/plans/` only after the durable outcome exists in a spec.
- Keep ADRs in `docs/adr/`.
- Prefer canonical reference data in `docs/reference/`, but treat `ssot` as a canonical role, not a folder mandate.
- Use `kind: [spec, ssot]` for a durable spec that is also the canonical source.
- Treat `kind: [ssot]` outside `docs/reference/` or `docs/specs/` as a placement review unless the content contradicts its metadata.
- Archived docs are historical. They should have `replacedBy` when a current doc supersedes them.

## Audit behavior

Assess structure during audit/triage, but do not restructure automatically.

Flag structure issues when:

- active plans and archived plans are mixed together
- specs are missing and completed plans are acting as current truth
- current docs depend on `docs/archive/**`
- many docs with the same purpose are scattered across inconsistent folders
- folder names are unusual enough that placement is unclear
- `kind: [ssot]` appears in an explanatory or unusual folder without clear canonical intent

If structure is poor enough to need changes, ask the user before restructuring. Do not force this layout onto a project that already has a clear convention. Do not treat unusual `ssot` placement as a frontmatter format error; classify it as a structure or intent review unless it violates the content contract.

## Create behavior

When creating a new doc:

1. Prefer the existing project convention.
2. If no convention exists, use the preferred folders above.
3. If placement is ambiguous, ask before writing.
