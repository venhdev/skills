# Output Template: Normalize Frontmatter

Use after fixing metadata on an existing doc.

```markdown
Updated: `<path>`

Metadata changes:
- `field`: `old` → `new`
- `field`: added `value`

Body changes: none / minimal: `<one-line reason>`
Validation: `scripts/check_frontmatter.py <path>` passed
Cascade followups: none / `<paths>`
```

Keep the report to the fields that changed. Do not restate the whole frontmatter block unless the user asks.
