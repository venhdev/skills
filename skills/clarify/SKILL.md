---
name: clarify
description: Use when requirements are ambiguous, architectural trade-offs require alignment, latent risks or edge cases need surfacing, or before planning multi-file mutations.
---

# clarify — Architectural Deliberation Engine

Uncover hidden assumptions, evaluate architectural trade-offs, and establish definitive system invariants before implementation planning begins.

## Operating Principles

- **Read-Only Enclosure**: Maintain zero filesystem mutations. Never edit files, draft code diffs, or execute build commands.
- **Root-First Traversal**: Identify and resolve the highest-impact architectural branching point before exploring dependent details.
- **Dynamic Branch Pruning**: Update the decision tree immediately upon receiving user input; eliminate branches made obsolete by settled answers.
- **High-Density Payload**: Keep each question concise, structured, and under 15 lines. Omit conversational preamble and decorative emojis.

## Process

**OPTIONAL SUB-SKILL:** ssot

### Phase 1: Surface & Gap Ingestion
1. Inspect the user proposal, referenced specifications, and authoritative contracts using available non-mutating capabilities (environment inspection tools, discovery scripts, or delegated exploratory subagents).
2. Ground inspection in authoritative repository documentation or architectural schemas. If documentation boundaries or canonical SSOTs require auditing, leverage sub-skill `ssot` when available.
3. Identify latent assumptions, failure modes, data contracts, and backward compatibility risks.
4. If the scope contains zero architectural ambiguities, proceed immediately to Phase 3.

### Phase 2: Dynamic Socratic Interview
1. Select the single most impactful unresolved architectural dilemma.
2. Formulate strictly **one** question per turn, labeled sequentially (`Q1`, `Q2`, ...).
3. Present the question using the High-Density Micro-Payload format:

   ```text
   ### Q<N>: <Architectural Dimension>
   Bottleneck: <Single-sentence technical tension or edge-case risk>

   • [A] <Option Title> ── Trade-off: <Advantage> vs. <Disadvantage>
   • [B] <Option Title> ── Trade-off: <Advantage> vs. <Disadvantage>
   • [C] <Option Title> ── Trade-off: <Advantage> vs. <Disadvantage>

   Recommendation: [Option] because <Single-sentence technical rationale>
   ```

4. Halt turn immediately after asking. Wait for user response.
5. Ingest user answer, prune obsolete branches, and repeat Phase 2 until all critical vectors are settled (maximum 3 to 5 questions), then proceed to Phase 3.

### Phase 3: Decision Matrix Synthesis
1. Compile all settled decisions into the canonical `Decision Matrix` table:

   ```markdown
   # Architectural Decision Matrix: <Feature / Scope Name>

   | Ref | Architectural Dimension | Chosen Decision | Rationale & Trade-off | Invariants & Guardrails |
   | :--- | :--- | :--- | :--- | :--- |
   | Q1 | <Dimension> | <Selected Option> | <Core Benefit & Accepted Cost> | <Technical Invariant> |
   ```

2. **Terminal Delivery Gate**: Deliver the completed `Decision Matrix` and halt turn immediately. Forbid proposing file mutations, generating diffs, or transitioning to implementation within this turn.
