# Read Mode

Use this mode when the user asks whether a doc is current, what the current decision is, what another doc depends on, or which doc is canonical.

## Behavior

1. Read frontmatter before body.
2. Determine lifecycle state: current, draft, stale, archived, completed-plan, or superseded ADR.
3. Follow replacement pointers before summarizing current truth:
   - ADR: `supersededBy`
   - archived plan: `replacedBy`
4. Separate historical context from current guidance.
5. Do not mutate files.

## What to report

Use `templates/reports/read-status.md`.

Prioritize:

- whether the requested doc is current
- what the actual current target is if not
- key dependencies if they matter to the answer
- historical context only after current guidance is clear
