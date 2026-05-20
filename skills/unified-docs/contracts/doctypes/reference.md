# Doc Type: Reference

```yaml
type: reference
kind: []  # no lifecycle role by default
```

## Description

`type: reference` is the default type for durable technical documentation that describes "what is" rather than "how to" or "why". It covers API references, configuration docs, terminology, and similar content that is meant to be looked up rather than read sequentially.

## Rules

- **Current-truth**: Reference docs describe current state, not future work or decisions.
- **Stable structure**: Reference docs should be organized for lookup efficiency (alphabetical, by component, by function).
- **No lifecycle role**: A `type: reference` doc with `kind: []` has no lifecycle obligations — no status, no superseded path unless it also carries `kind: [spec]` or `kind: [adr]`.
- **Templates**: Reference docs may be authored from `templates/authoring/spec.md` if they describe durable technical truth, or from a bare skeleton if they are simple configuration/API dumps.

## Relationship to Spec

A `type: reference` doc with `kind: [spec]` is a **spec** — it has lifecycle obligations (draft/accepted) and cascade rules from `contracts/doctypes/spec.md`.

A `type: reference` doc with `kind: []` and no spec designation is a plain reference doc with no lifecycle role. It should still be kept current, but the skill does not enforce status transitions or cascade obligations for it.

## Cascade rules

- Reference docs may be listed in `depends-on` or `updates` of plans, ADRs, how-tos, and explanations.
- Reference docs with no lifecycle role (`kind: []`) should not be dependency targets for durable docs — durable docs should depend on specs or ADRs, not on raw reference material.