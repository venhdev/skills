# Frontmatter Contract

This is the schema source for `unified-docs` metadata.

## Universal fields

```yaml
---
title: [Short descriptive title]
type: [how-to | reference | explanation | decision]
kind: []  # or [plan], [spec], [adr], [ssot], [til], or valid combinations
created: [ISO date]
updated: [ISO date]
depends-on: []
updates: []
---
```

Required universal fields:

- `title`
- `type`
- `kind`
- `created`
- `updated`
- `depends-on`
- `updates`

## Field definitions

- `created` — ISO date when the doc was first written
- `updated` — ISO date when the doc was last meaningfully changed (tracking only, not computed)

Never persist `kind: [stale]` or `kind: stale`.

## List policy

`kind`, `depends-on`, and `updates` should be represented as YAML lists. `kind` is required as a field, but `kind: []` is valid for docs without a lifecycle role. Empty lists are valid:

```yaml
kind: [spec, ssot]
depends-on: []
updates: []
```

## Lifecycle-specific fields

ADR fields and plan fields are defined in `contracts/lifecycle.md` because their validity depends on status transitions.

## Dependency fields

`depends-on` and `updates` are defined in `contracts/cascade.md`. They describe the current documentation graph, not historical lineage.
