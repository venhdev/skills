# ZenForge Construction Rules

Authoritative architectural laws and quality standards for all ZenForge skills and templates.

---

## 1. Universal & Decoupled

- **Stack & Host Agnostic**: Author all instructions independent of specific programming languages, frameworks, or host repositories.
- **Cross-LLM Determinism**: Ensure high-fidelity execution across all frontier LLM families without model-specific quirks.
- **Self-Contained Encapsulation**: Package all references, templates, and scripts locally within the skill directory; forbid relative links or dependencies to paths outside the skill directory.
- **Zero-Secret Exposure & Environment Isolation**: Forbid inspecting (`read_file`, `cat`, grep) secret files (`.env*` except `.env.example`, `*.pem`, `*.key`, `id_rsa`); ground local execution in `.agents/environment.md` (fallback to `.agents/dev-environment.md` if present) and isolate machine-local profiles in `.gitignore`.
- **Path Portability Boundary**: Confine `file:///<abs_path>#L<N>` strictly to ephemeral chat streams for IDE jump-to-definition; mandate repository-relative POSIX paths for all persisted artifacts (tasks, documentation, commits, remote trackers).

## 2. Zero-Waste & Anti-No-Op

- **Observable Behavioral Delta**: Every directive must measurably constrain output distribution. Ruthlessly prune conversational filler, preambles, persona roleplay, and generic advice modern LLMs follow by default.
- **Qualitative Density**: Enforce brevity through explicit scope exclusions; forbid brittle numerical quotas (line counts, arbitrary step numbers).
- **Decision-Sufficient Reports**: Keep conclusions short to medium according to complexity; include the evidence, conditions, trade-offs, or gaps the caller needs to decide rather than enforcing one-sentence output.
- **Selective Tables**: Use tables only for useful comparisons or mappings, with at most four columns; use blocks or prose when extra columns would obscure the decision.
- **Conditional Section Omission**: Explicitly mark non-universal or negative-candidate sections in output contracts as omissible (`*(omit if ...)*`, e.g., `*(omit if no candidate patterns pruned)*`); forbid forcing synthetic filler, boilerplate placeholders, or hallucinated entries to satisfy an empty contract section.
- **Tool-Aware Zero-Waste**: Forbid defensive rules, speculative circuit breakers, or retry instructions for behaviors that underlying tooling, permissions, or native error returns already handle.
- **Zero Transitional Glue**: Attach templates and schemas directly beneath governing headings without conversational filler.
- **Zero-Emoji & Token Economy**: Forbid decorative emojis across all skills, templates, and staging artifacts to conserve context tokens and eliminate visual noise.

## 3. Affirmative Framing & Paired Directives

- **Paired Action & Boundary**: State direct operational actions paired with explicit negative boundaries (*what to produce vs. what never to exceed*), using natural imperatives (`Stage`, `Verify`, `Confine`, `Never`, `Only`).
- **Anti-Priming ("Instead of X, do Y")**: Never state negative prohibitions in isolation without immediately defining the approved alternative. Forbid naked "Do not..." directives.
- **Show, Don't Tell (Micro-Snippets)**: Prefer concrete micro-snippets (`Smell vs. Remedy` or `Bad vs. Good`) over verbose explanatory prose. Snippets provide unambiguous grounding across all frontier LLMs.

## 4. Archetype-Driven Topology & Modular Blocks

- **Single Responsibility**: Each skill must perform exactly one job and deliver one canonical artifact.
- **Layout Derivation Heuristic**: Derive skill structure directly from operational posture instead of imposing a rigid 5-part template:
  - *Writing code, tracker records, or isolated scratch experiments?* ──> **Archetype C**: Mandates action boundaries, verification integrity, Rubric, an execution/proof report, and Phased protocol. Add staging and an authorization gate when the permitted action requires them; report-only scratch skills do not stage production fixes.
  - *Inspecting, framing through inquiry, or planning (Read-only)?* ──> **Archetype B**: Mandates Rubric (dimensions, grounding boundary), Report contract, and scope-adaptive execution. Add question/confirmation gates only when the job requires them; omit production mutation phases.
  - *Single-shot transform or cognitive lens?* ──> **Archetype A**: Mandates Output contract. Invariants contextual (only for tone/depth rigor). Omit multi-phase protocol.
