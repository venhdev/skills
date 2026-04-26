# Output: Health Report

Use this shape for Audit mode unless the user asks for another format.

```markdown
## Docs health audit

Scope: `<docs root or paths>`
Files checked: `<count or list>`

### Summary

- Critical: <n>
- Warnings: <n>
- Followups: <n>

### Findings

1. **<severity>: <short title>**
   - File: `<path>`
   - Evidence: <field, line, or observed mismatch>
   - Impact: <why it matters>
   - Repair: <specific action>

### Cascade status

- Repaired or valid reciprocal links: <list>
- Missing reciprocal links: <list>
- Intentional one-way links: <list with reason>

### Supersession status

- Complete chains: <list>
- Broken chains: <list>
- Superseded dependencies needing repair: <list>
```
