# Doc Type: How-To

```yaml
type: how-to
kind: []
```

## Purpose

Task-oriented guidance. Answers: **"How do I complete task X?"**

Step-based, practical, no explanation of why (beyond brief context).

## Rules

- **No inferred kind**: Don't add `kind: [spec]` just because the doc mentions current behavior.
- **No lifecycle**: No need for `status`, `replacedBy`. How-to docs are evergreen.
- **Dependencies**: If based on a decision → `depends-on` the corresponding ADR.
- **Not SSOT**: How-to provides guidance, not canonical truth.

## Cascade rules

- How-to must not be listed in `updates` of specs or ADRs.
- How-to may `depends-on` specs, ADRs, explanations.
