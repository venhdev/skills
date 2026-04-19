---
name: unified-docs-reader
description: >
  Reads documentation with lifecycle awareness. Checks document status, skips
  superseded ADRs, resolves depends-on chains, and flags stale docs before reading.
  Trigger when: "read doc", "check doc status", "is this doc current", "validate doc freshness".
---

# unified-docs-reader

Reads documents correctly — with lifecycle awareness, staleness checks, and dependency resolution.

> **Always run this before reading a critical SSOT doc.** It ensures you read the right version and don't follow superseded decisions.

---

## Step 1 — Read Frontmatter First

Always read frontmatter before reading body content. Check `status`, `kind`, `lastReviewed`, `reviewCadence`, `depends-on`, and `supersededBy` fields.

## Step 2 — Apply Skip Rules

**Activate this skill when:** reading a doc tagged `ssot`, reading a `type: decision` (ADR), or the user asks "is this doc current?", "check the status of this doc", or "validate this document".

| Condition | Action |
|-----------|--------|
| `status: superseded` | **Skip entirely.** Tell user: "This doc is superseded by [supersededBy]. Read [supersededBy] instead." |
| `status: deprecated` | Read for history context only. Warn user: "This doc is deprecated — do not follow." |
| `kind: [stale]` | Warn user: "This doc is flagged stale. lastReviewed: [date]. Proceed with caution." |
| `status: draft` | Tell user: "This doc is in draft — content may change." |
| `status: in-progress` | Note which milestones are complete vs. pending. Flag overdue milestones. |
| `kind: [plan]` | Check milestones table — report each milestone's completion status and target date. Flag overdue milestones. |

---

## Step 3 — Staleness Check

Compute days since `lastReviewed` vs `reviewCadence` (default: 180 days).

| Condition | Action |
|-----------|--------|
| `days_since_review > reviewCadence` | Warn: "Not reviewed in [N] days. May be outdated." |
| `days_since_review > reviewCadence * 2` | Flag as likely stale. Recommend running unified-docs-doctor audit. |

**Staleness one-liner:**

```bash
lastReviewed=$(grep 'lastReviewed:' doc.md | head -1 | awk '{print $2}')
days=$(( ($(date +%s) - $(date -d "$lastReviewed" +%s)) / 86400 ))
echo "Days since review: $days"
```

---

## Step 4 — Resolve depends-on

1. For each doc in `depends-on:`, read its frontmatter
2. Check if it is current (not stale, not superseded)
3. If superseded → warn: "Dependency [X] is superseded. Its content may be unreliable."
4. Read SSOT/dependency docs **first**, then the target doc
5. If any dependency is missing → warn: "depends-on: [X] not found. Doc may be incomplete."

---

## Step 5 — Read Body with Context

| Type | How to read |
|------|------------|
| **Tutorial** | Follow step-by-step. Build the concrete result. |
| **How-to** | Jump to steps. No preamble. |
| **Reference** | Scan tables, signatures. Dense. No opinions. |
| **Explanation** | Read "why" — conceptual understanding. |
| **Decision** | Read Context → Decision → Consequences. Check Status Log. |
| **Plan/Roadmap** | Read Goals → Milestones table (check completion %) → Timeline. Note which milestones are done vs. pending. |

---

## Output Format

After reading a doc, report:

```markdown
## Doc Status Report — [title]

| Field | Value |
|-------|-------|
| **Type** | [type] |
| **Kind** | [kind] |
| **Status** | [status] |
| **Owner** | [owner] |
| **Created** | [created] |
| **Last Reviewed** | [lastReviewed] — [N] days ago |
| **Staleness** | [OK | WARNING | STALE] |
| **Depends-on** | [list or "none"] |
| **supersededBy** | [ADR-XXX or "none"] |

### Skipped?
[Yes/No — reason]

### Dependency Status
- [doc-1]: [OK | STALE | NOT FOUND]
- [doc-2]: [OK | STALE | NOT FOUND]

### Milestone Status (plans only)
| Milestone | Target | Status |
|-----------|--------|--------|
| M1 | YYYY-MM-DD | [done / pending / overdue] |
| M2 | YYYY-MM-DD | [done / pending / overdue] |

### Recommendations
- [Any warnings or recommendations]
```

