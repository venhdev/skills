# ZenForge Construction Rules

Authoritative architectural laws and quality standards for all ZenForge skills and templates.

---

## 1. Universal & Decoupled

- **Stack & Host Agnostic**: Author all instructions independent of specific programming languages, frameworks, or host repositories.
- **Cross-LLM Determinism**: Ensure high-fidelity execution across all frontier LLM families without model-specific quirks.

## 2. Zero-Waste & Anti-No-Op

- **Observable Behavioral Delta**: Every directive must measurably constrain output distribution. Ruthlessly prune conversational filler, preambles, and generic advice modern LLMs follow by default.
- **Qualitative Density**: Enforce brevity through explicit scope exclusions; forbid brittle numerical quotas (line counts, arbitrary step numbers).
- **Tool-Aware Zero-Waste**: Forbid defensive rules, speculative circuit breakers, or retry instructions for behaviors that underlying tooling, permissions, or native error returns already handle.
- **Zero Transitional Glue**: Attach templates and schemas directly beneath governing headings without conversational filler.

## 3. Affirmative Framing & Paired Directives

- **Paired Action & Boundary**: State direct operational actions paired with explicit negative boundaries (*what to produce vs. what never to exceed*), using natural imperatives (`Stage`, `Verify`, `Confine`, `Never`, `Only`).
- **Anti-Priming**: Never state negative prohibitions in isolation without immediately defining the approved alternative.

## 4. Execution Topology & Pipeline Orchestration

- **Single Responsibility**: Each skill must perform exactly one job and deliver one canonical artifact.
- **Turn-Bounded Topology**:
  - *Multi-Phase Skills* (mutations, task publication): Structure phases strictly along natural turn boundaries (`Discovery → Staging & Authorization Gate → Atomic Mutation & Handoff`). Combine Staging and Gate in the same turn to eliminate artificial bureaucracy.
  - *Single-Turn Skills* (cartography, synthesis): Forbid multi-phase ceremony; structure strictly via Operating Invariants and Canonical Artifact Templates.
- **Universal Mention Convention**: Reference supporting sub-skills conditionally in templates and prose (`(via /<name> when available)`). Never treat sub-skills as hard blocking dependencies.

## 5. Invocation Gating & 3-Tier Economics

- **Invocation Governance**: Planning, scaffolding, destructive, and implementation skills must use `disable-model-invocation: true`. Only pure advisory/inquiry skills (`clarify`, `ssot`) allow autonomous model invocation.
- **Description Formula**: Frontmatter descriptions must stay under 25 words following: `[Imperative Action] + [Target Entity] + [Context / Purpose]`. Forbid negative clauses in frontmatter.
- **3-Tier Context Economics**: Tier 1 (Catalog: frontmatter < 50 tokens) ──> Tier 2 (Activation: `SKILL.md` < 150 lines) ──> Tier 3 (Execution on Demand: offload runbooks to `references/`).

## 6. SSOT & Markdown Integrity

- **Define Once in SSOT**: Every invariant, contract, or format must belong to exactly one authoritative location. Forbid duplicating invariants inside phase steps.
- **Pure GFM Standard**: Use standard GFM code fences (`text`, `markdown`, `bash`). Use 3-backticks baseline; restrict 4-backtick fences strictly to templates encapsulating nested code blocks.

## 7. Code Verification Integrity

- **Test-Driven Baseline**: Follow failing baseline test → minimal green code → verification cascade (typecheck → lint → test).
- **Code Circuit Breaker**: Confine Circuit Breakers strictly to post-mutation automated test and compiler regressions in code implementation skills (`forge`). If tests fail, halt immediately, report stderr, and prompt whether to debug or revert.

## 8. Canonical Artifacts & Lean Turn-Halt Gate

- **Canonical Completion Artifact**: Every skill must specify exactly one deterministic staging artifact (Changeset Tree, Decision Matrix, Task Staging Table). Forbid dumping raw, multi-page unified diffs or full file bodies into chat.
- **Lean Turn-Halt Gate**: Phase authorization gates must pair the staging artifact presentation directly with an immediate turn-halt imperative and a negative mutation boundary. Forbid micro-enumerating obvious user review choices.

## 9. Test-Driven Skill Authoring

- **Empirical Baseline Failures**: Author and modify skills strictly against observed agent failure traces without the rule.
- **Preserve Domain Intelligence**: Prune conversational ceremony ruthlessly, but strictly preserve domain-specific guardrails, query strategies, and safety budgets that prevent real-world failure modes.
