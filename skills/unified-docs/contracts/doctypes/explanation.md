# Doc Type: Explanation

```yaml
type: explanation
kind: []
```

## Purpose

Context and reasoning. Answers: **"Why is it this way?"**

Explains tradeoffs, context, rationale. Not a task guide (how-to) and not a decision record (ADR).

## Rules

- **No inferred kind**: 
  - Don't add `kind: [plan]` unless this doc actually has a plan lifecycle.
  - Don't add `kind: [spec]` just because it describes current state.
  - Don't add `kind: [til]` unless it's a Today I Learned snippet.
- **No mandatory lifecycle**: Explanation may have status if it's also a plan, but not required.

## Cascade rules

- Explanation may `depends-on` ADRs, specs, plans.
- Other docs may `depends-on` explanation to understand the reasoning.
