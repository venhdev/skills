# ZenForge Construction Rules

Authoritative architectural principles and engineering quality criteria for the ZenForge ecosystem (SOUL, skills, governance, and templates). Every instruction, skill definition, and documentation artifact must strictly adhere to these rules.

---

## 1. Universal & Decoupled
- **Stack-Agnostic Core**: Author all rules, skills, and prompts to be entirely independent of specific programming languages, frameworks, or project setups.
- **Cross-LLM Determinism**: Ensure instructions execute with high fidelity across all frontier LLM families (Claude 3.5/3.7, Gemini 2.0/3.0, GPT-4o/o3, and top open-weights).
- **External Host Repository**: Treat the target codebase as an external, mutable environment; never couple the agent's core identity to any single project.

## 2. Zero-Waste & Anti-No-Op
- **Observable Behavioral Delta**: Every single directive must measurably constrain or direct the model's output distribution. If removing a sentence produces zero regression in empirical testing, delete it immediately.
- **Instruction Debt Pruning**: Ruthlessly purge conversational filler, flowery adjectives, moralizing preambles, and generic advice modern LLMs obey by default (e.g., "write clean code", "be helpful", "think step-by-step").
- **High-Density Terse Register**: System instructions written in a compact, authoritative, technical register force the model to mirror the same concise, high-density precision.

## 3. Affirmative Framing & Paired Directives (Mandate–Forbid)
- **Positive Operational Framing**: Formulate procedures as direct constructive actions stating precisely what to produce and which tools to invoke.
- **Paired Directive Syntax**: For non-negotiable gates and boundary enforcement, always pair the affirmative requirement directly with the explicit prohibition:
  > *"Mandate [Required Action / Output Artifact]; Forbid [Prohibited Behavior / Premature Cascade]."*
- **Anti-Priming Rule**: Never state negative prohibitions in isolation without immediately defining the approved alternative.

## 4. Single-Skill Isolation & Pipeline Orchestration
- **Sandboxed Micro-Engines**: Each skill must do exactly one job and deliver one canonical artifact. Forbid monolithic "all-in-one" workflows.
- **Tracer-Bullet Vertical Slicing**: Mandate end-to-end vertical capabilities per task (schema → logic → UI → test); forbid horizontal layer sprawl across multiple subsystems in one turn.
- **Expand–Contract Sequence**: For wide refactors with large blast radiuses, mandate the 3-phase sequence: Expand interface → Migrate callers in batches → Contract obsolete interfaces.
- **Turn-Halt Execution Gates**: When a phase meets its completion criteria, halt the conversation turn immediately. Forbid cascading into downstream phases or mutating files without affirmative human authorization.
- **Standard Dependency Markers**: Declare prerequisite skills using the standard marker (`**SUB-SKILL:** <name>`); reference by identifier only without hardcoding syntax.

## 5. Invocation Gating & 3-Tier Progressive Disclosure
- **Invocation Governance**:
  - Assign `disable-model-invocation: true` to all planning, scaffolding, or destructive skills (`zenforge-init`, `changeset`, `to-tasks`, `forge`). They must run exclusively via human slash commands.
  - Skill frontmatter `description` must follow a concise, high-density affirmative formula:
    $$\text{[Imperative Action]} + \text{[Target Entity]} + \text{[Context / Purpose]}$$
    Keep descriptions compact (under 25 words / 35 tokens). Forbid negative boundary clauses (`Do NOT use for...`) in frontmatter to prevent catalog bloat and negative priming; enforce negative boundaries strictly via Operating Invariants inside the skill body.
- **3-Tier Context Economics**:
  - **Tier 1 (Catalog)**: Frontmatter (`name`, `description`) must remain under 50 tokens.
  - **Tier 2 (Activation)**: Top-level `SKILL.md` must stay lean (under 150 lines), focusing strictly on Operating Invariants and Phase Protocols.
  - **Tier 3 (Execution on Demand)**: Offload runbooks, schemas, and templates to `references/`, and deterministic logic to `scripts/`. Load on demand via explicit path pointers.

## 6. Clean Markdown Integrity & Template Isolation
- **Pure GFM Standard**: Mandate standard GitHub Flavored Markdown (GFM) fenced code blocks (` ```text `, ` ```markdown `, ` ```bash `) for all templates, schemas, and staging formats; forbid arbitrary HTML/XML wrapper tags.

## 7. Define Once in SSOT (Single Source of Truth)
- **Canonical Ownership**: Every invariant, contract, business rule, and tracker convention must belong to exactly one authoritative file.
- **Pointer-Based Authority**: Link to canonical definitions via relative paths or URI pointers (`file:///path#L10-L25`); forbid duplicating, paraphrasing, or inlining definitions.
- **Contradiction Circuit Breaker**: If proposed work conflicts with an established SSOT document or ADR, halt execution immediately and request human resolution before proceeding.

## 8. Verification Integrity & Error Circuit Breaker
- **Test-Driven Verification**: Enforce a Red-Green-Refactor sequence for implementation skills: failing test baseline → minimal green code → verification cascade (typecheck → lint → test).
- **Verification Circuit Breaker**: If any compilation, lint, or test check fails post-mutation:
  - Halt execution immediately.
  - Report exact `stderr` output with line pointers.
  - Prompt user whether to debug or revert.
  - Forbid silent lossy reversions, unguided retry loops, or altering test assertions to pass CI.

## 9. Canonical Artifact Contract & Delivery Boundary
- **Defined Completion Artifact**: Every skill must specify exactly one structured, deterministic artifact that marks its phase completion (e.g., Changeset Tree, Decision Matrix, Task Staging Table).
- **Staging-Before-Mutation**: Output representations must be optimized for fast human review (structured markdown tables under 15–20 lines, high-density bullet trees, or Conventional Commit handoffs).
- **Prohibition of Raw Dumps**: Forbid emitting raw, multi-page unified diffs or full file bodies into the conversation stream during staging phases. Reserve code blocks exclusively for newly authored contracts or minimal diffs.

## 10. Test-Driven Skill Authoring (Eval-Driven Refinement)
- **Empirical Baseline Failures**: Author and modify skills strictly against observed agent rationalizations or failure traces without the rule.
- **Minimal Loophole Directives**: Deploy the absolute minimal instruction needed to plug the specific failure loophole; reject speculative rules that address hypothetical problems.
