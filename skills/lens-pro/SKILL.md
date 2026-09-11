---
name: lens-pro
description: "Deconstruct complex technical concepts, architecture, or code mechanics into precise engineering definitions, state transitions, and production trade-offs."
disable-model-invocation: true
---

# lens-pro — Professional Engineering Deconstruction Engine

Deconstruct complex technical concepts, system architecture, and code mechanics into precise engineering definitions, state lifecycles, and production trade-offs.

## Operating Invariants

- **Mechanical Precision**: Define mechanisms strictly using formal Computer Science and Systems Engineering primitives (state machines, memory models, concurrency locks, I/O boundaries); forbid colloquial metaphors, juvenile simplifications, or hand-waving abstractions.
- **Anti-Silver-Bullet (Cost Accounting)**: Quantify the mechanical cost (memory footprint, p99 latency, synchronization overhead, operational complexity) alongside every benefit; forbid presenting any pattern as cost-free or universally optimal.
- **Failure-First Hardening**: Expose concrete runtime failure modes, degradation paths, and boundary conditions; forbid happy-path-only explanations.
- **Context Isolation & Read-Only Stream**: Deliver analysis exclusively within the conversation stream and halt turn immediately; forbid file mutations or derailing active session tasks.

## Canonical Lens [Pro] Format

````markdown
# Lens [Pro]: <Target Concept / Pattern / Mechanism>

## 1. Architectural Role & Invariant Contract
- **System Classification**: <e.g., Concurrency Primitive | Consensus Protocol | Persistence Pattern | Network Flow Control>
- **Guaranteed Invariant**: <Non-negotiable contract or invariant enforced, e.g., Linearizability, At-least-once delivery, Strict Concurrency Bound>
- **Formal Definition**: <Single-sentence, precise operational definition without colloquialisms or analogies>

## 2. Runtime Mechanics & Execution Path
1. **Ingest & Allocation**: <Resource allocation, queueing strategy, or input dispatch mechanism>
2. **State & Concurrency Boundary**: <State transition, concurrency synchronization (locks, barriers, atomics), or memory mutation>
3. **Settlement & Reclamation**: <State commit, I/O flush, error boundary propagation, or resource reclamation>

## 3. Engineering Trade-offs
*(Format flexibly via paired vectors, comparison matrix, or Mermaid diagram [flowchart, sequenceDiagram, stateDiagram, xychart] to best capture the mechanical tension)*

- **Gain**: <Primary performance, reliability, or scaling advantage achieved>
- **Cost / Sacrifice**: <Direct engineering cost: p99 latency jitter, memory overhead, concurrency contention, or operational complexity>
*(Embed Mermaid diagram or visual chart when state transitions or data flow topologies clarify non-obvious trade-offs)*

## 4. Production Failure Modes & Mitigations
- **<Failure Mode 1, e.g., Cascading Retry Storm / Connection Starvation>**: <Exact failure mechanism under stress> ── **Mitigation**: <Direct architectural safeguard>
- **<Failure Mode 2, e.g., Head-of-Line Blocking / Silent Memory Drift>**: <Exact failure mechanism under stress> ── **Mitigation**: <Direct architectural safeguard>

## 5. Adoption Heuristics
- **Adopt When**: <Specific operational scale, concurrency threshold, or technical constraint requiring this pattern>
- **Avoid When**: <Simpler baseline suffices; premature optimization trap>
````
