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
- Do not add plan files to a spec or SSOT doc's `updates` merely because the plan may later affect that durable doc. Plans are temporary execution artifacts and should not become durable inbound graph targets for specs/SSOT docs.
- A plan may list durable docs in its own `updates` when those docs should be revisited as the plan changes status, but that relationship is lifecycle guidance, not a reciprocal dependency to patch onto the durable docs.
- If the relation is intentionally one-way, report the exception and the reason.
- If the target is archived, superseded, or ambiguous, prefer report/ask over silent mutation.

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

## When scripts are unavailable

If `cascade_targets.py` cannot run (Python unavailable, shell execution blocked, etc.), trace the cascade graph manually:

**Step 1 — Read outgoing links from the target file's frontmatter:**
- `depends-on:` — docs this file relies on
- `updates:` — docs to revisit when this file changes

**Step 2 — Find incoming references (who depends on or updates this file):**
Search all `.md` files in the docs tree for the target file's name in their `depends-on` or `updates` fields.

Use any available tool:
- `grep -r "target-filename.md" docs/ --include="*.md" -l`
- Read frontmatter of files you suspect reference this one
- Ask the user if a search tool is unavailable

**Step 3 — Produce the equivalent of the JSON output (mentally or in your report):**
- Outgoing: `depends-on` and `updates` lists from the target's frontmatter
- Incoming: files found in Step 2

**Exclusions**: Skip files under `.claude/`, `.github/`, `node_modules/`, `tmp/`, and similar tool/vendor paths.

The manual result should be equivalent to what `cascade_targets.py` would produce. Apply the same repair/report logic from the Repair policy section above.
