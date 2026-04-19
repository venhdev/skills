# Unified Docs Creator

A Claude skill that guides agents to create, audit, and refactor documentation using the **Diátaxis framework** — a systematic approach to technical documentation that separates content by user intent.

---

## What It Does

Every document answers one of four questions. This skill classifies documents accordingly and enforces the correct structure, tone, and format for each type:

| Type | Answers | Audience |
|------|---------|----------|
| **Tutorial** | "How do I learn this?" | Beginners, step-by-step build |
| **How-to Guide** | "How do I accomplish X?" | Practitioner, task-focused |
| **Reference** | "What does X do?" | Lookup, exhaustive detail |
| **Explanation** | "Why is it designed this way?" | Conceptual, architectural |

---

## Trigger Phrases

Activate the skill when the user says:

- `"document this"` / `"write a guide"` / `"organize docs"`
- `"docs audit"` / `"fix the docs structure"` / `"apply Diátaxis"`
- `"cleanup docs"` / `"clear deprecated docs"`
- Any task involving `.md` files, READMEs, onboarding guides, or architecture records

---

## How It Works

1. **Auto-detect** — Scans the project structure and existing docs before writing anything
2. **Classify** — Applies the Keyword Matrix + Title-Pattern priority to determine document type
3. **Create or Update** — Writes new docs or refactors existing ones with correct frontmatter
4. **Audit / Cleanup** — Flags misplaced, duplicate, orphaned, or stale docs (acts only on approval)

---

## Required Frontmatter

Every document must include:

```yaml
---
title: [Short descriptive title]
description: [One-sentence compressed infinitive, 10–15 words max]
type: [tutorial | how-to | reference | explanation]
audience: [new contributors | developers | api consumers | operators | maintainers]
created: [ISO date]
lastReviewed: [ISO date]
---
```

---

## Quality Standards

- **No teaching preambles in how-tos** — jump straight to steps
- **No opinions in references** — dense tables and lists only
- **SSOT compliance** — link to authoritative sources instead of inlining
- **Contextual cross-references** — every "See X" includes a one-line reason why
- **No bare URLs** — always use relative links with descriptive titles

---

## License

MIT
