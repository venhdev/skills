# Classification Contract

Classify a doc before creating or substantially reworking it. Output `type`, `kind`, and any lifecycle route.

## Allowed `type` values

| Type | Use when the doc answers | Notes |
|---|---|---|
| `tutorial` | How do I learn this from the beginning? | Hands-on learning path. |
| `how-to` | How do I accomplish this task? | Steps toward a concrete outcome. |
| `reference` | What is the exact contract or field? | Dense, neutral, current facts. |
| `explanation` | Why is it this way? | Context, rationale, tradeoffs. |
| `decision` | What was decided and why? | ADR only. |

## Allowed `kind` values

| Kind | Use when |
|---|---|
| `plan` | Temporary execution plan, rollout, roadmap, migration, or milestone doc. |
| `spec` | Durable current behavior, requirements, or system contract. |
| `adr` | Architectural decision record. |
| `ssot` | Canonical source of truth; other docs link instead of copying. |
| `draft` | Work is not stable yet. |
| `til` | Short practical note from a real issue or task. |

`kind` is always a list, even for one value.

## Multi-kind rules

Use multiple values when each is true:

- `kind: [spec, ssot]` for a durable spec that is also canonical truth.
- `kind: [draft, spec]` for a not-yet-accepted spec draft.

Do not normalize away valid secondary kinds.

## SSOT placement

`ssot` describes canonical role, not a folder mandate. `docs/reference/` and `docs/specs/` are common homes, but unusual placement is a review signal rather than a format error unless content contradicts metadata.

## TIL role

`kind: [til]` is a practical note. It may link to a spec, ADR, or guide, but it is not durable canonical guidance by itself.

## Kind defaults by doc shape

- `how-to` docs usually use `kind: []` unless the user explicitly needs another lifecycle role.
- `explanation` docs usually use `kind: []` unless they are specifically a `plan` or `til`.
- Do not infer `kind: [spec]` just because a how-to or explanation mentions current behavior, validation, configuration, or canonical docs.
- Use `kind: [draft]` only when the user clearly wants to mark unstable status in metadata.

## Title and intent signals

| Signal | Classification |
|---|---|
| getting started, walkthrough | `type: tutorial` |
| how to, guide, run steps | `type: how-to` |
| API, configuration, schema, field list | `type: reference` |
| spec, requirements, behavior, contract | `type: reference`, `kind: [spec]` |
| overview, why, rationale, architecture | `type: explanation` |
| plan, rollout, migration, milestones | `type: explanation`, `kind: [plan]` |
| ADR, decision record | `type: decision`, `kind: [adr]` |
| TIL, today I learned, incident note | add `kind: [til]` |

## Intake block classification

Use this when Create mode receives extracted external source material such as Markdown/text from a PDF, DOCX, API JSON, meeting notes, or partner spec, or when the user explicitly marks a small unclear idea with `--in` / `--intake`.

Classify the source before authoring:

| Intake block | Use for |
|---|---|
| Requirement | Desired behavior, capability, outcome, or acceptance expectation. |
| Constraint | Technical, business, security, compliance, budget, deadline, platform, or operational limit. |
| Milestone | Sequenced delivery slice, rollout step, integration phase, or target date. |
| Dependency | Current doc, API, system, team, access, decision, or prerequisite. |
| Risk | Uncertainty that may change delivery, scope, quality, security, cost, or operations. |
| Open question | Missing information that does not yet have a reliable answer. |
| User-idea intake | Small unclear request explicitly marked with `--in` or `--intake`; clarify before authoring. |
| Follow-up doc | Spec, ADR, how-to, or reference doc that should be split out later. |

Clarification policy:

- For `--in` / `--intake` user-idea intake, clarify outcome, users, scope boundaries, success criteria, constraints, risks, dependencies, and output shape before authoring.
- Ask before creating when a gap or conflict changes the goal, scope, milestone order, gate conditions, ownership, go/no-go direction, ADR-level decision, or whether durable behavior belongs in a spec.
- Do not convert blockers into assumptions.
- Non-blocking gaps may remain as open questions in the generated doc.

Enterprise/source mapping:

| Source signal | Default handling |
|---|---|
| BRD or high-level business requirements | Plan goal, scope, milestones, risks. |
| SRS or detailed requirements | Plan first; recommend a spec for durable detailed behavior. |
| HLD or architecture design | Plan plus ADR/spec recommendation when decisions are stable or blocking. |
| LLD or detailed module design | Recommend a spec; use plan only for implementation sequencing. |
| API JSON, OpenAPI, or partner API text | Plan milestones by endpoint/domain/auth/testing; recommend API reference or spec. |
| Test plan or test cases | Plan validation/risk items; recommend spec or how-to when durable. |
| Deployment or ops guide | Plan dependency/risk items; recommend how-to. |
| User manual | Usually how-to; plan only when implementation/change work is implied. |
| Meeting notes or transcript | Plan when action-oriented; explanation or TIL when mainly context. |
