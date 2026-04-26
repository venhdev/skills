# Output Template: Supersede ADR

Use after creating a replacement ADR for an accepted decision.

```markdown
Superseded: `<old ADR>` → `<new ADR>`

Old ADR changes:
- `status`: `accepted` → `superseded`
- `supersededBy`: `<new ADR id>`
- Status Log appended

New ADR:
- `adr-id`: `<new ADR id>`
- `status`: `draft | accepted`
- `supersedes`: `<old ADR id>`

Validation: old and new ADR frontmatter passed
Cascade followups: none / `<paths>`
```

Never summarize the accepted ADR body as if it was edited; only metadata and Status Log should change.
