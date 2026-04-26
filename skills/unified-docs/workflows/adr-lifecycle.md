# ADR Lifecycle

ADRs record architectural decisions with lasting consequences. They are write-once artifacts: once `status: accepted`, never edit the decision body in place. Create a new ADR and supersede the old one.

See `contracts/frontmatter.md` for ADR-specific fields.

## When to write an ADR

Write an ADR for: naming conventions, authentication strategy, protocol choices, database architecture, event schema conventions, service boundary decisions, integration patterns.

Do not write an ADR for: routine refactors, single-service implementation details, or tooling notes without architectural consequence.

## Body structure

```markdown
# ADR-NNN: [Decision title]

## Context
[Problem, constraints, alternatives considered.]

## Decision
[The chosen option, described clearly.]

## Consequences
[Positive, negative, neutral consequences. Migration cost.]

## Status Log
- YYYY-MM-DD - draft
- YYYY-MM-DD - accepted
```

## Status transitions

| From | To | Action |
|---|---|---|
| absent | `draft` | Create ADR with next `adr-id`; set `status: draft` |
| `draft` | `accepted` | Set `decided:` to today; append Status Log entry; update cascade targets |
| `accepted` | `completed` | Append Status Log entry; refresh `updates:` docs |
| `accepted` | `superseded` | Create a new ADR; set old ADR `status: superseded` and `supersededBy: ADR-NNN`; set new ADR `supersedes: ADR-OLD`; do not mutate old ADR body beyond Status Log and metadata |

## Supersession rules

- Use `supersedes` / `supersededBy` for ADR lineage.
- Do not add the superseded ADR to the replacement ADR's `depends-on` as historical context.
- If other docs currently depend on the old ADR, update them to depend on the replacement ADR or a stable SSOT/reference doc.
- Preserve the accepted ADR body. Only metadata and Status Log can change.
- Run cascade tracing for both old and new ADRs after supersession.

## Rules

- `type` must be `decision`, never `explanation`.
- `kind` must include `adr`.
- ADR ids are sequential and zero-padded (`ADR-001`, `ADR-002`).
- The Status Log is the only body section that grows after acceptance.
- Always update `lastReviewed` when the Status Log is appended.

## ADR register

If the project has an ADR register such as `docs/adr/README.md` or `docs/adr/index.md`, update it when you create or supersede an ADR.
