# Cascade Contract

This file defines how `depends-on` and `updates` describe the current documentation graph.

## Field meanings

- `depends-on`: current docs this doc relies on to stay correct.
- `updates`: current docs that should be revisited when this doc changes.

These fields describe current graph relationships, not historical notes.

## Inverse relationship

The relationship is usually represented by inverse fields, not identical fields:

- If A `depends-on` B, B will usually list A in `updates`.
- If A `updates` B, B will usually list A in `depends-on`.

Do not mirror the same field onto both docs just to make the graph look symmetrical.

## Repair policy

When maintaining docs:

- Repair missing inverse links when the relationship is current and unambiguous.
- If the relation is intentionally one-way, report the exception and the reason.
- If the target is stale, archived, superseded, or ambiguous, prefer report/ask over silent mutation.

## Current-truth restrictions

- Current docs should not depend on `docs/archive/**`.
- Current docs should not depend on superseded ADRs as if they were still active guidance.
- If a current doc depends on a superseded ADR, recommend the replacement ADR via `supersededBy` or a stable current spec/SSOT doc.

## Validation habit

Before editing cascade metadata, trace:

1. outgoing `depends-on`
2. outgoing `updates`
3. incoming references from other docs

Use `scripts/cascade_targets.py` when practical.
