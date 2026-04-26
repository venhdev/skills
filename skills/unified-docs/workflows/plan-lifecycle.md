# Plan / Roadmap Lifecycle

Plans are temporary execution artifacts. Specs are durable current truth.

See `contracts/frontmatter.md` for fields.

## When to write a plan

Write `kind: [plan]` for roadmap, rollout, migration, milestone, implementation, adoption, or strategy docs.

Do not use a plan as long-term product truth. When the work completes, promote durable outcomes into a `kind: [spec]` doc.

## Plan completion rule

When a plan is completed:

1. Set `status: completed`, `completed: YYYY-MM-DD`, and bump `lastReviewed`.
2. Update or create the related spec under `docs/specs/`.
3. Move durable outcomes into the spec: current behavior, requirements, accepted constraints, interfaces, acceptance criteria, and shipped decisions not covered by ADRs.
4. Archive the plan if the user asks or approves.

## Archive rule

Archive path:

```text
docs/archive/plans/<plan-name>.md
```

Archived plans are historical artifacts and leave the current dependency graph.

When archiving:

1. Set `status: archived`.
2. Set `archived: YYYY-MM-DD`.
3. Set `replacedBy: docs/specs/<spec>.md`.
4. Add a short banner at the top of the body:
   `Archived plan. Current behavior lives in <spec path>.`
5. Move the file to `docs/archive/plans/`.
6. Update current inbound references to point to the spec or remove them.
7. Do not repair every internal relative link inside the archived plan unless the user asks.

## Status transitions

| From | To | Action |
|---|---|---|
| absent | `draft` | Create plan; set `status: draft` |
| `draft` | `in-progress` | At least one milestone started; append Status Log entry |
| `in-progress` | `completed` | All milestones shipped; promote durable outcome into spec |
| `completed` | `archived` | Move plan to archive after spec exists and refs are updated |

## Rules

- Milestones are deliverables, not tasks.
- Goals must be measurable.
- `updates:` holds current doc paths, never prose.
- Current docs should not depend on archived plans.
- Archive is not delete: keep enough history to understand why work happened, but route readers to spec for current truth.
