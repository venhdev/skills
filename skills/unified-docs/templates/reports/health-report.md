# Health Report

Use for Audit mode. Group findings by severity and avoid mutating files.

```markdown
Mode: Audit
Scope: [targeted | full corpus]
Status: [healthy | issues found | blocked]

Critical:
- [Broken current-truth issue, or none]

Warnings:
- [Stale, unusual placement, missing inverse cascade, or none]

Info:
- [Non-blocking observations]

Excluded scope:
- [Tool/runtime/generated/vendor paths intentionally skipped, or none]

Recommended fixes:
- `[path]` — [exact change or replacement target]

Validation:
- [scripts used, files checked, or not run]
```
