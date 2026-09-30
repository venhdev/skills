---
name: spike
description: "Prove technical feasibility, algorithmic correctness, and implementation suitability through isolated scratch harnesses before codebase modification."
disable-model-invocation: true
---

# spike — Empirical Feasibility & Algorithm Proof

## Operating Invariants

- **Scratch Isolation**: Author all prototype logic, experimental harnesses, and proof scripts strictly inside `.agents/scratch/`; forbid creating or modifying production codebase files or permanent tests.
- **Empirical Proof Gate**: Demonstrate suitability by executing a working scratch script; forbid declaring an approach viable based solely on speculative reasoning or static inspection.
- **Pristine Working Tree**: Confine all experimental scripts, mock data, and test logs strictly inside `.agents/scratch/`; keep repository source directories untouched (`git status` clean).

## Domain Rubric

### 1. Candidate Approach Archetypes

Evaluate technical approaches across 4 common archetypes:

- **Algorithm & Logic**: Data structure selection, algorithmic complexity, boundary invariants, and edge-case handling.
- **Technology & Integration**: Third-party library suitability, external API contracts, protocol interop, and runtime compatibility.
- **Design Pattern & Concurrency**: Async lifecycles, event flows, state machine transitions, and race condition resilience.
- **Migration & Parity**: Core interface refactoring, adapter layers, and behavior parity with legacy implementations.

### 2. Empirical Grounding

- Instead of debating hypothetical edge cases or theoretical trade-offs, author a minimal scratch script to let runtime execution expose boundary constraints, quirks, or performance ceilings.
- Allow the agent full discretion over evaluation metrics (correctness, complexity, memory, or ergonomics) based on the specific problem context; forbid rigid numerical quotas when qualitative proofs suffice.

## Canonical Output Contract

```markdown
# Feasibility Dossier: <Topic / Candidate Approach>

## 1. Candidate Approach & Verdict
- **Candidate Approach**: <Description of evaluated implementation path or comparison between Options A vs B>
- **Verdict**: <VIABLE | CONDITIONAL | FLAWED>
- **Summary**: <High-level judgment on whether the approach is suitable and why>
- **Scratch Proof**: `<command executed against .agents/scratch/<script>>`

## 2. Findings & Observed Behavior
- **Proven Capabilities**: <Key behaviors, correctness checks, or edge cases verified by the scratch harness>
- **Discovered Constraints**: <Quirks, bottlenecks, trade-offs, or runtime limits revealed during execution>

## 3. Recommended Implementation Path
- **Integration Seam**: <Target module or subsystem where this approach should be implemented>
- **Next Step**: Route to `/changeset` to plan filesystem impact, or `/clarify` if trade-offs require human decision.
```

## Execution Protocol

### Phase 1: Candidate Framing
1. Identify the candidate approach, proposed algorithm, or technical unknown requiring proof.
2. Isolate the primary risk, assumption, or boundary condition that must be empirically verified.

### Phase 2: Isolated Scratch Experimentation
1. Author a minimal standalone proof script inside `.agents/scratch/spike_<slug>.<ext>`.
2. Execute the script to empirically observe runtime behavior, edge handling, and suitability.

### Phase 3: Feasibility Synthesis & Turn-Halt Gate
1. Synthesize observations and discovered constraints into the Canonical Output Contract.
2. Present the `Feasibility Dossier` and halt turn immediately for human review.
