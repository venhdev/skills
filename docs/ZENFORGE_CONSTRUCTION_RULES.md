# ZenForge Construction Rules

Authoritative architectural laws and quality standards for all ZenForge skills and templates.

---

## 1. Universal & Decoupled

- **Stack & Host Agnostic**: Author all instructions independent of specific programming languages, frameworks, or host repositories.
- **Cross-LLM Determinism**: Ensure high-fidelity execution across all frontier LLM families without model-specific quirks.
- **Self-Contained Encapsulation**: Package all references, templates, and scripts locally within the skill directory; forbid relative links or dependencies to paths outside the skill directory.

## 2. Zero-Waste & Anti-No-Op

- **Observable Behavioral Delta**: Every directive must measurably constrain output distribution. Ruthlessly prune conversational filler, preambles, and generic advice modern LLMs follow by default.
- **Qualitative Density**: Enforce brevity through explicit scope exclusions; forbid brittle numerical quotas (line counts, arbitrary step numbers).
- **Tool-Aware Zero-Waste**: Forbid defensive rules, speculative circuit breakers, or retry instructions for behaviors that underlying tooling, permissions, or native error returns already handle.
- **Zero Transitional Glue**: Attach templates and schemas directly beneath governing headings without conversational filler.

## 3. Affirmative Framing & Paired Directives

- **Paired Action & Boundary**: State direct operational actions paired with explicit negative boundaries (*what to produce vs. what never to exceed*), using natural imperatives (`Stage`, `Verify`, `Confine`, `Never`, `Only`).
- **Anti-Priming**: Never state negative prohibitions in isolation without immediately defining the approved alternative.

## 4. Archetype-Driven Topology & Modular Blocks

- **Single Responsibility**: Each skill must perform exactly one job and deliver one canonical artifact.
- **Contextual Structural Blocks**: Structural components (`Operating Invariants`, `Domain Engine / Rubric`, `Execution Protocol`, `Pre-Mutation Gate`) are modular building blocks applied strictly by operational archetype; forbid imposing universal 4-layer boilerplate or empty phantom sections onto every skill.
- **Operational Archetypes**:
  - *Archetype A (Deterministic Utilities & Direct Transforms)* (e.g., `distill`, format converters, CLI runners): Single-shot input-to-output delivery. Structure: Frontmatter + Input/Output Contract + Transformation Steps/Heuristics. Forbid pre-mutation gates, invariants sections, or multi-phase ceremony.
  - *Archetype B (Exploratory Cartography & Deliberation)* (e.g., `recon`, `scout`, `clarify`, `lens`): Read-only codebase inspection, external research, or conceptual deliberation. Structure: Frontmatter + Scope/Search Budget + Domain Rubric / Deliberation Dimensions + Canonical Report Template + Delegation/Synthesis Protocol. Forbid filesystem mutation gates or test verification cascades.
  - *Archetype C (Stateful & Mutating Workflows)* (e.g., `changeset`, `forge`, `simplify`): Filesystem modifications, state transitions, and test runs. Structure: Frontmatter + Operating Invariants (Pre-mutation gate, scope discipline, behavioral invariance) + Blast Radius / Refactoring Rubric + Turn-Bounded Execution Protocol (`Discovery` → `Staging & Authorization Gate` in single turn → `Mutation & Test Cascade`).
- **Turn-Bounded Execution**: Combine Discovery, Staging, and Authorization Gate into a single turn whenever feasible to eliminate artificial conversational bureaucracy. Halt turn immediately at the authorization gate.
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
