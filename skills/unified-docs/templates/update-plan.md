# Output Template: Update Plan

Use after updating a living plan, roadmap, or migration plan.

```markdown
Updated: `<path>`

Plan status:
- `status`: `old` → `new`
- `lastReviewed`: `old` → `YYYY-MM-DD`

Plan body:
- Milestones updated: `<short list>`
- Status Log appended: `YYYY-MM-DD - <reason>`

Cascade followups:
- `<path>` — `<why revisit>`

Validation: `scripts/check_frontmatter.py <path>` passed
```

Mention only changed milestones and docs needing review. Avoid a full plan recap.
