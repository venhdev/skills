---
name: clarify
description: "Resolve architectural ambiguities, edge cases, and system trade-offs for proposed features or refactors before implementation planning."
---

# clarify — Architectural Deliberation Engine

Uncover hidden assumptions, evaluate architectural trade-offs, and establish definitive system invariants before implementation planning begins.

## Operating Invariants

- **Read-Only Enclosure**: Mandate maintaining zero filesystem mutations; forbid editing files, drafting code diffs, or executing mutating commands.
- **Scope Discipline**: Mandate confining inquiry strictly to high-impact architectural decisions, failure modes, and system invariants; forbid syntax bikeshedding, micro-optimizations, or tangential scope expansion.
- **Sequential Turn Discipline**: Mandate formulating strictly one question per turn and halting immediately to await user response; forbid asking multi-question lists or cascading without user answers.
- **Deliberation Budget**: Mandate limiting inquiries to a maximum of 3 to 5 questions; forbid open-ended interrogation loops. Short-circuit immediately to synthesis when no architectural ambiguities remain.
- **Contradiction Circuit Breaker**: If proposed requirements conflict with established SSOT specifications or ADRs, mandate halting immediately, citing the contradiction with file pointers, and requesting resolution before proceeding; forbid continuing execution on contradictory specifications.

## Domain Engine & Standards

### 1. Deliberation Dimensions
Evaluate proposals against 5 core architectural dimensions:
- **Data Contracts & Schemas**: Boundaries, serialization, entity models, and migration compatibility.
- **State & Lifecycle**: Source of truth, mutation paths, concurrency, and persistence.
- **Failure Modes & Resilience**: Error boundaries, partial failure, fallback strategies, and recovery.
- **System Boundaries**: Service responsibilities, dependency blast radius, and external integrations.
- **Non-Functional Guardrails**: Latency, throughput limits, scalability constraints, and security.

### 2. Traversal & Anti-Bikeshedding Guardrails
- **Root-First Traversal**: Identify and resolve the highest-impact architectural branching point before exploring dependent details.
- **Dynamic Branch Pruning**: Update the decision tree immediately upon receiving user input; eliminate branches rendered obsolete by settled answers.
- **Guardrail (When NOT to Ask)**: If a decision is an idiomatic implementation detail with no observable cross-module impact or system trade-off, adopt the standard convention without prompting the user.

### 3. Canonical Question Payload Format
Present each question using the high-density structured format (under 15 lines, omitting conversational preamble and decorative emojis):

```text
### Q<N>: <Architectural Dimension>
Bottleneck: <Single-sentence technical tension or edge-case risk>

• [A] <Option Title> ── Trade-off: <Advantage> vs. <Disadvantage>
• [B] <Option Title> ── Trade-off: <Advantage> vs. <Disadvantage>
• [C] <Option Title> ── Trade-off: <Advantage> vs. <Disadvantage>

Recommendation: [Option] because <Single-sentence technical rationale>
```

### 4. Canonical Decision Matrix Format
Compile settled decisions into the canonical tabular format:

```markdown
# Architectural Decision Matrix: <Feature / Scope Name>

| Ref | Architectural Dimension | Chosen Decision | Rationale & Trade-off | Invariants & Guardrails |
| :--- | :--- | :--- | :--- | :--- |
| Q1 | <Dimension> | <Selected Option> | <Core Benefit & Accepted Cost> | <Technical Invariant> |
```

## Execution Protocol

**SUB-SKILL:** changeset, ssot

### Phase 1: Surface & Gap Ingestion
1. Inspect the user proposal, referenced specifications, and authoritative contracts using non-mutating capabilities (inspection tools, discovery scripts, or delegated exploratory subagents).
2. Ground inspection in authoritative repository documentation or architectural schemas. If documentation boundaries or canonical SSOTs require auditing, leverage sub-skill `ssot` when available.
3. Identify latent assumptions, failure modes, data contracts, and backward compatibility risks against Deliberation Dimensions.
4. **Clean-Pass Short-Circuit**: If the scope contains zero architectural ambiguities, proceed immediately to Phase 3.
5. Otherwise, select the primary architectural dimension and proceed to Phase 2.

### Phase 2: Sequential Deliberation Interview
1. Select the single most impactful unresolved architectural dilemma using Root-First Traversal.
2. Formulate strictly **one** question per turn labeled sequentially (`Q1`, `Q2`, ...) adhering to the Canonical Question Payload Format.
3. Halt turn immediately after asking. Wait for user response.
4. Ingest user answer, apply Dynamic Branch Pruning, and evaluate remaining ambiguities.
5. Repeat Phase 2 until all critical vectors are settled (maximum 3 to 5 questions), then proceed to Phase 3.

### Phase 3: Decision Matrix Synthesis
1. Compile all settled decisions into the Canonical Decision Matrix Format.
2. **Terminal Delivery Gate**: Deliver the completed `Decision Matrix` and halt turn immediately. Forbid proposing file mutations, generating diffs, or transitioning to implementation within this turn.
3. **Pipeline Transition**: Guide user to proceed to blast-radius planning via `/changeset`.
