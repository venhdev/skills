# Writing Rules per Type

These rules apply to the body only. Frontmatter rules live in
`contracts/frontmatter.md`. Lifecycle rules live in `workflows/adr-lifecycle.md` and
`workflows/plan-lifecycle.md`.

| Type | Opening | Body expectations | Avoid |
|---|---|---|---|
| tutorial | First line after frontmatter is `## Step 1`. | Concrete, hands-on steps that end in a working result. | Welcome paragraphs, "In this tutorial" prose, unnumbered intros. |
| how-to | First line after frontmatter is `## Step 1` or `## Prerequisites`. | Task-oriented commands, precise actions, concrete outcome. | Narrative overviews, "This guide covers...", "Before you begin..." prose. |
| reference | Dense tables and lists; signatures and fields only. | Exhaustive, neutral. | Opinions, tutorial-style prose, duplicate content that belongs in an SSOT. |
| explanation | Narrative exposition of "why". | May reference tutorials, how-tos, or decisions. | Step lists (that is how-to territory). |
| decision | Context -> Decision -> Consequences -> Status Log. | See `workflows/adr-lifecycle.md`. | Editing accepted ADR body in place. |
| plan | See `workflows/plan-lifecycle.md`. | Deliverable-oriented milestones, measurable goals. | Task-step lists, vague goals, `reviewCadence > 90`. |
| runbook | `## Prerequisites` -> numbered procedure -> decision table -> rollback -> post-action checklist. | Operational, reproducible, safe to execute under incident pressure. | Narrative overview before the first heading; no rollback section. |

## Cross-type rules

- Never copy authoritative content. Link to the SSOT doc instead.
- Never leave bare external URLs without descriptive text.
- Internal links use relative paths.
- Use short paragraphs and prefer tables when the data is tabular.
