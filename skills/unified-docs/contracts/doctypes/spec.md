# Doc Type: Spec (Specification)

```yaml
type: reference
kind: [spec]
# or
kind: [spec, ssot]
```

## Additional fields

- `status`: `draft` | `accepted`

## Lifecycle

```
draft → accepted (stable; no terminal state)
```

## Rules

- **Current-truth**: Spec describes current truth (durable), not temporary like a plan.
- **SSOT (Single Source of Truth)**: 
  - `kind: [spec, ssot]` only when spec is the canonical authority **only one** for that topic.
  - No two specs should have `kind: [spec, ssot]` for the same topic.
- **Replacement**: Superseded ADRs and completed plans must not be current dependency targets when an accepted spec replacement exists.

## Cascade rules

- When spec changes → check all docs in incoming cascade (docs that `depends-on` this spec).
- If spec is deleted → docs must update their `depends-on`.
