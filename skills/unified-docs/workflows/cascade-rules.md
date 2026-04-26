# Cascade Rules

Cascade metadata lets Read mode know what context is required and lets Audit mode trace what must be revisited after a change.

## Field meanings

- `depends-on`: docs this doc currently relies on.
- `updates`: docs that should be revisited when this doc changes.

## Reciprocity

The relationship is normally represented by inverse fields, not duplicate fields:

- If A lists B in `updates`, B should normally list A in `depends-on`.
- If B lists A in `depends-on`, A should normally list B in `updates`.

Do not mirror the same field onto both files just to make the graph look bidirectional. `A depends-on B` means A needs B as current context; the reciprocal trace is usually `B updates A`, meaning A should be revisited when B changes.

Repair missing inverse links when the relationship is current and specific enough. If it is intentionally one-way, report the exception and the reason.

## ADR supersession is not dependency

Use ADR fields for ADR lineage:

- old ADR: `supersededBy: ADR-NNN`
- new ADR: `supersedes: ADR-OLD`

Do not add the old superseded ADR to the new ADR's `depends-on` just to preserve history. If another doc depends on the old ADR, update that dependency to the replacement ADR or to a stable SSOT/reference doc.

## Before changing cascade fields

1. Run or mentally model `scripts/cascade_targets.py <file> [root]`.
2. Identify outgoing `depends-on` and `updates`.
3. Identify incoming docs that reference the file.
4. Decide whether reciprocal links should be patched or explicitly waived.

## After changing cascade fields

Report:

- changed cascade fields
- reciprocal links repaired
- one-way exceptions
- remaining followups
