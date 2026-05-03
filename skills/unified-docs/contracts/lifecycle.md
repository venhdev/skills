# Lifecycle Contract

This file defines ADR, plan, and spec lifecycle rules.

## ADR contract

ADR metadata:

```yaml
type: decision
kind: [adr]
adr-id: ADR-NNN
status: [draft | accepted | completed | superseded]
deciders: [name, name]
decided: [ISO date when accepted]
supersededBy:
supersedes:
```

Rules:

- `type` must be `decision`.
- `kind` must include `adr`.
- `adr-id` is sequential and zero-padded.
- `supersededBy` and `supersedes` use ADR IDs like `ADR-002`, not file paths.
- Once accepted, the decision body is write-once; only metadata and Status Log may change.
- ADR lifecycle is terminal: `draft → accepted → superseded`. ADRs are not "completed" (that's a plan lifecycle concept).

Status transitions:

| From | To | Action |
|---|---|---|
| absent | `draft` | Create ADR with next `adr-id`. |
| `draft` | `accepted` | Set `decided:` and append Status Log. |
| `accepted` | `superseded` | Create a new ADR, set old `supersededBy`, set new `supersedes`, keep old body intact except Status Log and metadata. |

## Plan contract

Plan metadata:

```yaml
type: explanation
kind: [plan]
status: [draft | in-progress | completed | archived]
completed:
archived:
replacedBy:
```

Rules:

- Plans are temporary execution artifacts.
- Milestones should be deliverables, not task crumbs.
- Goals should be measurable.
- Completed plans promote durable truth into a spec and set `replacedBy` once the accepted current spec exists.
- Archived plans leave the current dependency graph.
- Plans should not be added as inbound `updates` targets on durable spec or SSOT docs. If a plan needs to remind maintainers to refresh a durable doc at completion/archive time, keep that reference on the plan's own `updates` field or in its lifecycle notes.

Why `type: explanation`? Plans explain *how and why* an initiative will be executed: sequence, milestones, constraints, risks, dependencies. They provide context and rationale for upcoming work, not a permanent reference contract like a spec. Once the plan is done, its content promotes into a durable spec or retires to archive.

Status transitions:

| From | To | Action |
|---|---|---|
| absent | `draft` | Create plan. |
| `draft` | `in-progress` | At least one milestone is active. |
| `in-progress` | `completed` | Durable outcome has shipped; promote current truth into spec. |
| `completed` | `archived` | Move to archive only after a current spec exists. |

Archive rules:

- Suggested archive path: `docs/archive/plans/<plan-name>.md`
- Set `replacedBy` to the current spec path.
- Add a short top-of-body banner to the archived plan pointing readers to the spec.
- Update current inbound references to point to the spec or remove them.

## Spec contract

Spec metadata:

```yaml
type: reference
kind: [spec]              # may be [spec, ssot]
status: [draft | accepted]
```

Rules:

- Specs describe current durable truth: behavior, requirements, contracts, acceptance criteria, and operational constraints.
- `kind: [spec, ssot]` is valid when the spec is also canonical truth.
- Completed plans and superseded ADRs should not remain the current dependency target when an accepted current spec exists.
- For plan archival, a current spec means a non-archived spec with `status: accepted`; draft specs may be linked as work in progress but should not be the final `replacedBy` target for archived plans.
