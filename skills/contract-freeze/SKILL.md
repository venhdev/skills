---
name: contract-freeze
description: "Partition system boundaries into isolated bounded contexts and freeze explicit cross-module interface contracts before changeset planning or implementation."
disable-model-invocation: true
---

# contract-freeze — Boundary Partitioning & Interface Contract Locking

## Domain Rubric

### 1. Bounded Context Isolation
Isolate domain boundaries so each context exclusively owns its datastore and internal state. Forbid direct cross-database queries or foreign entity mutations; require all cross-boundary interactions to traverse explicit Public Interfaces, Domain Events, or Anti-Corruption Layers (ACL).

### 2. Frozen Interface Specifications
Define cross-module contracts using strict typed declarations (TypeScript types, Protocol Buffers, JSONSchema, OpenAPI, or Go interfaces). Every contract must declare:
- *Direction & Protocol*: In-process interface, sync RPC, or async event.
- *Strict Payloads*: Input command structure and output event/return type.
- *Invariants & Semantics*: Idempotency requirements, failure modes, and deterministic error codes.

### 3. Consumer-Driven Mockability
Generate or specify mock provider fixtures for each frozen contract. Downstream contexts must be capable of building and executing against mock fixtures without waiting for upstream implementations.

### 4. State
- `STAGED`: Bounded contexts mapped, public interfaces frozen, and mock fixtures reviewable.
- `UNRESOLVED`: Domain boundary ambiguity or conflicting data ownership prevents clean separation.

## Canonical Output Contract

```markdown
# Contract Freeze: <System / Domain Slug> · <STAGED | UNRESOLVED>

Basis: <Architectural direction or domain specifications reference>

## 1. Bounded Context Topology
- Context: `<ContextName>` | Data Ownership: `<Tables / Entities>` | Public Port: `<InterfaceName>`
- Boundary Rules: <Encapsulation constraints and ACL mapping>

## 2. Frozen Interface Specifications
```text
[CONTRACT: <UpstreamContext> -> <DownstreamContext>]
Protocol: <In-process Interface | REST | gRPC | Domain Event>
Direction: <Sync Call | Async Event>
Payload:
  - Input / Command: <Typed structure or schema reference>
  - Output / Event: <Typed structure or schema reference>
Invariants:
  - Idempotency: <Required / Optional / N/A>
  - Error Model: <Deterministic error codes and retry semantics>
```

## 3. Harness Stubs & Consumer Fixtures
- Mock Provider: <Repo-relative path for mock stubs or contract fixtures>
- Verification Target: <Consumer test criteria validating against this contract>

Summary: <N> contexts defined, <M> cross-boundary contracts frozen.
*(omit if clean)* Gap: <Conflicting domain ownership or ambiguous contract boundary.>
```

### Routing

- Greenfield plumbing verification across frozen contracts → (via /skeleton-spike).
- Large-scale refactor with frozen contracts → (via /seam-plan).
- Single-module feature with frozen contracts ready for file impact planning → (via /changeset).
- Domain boundary or schema ambiguities unresolved → (via /clarify).

## Execution Protocol

1. Analyze architectural proposals and codebase module boundaries.
2. Partition domain responsibilities into isolated Bounded Contexts with single data ownership.
3. Draft typed interfaces, payloads, error models, and mock stubs for all cross-boundary traffic.
4. Output the frozen contract specification and halt turn for human authorization before any implementation.
