# Doc Type: Plan

```yaml
type: explanation
kind: [plan]
```

## Additional required fields

- `status`: `draft` | `in-progress` | `completed` | `archived`
- `completed`: [ISO date] — required when `status: completed`
- `archived`: [ISO date] — required when `status: archived`
- `replacedBy`: [file path] — required when `status: completed` or `archived`

## Lifecycle

```
draft → in-progress → completed → archived
```

**Rules**:
- **Completed**: Promote durable outcomes into a spec (new or existing).
- **replacedBy**: Must point to a non-archived accepted spec, not a draft.
- **Archive-ready**: When `replacedBy` correctly points to an accepted spec; if current inbound references are already moved or absent → ready.
- **Archive banner**: Add banner to the plan → directing readers to the spec. Do not add banner to the spec.

## Cascade rules

- Plan **MAY** list durable docs in `updates` (lifecycle reminder).
- Durable docs **MUST NOT** list plan in `updates` or `depends-on`.
- Plan **MUST NOT** be added to `updates` of spec/SSOT docs.

## Gate conditions

After creating a plan, suggest the user add gate conditions if needed. This is a hint, not required.
