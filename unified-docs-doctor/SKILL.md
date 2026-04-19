---
name: unified-docs-doctor
description: >
  Performs comprehensive documentation health audits: stale detection, orphan
  detection, duplicate detection, cascade tracing, and SSOT violation reporting.
  Also verifies consistency between doc outputs from unified-docs-creator and
  unified-docs-reader. Includes cascade update workflow for post-audit fixes.
  Trigger when: "cleanup", "audit", "fix scattered docs", "check docs health",
  "stale", "orphan", "doc health report", "run audit".
---

# unified-docs-doctor

The documentation health inspector.

> **Verifies outputs from unified-docs-creator and unified-docs-reader.** When a doc task completes, run an audit to verify consistency.

---

## Activation Triggers

Activate unified-docs-doctor when:
- User says "cleanup", "audit", "find duplicates", "fix scattered docs"
- A doc task completes → run audit to verify consistency
- User asks "what's the health of our docs?"
- Before a release or major change → verify doc consistency
- After any doc creation/update → verify cascade was applied

---

## Audit Workflow

### Phase 1 — Discovery

Run before touching any doc. Detects the current state.

#### 1.1 Scan README files

```
/README.md
/docs/README.md
/docs/index.md
[docs-folder]/SUMMARY.md       ← MkDocs/Docusaurus
[docs-folder]/_Sidebar.md
```

From these, extract:
- Docs folder name
- Project conventions
- Existing type categorization

#### 1.2 List all docs

```bash
find . -name "*.md" -not -path "*/node_modules/*" -not -path "*/dist/*" -not -path "*/build/*" | sort
```

For each doc, extract:
- File path
- Title (from frontmatter or first H1)
- Type (from frontmatter)
- Kind (from frontmatter)
- Status (from frontmatter)
- Owner (from frontmatter)
- `lastReviewed` date
- `depends-on:` list
- `updates:` list

#### 1.3 Classify all docs

For each doc, apply the keyword matrix:

| Title pattern / body keyword | → Type |
|------------------------------|--------|
| Tutorial patterns | Tutorial |
| How-to patterns | How-to |
| Reference patterns | Reference |
| Explanation patterns | Explanation |
| `type: decision` in frontmatter | Decision |
| `kind: [plan]` in frontmatter | Plan |
| No match | Flag as "unclassified" |

Flag mismatches: doc says `type: how-to` but content looks like `explanation`.

---

### Phase 2 — Violation Detection

#### 2.1 Stale Detection

A doc is stale when either condition is true:

| Condition | Result |
|-----------|--------|
| `lastReviewed` older than `reviewCadence` days (default: 180) | Stale |
| `kind: [stale]` present in frontmatter | Stale |

**Script:**

```bash
for doc in $(find . -name "*.md" -not -path "*/node_modules/*" -not -path "*/dist/*" -not -path "*/build/*"); do
  lastReviewed=$(grep 'lastReviewed:' "$doc" | head -1 | awk '{print $2}')
  if [ -n "$lastReviewed" ]; then
    # Respect per-doc-type cadence: plans default to 90 days, others to 180
    cadence=$(grep 'reviewCadence:' "$doc" | head -1 | awk '{print $2}' | tr -d ' ')
    if [ -z "$cadence" ]; then
      kind=$(grep 'kind:' "$doc" | head -1)
      if echo "$kind" | grep -q 'plan'; then
        cadence=90
      else
        cadence=180
      fi
    fi
    days=$(( ($(date +%s) - $(date -d "$lastReviewed" +%s)) / 86400 ))
    if [ $days -gt "${cadence:-180}" ]; then
      echo "STALE: $doc (${days} days, cadence ${cadence})"
    fi
  fi
done
```