- **Sequential Heading Flow**: When present, sections strictly follow: `Frontmatter` ──> `# title — mission` ──> `## Operating Invariants` ──> `## Domain Rubric` ──> `## Canonical Output Contract` ──> `## Execution Protocol`.
- **Contextual Anatomy Matrix**:

| Section Heading | Archetype A (Utility / Lens) | Archetype B (Audit / Map) | Archetype C (Mutating Workflow) |
|---|---|---|---|
| `Frontmatter` | Mandatory | Mandatory | Mandatory |
| `## Operating Invariants` | Contextual (Framing limits) | Omit (Tool read-only) | Mandatory (Action and proof boundaries) |
| `## Domain Rubric` | Contextual (Taxonomy) | Mandatory (Dimensions, grounding) | Mandatory (Smells, rules) |
| `## Canonical Output Contract` | Mandatory (Dossier template) | Mandatory (Report or conditional staging) | Mandatory (Execution report) |
| `## Execution Protocol` | Omit / Linear | Mandatory (Inquiry, inspection, or planning) | Mandatory (Authorized execution + verification) |

- **Operational Archetypes**:
  - *Archetype A (Utilities & Cognitive Lenses)* (e.g., `lens-pro`, format converters): Single-shot input-to-output delivery without multi-phase ceremony.
  - *Archetype B (Inquiry, Cartography, Audit & Planning)* (e.g., `distill`, `recon`, `scout`, `assay`, `clarify`, `changeset`, `simplify-audit`): Read-only grounding and a report or staged plan; delegate only broad inspection slices. An embedded changeset is an internal artifact, not a mandatory invocation of another skill.
  - *Archetype C (Stateful & Mutating Workflows)* (e.g., `forge`, `to-tasks`): Authorized filesystem/tracker modifications or isolated scratch experimentation (e.g., `diagnose`, `spike`), with explicit boundaries and verification. Scratch permission never implies permission to patch production.
- **Turn-Bounded Execution**: Combine Discovery, Staging, and Authorization Gate whenever feasible. Halt for required answers or missing authorization; continue authorized scope without re-asking or imposing one phase per turn.
- **Universal Mention Convention**: Reference supporting sub-skills conditionally in templates and prose (`(via /<name>)`). Never treat sub-skills as hard blocking dependencies.

## 5. Invocation Gating & 3-Tier Economics

- **Invocation Governance**: Planning, scaffolding, destructive, and implementation skills must use `disable-model-invocation: true`. Only pure advisory/inquiry skills (`clarify`, `ssot`) allow autonomous model invocation.
- **Description Formula**: Frontmatter descriptions must stay under 25 words following: `[Imperative Action] + [Target Entity] + [Context / Purpose]`. Forbid negative clauses in frontmatter.
- **3-Tier Context Economics**: Tier 1 (Catalog: frontmatter < 50 tokens) ──> Tier 2 (Activation: `SKILL.md` < 150 lines, 40–120 lines sweet spot) ──> Tier 3 (Execution on Demand: offload runbooks to `references/`).

### Invocation Tiers & Output Adaptability

Classify all skills into one of three operational invocation tiers:

