---
name: unified-docs-creator
description: >
  Creates new documentation with correct Diátaxis type, kind label, and frontmatter.
  Handles non-ADR doc creation and full ADR lifecycle management (draft → accepted →
  completed → superseded). Trigger when: "document this", "write a guide", "create doc",
  "new documentation", "create ADR", "decision record".
---

# unified-docs-creator

Creates new documentation — classified correctly, placed properly, frontmattered completely.

---

## When to Use

Activate unified-docs-creator when:
- User asks to "document this", "write a guide", "create doc"
- A new feature, component, or concept needs documentation
- A plan or roadmap doc is needed
- User asks to "create ADR", "decision record"

**For document type guidance (tutorial vs how-to, when to write an ADR), see "Document or ADR Decision" below.**

---

## Document or ADR Decision

| Situation | Action |
|-----------|--------|
| Non-ADR doc (tutorial, how-to, reference, explanation) | Follow Type System + Classification below |
| Architectural decision with lasting consequences | Use ADR Lifecycle section instead |

**Write an ADR for:** naming conventions, authentication strategy, protocol choices (REST vs gRPC), database architecture, event schema conventions, service boundary decisions, integration patterns.

**Do NOT write an ADR for:** routine refactors, single-service details, tooling changes (use a TIL instead).

---

## ADR Lifecycle

**CRITICAL — ADR Classification Check:** If the user asks to document an architectural decision with lasting consequences, an authentication strategy, naming convention, protocol choice, database architecture, service boundary decision, or integration pattern — the type MUST be `decision`, NOT `explanation`. The skill must NOT classify an ADR as `explanation` under any circumstances.

### ADR Frontmatter (Required)

When creating an ADR, the frontmatter MUST include:

```yaml
---
title: [Short descriptive title — noun phrase, not verb phrase]
type: decision                              # ← MUST be "decision", NOT "explanation"
kind: [adr]                                 # ← MUST include "adr"
adr-id: ADR-[NNN]                           # ← sequential, zero-padded
status: draft                               # ← start as draft
deciders: [name, name]
decided:                                    # ← filled when status becomes accepted
supersededBy:
depends-on: []
updates: []
---
```

> **Common mistake to avoid:** The skill frequently misclassifies ADR content as `type: explanation`. ADRs are **decisions**, not explanations. An ADR explains the reasoning *for* a decision, but its type is `decision`. Do NOT set `type: explanation` for an ADR.

| Canonical type | What it answers | Alternative names |
|----------------|-----------------|-------------------|
| `tutorial` | "How do I learn this?" | walkthrough, getting-started |
| `how-to` | "How do I accomplish X?" | howto, guide, howto-guide |
| `reference` | "What does X do?" | ref, api, spec |
| `explanation` | "Why is it designed this way?" | explain, overview, concept |
| `decision` | "What design choice was made and why?" | adr |

> **Type aliases are accepted on read, normalized on write.**

---

## Document Classification

### Step 1 — Title Pattern (Priority)

If the title matches, classify by title regardless of body keywords:

| Title pattern | → Type |
|---------------|--------|
| "getting started", "introduction to", "learn", "basics", "first steps", "beginner", "walkthrough", "hands-on", "step by step", "build your first", "from scratch", "crash course", "prerequisites", "setup from zero" | Tutorial |
| "how to", "how do I", "guide to" | How-to |
| "api", "reference", "specification", "changelog", "migration guide" | Reference |
| "why", "architecture", "design decisions", "rationale", "overview" | Explanation |
| "plan", "roadmap", "test-plan", "qa-plan", "strategy" | Explanation — see Kind Assignment for `kind: [plan]` details |

### Step 2 — Keyword Matrix

If title doesn't match, check body keywords:

| Type | Keywords |
|------|----------|
| Tutorial | "getting started", "introduction", "learn", "basics", "first steps", "walkthrough", "hands-on", "step by step", "build", "create from scratch", "crash course" |
| How-to | "how to", "guide to", "how do I", "setting up", "configure", "install", "deploy", "enable", "disable", "manage", "add", "remove", "integrate", "authenticate", "implement", "build with" |
| Reference | "api", "reference", "specification", "configuration", "options", "parameters", "flags", "cli", "commands", "schema", "types", "interface", "endpoints", "changelog" |
| Explanation | "why", "concept", "architecture", "design", "decisions", "rationale", "theory", "overview", "background", "understanding", "tradeoffs", "compared to", "vs", "history" |

### Step 3 — Type Resolution for Ambiguous Docs

| Dominant content | → Type |
|-----------------|--------|
| Steps with concrete result | Tutorial |
| Commands, task-oriented steps | How-to |
| Tables, signatures, exhaustive lists | Reference |
| Rationale, concepts, "why" | Explanation |

---

## Kind Assignment

| kind | Use when |
|------|----------|
| `ssot` | Canonical source — never copy, always link |
| `cross-cutting` | Referenced from multiple services or packages |
| `plan` | A plan, roadmap, test-plan, or strategy document |
| `runbook` | Operational procedures |
| `til` | Today I learned entry |
| `draft` | Work-in-progress, not yet stable |
| `stale` | Outdated, flagged for review (set by unified-docs-doctor) |

