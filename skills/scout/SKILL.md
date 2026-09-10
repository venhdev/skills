---
name: scout
description: "Investigate real-system best practices, production archetypes, and failure modes before architectural commitment."
disable-model-invocation: true
---

# scout — Real-System Best Practices & Production Archetype Engine

Investigate battle-tested production archetypes, real-world failure modes, and boring tech baselines via an autonomous research subagent before committing to architectural decisions.

## Domain Engine & Standards

### 1. Source Authority Hierarchy

Triangulate technical inquiries against authoritative production evidence:

- **Tier 1 (Canonical Standards & Official Specs)**: RFCs, language specifications, cloud vendor reference architectures, and formal technical papers.
- **Tier 2 (High-Scale Production Engineering)**: Tech blogs and architecture disclosures from proven high-scale engineering organizations (e.g., Stripe, Netflix, Cloudflare, Discord, Uber, Meta).
- **Tier 3 (Battle Scars & Postmortems)**: Documented production outages, postmortems, and root-cause analyses revealing edge cases, operational bottlenecks, and failure modes.
- **Exclusion Filter**: Exclude SEO-optimized content farms, unverified personal blog posts, vendor marketing collateral, and unvetted AI summaries.

### 2. Search & Inclusion Budget

- **Exploration Ceiling**: Maximum 8 search queries and 10 fetched pages per scouting run.
- **Inclusion Budget**: Synthesize from 3 to 5 premier, high-signal sources.
- **Short-Circuit Stop**: Terminate search as soon as the canonical pattern and known failure modes are triangulated with primary evidence.

## Canonical Scout Report Format

```markdown
# Scout Report: <Topic / Inquiry>

## 1. Executive Summary
- **Canonical Production Archetype**: <Boring, battle-tested standard approach adopted by industry leaders>
- **Operational Reality**: <Key complexity, maintenance overhead, or scaling bottleneck in practice>

## 2. Industry Precedents & Battle Scars
| Source / Organization | Pattern Deployed | Production Scale / Context | Known Failure Mode / Trade-off |
| :--- | :--- | :--- | :--- |
| <Org / Paper> | <Architecture / Tool> | <Traffic, Data Volume, SLA> | <Failure point, latency spike, cost trap> |

## 3. Recommended Baseline vs. Traps
- **Recommended Baseline**: <Simplest robust architecture meeting requirements without premature optimization>
- **Premature Traps to Avoid**: <Over-engineered patterns or hype-driven tools that introduce unnecessary failure modes>

## 4. Primary Citations
- [<Source Title>](<Verified URL>) — <Core insight or concrete data point extracted>

## 5. Pipeline Routing
- `/clarify` ── Deliberate trade-offs or evaluate architectural fit for the project.
- `/changeset` ── Archetype is selected and baseline approach is ready for change planning.
```

## Delegation & Synthesis Protocol

**SUB-SKILL:** clarify, changeset

1. **Query Triangulation & Dispatch**:
   - Formulate 2 to 3 targeted search queries crossing: canonical standards/RFCs, high-scale engineering cases, and failure postmortems.
   - Launch a `research` subagent (Role: `Real-System Research Scout`) supplying:
     - The triangulated queries and Search Budget (max 8 queries, 10 fetched pages).
     - The **Canonical Scout Report Format** above as the required output contract.
   - Mandate the subagent to inspect full page bodies via URL tools before synthesizing; cite strictly verified URLs and never synthesize from search snippets alone.

2. **Synthesis & Halt**:
   - Emit the completed Canonical Scout Report into the conversation stream and halt turn immediately. Never create, modify, or delete repository files.
