# Quality Checklists

Use checklists as completion criteria. Pick only the relevant checklist(s) for the current task and keep scope minimal.

## Universal checklist (every doc change)

| # | Check | Pass condition |
|---|---|---|
| 1 | Type/kind valid | Matches contract in `contracts/frontmatter.md` |
| 2 | Frontmatter valid | `check_frontmatter.py` passes |
| 3 | Dependency targets valid | Every `depends-on` target exists and is current |
| 4 | Cascade intent clear | Reciprocal link repaired or explicit one-way note |
| 5 | `lastReviewed` handled | Updated on meaningful change |
| 6 | Scope respected | Only requested files/scope touched |
| 7 | Output complete | Mode, status, changed files, findings, followups, validation |
| 8 | Structure respected | Existing convention followed; restructure proposed only with user approval |

## ADR checklist (extra)

| # | Check | Pass condition |
|---|---|---|
| 1 | ADR typing | `type: decision`, `kind` includes `adr` |
| 2 | ADR id/status fields | `adr-id`, `status`, `deciders`, `decided` valid |
| 3 | Accepted body immutability | Accepted ADR body unchanged; only metadata/Status Log updated |
| 4 | Supersession linkage | Old ADR has `supersededBy`, new ADR has `supersedes` |
| 5 | Dependency semantics | Replacement ADR does not depend on superseded ADR as current prerequisite |

## Plan checklist (extra)

| # | Check | Pass condition |
|---|---|---|
| 1 | Plan typing | `type: explanation`, `kind` includes `plan` |
| 2 | Status coherence | `status` aligns with milestones/progress |
| 3 | Cadence/freshness | Effective cadence applied (explicit or default) |
| 4 | Cascade impact | Downstream docs identified via `updates`/incoming links |

## Completed plan → spec + archive checklist

| # | Check | Pass condition |
|---|---|---|
| 1 | Related spec exists | `docs/specs/<topic>.md` exists or is created |
| 2 | Spec typing | Spec has `type: reference`, `kind` includes `spec` |
| 3 | Durable outcome promoted | Current behavior/requirements/accepted constraints moved from plan into spec |
| 4 | Plan completion marked | Plan has `status: completed`, `completed`, and updated `lastReviewed` |
| 5 | Archive marked | Archived plan has `status: archived`, `archived`, and `replacedBy` |
| 6 | Archive location | Plan moved to `docs/archive/plans/` when archive is requested/approved |
| 7 | Current refs updated | Current docs no longer depend on archived plan; refs point to spec or are removed |
| 8 | Archive graph excluded | Do not block current health on broken internal links inside archived plan |

## Spec checklist (extra)

| # | Check | Pass condition |
|---|---|---|
| 1 | Purpose | Describes current durable behavior, not execution history |
| 2 | Scope | Includes requirements/behavior/contracts/acceptance criteria that are current |
| 3 | Source of truth | Replaces completed plan as current reader entrypoint |
| 4 | Links | Depends only on current docs, not archived plans |

## TIL checklist (extra)

| # | Check | Pass condition |
|---|---|---|
| 1 | Purpose | Captures a short practical lesson from real issue/task |
| 2 | Brevity | Short note; avoids tutorial/reference bloat |
| 3 | Reuse value | Includes actionable fix/pattern to prevent repeat |
| 4 | Metadata | `kind` includes `til`, frontmatter valid |
