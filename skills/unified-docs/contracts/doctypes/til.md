# Doc Type: TIL (Today I Learned)

```yaml
type: explanation
kind: [til]
```

## Purpose

Quick note about a finding, learning, or discovery. Not canonical guidance.

## Rules

- **Not SSOT**: Do not treat as SSOT or canonical reference.
- **Not dependency target**: Other docs must not `depends-on` TIL.
- **Promotion path**: If insight is important → promote to a spec or standalone explanation doc.

## Cascade rules

- TIL must not be listed in `updates` of specs or ADRs.
- Docs must not `depends-on` TIL.
- TIL may reference specs, ADRs, explanations to provide context.
