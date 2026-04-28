# Frontmatter Contract

This is the schema source for `unified-docs` metadata.

## Universal fields

```yaml
---
title: [Short descriptive title]
type: [tutorial | how-to | reference | explanation | decision]
kind: []  # or [plan], [spec], [adr], [ssot], [draft], [til], or valid combinations
audience: [new contributors | developers | api consumers | operators | maintainers]
owner: [team-name | agent-name | unassigned]
created: [ISO date]
lastReviewed: [ISO date]
reviewCadence: [optional integer days]
depends-on: []
updates: []
---
```

Required universal fields:

- `title`
- `type`
- `kind`
- `audience`
- `owner`
- `created`
- `lastReviewed`
- `depends-on`
- `updates`

Optional universal field:

- `reviewCadence`

## Cadence policy

`reviewCadence` is optional. If missing, compute stale status with defaults:

- `plan`: 90 days
- all other docs: 180 days

A doc is stale when:

```text
today - lastReviewed > effectiveCadence
```

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
