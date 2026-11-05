---
name: conform-design
description: "Design a solution to a confirmed problem that conforms to the project's established architecture, rules, contracts, and existing patterns."
disable-model-invocation: true
---

# conform-design — Eco-Conformant Solution Design

## Operating Invariants

- **Problem Gate**: Halt immediately if the problem statement is ambiguous, unconfirmed, or unscoped. Route to clarifying skills and stop — do not proceed until the problem is explicit and user-confirmed.
- **Context Separation**: Treat *problem context* and *documentation context* as distinct. Problem context requires human confirmation before proceeding. Documentation context (SSOT, rules, contracts, structure, patterns) may be explored autonomously via subagent or direct inspection — never block on it.
- **Conformance Mandate**: Design exclusively within the established architecture. Never invent new patterns, layers, or conventions; reuse existing components, helpers, utilities, and seams. If the settled architecture cannot solve the problem, surface the gap and halt.

---

## Execution Protocol

### 1. Problem Gate *(halt if fails)*

Verify the problem is:
- **Explicit**: observable behavior, affected seam, or objective stated
- **Scoped**: bounded — not an open-ended investigation
- **Confirmed**: user or prior skill output has affirmed this is the target

If any condition fails → state which is missing → route and **stop**:
- Vague problem → `/clarify`
- Root-cause unknown → `/diagnose`
- Codebase unfamiliar → `/recon`

### 2. Architecture Surface *(autonomous)*

Explore without blocking on human input:
- Scan `.agents/`, `docs/`, ADRs, rules files for governing constraints relevant to the problem scope
- Identify: layer rules, naming conventions, existing abstractions, reusable components/helpers/utils
- Note frozen contracts and schemas that must survive unchanged
- Use subagent for broad SSOT exploration if scope is large

### 3. Solution Design

The established architecture largely determines the solution. Design directly within surfaced constraints; do not brainstorm alternatives for the sake of it.

Only surface a secondary option when two genuinely distinct conformant approaches exist within the eco — and always resolve to one selected direction with rationale. Never leave the choice to downstream.

---

## Canonical Output Contract

```markdown
# conform-design: <Problem Slug> · <SETTLED | UNRESOLVED>

## Problem
<One sentence: what is broken/needed, which seam, which layer>

## Governing Rules
- <rule/contract/SSOT ref>: <constraint it imposes>

## Design
<Mechanism specific to this codebase — layers, components, seams to touch>
<If two valid conformant paths exist, note the rejected one and why>

## Constraints for Downstream
- MUST: <rule ref>
- MUST NOT: <boundary>

## Next
<Immediate next skill: /changeset | /spike | /split-tasks | /clarify — with reason>
```

**States**: `SETTLED` — ready for downstream. `UNRESOLVED` — blocked; reason and route stated.
