# Classification

Classify a doc before writing. Output: `type`, `kind`, and optional lifecycle route.

See `contracts/frontmatter.md` for field definitions and `contracts/docs-structure.md` for placement guidance.

## Canonical Diataxis types

| Type | Answers | Notes |
|---|---|---|
| `tutorial` | "How do I learn this?" | Hands-on, learning-oriented |
| `how-to` | "How do I accomplish X?" | Task-oriented, steps |
| `reference` | "What does X do?" | Dense, exhaustive, current truth |
| `explanation` | "Why is it designed this way?" | Background, tradeoffs, concepts |
| `decision` | "What was chosen and why?" | ADR only |

## Kind assignment

| kind value | Use when |
|---|---|
| `plan` | Temporary execution plan: roadmap, rollout, migration, milestones |
| `spec` | Durable current behavior/requirements/contracts for a feature or capability |
| `adr` | Architectural decision record |
| `ssot` | Canonical source; other docs link, not copy |
| `draft` | Work-in-progress, not yet stable |
| `til` | Short practical note from a real issue/task |

Multiple `kind` values are allowed when useful, e.g. `kind: [spec, ssot]`.

## Title signals

| Title pattern | Classification |
|---|---|
| getting started, introduction, walkthrough | `type: tutorial` |
| how to, guide to | `type: how-to` |
| api, reference, configuration, schema | `type: reference` |
| spec, specification, requirements, behavior, contract | `type: reference`, `kind: [spec]` |
| why, architecture, rationale, overview | `type: explanation` |
| plan, roadmap, rollout, migration, milestones | `type: explanation`, `kind: [plan]` |
| ADR, decision record | `type: decision`, `kind: [adr]` |
| TIL, today I learned, note from issue | add `kind: [til]` |

## Lifecycle routes

| Condition | Route |
|---|---|
| Type is `decision` or kind includes `adr` | `workflows/adr-lifecycle.md` |
| Kind includes `plan` | `workflows/plan-lifecycle.md` |
| Kind includes `spec` | `contracts/quality-checklist.md` spec checklist |
| Otherwise | Main workflow in `SKILL.md` |