1. **User-Invoked (Interactive Skills)**: Direct human engagement (e.g., `/clarify`, `/architect-decision`, `/changeset`, `/contract-freeze`, `/seam-plan`). Mandates a rich Canonical Output Contract (Markdown tables, ASCII trees, structured options) for human review and authorization gates.
2. **Dual-Invoked (Pipeline & Execution Skills)**: Callable either directly by human operators or autonomously by runners and subagents (e.g., `/forge`, `/assay`, `/split-tasks`, `/tracking-tasks`, `/env-doctor`, `/skeleton-spike`). Supports **Adaptive Output**:
   - *Human Caller*: Renders the full Canonical Output Contract for audit and inspection.
   - *Model / Runner Caller*: Returns a lean **Micro-Summary / Exit Status** (`PASS | commit: <sha> | <stats>` or compact JSON < 50 tokens) to prevent context bloat and token churn.
3. **Model-Invoked (Behavioral Protocols)**: Cognitive guidelines, heuristics, and execution guardrails (e.g., `/tdd`, coding conventions, UI/UX heuristics). **Canonical Output Contract is Omitted by Design**; its artifact is the verified production code and test delta itself. Governed by Domain Rubric and Smell vs. Remedy snippets.

*Progressive Adoption Policy*: Apply these invocation tiers immediately to all newly authored or modified skills; legacy skills retain their existing contracts until scheduled for revision.

## 6. SSOT & Markdown Integrity

- **Define Once in SSOT**: Every invariant, contract, or format must belong to exactly one authoritative location. Forbid duplicating invariants inside phase steps.
- **Local Execution Criteria**: Keep each skill's input requirements, state criteria, and handoff payload self-contained. Apply the authoring semantics in this document without making an installed skill depend on repository-wide documentation; local criteria specialize the shared meaning rather than redefine it.
- **Pure GFM Standard**: Use standard GFM code fences (`text`, `markdown`, `bash`) and `mermaid` for connection/state diagrams. Use 3-backticks baseline; restrict 4-backtick fences strictly to templates encapsulating nested code blocks.

## 7. Code Verification Integrity

- **Test-Driven Baseline**: For a bugfix, establish the original failing assertion before minimal green code; for a feature, verify agreed acceptance criteria. Use applicable typecheck, lint, and test checks; prefer meaningful existing verification for behavior-preserving changes over shallow new tests.
- **Autonomous Verification Cascade**: Correct ordinary compiler, linter, and test failures within authorized scope until required checks pass. Stop mutations and report `UNRESOLVED` when success requires redesign, substantial scope expansion, or unavailable required verification; preserve evidence and partial edits rather than silently reverting or weakening assertions.

## 8. Canonical Artifacts & Lean Turn-Halt Gate

