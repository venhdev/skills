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
- **Qualitative Density over Arbitrary Counts**: Forbid brittle numerical micro-quotas (e.g., "strictly under 35 lines", "exactly 3 steps"). Frontier LLMs lack lookahead line counters; hard quotas induce truncation panic and distorted grouping. Enforce brevity qualitatively through explicit scope exclusion (omit historical chronology, theoretical proofs, and tangential edge cases) and natural structural progression.
- **No-Op Transitional Glue**: Forbid conversational filler between section headers and artifact templates (e.g., "Render the explanation directly using this format:"). Section headers are self-describing; attach artifact templates directly beneath their governing headings.
- **Tool-Aware Zero-Waste**: Do not instruct models or subagents to manually avoid what underlying tooling (`fd`, `ripgrep` default gitignore handling) or subagent permission boundaries (read-only agents lacking write tools) already enforce by design.

## 3. Affirmative Framing & Paired Directives (Action & Boundary)

- **Positive Operational Framing**: Formulate procedures as direct constructive actions stating precisely what to produce and which tools to invoke.
- **Paired Boundary Enforcement**: Pair every non-negotiable action with its explicit negative boundary (*what to do vs. what never to exceed*), using varied, natural engineering imperatives (`Confine`, `Verify`, `Stage`, `Omit`, `Never`, `Only`).
- **Anti-Priming Rule**: Never state negative prohibitions in isolation without immediately defining the approved alternative.

## 4. Single-Skill Isolation & Pipeline Orchestration

- **Sandboxed Micro-Engines**: Each skill must do exactly one job and deliver one canonical artifact. Forbid monolithic "all-in-one" workflows.
- **Execution Topology (Single-Turn vs. Multi-Phase)**:
  - **Multi-Phase Skills** (state mutation, interactive deliberation, code refactoring): Mandate explicit gated phases (`Phase 1: Inspect → Phase 2: Stage → Phase 3: Authorize → Phase 4: Mutate & Verify`).
  - **Single-Turn Skills** (conceptual translation, problem framing, cartography): Forbid synthetic multi-phase workflow ceremony (`Ingestion → Synthesis → Delivery`). Structure single-turn skills strictly via Operating Invariants and Canonical Artifact Templates.
- **Subagent Delegation Architecture (Fan-out / Fan-in)**: When a single-turn skill delegates deep exploration or research to subagents:
  - *Fan-out*: Partition by architectural boundary or query vector, passing the target scope and inlining the Canonical Artifact Template directly into the subagent prompt as the mandatory output contract.
  - *Fan-in*: Reconcile and merge findings into a single unified artifact, emit to stream, and halt turn immediately without cascading into file mutations.
- **Tracer-Bullet Vertical Slicing**: Mandate end-to-end vertical capabilities per task (schema → logic → UI → test); forbid horizontal layer sprawl across multiple subsystems in one turn.
- **Expand–Contract Sequence**: For wide refactors with large blast radiuses, mandate the 3-phase sequence: Expand interface → Migrate callers in batches → Contract obsolete interfaces.
- **Turn-Halt Execution Gates**: When a phase meets its completion criteria, halt the conversation turn immediately. Forbid cascading into downstream phases or mutating files without affirmative human authorization.
- **Standard Dependency Markers**: Declare prerequisite skills using the standard marker (`**SUB-SKILL:** <name>`); reference by identifier only without hardcoding syntax.

## 5. Invocation Gating & 3-Tier Progressive Disclosure

- **Invocation Governance**:
  - Assign `disable-model-invocation: true` to all planning, scaffolding, destructive, refactoring, diagnostic, or worktree skills (`zenforge-init`, `changeset`, `to-tasks`, `forge`, `simplify`, `diagnose`, `worktree`, `recon`, `scout`, `distill`, `lens`). They must run exclusively via human slash commands.
  - Advisory or inquiry-driven skills (`clarify`, `ssot`) permit model invocation so the agent can autonomously request architectural clarification on ambiguous requirements or verify authoritative documentation contracts.
  - Skill frontmatter `description` must follow a concise, high-density affirmative formula:
    `[Imperative Action] + [Target Entity] + [Context / Purpose]`
    Keep descriptions compact (under 25 words / 35 tokens). Forbid negative boundary clauses (`Do NOT use for...`) in frontmatter to prevent catalog bloat and negative priming; enforce negative boundaries strictly via Operating Invariants inside the skill body.
- **3-Tier Context Economics**:
  - **Tier 1 (Catalog)**: Frontmatter (`name`, `description`) must remain under 50 tokens.
  - **Tier 2 (Activation)**: Top-level `SKILL.md` must stay lean (under 150 lines), focusing strictly on Operating Invariants and Phase Protocols (or Canonical Artifact Templates for single-turn skills).
  - **Tier 3 (Execution on Demand)**: Offload runbooks, schemas, and templates to `references/`, and deterministic logic to `scripts/`. Load on demand via explicit path pointers.

## 6. Clean Markdown Integrity & Template Isolation

- **Pure GFM Standard**: Mandate standard GitHub Flavored Markdown (GFM) fenced code blocks (`` ```text ``, `` ```markdown ``, `` ```bash ``) for all templates, schemas, and staging formats; forbid arbitrary HTML/XML wrapper tags.
- **Fencing Depth Discipline**: Use standard 3-backtick fences (`` ``` ``) as the canonical baseline. Restrict 4-backtick fences (`` ```` ``) strictly to templates that encapsulate nested code blocks (e.g., mermaid diagrams, bash scripts, or raw text blocks) to prevent syntax parsing collisions.

## 7. Define Once in SSOT (Single Source of Truth)

- **Canonical Ownership**: Every invariant, contract, business rule, and tracker convention must belong to exactly one authoritative file.
- **Pointer-Based Authority**: Link to canonical definitions via relative paths or URI pointers (`file:///path#L10-L25`); forbid duplicating, paraphrasing, or inlining definitions.
- **Internal Skill SSOT**: Never define output representations twice within the same skill (e.g., listing rubric points in prose and then duplicating them inside a markdown template). The Canonical Artifact Template is the sole authoritative definition of the output structure.
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
- **Staging-Before-Mutation**: Output representations must be optimized for fast human review (compact markdown tables, high-density bullet trees, or Conventional Commit handoffs).
- **Prohibition of Raw Dumps**: Forbid emitting raw, multi-page unified diffs or full file bodies into the conversation stream during staging phases. Reserve code blocks exclusively for newly authored contracts or minimal diffs.

## 10. Test-Driven Skill Authoring (Eval-Driven Refinement)

- **Empirical Baseline Failures**: Author and modify skills strictly against observed agent rationalizations or failure traces without the rule.
- **Minimal Loophole Directives**: Deploy the absolute minimal instruction needed to plug the specific failure loophole; reject speculative rules that address hypothetical problems.
- **Preservation of Domain Intelligence**: Prune conversational ceremony and no-op pseudo-steps ruthlessly, but strictly preserve domain-specific guardrails, authority hierarchies, query strategies, and safety budgets that prevent real-world failure modes.
