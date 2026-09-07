---
name: distill
description: "Deconstruct raw intent, expose XY problems, and isolate latent assumptions to establish unambiguous problem boundaries before architectural design."
disable-model-invocation: true
---

# distill — Intent Deconstruction & Problem Framing Engine

Deconstruct raw inquiries, expose XY problems, isolate latent assumptions, and establish crisp problem boundaries before architectural design or implementation planning.

## Domain Engine & Standards

### 1. The XY Problem Rubric
Deconstruct proposals across two distinct layers:
- **Root Objective ($X$)**: The core business outcome, operational bottleneck, or defect requiring resolution.
- **Proposed Mechanism ($Y$)**: The specific tool, algorithm, framework, or architectural pattern requested.
- **Diagnostic Categories**:
  - `Aligned`: Mechanism $Y$ directly and proportionally resolves Objective $X$ with minimal overhead.
  - `Overkill`: Mechanism $Y$ resolves Objective $X$ but introduces disproportionate accidental complexity or operational toil.
  - `Misaligned`: Mechanism $Y$ treats superficial symptoms without resolving the root cause of Objective $X$.
  - `Premature`: Mechanism $Y$ optimizes for hypothetical future requirements without empirical justification.

### 2. Latent Assumption Dimensions
Surface unstated premises across 4 core vectors:
- **Scale & Volume**: Traffic, data volume, and concurrency expectations (e.g., distributed architecture for single-node workloads).
- **Consistency & Latency**: Synchronization models, eventual consistency tolerances, and real-time demands.
- **Operational Capacity**: Maintenance bandwidth, infrastructure costs, and cognitive complexity.
- **User & Environmental Reality**: Access patterns, failure tolerance, and real-world client constraints.

### 3. Canonical Framing Dossier Format
Present the distillation in the structured, high-density format:

````markdown
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
````

## Workflow

**SUB-SKILL:** scout, clarify, changeset

### Phase 1: Symptom & Intent Extraction
1. Isolate the underlying operational symptoms or business goals from the proposed technology choice.

### Phase 2: XY Deconstruction & Boundary Analysis
1. Map the proposal against the XY Problem Rubric to separate Root Objective ($X$) from Proposed Mechanism ($Y$).
2. Surface unstated premises against Latent Assumption Dimensions and define strict In-Scope vs. Out-of-Scope boundaries.

### Phase 3: Delivery & Turn Halt
1. Synthesize findings into the Canonical Framing Dossier format and deliver strictly in the conversation stream in a single turn. Never create, modify, or delete repository files or propose code diffs.
2. Halt turn immediately to await user review before any downstream architectural commitment.
