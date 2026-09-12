# ZenForge Construction Rules

Authoritative architectural laws and quality standards for all ZenForge skills and templates.

---

## 1. Universal & Decoupled

- **Stack & Host Agnostic**: Author all instructions independent of specific programming languages, frameworks, or host repositories.
- **Cross-LLM Determinism**: Ensure high-fidelity execution across all frontier LLM families without model-specific quirks.
- **Self-Contained Encapsulation**: Package all references, templates, and scripts locally within the skill directory; forbid relative links or dependencies to paths outside the skill directory.
- **Path Portability Boundary**: Confine `file:///<abs_path>#L<N>` strictly to ephemeral chat streams for IDE jump-to-definition; mandate repository-relative POSIX paths for all persisted artifacts (tasks, documentation, commits, remote trackers).

## 2. Zero-Waste & Anti-No-Op

- **Observable Behavioral Delta**: Every directive must measurably constrain output distribution. Ruthlessly prune conversational filler, preambles, persona roleplay, and generic advice modern LLMs follow by default.
- **Qualitative Density**: Enforce brevity through explicit scope exclusions; forbid brittle numerical quotas (line counts, arbitrary step numbers).
- **Tool-Aware Zero-Waste**: Forbid defensive rules, speculative circuit breakers, or retry instructions for behaviors that underlying tooling, permissions, or native error returns already handle.
- **Zero Transitional Glue**: Attach templates and schemas directly beneath governing headings without conversational filler.

## 3. Affirmative Framing & Paired Directives

- **Paired Action & Boundary**: State direct operational actions paired with explicit negative boundaries (*what to produce vs. what never to exceed*), using natural imperatives (`Stage`, `Verify`, `Confine`, `Never`, `Only`).
- **Anti-Priming ("Instead of X, do Y")**: Never state negative prohibitions in isolation without immediately defining the approved alternative. Forbid naked "Do not..." directives.
- **Show, Don't Tell (Micro-Snippets)**: Prefer concrete micro-snippets (`Smell vs. Remedy` or `Bad vs. Good`) over verbose explanatory prose. Snippets provide unambiguous grounding across all frontier LLMs.

## 4. Archetype-Driven Topology & Modular Blocks

- **Single Responsibility**: Each skill must perform exactly one job and deliver one canonical artifact.
- **Layout Derivation Heuristic**: Derive skill structure directly from operational posture instead of imposing a rigid 5-part template:
  - *Mutating workspace or code?* ──> **Archetype C**: Mandates Invariants (Pre-mutation gate, breaker), Rubric, Staging contract, and Phased protocol.
  - *Inspecting codebase or research (Read-only)?* ──> **Archetype B**: Mandates Rubric (dimensions, search budget), Report contract, and Subagent delegation protocol. Omit invariants and mutation gates.
  - *Single-shot transform or cognitive lens?* ──> **Archetype A**: Mandates Output contract. Invariants contextual (only for tone/depth rigor). Omit multi-phase protocol.
- **Sequential Heading Flow**: When present, sections strictly follow: `Frontmatter` ──> `# title — mission` ──> `## Operating Invariants` ──> `## Domain Rubric` ──> `## Canonical Output Contract` ──> `## Execution Protocol`.
- **Contextual Anatomy Matrix**:

| Section Heading | Archetype A (Utility / Lens) | Archetype B (Audit / Map) | Archetype C (Mutating Workflow) |
|---|---|---|---|
| `Frontmatter` | Mandatory | Mandatory | Mandatory |
| `## Operating Invariants` | Contextual (Framing limits) | Omit (Tool read-only) | Mandatory (Gate, breaker) |
| `## Domain Rubric` | Contextual (Taxonomy) | Mandatory (Dimensions) | Mandatory (Smells, rules) |
| `## Canonical Output Contract` | Mandatory (Dossier template) | Mandatory (Report format) | Mandatory (Staging tree) |
| `## Execution Protocol` | Omit / Linear (1–3 steps) | Mandatory (Fan-out/in) | Mandatory (Phased + Turn-Halt) |

- **Operational Archetypes**:
  - *Archetype A (Utilities & Cognitive Lenses)* (e.g., `distill`, `lens-pro`, format converters): Single-shot input-to-output delivery without multi-phase ceremony.
  - *Archetype B (Cartography, Scout & Review)* (e.g., `recon`, `scout`, `assay`, `clarify`): Read-only codebase inspection, external research, or mutation critique via subagent fan-out/in.
  - *Archetype C (Stateful & Mutating Workflows)* (e.g., `changeset`, `forge`, `simplify`): Filesystem modifications, state transitions, and test runs governed by explicit turn-halt gates and circuit breakers.
- **Turn-Bounded Execution**: Combine Discovery, Staging, and Authorization Gate into a single turn whenever feasible to eliminate artificial conversational bureaucracy. Halt turn immediately at the authorization gate.
- **Universal Mention Convention**: Reference supporting sub-skills conditionally in templates and prose (`(via /<name> when available)`). Never treat sub-skills as hard blocking dependencies.

## 5. Invocation Gating & 3-Tier Economics

- **Invocation Governance**: Planning, scaffolding, destructive, and implementation skills must use `disable-model-invocation: true`. Only pure advisory/inquiry skills (`clarify`, `ssot`) allow autonomous model invocation.
- **Description Formula**: Frontmatter descriptions must stay under 25 words following: `[Imperative Action] + [Target Entity] + [Context / Purpose]`. Forbid negative clauses in frontmatter.
- **3-Tier Context Economics**: Tier 1 (Catalog: frontmatter < 50 tokens) ──> Tier 2 (Activation: `SKILL.md` < 150 lines, 40–120 lines sweet spot) ──> Tier 3 (Execution on Demand: offload runbooks to `references/`).

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
