---
name: skeleton-spike
description: "Construct and verify a minimal end-to-end walking skeleton across distributed modules or bounded contexts to prove architectural integration before detailed implementation."
disable-model-invocation: true
---

# skeleton-spike — Multi-Module Walking Skeleton & Plumbing Proof

## Operating Invariants

- **Thin Tracer Only**: Transmit only minimal dummy payloads (ping-pong tracer); forbid implementing complex business logic inside the skeleton.
- **All-Seam Traversal**: The skeleton must traverse the complete end-to-end pipeline: `Input Port -> Adapter -> Serialization -> Transport / Bus -> Target Module -> Persistence Stub`.
- **Sandbox & Environment Boundary**: Confine all exploratory scaffolding code strictly to `.agents/scratch/` or dedicated git worktrees (`.agents/worktree/`); ground runners and services in `.agents/environment.md` (fallback to `.agents/dev-environment.md`). Never contaminate the main branch history with unapproved spikes.
- **Adaptive Invocation Output**: Render the full proof report for humans; emit an integration status micro-token for autonomous runners.

## Domain Rubric

### 1. Plumbing Verification Target
Verify that architectural plumbing functions under real runtime conditions:
- Dependency injection and service bootstrapper wiring.
- Serialization and schema compatibility between bounded contexts.
- Configuration loading and port bindings.
- Basic latency and inter-module communication protocols (HTTP, gRPC, event bus).

### 2. State
- `CONFIRMED`: End-to-end trace executes successfully with passing integration assertions.
- `INFERRED`: Trace succeeds partially but relies on mocked transport or unverified runtime assumptions.
- `REJECTED`: Incompatible protocols, serialization failure, or runtime blockers rule out the candidate architecture.
- `UNRESOLVED`: Environment limitations or missing permissions prevent end-to-end execution.

## Canonical Output Contract

```markdown
# Walking Skeleton Proof Report: <Architecture Slug> · <CONFIRMED | INFERRED | REJECTED | UNRESOLVED>

Conclusion: <Observable outcome proving whether multi-module plumbing holds or fails>
Exercised Trace: `<Module A> -> <Transport / Seam> -> <Module B> -> <Sink>`
Governing Contracts: <Pointers to frozen interface contracts validated>

## Execution Evidence
- Harness Directory: <Repo-relative path in scratch or worktree>
- Verification Command: `<command executed>`
- Telemetry & Output:
  ```text
  <Sanitized execution trace showing end-to-end traversal>
  ```

*(omit if all pass)* ## Boundary Findings
- Inter-Module Tension: <Description of failure at specific connection seam>

Conditions: <Environment constraints, runtime requirements, and verification limits>
```

*Autonomous Micro-Token Contract*:
```text
CONFIRMED | trace: <seam-path> | runtime: passed | fixtures: verified
```

### Routing

- Walking skeleton CONFIRMED, proceed to slice implementation tasks → (via /split-tasks).
- Architectural plumbing REJECTED or unworkable → (via /architect-decision).
- Contract mismatch detected during traversal → (via /contract-freeze).
- Unready host runtime or failing service dependencies prevent traversal → (via /env-doctor).

## Execution Protocol

1. Read governing frozen contracts and target module interfaces.
2. Scaffold a minimal end-to-end walking skeleton inside `.agents/scratch/` or a worktree.
3. Wire transport, serialization, and persistence stubs with dummy ping payloads.
4. Execute integration run command and verify complete traversal.
5. Report findings using Canonical Proof Report or autonomous micro-token.
