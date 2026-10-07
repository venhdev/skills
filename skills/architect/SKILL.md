---
name: architect
description: "Narrow open-ended problems into locked design direction, then converge verified evidence into one approach or an explicit no-build verdict."
disable-model-invocation: true
---

# architect — Design Direction Framing & Approach Convergence Engine

## Domain Rubric

### 1. Framing Axes

Establish direction before searching. An axis with no bearing on the work resolves to `N/A` with a one-line reason rather than an invented constraint. Scoping out two or more axes signals this is not an architecture decision — route to (via /distill) or (via /clarify). Forbid implementation selection until every axis is answered or scoped out:

- **Problem Boundary**: The operational outcome and the observable signal that declares it met, isolated from the requested mechanism.
- **Hard Constraints**: Non-negotiables that eliminate whole approach families — time, money, compliance, data safety, platform, or skill absent from the team.
- **Frozen Surface**: Surfaces that must survive unchanged, whether by contract, by data already written, or by someone else's reliance.
- **Reversibility Class**: One-way door (acting forecloses the undo) versus two-way (revertible at no external cost).

### 2. Framing Discipline

- **Inference Before Inquiry**: Derive each technical constraint from the stated business consequence first, then offer the derivation as a default for confirmation.

  ```text
  Bad:  "Expected throughput? What p99 latency budget?"
  Good: "When an order fails, what actually happens — lost revenue, or just slower processing?"
        (Derive the durability requirement from the answer, then offer it as the default.)
  ```

- **User-Answerable Framing**: Phrase every axis question in terms the requester can answer from experience, attaching the inferred assumption inside the question so they correct the premise instead of supplying a metric. Replace any request for throughput, latency percentiles, replication factors, or delivery semantics with a question about the consequence behind it.
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
- **Adversarial Self-Review**: State the strongest sourced case that the selection is wrong, plus the criterion that would flip it. A verdict without a credible counter-case is unreviewed, not validated.
- **Falsification Triggers**: Name the observable condition that would overturn the verdict, making it testable rather than rhetorical.

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
- **Open Gaps**: <ASSUMPTION and UNPROVEN claims, and what would settle them>

## 4. Rejected Paths
- **<Candidate>**: Eliminated by <Criterion> ── <specific disqualifying evidence>

## 5. Next Step
- `/spike` ── settle UNPROVEN viability empirically before committing.
- `/changeset` ── selection accepted; map filesystem impact. Explicitly bind all findings and implementation-phase gaps into target acceptance criteria.
- No handoff ── on `DO-NOT-BUILD`, file the dossier with its justification and end.
```

## Execution Protocol

### Phase 1: Framing Interview & Turn-Halt Gate

1. Convert every engineering detail the requester supplied into its underlying business consequence; separate outcome from mechanism.
2. Evaluate the four Framing Axes and present only the unanswered ones in Framing Payload Format, each carrying a default. Batch independent axes in one turn and isolate causal branches (via /clarify) where a later answer depends on an earlier one.
3. Halt turn immediately. Absorb batch approvals, prune branches settled by prior answers, and repeat until every axis is fixed or the requester directs convergence. Lock the frame from their responses without a separate confirmation turn.

### Phase 2: Evidence Fan-Out

1. Derive search targets from unresolved unknowns in the locked frame rather than from criteria. Inspect repository files directly to exhaust static codebase facts before declaring gaps.
2. Scope-adaptive dispatch:
   - *In-Turn Harvest* (single domain, narrow constraints): gather evidence directly to eliminate subagent latency.
   - *Subagent Fan-Out* (multi-layer, cross-domain, or exceeding single-turn budget): launch 2–3 concurrent `research` subagents, one per cluster.
3. Seed each subagent per the Context Seeding Contract, binding its Output Contract to one row of the §2 Evidence Base table.
4. Delegate source authority and search budgets conditionally (via /scout); harvest in-turn only what that budget already covers.

### Phase 3: Convergence Synthesis & Turn-Halt Gate

1. Score candidates on all five Convergence Criteria; enforce the disqualifying power of Constraint Fitness.
2. Apply the tie-break chain and the Anti-No-Op Guardrails to produce one approach, or `DO-NOT-BUILD` when none survives. Carry all implementation findings directly into §5 Next Step acceptance criteria for `(via /changeset)`.
3. Run the Adversarial Self-Review; revise the selection when the counter-case wins on a Convergence Criterion.
4. Present the completed `Approach Convergence` dossier and halt turn immediately for human decision. Never modify workspace files, author code, or begin implementation within this turn.