**Action:** Set `kind: [stale]` on stale docs (add, don't replace existing kind).

#### 2.2 Orphan Detection

A doc is an orphan when any of these conditions hold:

| Condition | Meaning |
|-----------|---------|
| No keywords match any type | Unclassifiable |
| No clear audience | Directionless |
| NOT reachable from any index/SUMMARY | Disconnected |
| Links to nothing | Floating |
| Nothing links to it | Unreferenced |

> **Note:** Orphan detection runs after Discovery — the "NOT reachable" check uses docs found in `#### 1.1 Scan README files`.

**Script:**

```bash
# 1. Get all docs listed in SUMMARY/index
summary_docs=$(grep -Eo '\]\([^)]+\.md\)' docs/SUMMARY.md 2>/dev/null | sed 's/.*(\([^)]*\)).*/\1/')
# 2. Get all docs linked from any other doc
linked_docs=$(grep -rho '\]\([^)]*\.md\)' docs/ --include="*.md" 2>/dev/null | sed 's/.*(\([^)]*\)).*/\1/' | sort -u)
# 3. Docs in neither list → orphan candidates
```

**Action:** Delete or mark `kind: [stale]`. Ask user before deleting.

#### 2.3 Duplicate Detection

Use this 3-tier scan:

1. **Filename scan** (fast): Matching or near-matching file names
2. **Title/heading scan** (medium): Matching title keywords in all doc titles and first headings
3. **Full-text scan** (deep, for ambiguous cases): Core concept in all doc bodies

```bash
grep -r "^title:" docs/ --include="*.md" | grep -i "auth"
grep -ri "authentication" docs/ --include="*.md" | grep -v "binary file"
```

**Action:** For duplicates, keep one canonical doc (`kind: [ssot]`), add redirect links elsewhere.

#### 2.4 SSOT Violation Detection

A doc violates SSOT when:

| Check | Violation |
|-------|-----------|
| Config values or contract specs copied verbatim | SSOT violation — use link instead |
| "see [source]" absent for authoritative content | SSOT violation |

**Action:** Replace duplicated content with "see [source]" + link.

#### 2.5 Plan Violation Detection

A `kind: [plan]` doc violates its contract when:

| Check | Violation |
|-------|-----------|
| No milestones table | Plans must have a milestones table with target dates |
| Milestones are task-lists, not deliverables | "Install PostgreSQL" is a task; "PostgreSQL deployed to staging" is a deliverable |
| `status` field is absent | Plans must have a status (draft / in-progress / completed) |
| Overdue milestones not flagged | Any milestone with a past target date that is not completed |
| `reviewCadence` > 90 days | Plans drift — use 90-day cadence minimum |

**Action:** Flag each violation in the audit report. For overdue milestones, compute days overdue and recommend a plan refresh.

#### 2.6 Broken Link Detection

A link is broken when its target file does not exist or its anchor does not exist in the target file.

**Two types of broken links:**

| Type | Example | Detection |
|------|---------|-----------|
| Dead cross-file | `[ADR-003](./decisions/ADR-003.md)` — target file does not exist | Resolve relative path; check file exists on disk |
| Dead anchor | `[Step 3](#nonexistent-anchor)` — heading does not exist | Extract all `## ` headings from target file; check anchor exists |

**Cross-file link script:**

```bash
# Find all markdown links to .md files
grep -rho '\]\([^)]*\.md\)' docs/ --include="*.md" | sort -u | while read link; do
  target=$(echo "$link" | sed 's/.*(\([^)]*\)).*/\1/')
  # Resolve relative to the doc that contains the link
  if [ ! -f "docs/$target" ] && [ ! -f "$target" ]; then
    echo "BROKEN: $link (file not found)"
  fi
done
```

**Anchor link script:**

```bash
# For each .md file, extract all ## headings and check for dead anchors
for doc in docs/*.md docs/**/*.md; do
  anchors=$(grep '^## ' "$doc" | sed 's/## //' | sed 's/ /-/g' | tr '[:upper:]' '[:lower:]')
  # Find all #anchor references in other docs and cross-reference
done
```

**Fix action for broken links:**
- Dead cross-file link → fix the relative path if possible, otherwise **strike through or remove** the link
- Dead anchor link → fix the anchor reference to match an existing heading, otherwise **strike through or remove** the link
- Do NOT leave broken links in the document — they erode reader trust

> **Strike-through format for unfixable links:**
> `~~[Auth Setup](./auth-setup.md)~~ — removed (file does not exist)`

---

### Phase 3 — Cascade Tracing

#### 3.1 Trace Dependency and Updates Chains

1. Find all SSOT docs: `grep -rl "kind:.*ssot" docs/`
2. For each SSOT doc, echo the file path and basename
3. Trace dependents: `xargs grep -l "$basename"` from docs referencing it
4. Trace updates: find docs with `status: completed` and empty `updates:` — flag if SSOT was updated without cascade

**Script:**

```bash
for doc in $(grep -rl "kind:.*ssot" docs/ --include="*.md" 2>/dev/null); do
  basename=$(basename "$doc")
  echo "SSOT: $doc"
  grep -rl "depends-on:" docs/ --include="*.md" 2>/dev/null | xargs grep -l "$basename" && echo "  ↑ depends on this"
done
```

#### 3.3 Flag cascade gaps

| Condition | Flag |
|-----------|------|
| SSOT updated but `updates:` not set | "Missing cascade field" |
| `depends-on:` doc doesn't exist | "Broken dependency" |
| `depends-on:` doc is `superseded` | "Dependency superseded" |
| `depends-on:` doc is stale | "Dependency stale — may be unreliable" |

---

### Phase 4 — Inter-Skill Consistency Check

Verifies outputs from unified-docs-creator and unified-docs-reader are consistent.

#### 4.1 unified-docs-creator output check

For each doc created:
- Frontmatter complete? (all required fields present)
- Type matches content?
- `depends-on:` and `updates:` set where applicable?
- Placement confirmed?

#### 4.2 unified-docs-reader output check

For each doc read:
- `lastReviewed` updated to today?
- `lastUpdatedBy` set?
- Cascade propagated correctly?
- No broken links after move?

---

### Phase 5 — Report

Present findings in this format:

```markdown
## Documentation Health Report — [date]

### Summary

| Metric | Count |
|--------|-------|
| Total docs | N |
| Stale | N |
| Orphan | N |
| Duplicate | N |
| SSOT violations | N |
| Broken dependencies | N |
| Health score | [OK / WARNING / CRITICAL] |

### Stale Docs (lastReviewed > 180 days)
| Doc | Last Reviewed | Days ago | Owner |
|-----|--------------|---------|-------|
| [doc-1] | 2025-10-01 | 197 | @user |

### Orphan Docs
| Doc | Reason | Action |
|-----|--------|--------|
| [doc-1] | No audience, no links | Mark stale? Delete? |

### Cascade Gaps
| Doc | Gap | Recommended Action |
|-----|-----|-------------------|
| [doc-1] | Missing updates: field | Add updates: [affected doc] |

### Inter-Skill Violations
| Skill | Violation | Doc |
|-------|-----------|-----|
| unified-docs-creator | Frontmatter incomplete | [doc-1] |
| unified-docs-reader | lastReviewed not updated | [doc-2] |

### Recommendations
1. [Top 3 actions by priority]
```

---

## Fix Workflow (Post-Audit)

After audit completes and user approves recommended actions, apply fixes:

### Cascade Update

When an SSOT or completed ADR is updated:

1. **Read the updated doc** — check frontmatter and status
2. **Trace updates chain** — find all docs in `updates:` field
3. **Propagate to downstream docs** — for all docs that have this in `depends-on:`

For each doc in `updates:` or `depends-on:`:
- Read the target doc
- Apply necessary changes
- Update `lastReviewed` to today
- Set `lastUpdatedBy` to agent name
- Warn if downstream doc's `lastReviewed` is older than the updated doc

### Mark Doc as Completed

When an ADR or plan transitions to `status: completed`:

1. Update the doc itself:
   ```yaml
   status: completed
   lastReviewed: {{today}}
   lastUpdatedBy: {{agent-name}}
   ```
2. Run cascade update for all docs in `updates:`
3. Notify downstream docs (those with this in `depends-on:`)

### Update Plan Milestones

When a plan milestone completes (or is overdue):

1. Read the plan doc's milestones table
2. For each completed milestone:
   - Add completion date to the table row (add a `Completed` column)
   - Add entry to **Status Log**: `YYYY-MM-DD — Milestone [name] completed`
   - Update `lastReviewed` to today
3. For each overdue milestone (target date passed, not completed):
   - Flag in audit report: `Milestone [name] is [N] days overdue — target was YYYY-MM-DD`
   - Add entry to Status Log: `YYYY-MM-DD — Milestone [name] overdue`
4. Cascade: if plan `status: completed`, run cascade update for all `updates:` docs

### Frontmatter Patch

When only frontmatter needs updating (no body change):

1. Identify the field(s) to change
2. Use Edit to update frontmatter only
3. Update `lastReviewed` to today
4. No cascade needed for pure frontmatter changes

---

## Health Score Formula

```
health_score = (total_docs - stale - orphan - duplicate - violations) / total_docs * 100

[OK]       → health_score >= 90
[WARNING]  → health_score >= 70
[CRITICAL] → health_score < 70
```

---

## Safety and Operational Rules

1. **Act only on explicit user approval.** Never auto-delete, auto-move, or auto-merge. Present findings and ask: "Should I apply these changes?"
2. **Preserve historical context.** When marking a doc as `stale`, add a note: "Marked stale on [date]. See [replacement] for current content."
3. **Cascade before reporting.** When an SSOT is flagged stale, trace all dependent docs and include them in the report.
4. **Verify superseded chains.** Make sure no ADR is linked to a non-existent replacement.

---

## Post-Audit Actions

After audit completes:
1. Ask user to approve recommended actions
2. If approved → apply fixes using Fix Workflow above
3. Re-run audit to confirm improvements
4. Update health score in report