---
name: scout
description: "Investigate real-system best practices, battle-tested production archetypes, and failure modes via an autonomous research subagent before architectural commitment."
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

### 3. Canonical Scout Report Format
Present findings in the structured, high-density format:

````markdown
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
````

## Workflow

**SUB-SKILL:** clarify, changeset

### Phase 1: Query Triangulation
1. Ingest the user's technical inquiry, architectural dilemma, or technology choice.
2. Formulate 2 to 3 targeted search queries crossing canonical standards, high-scale engineering cases, and failure postmortems.

### Phase 2: Autonomous Research Delegation
1. Spawn the read-only `research` subagent (Role: `Real-System Research Scout`) with the formulated queries and search budget. Never execute web searches, page reads, or exploration traces in the parent conversation stream.
2. Direct the subagent to inspect page bodies directly via URL reading tools before synthesizing. Cite strictly verified URLs returned from tool execution; never synthesize from search snippets alone or fabricate URL paths from memory.
3. Collect the subagent's structured research findings via the message channel.

### Phase 3: Delivery & Staging
1. Synthesize the collected findings into the Canonical Scout Report format and deliver strictly in the conversation stream. Never create, modify, or delete repository files.
2. Offer optional persistence to `.agents/scratch/scout-<slug>.md`. Halt turn immediately to await user review before any downstream architectural commitment.
3. Guide the user to proceed to architectural clarification via `/clarify` or change planning via `/changeset`.