- **Canonical Completion Artifact**: Specify one deterministic artifact appropriate to the job (report, decision, Changeset Tree, or Task DAG), with conditional sections for clean, unresolved, or staged outcomes. Keep full file bodies and raw multi-page diffs out of chat. Model-invoked behavioral skills omit this section entirely; dual-invoked skills collapse into micro-tokens when dispatched autonomously.
- **ASCII Tree Staging Rigor**: Mandate ASCII tree representation inside ```text code fences for changeset staging; forbid markdown bullet lists (lacks structural hierarchy) and bare un-fenced ASCII pipes (triggers markdown table parsing collisions).
- **Lean Turn-Halt Gate**: Pair a staged artifact needing authorization with an immediate halt and a mutation boundary. Honor authorization already provided for that concrete scope; omit review-choice menus and redundant approval turns.

## 9. Test-Driven Skill Authoring

- **Empirical Baseline Failures**: Author and modify skills strictly against observed agent failure traces without the rule.
- **Preserve Domain Intelligence**: Prune conversational ceremony ruthlessly, but strictly preserve domain-specific guardrails, query strategies, and safety budgets that prevent real-world failure modes.
- **Behavioral Validation**: Check independent invocation and handoff scenarios against observed failure modes. Structural validators supplement scenario review; they do not establish that a skill follows its contract.

## 10. Subagent Delegation & Orchestration Architecture

- **Two-Tier Discovery (Macro Grounding vs. Micro Execution)**:
  - *Main Agent (Orchestrator)*: Pre-scouts high-level architectural anchors (shared primitives, governing specs, boundary coordinates) before dispatching to ground subagent scope. Forbid blind dispatch without grounded anchor context.
  - *Subagents (Workers)*: Execute deep, parallel, file-by-file traversal or multi-query search strictly within their partitioned slice.
- **Context Seeding Contract**: Subagents run in isolated conversation contexts and do not inherit active memory. The Main Agent must explicitly seed each subagent with:
  1. *Target Slice*: Exact directory or file boundary.
  2. *Discovered Anchors*: Pre-scouted primitives, interfaces, or specs to audit against.
  3. *Governing Rubric & References*: Relevant evaluation rules and the exact path to any Tier 3 reference files (`references/*.md`) for the subagent to view on demand.
  4. *Output Contract*: Mandatory sub-report schema for deterministic fan-in.
- **Scale-Adaptive Execution Threshold**:
  - *In-Turn Execution*: If the target scope is localized (e.g., 1–5 files, single cohesive module), the Main Agent executes the audit directly in-turn to eliminate subagent spawning latency and token overhead.
  - *Subagent Fan-Out*: Launch 2–3 concurrent subagents strictly when the codebase spans multiple architectural layers, cross-package subsystems, or exceeds single-turn inspection budgets.

## 11. Input, State & Routing

- **Independent Invocation**: Define minimum input and what the skill can discover itself. Accept equivalent evidence/artifacts from the caller; never require a ceremonial upstream skill run.
- **Bounded Responsibility**: State the completion target, permitted actions, and stop conditions. Carry the caller's direction and constraints; route a necessary redesign instead of solving a different problem.

### Conclusion States

Use one `State` for the identified conclusion target; pair it with a decision-sufficient conclusion and evidence. Keep conditions and scope as facts, not a second `Fit` or `Result` enum.

| State | Meaning |
| :--- | :--- |
| `CONFIRMED` | Evidence supports the skill's target conclusion within the stated scope. |
| `INFERRED` | Evidence supports a reasoned conclusion whose material claims remain unverified. |
| `REJECTED` | Evidence rules out the candidate or proposal being evaluated; name the rejected scope. |
| `UNRESOLVED` | Input, conflict, access, or evidence prevents a defensible conclusion. |

- **Proof Boundaries**: RED establishes a symptom, not a cause; scratch GREEN establishes tested behavior, not an unverified production fix. Intent confirmation establishes the caller's meaning, not feasibility or mutation permission. Architecture confirmation establishes a supported choice, not implementation GREEN.
- **Specialized States**: Use a separate family only when the conclusion states cannot express the job's result. Planning uses `STAGED / UNRESOLVED`; implementation uses `STAGED / IN_PROGRESS / DONE / UNRESOLVED`; simplify audit uses `PASS / STAGED / UNRESOLVED`. Keep established review verdicts and task/document lifecycles until their contracts are explicitly revised.
- **Completion Scope**: Tie `DONE` to the authorized endpoint. Published tasks can be `ready`; verified implementation can remain uncommitted; release readiness can remain unpublished. Dispatching a worker does not prove its work complete.

### Routing

- **Contextual Guidance**: Write routing under `### Routing` as `<Context / gap> → (via /<skill>)`. Let the agent choose relevant handoffs from the actual outcome; keep routing guidance outside output templates and omit forced routing placeholders or numerical limits. A connection map is not a required pipeline.
- **Payload Continuity**: Carry problem/outcome, candidate or settled direction, constraints, acceptance criteria, proof/references and their conditions, current checkout scope, and applicable authorization. Preserve identifiers for distinct intents and task dependencies when relevant.
- **Gate Fidelity**: Keep confirmation, evidence, decision, publication approval, and implementation authorization distinct. A positive state or review verdict never grants permission for an external action.
- **Evidence Freshness**: Reconcile handoff evidence with current code and environment. Revalidate only affected claims when assumptions change; keep unaffected findings rather than discarding the entire context.
