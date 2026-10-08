---
name: architect
description: "Narrow open-ended problems into locked design direction, then converge verified evidence into one approach or an explicit no-build verdict."
disable-model-invocation: true
---

# architect — Design Direction Framing & Approach Convergence Engine

## Domain Rubric

### 1. Framing Axes

Establish grounded direction before implementation. An axis with no bearing on the work resolves to `N/A` with a one-line reason rather than an invented constraint. Scoping out two or more axes signals this is not an architecture decision — route to (via /distill) or (via /clarify). Forbid implementation selection until every axis is answered or scoped out:

- **Problem Boundary**: The operational outcome and the observable signal that declares it met, isolated from the requested mechanism.
- **Hard Constraints**: Non-negotiables that eliminate whole approach families — time, money, compliance, data safety, platform, or skill absent from the team.
- **Frozen Surface**: Surfaces that must survive unchanged, whether by contract, by data already written, or by someone else's reliance.
- **Reversibility Class**: One-way door (acting forecloses the undo) versus two-way (revertible at no external cost).

### 2. Framing Discipline

- **Inference Before Inquiry**: Phrase axis questions around business consequences rather than technical metrics (throughput, latency, delivery semantics). Infer technical constraints from consequence and provide as default for confirmation:

  ```text
  Bad:  "Expected throughput? What p99 latency budget?"
  Good: "When an order fails, what actually happens — lost revenue, or just slower processing?"
        (Derive durability requirement from answer, then offer as default.)
  ```

- **Zero-Ambiguity Fast Path**: When every axis is already fixed by the inquiry, lock the frame directly.
- **Framing Payload Format**:

  ```text
  ### Axis<N>: <Framing Axis>
  Bottleneck: <Single unresolved tension that shapes the design space>

  • [A] <Option> ── Trade-off: <advantage> vs. <disadvantage>
  • [B] <Option> ── Trade-off: <advantage> vs. <disadvantage>

  Default: [Option] because <single-sentence rationale>
  ```

### 3. Convergence Criteria

Score every candidate against all five; emit one approach or `DO-NOT-BUILD`:

- **Problem Closure**: Resolves the framed outcome rather than relocating it downstream.
- **Constraint Fitness**: Zero Hard Constraint violations; any violation disqualifies the candidate regardless of merit.
- **Precedent Backing**: Attested in production systems at disclosed scale, with known failure modes named.
- **Reversibility**: At equal merit prefer the two-way door; at equal reversibility prefer the narrower Frozen Surface.
- **Evidence Strength**: Every load-bearing claim carries a code pointer, verified citation, or measured result. Resolve discoverable codebase facts directly against source files; forbid deferring local repository facts as Open Gaps. Confine Open Gaps to external unknowns or unmeasured runtime risks.

### 4. Anti-No-Op Guardrails

- **Single Convergence, or None**: Emit one approach, or `DO-NOT-BUILD` when no candidate clears Problem Closure and Constraint Fitness. Replace hedging ("it depends", "Option A or B") with one ranked pick; break ties by two-way door, then narrower Frozen Surface, then duller precedent.
- **Adversarial Self-Review & Falsification**: State the strongest sourced case that the selection is wrong, paired with the observable condition that would flip the verdict to an alternative. A verdict without a credible counter-case is unreviewed, not validated.

## Canonical Output Contract

```markdown
# Approach Convergence: <Problem Slug>

## 1. Locked Direction
- **Outcome**: <Operational result in one sentence>
- **Hard Constraints**: <Non-negotiables>
- **Frozen Surface**: <Surfaces that must not change>
- **Reversibility Class**: <One-way | Two-way>

## 2. Evidence Base
| Source | What It Attests | Scale / Context | Known Failure Mode |
| :--- | :--- | :--- | :--- |
| <Code pointer, publication, or measured harness> | <Specific claim supported> | <Load or environment> | <Breakage point, or 'None disclosed'> |

## 3. Verdict
- **Outcome**: <Single approach as a concrete path, or `DO-NOT-BUILD` with justification>
- **Why It Wins**: <Criterion-by-criterion justification against the strongest alternative>
- **Accepted Cost**: <Sacrifice knowingly taken on>
- **Counter-Case**: <Strongest sourced argument the selection is wrong> ── flips to <alternative> when <observable condition>
- **Open Gaps**: <External unknowns or unmeasured runtime risks, and what settles them | 'None'>

## 4. Rejected Paths
- **<Candidate>**: Eliminated by <Criterion> ── <specific disqualifying evidence>

## 5. Next Step
- `(via /spike)` ── settle UNPROVEN viability empirically before committing.
- `(via /changeset)` ── selection accepted; map filesystem impact. Explicitly bind all findings and implementation-phase gaps into target acceptance criteria.
- No handoff ── on `DO-NOT-BUILD`, file the dossier with its justification and end.
```

## Execution Protocol

### Phase 1: Grounded Framing & Evidence Loop

1. Pre-scout architectural anchors: inspect codebase source files, schemas, and ADRs (via /ssot) alongside inquiry to ground technical defaults in system reality.
2. Convert engineering details into business consequences; evaluate the four Framing Axes using pre-scouted facts.
3. Present unanswered axes in Framing Payload Format, pairing each option with an evidence-grounded default. Batch independent axes in one turn and isolate causal branches (via /clarify).
4. Halt turn immediately for human response.
5. Adaptive Deliberation Loop:
   - *Requirement Shift*: If user response alters foundational premises, invalidate settled axes and re-scout from scratch.
   - *Branch Refinement*: Ingest answers, inspect newly exposed code paths, and iterate until all 4 axes are locked.
   - *Scope-Adaptive Fan-Out*: Gather evidence in-turn per §3 Evidence Strength; launch 2–3 concurrent `research` subagents (seeded with target slice and §2 schema) strictly when scope exceeds single-turn budget.

### Phase 2: Convergence Synthesis & Turn-Halt Gate

1. Score candidates on all five Convergence Criteria; enforce the disqualifying power of Constraint Fitness.
2. Apply the tie-break chain and the Anti-No-Op Guardrails to produce one approach, or `DO-NOT-BUILD` when none survives. Carry all implementation findings directly into §5 Next Step acceptance criteria for `(via /changeset)`.
3. Run the Adversarial Self-Review; revise the selection when the counter-case wins on a Convergence Criterion.
4. Present the completed `Approach Convergence` dossier and halt turn immediately for human decision. Never modify workspace files, author code, or begin implementation within this turn.
