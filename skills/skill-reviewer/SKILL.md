---
name: skill-reviewer
description: >
  Audits and refines skill files for signal-to-noise quality. Use when auditing skill
  health, before splitting a growing skill, or when an agent reports a skill is hard to
  follow. Trigger phrases: "review this skill", "audit skills", "refine this skill".
---

# skill-reviewer

Audits and refines skill files to maximize signal-to-noise for agent consumption.

---

## When to Trigger

> See frontmatter `description` for primary trigger conditions.

Use this section for operational signals only:

- An agent reports a skill is "hard to follow"
- After adding a new section to any skill
- When signal-to-noise visibly degrades

> **Do NOT trigger just because a skill is long.** Line count is not a quality signal.
> A 1000-line skill covering 5 distinct concerns is better than 5 separate skills
> an agent must context-switch between. Only act when signal-to-noise is genuinely low.

---

## What Actually Hurts Agents

| Problem | Why it hurts | Detection |
|---------|-------------|-----------|
| **Vague section headers** | Agents can't quickly locate the right section | Header is a phrase, not a noun |
| **Prose > bullets** | Rules buried in paragraphs agents skim past | > 3 sentences explaining a single rule |
| **Anti-patterns as paragraph** | Hardest section to skim — agents ignore it | Listed as numbered items or paragraphs |
| **Duplicated rules** | Same rule stated 3x → agent confused which is authoritative | Same idea in 2+ sections |
| **Template inline in prose** | Agent can't copy-paste cleanly | Code mixed inside sentences |
| **Missing frontmatter** | Agent doesn't know when to trigger | No YAML frontmatter block |
| **Mixed concerns in one section** | Forces agents to read everything to find the rule | Section covers both "what" and "how to apply it" |

---

## What Does NOT Hurt Agents

- Long skills with distinct sections covering distinct concerns
- Many lines of tables or code (high density, easy to skim)
- Sections > 100 lines if they cover one concern thoroughly
- Repetition across skills (different triggers, different contexts)

---

## Audit Steps

### Step 1 — Scan for Violation Signals

Use the detection column in "What Actually Hurts Agents" as your scan checklist. For each section, check whether any detection signal is true. If none fire, skip the section.

| # | Check | Problem | Fix | Detection Signal |
|---|-------|---------|-----|-------------------|
| 1 | Vague header? | "Notes", "Other", "Further info" | Rename to descriptive noun | Header is a phrase, not a noun |
| 2 | Rule as paragraph? | > 3 sentences for one rule | Convert to bullet or table | Rule not in bullet or table form |
| 3 | Anti-patterns as paragraph? | Numbered list or wall of text | Pure bullet list | No bullet list for anti-patterns |
| 4 | Duplicate rule? | Same idea in 2+ sections | De-duplicate, keep one, link elsewhere | Duplicate semantic rule exists |
| 5 | Template inline? | Code inside sentences | Extract to code block | Code fences not used for examples |
| 6 | Mixed concerns? | Section covers unrelated things | Split into sub-sections | Contains > 1 independent rule set |
| 7 | Missing frontmatter? | No YAML block | Add before refining | No `---` block at file top |

Apply fixes only to rows where the Detection Signal column is false. Do not rewrite well-structured sections just because they're long.

---

## Refinement Tactics

### Tactic 1 — Prose → Table or Bullet

```
Before:
"The following naming patterns are correct for domain services:
<domain>-service is the standard. Examples include..."

After: use a table instead:
| Category | Pattern | Examples |
|---|---|---|
| Domain services | `<domain>-service` | auth-service, payment-service |
```

### Tactic 2 — Anti-Patterns as Pure Bullet List

Anti-patterns that hurt agents most:

- **Teaching preamble** — "First let me explain..." → Cut. How-tos jump to steps.
- **Verbose rule** — > 3 sentences → Bullet or table.
- **Duplicate rules** — Same idea in 2+ sections → Keep one authoritative instance, link elsewhere.
- **Numbered wall of text** → Convert to pure bullet list.

### Tactic 3 — Navigation Headers

Prefer descriptive noun headers or numbered steps over vague phrases:

| Before (vague) | After (scannable) |
|---|---|
| "Additional Notes" | `## 7. Templates` |
| "Other Considerations" | `## 10. Decision Records (ADRs)` |
| "Notes" | `## 5. Duplicate Scan Rules` |

### Tactic 4 — Extract Inline Templates

Move embedded examples into standalone code blocks. Use indented blocks for the example:

```
## Frontmatter Template

Correct format:

    ---
    title: [Short title]
    type: tutorial | how-to | reference | explanation
    ---
```

### Tactic 5 — Markdown Safety Rule

- Avoid nested backtick fences (` ```yaml inside ``` `) — many markdown parsers break on it.
- Prefer indented blocks for embedded examples in code block sections.

### Tactic 6 — Split Only When Genuinely Mixed

Split a section only when it covers two or more distinct concerns that:
- Have different trigger conditions
- Need to be applied independently

Do NOT split a long section that covers one concern thoroughly.

---

## Output Format

After auditing, produce this report:

    ## Skill Quality Report — `<skill-name>`
    
    **Path:** `<path>`
    **Signal-to-Noise:** ✅ Good | ⚠️ Needs cleanup | 🔴 Hard to follow
    
    ### Violations Found
    
    1. **[Section 3]** "Notes" header — vague → rename to "Duplicate Scan Rules"
    2. **[Section 7]** Rule stated as paragraph → convert to bullet list
    3. **[Anti-patterns]** Numbered list → convert to pure bullet list
    4. **[Section 4]** Mixed concerns → split into sub-sections
    
    ### Refinements Applied
    
    | # | Section | Fix |
    |---|---------|-----|
    | 1 | Section 3 | Renamed header |
    | 2 | Section 7 | Prose → bullet list |
    | 3 | Anti-patterns | Numbered → bullet list |
    | 4 | Section 4 | Split into sub-sections |
    
    ### Remains Unchanged
    
    - Sections 1, 2, 5, 6 — well-structured, no changes needed
    - All templates — correctly in code blocks

---

## Refine Checklist

Before marking a refinement done, confirm all of the following:

- No vague section headers — agent can locate any section by scanning
- Rules use bullets or tables — no prose paragraphs hiding a single rule
- Anti-patterns section → pure bullet list
- No duplicate rules across sections
- Templates are in code blocks, never inline in prose
- Frontmatter present with `name` and `description`
- Did NOT rewrite well-structured sections just for being long

---

## Self-Refinement Loop

When refining this skill with itself:

> **Save a backup as `SKILL.md.bak` before applying any changes.**

Everything else — the audit steps, the tactics, the checklist — is already in this skill. Apply them as-is.