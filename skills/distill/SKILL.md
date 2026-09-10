---
name: distill
description: "Deconstruct raw intent, expose XY problems, and isolate latent assumptions to establish unambiguous problem boundaries before architectural design."
disable-model-invocation: true
---

# distill — Intent Deconstruction & Problem Framing Engine

Deconstruct raw inquiries, expose XY problems, isolate latent assumptions, and establish crisp problem boundaries before architectural design or implementation planning.

## Operating Invariants

- **Read-Only Stream**: Deliver analysis exclusively within the conversation stream and halt turn immediately. Never create, modify, or delete workspace files or propose code diffs.
- **Root-First Isolation**: Separate the underlying operational objective ($X$) from the requested tool or mechanism ($Y$); challenge premature technical commitments.
- **Context Preservation**: Frame problem boundaries in isolation without derailing active session context or mutating project scope.

## Domain Standards

### 1. Diagnostic Categories

- `Aligned`: Mechanism $Y$ directly and proportionally resolves Objective $X$ with minimal overhead.
- `Overkill`: Mechanism $Y$ resolves Objective $X$ but introduces disproportionate accidental complexity.
- `Misaligned`: Mechanism $Y$ treats superficial symptoms without resolving the root cause of Objective $X$.
- `Premature`: Mechanism $Y$ optimizes for unverified future requirements without empirical justification.

### 2. Latent Assumption Vectors

- **Scale & Volume**: Traffic, data volume, and concurrency expectations.
- **Consistency & Latency**: Synchronization models, eventual consistency tolerances, and real-time demands.
- **Operational Capacity**: Maintenance bandwidth, infrastructure costs, and cognitive complexity.
- **User & Environmental Reality**: Access patterns, failure tolerance, and real-world client constraints.

## Canonical Framing Dossier Format

```markdown
# Framing Dossier: <Topic / Problem Slug>

## 1. Intent Deconstruction (XY Analysis)
- **Root Objective (X)**: <Core outcome or problem requiring resolution>
- **Proposed Mechanism (Y)**: <Initial approach, requested tool, or suggested architecture>
- **Diagnostic**: <Aligned | Overkill | Misaligned | Premature>
- **Diagnostic Rationale**: <Single-sentence technical explanation of fit or tension>

## 2. Latent Assumptions & Risks
| Vector | Unstated Assumption | Empirical Reality / Edge Case | Invalidation Impact |
| :--- | :--- | :--- | :--- |
| <Scale / State / Ops> | <Implicit premise> | <What must be proven or verified> | <Risk if assumption is false> |

## 3. Scope Boundaries
- **In-Scope**: <Minimal essential problem surface necessary to achieve Objective X>
- **Out-of-Scope**: <Premature generalizations, secondary features, or accidental complexity>

## 4. Pipeline Routing
- `/scout <inferred_topic>` ── Real-world production archetypes, scale metrics, or failure modes for Objective X are unknown.
- `/clarify` ── Objective X requires multi-turn deliberation across architectural trade-offs.
- `/changeset` ── Problem boundaries and baseline approach for Objective X are unambiguous.
```

## Execution Protocol

**SUB-SKILL:** scout, clarify, changeset

1. Ingest the user inquiry, proposed solution, or problem statement.
2. Evaluate against Diagnostic Categories and Latent Assumption Vectors.
3. Present the completed Framing Dossier into the conversation stream and halt turn immediately. Never create, modify, or delete repository files.