Multiple `kind` values allowed: `kind: [plan, ssot]` is valid.

---

## Project Structure Detection

Before creating, detect the project structure. Extract: docs folder name, existing categorization, conventions.

**Placement uncertainty → Ask user.** Do not guess a path.

```
The project structure doesn't have a clear location for a [type] doc about [topic].
Should I place it in [option A], [option B], or somewhere else?
```

Standard structure patterns:

```
/README.md
/docs/README.md
/docs/index.md
[docs-folder]/SUMMARY.md       ← MkDocs/Docusaurus
[docs-folder]/_Sidebar.md
```

---

## Required Frontmatter

Every new document **must** start with this YAML frontmatter:

```yaml
---
title: [Short descriptive title — noun phrase]
type: [tutorial | how-to | reference | explanation | decision | runbook]
kind:   [plan, adr, runbook, til, ssot, cross-cutting, draft, ...]
audience: [new contributors | developers | api consumers | operators | maintainers]
owner: [team-name | agent-name | unassigned]
created: [ISO date — e.g., 2026-04-18]
lastReviewed: [ISO date — same as created initially]
reviewCadence: [30 | 90 | 180]   # plans: 90 | how-to/reference: 180 | tutorial: 180
depends-on: []    # list docs this one depends on
updates: []       # list docs this will update when completed
---
```

> **Critical for plans:** Always set `reviewCadence: 90`. Plans drift fast — 90-day review cadence is the minimum. For plans with multiple milestones that span >3 months, also set a `reviewCadence: 30` warning in the plan body.

> **Critical for runbooks:** When creating a runbook, `type` MUST be `runbook` — NOT `how-to` or `explanation`. The Diátaxis runbook type is distinct from how-to. Do NOT set `type: how-to` for a runbook.

> **Note:** `depends-on` and `updates` are valid metadata fields for **doc content** frontmatter. They are NOT skill frontmatter fields.

---

## Writing Rules per Type

| Type | Structure | What to avoid |
|------|-----------|---------------|
| **Tutorial** | **The very first line after frontmatter must be `## Step 1`.** No numbered list, no paragraph, no sentence before the first heading. Concrete result at end. | NEVER write: "This tutorial...", "In this guide...", "Welcome to...", or any paragraph before `## Step 1`. The doc starts with the heading `## Step 1`, not with prose or an unnumbered list. |
| **How-to** | **The very first line after frontmatter must be `## Step 1` or `## Prerequisites`.** No paragraph, no sentence, no dash, no list before the first heading. Jump straight to the first heading. | NEVER write: "This guide covers...", "In this how-to...", "Before you begin...", or any paragraph before `## Step 1`. The doc starts with the first heading, not with prose. |
| **Reference** | Dense tables/lists. No opinions. | No personal takes on quality |
| **Explanation** | Narrative, "why". May reference other types. | Not a how-to — no step lists |
| **Plan / Roadmap** | Structured sections: Context → Goals → Milestones → Timeline → Risks. Table of milestones preferred. | Not a how-to — no task-step lists; milestones are deliverables, not actions |
| **Runbook** | `## Prerequisites` → numbered procedure → decision table → rollback section → post-action checklist. Jump straight to `## Prerequisites` — no intro prose. | No narrative overview before the first heading. Decision table is required for operational runbooks. |

---

## Plan / Roadmap Lifecycle

Use this section when the user asks to write a plan, roadmap, strategy, test-plan, qa-plan, or migration plan.

### When to Write a Plan

Write a `kind: [plan]` doc when the title or request contains:
- "roadmap", "plan", "strategy", "test-plan", "qa-plan", "migration plan"
- "milestones", "timeline", "phases", "stages"
- "implementation plan", "adoption plan", "rollout plan"

**Do NOT write a plan when** the user wants steps to accomplish a one-time task (use a how-to instead).

### Plan Frontmatter (Required)

```yaml
---
title: [Short descriptive title — noun phrase]
type: explanation
kind: [plan]                                  # ← MUST include "plan"
status: [draft | in-progress | completed]
owner: [team-name | agent-name | unassigned]
audience: [who needs to know this plan]
created: [ISO date]
lastReviewed: [ISO date]
reviewCadence: 90                             # ← plans need more frequent review
depends-on: []
updates: []
---
```

### Plan Body Structure

```markdown
# [Title]

## Context
[Why this plan exists. What problem it solves or what goal it advances. What constraints apply.]

## Goals
[What success looks like. Numbered or bulleted — concrete, measurable outcomes.]

## Milestones
[Deliverable-oriented milestones with target dates. Use a table:]

| Milestone | Description | Target Date | Owner |
|----------|-------------|------------|-------|
| M1 | [First milestone deliverable] | YYYY-MM-DD | @name |
| M2 | [Second milestone deliverable] | YYYY-MM-DD | @name |

## Timeline
[Phase-by-phase breakdown or Gantt-style overview. Include dependencies between milestones.]

## Risks & Mitigations
| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| [risk] | [H/M/L] | [H/M/L] | [mitigation] |

## Status Log
- YYYY-MM-DD — Draft
- YYYY-MM-DD — [status change with reason]
```

