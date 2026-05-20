# Doc Type: Plan

```yaml
type: explanation
kind: [plan]
```

## Additional required fields

- `status`: `draft` | `in-progress` | `completed` | `superseded` | `archived`
- `completed`: [ISO date] — required when `status: completed`
- `superseded`: [ISO date] — required when `status: superseded`
- `archived`: [ISO date] — required when `status: archived`
- `replacedBy`: [file path] — required when `status: completed`, `superseded`, or `archived`

## Lifecycle

```
draft → in-progress → completed → superseded → archived
```

**Status semantics**:
- **draft** — intended work not yet started
- **in-progress** — work actively being executed
- **completed** — work finished and outcomes promoted to a spec
- **superseded** — work abandoned mid-flight; a different approach was chosen and the rest of the plan should never be implemented. Must set `replacedBy`.
- **archived** — completed plan whose spec is now accepted and stable. Must set `replacedBy`.

**Rules**:
- **Completed**: Promote durable outcomes into a spec (new or existing).
- **Superseded**: When the plan approach is abandoned and no continuation is expected. The `replacedBy` target can be any doc (spec, TIL, explanation, etc.) representing the chosen alternative.
- **replacedBy**: Must point to a resolvable file. No longer required to be an accepted spec.
- **Archive-ready**: When `replacedBy` correctly points to a doc; if current inbound references are already moved or absent → ready.
- **Archive banner**: Add banner to the plan → directing readers to the spec. Do not add banner to the spec.

## Cascade rules

- Plan **MAY** list durable docs in `updates` (lifecycle reminder).
- Durable docs **MUST NOT** list plan in `updates` or `depends-on`.
- Plan **MUST NOT** be added to `updates` of spec/SSOT docs.

## Gate conditions

After creating a plan, suggest the user add gate conditions if needed. This is a hint, not required.
