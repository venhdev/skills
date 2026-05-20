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
draft → accepted (stable)
accepted → superseded (terminal)
```

**Status semantics**:
- **draft** — spec is being written or reviewed
- **accepted** — spec is the current authoritative truth
- **superseded** — a new spec has replaced this one; do not use this spec for new work

**Supersession**: When a spec is superseded, create a new spec with the updated truth and set `supersededBy` on the old spec. Do not rewrite the old spec's body; add a deprecation banner pointing to the new spec.

## Rules

- **Current-truth**: Spec describes current truth (durable), not temporary like a plan.
- **SSOT (Single Source of Truth)**: 
  - `kind: [spec, ssot]` only when spec is the canonical authority **only one** for that topic.
  - No two specs should have `kind: [spec, ssot]` for the same topic.
- **Replacement**: Superseded ADRs and completed plans must not be current dependency targets when an accepted spec replacement exists.

## Cascade rules

- When spec changes → check all docs in incoming cascade (docs that `depends-on` this spec).
- If spec is deleted → docs must update their `depends-on`.