### Plan Writing Rules

- **Milestones are deliverables, not tasks.** Write "API gateway deployed" not "Deploy API gateway".
- **Goals are measurable.** Vague goals ("improve observability") → specific ("P99 latency < 200ms across all endpoints").
- **No task-step lists.** The body is NOT a how-to. Milestones advance when deliverables ship, not when someone does a step.
- **Risks table required** for plans with more than one milestone.

### Updating a Plan

When a milestone completes:

1. Update the milestone's row in the table (add completion date or strike-through)
2. Add a new entry to **Status Log** with today's date
3. If `status: completed` → run cascade update for all docs in `depends-on:`
4. Update `lastReviewed` to today

> **Critical — `updates:` field is for document paths only.** Never put status log strings or prose in `updates:`. The field holds YAML list items pointing to doc files (e.g., `- docs/architecture/sys-arch.md`). Status Log entries belong in the **Status Log section of the body**, not in frontmatter.

---

## Duplicate Scan

**Do NOT create a doc without checking for duplicates first.**

| Tier | Method | Scope |
|------|--------|-------|
| 1 | Filename scan | Matching or near-matching file names (case-insensitive) |
| 2 | Title/heading scan | Matching title keywords in all doc titles and first headings |
| 3 | Full-text scan *(only when ambiguous)* | Core concept in all doc bodies |

- Only after all 3 tiers come back clean → proceed as new doc.
- If tier 2 or 3 finds a match → link to the existing doc instead of creating a new one.
- **Topic overlap check (mandatory for Tier 2):** Even if the title doesn't match, check whether the requested doc topic is semantically covered by an existing doc. Example: a request for "Authentication Setup Guide" overlaps with "Auth Implementation Guide" (both about auth) — this is a Tier 2 match. When in doubt, scan the first 200 lines of docs in the same folder for related keywords.

### Cascade Field Setup

Before creating, set `depends-on:` and `updates:` in the doc's frontmatter:

| Field | Meaning |
|-------|---------|
| `depends-on:` | Which docs must be current before this doc is valid |
| `updates:` | Which docs will this doc update when completed |

```yaml
depends-on:
  - sys-arch.md
updates:
  - sys-arch.md
  - ref-auth.md
```

---

## Quality Checklist

Before marking a doc creation done:

| # | Check | How |
|---|-------|-----|
| 1 | **Correct type** | Matches its declared type's structure rules? |
| 2 | **Frontmatter complete** | Has title, type, kind, audience, owner, created, lastReviewed? |
| 3 | **No duplication** | Topic already covered elsewhere? Link instead. |
| 4 | **SSOT-aligned** | Config values → "see [source]". Never copy from authoritative source. |
| 5 | **Right level** | Scan headings and scope — do they match the declared audience? |
| 6 | **Actionable** | Can the reader do something after reading this? |
| 7 | **No bare URLs** | Internal links use relative paths |
| 8 | **Cascade fields set** | `depends-on:` and `updates:` filled where applicable |
| 9 | **Placement confirmed** | User was asked if placement was unclear |

For ADR creation, also verify:

| # | Check | How |
|---|-------|-----|
| 1 | **Correct type** | `type: decision` in frontmatter |
| 2 | **Frontmatter complete** | Has `adr-id`, `status`, `deciders`, `decided` |
| 3 | **Structure complete** | Context → Decision → Consequences → Status Log |
| 4 | **Status Log updated** | New status entry added with date |
| 5 | **No duplicate decision** | Same topic covered elsewhere? Link, don't duplicate |
| 6 | **Cascade prepared** | If `status: completed`, do `updates:` docs exist? |
| 7 | **Register updated** | ADR register reflects new status |

For Plan creation, also verify:

| # | Check | How |
|---|-------|-----|
| 1 | **`reviewCadence: 90`** | Plans must have `reviewCadence: 90` — not 30, not 180 |
| 2 | **Milestones table** | Has Description, Target Date, Owner columns; at least 3 rows |
| 3 | **Risks table** | Risks & Mitigations table present for plans with >1 milestone |
| 4 | **Goals are measurable** | Vague goals ("improve X") rewritten as specific outcomes |
| 5 | **Milestones are deliverables** | "PostgreSQL deployed" not "Deploy PostgreSQL" |

For Runbook creation, also verify:

| # | Check | How |
|---|-------|-----|
| 1 | **`type: runbook`** | `type` MUST be `runbook` — not `how-to`, not `explanation` |
| 2 | **Prerequisites section** | `## Prerequisites` present before the procedure |
| 3 | **Numbered procedure** | At least 5 steps with concrete actions |
| 4 | **Decision table** | Required for operational runbooks (failover, incident, rollback) |
| 5 | **Rollback section** | Present with at least 2 rollback steps |
| 6 | **Post-action checklist** | Final checklist section present |