# Classification Contract

Classify a doc before creating or substantially reworking it. Output `type`, `kind`, and any lifecycle route.

## Allowed `type` values

| Type | Use when the doc answers | Notes |
|---|---|---|
| `how-to` | How do I accomplish this task? | Steps toward a concrete outcome. For learning paths, use how-to with detailed prerequisites and setup steps. |
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
| `til` | Short practical note from a real issue or task. |

`kind` is always a list, even for one value.

## Multi-kind rules

Use multiple values when each is true:

- `kind: [spec, ssot]` for a durable spec that is also canonical truth (use `status: draft` on specs/plans instead of `kind: [draft, spec]`).
  - Example: `kind: [spec]` if the spec is living documentation for one team. `kind: [spec, ssot]` if other teams explicitly link to this spec as their authority on that topic.

Do not normalize away valid secondary kinds.

## SSOT placement

`ssot` describes canonical role, not a folder mandate. `docs/reference/` and `docs/specs/` are common homes, but unusual placement is a review signal rather than a format error unless content contradicts metadata.

## TIL role

`kind: [til]` is a practical note. It may link to a spec, ADR, or guide, but it is not durable canonical guidance by itself.

## Kind defaults by doc shape

- `how-to` docs usually use `kind: []` unless the user explicitly needs another lifecycle role.
- `explanation` docs usually use `kind: []` unless they are specifically a `plan` or `til`.
- Do not infer `kind: [spec]` just because a how-to or explanation mentions current behavior, validation, configuration, or canonical docs.
- For unstable docs without lifecycle roles (how-to, explanation, reference), use `status: draft` in frontmatter or a body callout instead of kind metadata.

## Title and intent signals

| Signal | Classification |
|---|---|
| getting started, walkthrough, learning path | `type: how-to` (with detailed prerequisites/setup) |
| how to, guide, run steps | `type: how-to` |
| API, configuration, schema, field list | `type: reference` |
| spec, requirements, behavior, contract | `type: reference`, `kind: [spec]` |
| overview, why, rationale, architecture | `type: explanation` |
| plan, rollout, migration, milestones | `type: explanation`, `kind: [plan]` |
| ADR, decision record | `type: decision`, `kind: [adr]` |
| TIL, today I learned, incident note | add `kind: [til]` |

## Discovery-first classification

Create mode must use discovery before authoring any new doc, including direct requests, rough notes, and source-material conversions such as extracted Markdown/text from a PDF, DOCX, API JSON, meeting notes, partner specs, imported requirements, or other attached documents.

During discovery, classify what you learn into:

| Discovery block | Use for |
|---|---|
| Requirement | Desired behavior, capability, outcome, or acceptance expectation. |
| Constraint | Technical, business, security, compliance, budget, deadline, platform, or operational limit. |
| Milestone | Sequenced delivery slice, rollout step, integration phase, or target date. |
| Dependency | Current doc, API, system, team, access, decision, or prerequisite. |
| Risk | Uncertainty that may change delivery, scope, quality, security, cost, or operations. |
| Audience signal | Who the document is for and what they need from it. |
| Scope boundary | What is in scope, what is out of scope, and what would change the artifact choice. |
| Artifact signal | Evidence that the request should become a plan, spec, ADR, how-to, explanation, or split set of docs. |

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

## Source material extraction protocol

When input contains structured source material (BRD, meeting notes, code), extract known facts directly — do not ask discovery questions about information already present in the source.

**Rule**: Never ask a discovery question about information already in the source material.

| Input type | Extract directly | Still ask about |
|---|---|---|
| BRD / SRS | Goals, scope, milestones, constraints | Milestone order, acceptance criteria, gate conditions |
| Meeting notes | Action items, decisions, owners, timeline | What is confirmed vs proposed; prioritization |
| Code files | Current behavior, API shape, existing patterns | Artifact choice (spec vs how-to); durable vs task-scoped |
| Short prompt / idea | *Nothing* — no structured content | Everything via discovery |

**When to use**: Recognize input type → extract what's present → ask only about what's missing → faster path to doc creation.

