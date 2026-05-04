# Doc Type: ADR (Architecture Decision Record)

```yaml
type: decision
kind: [adr]
```

## Additional required fields

- `adr-id`: Sequential, zero-padded (`ADR-001`, `ADR-002`, etc.)
- `status`: `draft` | `accepted` | `superseded`
- `deciders`: [list of people/roles]
- `decided`: [ISO date] — required when `status: accepted`
- `supersededBy`: [ADR ID] — required when `status: superseded` (format: `ADR-NNN`, not file path)
- `supersedes`: [ADR ID] — if this ADR supersedes an old one (format: `ADR-NNN`)

## Lifecycle

```
draft → accepted → superseded (terminal)
```

**Rules**:
- **Accepted**: Body is write-once. Do not rewrite the decision.
- **Supersession**: Create new ADR → update old ADR's frontmatter + Status Log. Do not rewrite body of old ADR.
- **IDs**: Use ADR ID (`ADR-002`), not file path, in `supersededBy` and `supersedes`.

## Cascade rules

- Current docs must not `depends-on` superseded ADRs as if still active.
- If docs are currently `depends-on` a superseded ADR → surface and ask user to handle.
