# Frontmatter Spec (SSOT)

Single contract for `unified-docs`.

## Universal fields

```yaml
---
title: [Short descriptive title]
type: [tutorial | how-to | reference | explanation | decision]
kind: [plan, spec, adr, ssot, draft, til]
audience: [new contributors | developers | api consumers | operators | maintainers]
owner: [team-name | agent-name | unassigned]
created: [ISO date]
lastReviewed: [ISO date]
reviewCadence: [optional integer days]
depends-on: []
updates: []
---
```

## Kind policy

`kind` is always a YAML list, even for one value: `kind: [spec]`, not `kind: spec`.

Allowed `kind` values:

- `plan`: temporary execution plan, roadmap, rollout, migration, or milestone doc
- `spec`: durable current behavior/requirements/source-of-truth for a feature or system capability
- `adr`: architectural decision record
- `ssot`: canonical source of truth; often lives in `docs/reference/` or `docs/specs/`, but the folder alone does not decide validity
- `draft`: not stable yet
- `til`: short practical note from a real issue/task

Multiple values are allowed when each value is true, for example `kind: [spec, ssot]` for a spec that is also the canonical source. Do not normalize away valid secondary kinds.

Do not use `stale` as persisted metadata. Stale is computed from review timestamps and cadence.

## Cadence policy

`reviewCadence` is optional. If missing, use defaults:

- `plan`: 90 days
- all other types: 180 days

Stale condition:

- `today - lastReviewed > effectiveCadence`

## Dependency/cascade semantics

- `depends-on`: current prerequisite docs.
- `updates`: downstream docs to revisit when this doc changes.

Normally reciprocal. If intentionally one-way, report why.

Archived docs leave the current dependency graph. Current docs should not depend on `docs/archive/**`.

ADR lineage is separate:

- old ADR: `supersededBy`
- new ADR: `supersedes`

Use ADR IDs such as `ADR-002` for lineage fields, not file paths. Do not use `depends-on` for ADR historical lineage.

## ADR fields

```yaml
type: decision
kind: [adr]
adr-id: ADR-NNN
status: [draft | accepted | completed | superseded]
deciders: [name, name]
decided: [ISO date when accepted]
supersededBy:
supersedes:
```

## Plan fields

```yaml
type: explanation
kind: [plan]
status: [draft | in-progress | completed | archived]
completed:        # ISO date when completed
archived:         # ISO date when moved to archive
replacedBy:       # current spec path after archive
```

## Spec fields

```yaml
type: reference
kind: [spec]              # may be [spec, ssot] when canonical
status: [draft | accepted]
```

Specs describe current durable behavior: requirements, accepted behavior, API/CLI contracts, acceptance criteria, constraints, and operationally relevant outcomes. Plans are temporary execution artifacts; specs are current truth.
