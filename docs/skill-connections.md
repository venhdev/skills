---
title: ZenForge Skill Connections
description: Canonical map of conditional handoffs and composition across ZenForge skills.
status: active
created: 2026-10-14
updated: 2026-10-14
---

# ZenForge Skill Connections

This document owns the connection map for all 27 skills. Three views show the same
network; repeated nodes refer to the same skill. Arrows describe conditional
handoffs, not a required sequence of invocations.

Input/output and state criteria belong to each skill. Shared authoring semantics
belong to [Construction Rules](ZENFORGE_CONSTRUCTION_RULES.md#11-input-state--routing).
The reviewed framing, proof, planning, and implementation contracts are applied
below; other skill responsibilities and specialist verdicts are retained.

## Reading the Map

Solid arrows describe handoffs or wrapper composition with the prerequisite on
the edge. Dashed arrows describe optional review, support, or caller-requested
follow-up. Potential graph edges are not an output menu: select actual handoffs
under the Construction Rules, and complete the request without another skill
when no gap remains. A handoff carries evidence, constraints, and the applicable
authorization; an arrow supplies none of these by itself.

## Framing, Proof, Planning, and Delivery

```mermaid
flowchart LR
    subgraph framing[Framing and decisions]
        distill["/distill"]
        clarify["/clarify"]
        architect["/architect-decision"]
    end
    subgraph proof[Proof and implementation]
        diagnose["/diagnose"]
        spike["/spike"]
        forge["/forge"]
    end
    subgraph planning[Planning and tracking]
        changeset["/changeset"]
        tasks["/to-tasks"]
        simplify["/simplify-audit"]
    end
    assay["/assay"]

    distill -->|Confirmed intent with unresolved requirements| clarify
    distill -->|Direction needs selection| architect
    distill -->|Accepted supported direction| changeset
    distill -->|Symptom needs investigation| diagnose
    distill -->|Supplied candidate needs proof| spike
    clarify -->|Settled direction| changeset
    clarify -->|Candidate needs proof| spike
    clarify -->|Architecture needs selection| architect
    architect -->|Candidate and direction need empirical proof| spike
    architect -->|Supported direction accepted| changeset
    diagnose -->|Candidate and direction supplied| spike
    diagnose -->|Solution direction needs selection| architect
    spike -->|Proven solution accepted| changeset
    spike -->|Requirements need resolution| clarify
    spike -->|Original direction cannot hold| architect
    changeset -->|Supported scope and authorized implementation| forge
    changeset -->|Tracking requested| tasks
    changeset -->|Stale plan needing requirement revision| clarify
    changeset -->|Stale plan needing direction redesign| architect
    tasks -.->|Eligible task with requested authorized execution| forge
    forge -->|Material requirements gap| clarify
    forge -->|Direction requires redesign| architect
    forge -->|Settled direction needing impact restaging| changeset
    simplify -->|Embedded changeset supported and authorized| forge
    simplify -->|Architectural requirements unresolved| clarify
    simplify -->|Architecture needs a design decision| architect
    changeset -.->|Plan review requested| assay
    tasks -.->|Task review requested| assay
    forge -.->|Implementation review requested| assay
    assay -->|Clear scope correction| changeset
    assay -->|Contract tension| clarify
    assay -->|PASS and supported unapplied scope authorized| forge
```

`diagnose` ends at a root-cause report; it does not fix production.
`spike` receives a problem and candidate/direction, and only explores alternatives
inside that direction. Unverified assumptions or missing access remain explicit
gaps rather than evidence of success or failure.

`simplify-audit` can embed its own changeset; it does not call `changeset` just to
stage that artifact. `to-tasks` records work and splits only when useful, ending
after verified publication unless implementation is separately requested.
`forge` ends at verified implementation and a proposed commit message; a
necessary redesign is a separate handoff, not an implementation adjustment.

## Inspection and Remediation

```mermaid
flowchart LR
    subgraph inspection[Review and audit specialists]
        harness["/harness-audit"]
        perimeter["/perimeter"]
        structure["/inspect-structure"]
        naming["/inspect-role-naming"]
        perf["/inspect-perf"]
        security["/inspect-security"]
    end
    init["/zenforge-init"]
    clarify["/clarify"]
    changeset["/changeset"]
    forge["/forge"]
    simplify["/simplify-audit"]
    diagnose["/diagnose"]

    harness -->|Missing harness anchor| init
    harness -->|Infrastructure trade-off| clarify
    harness -->|Clear remediation scope| changeset
    harness -->|Supported test fix already authorized| forge
    perimeter -->|Ownership or exception unresolved| clarify
    perimeter -->|Qualified boundary needs staging| changeset
    perimeter -->|Supported boundary changeset authorized| forge
    structure -->|Structural remediation| changeset
    structure -->|Suspected functional defect| diagnose
    naming -->|Structural or cross-file correction| changeset
    naming -->|Local behavior-preserving rename candidate| simplify
    perf -->|Systemic or behavior-changing remediation| changeset
    perf -->|Behavior-preserving optimization candidate| simplify
    security -->|Local or systemic security patch needs staging| changeset
    security -->|Supported security changeset authorized| forge
```

A clean audit can finish without remediation. Locality does not establish behavior
equivalence: security patches, new query limits, or public contract changes need
their own supported changeset. Routes to `simplify-audit` request inspection and
staging, never direct refactoring.

## Context, Governance, Isolation, and Release

```mermaid
flowchart LR
    recon["/recon"] -->|Settled direction with change requested| changeset["/changeset"]
    recon -->|Architectural tension| clarify["/clarify"]
    scout["/scout"] -->|Accepted supported baseline with change requested| changeset
    scout -->|Project fit needs deliberation| clarify
    distill["/distill"] -->|Repository capability unknown| recon
    distill -->|Production context unknown| scout
    ssot["/ssot"] -.->|Authoritative contracts| work["Applicable skill invocation"]
    soul["/zenforge-soul"] -.->|Context-based routing guidance| work
    doctor["/env-doctor"] -.->|Environment readiness verified| work
    init["/zenforge-init"] -.->|Harness audit requested| harness["/harness-audit"]
    init -.->|Documentation audit requested| ssot
    init -.->|Environment audit requested| doctor
    init -.->|Design deliberation requested| clarify
    eli5["/lens-eli5"] -.->|Explanation support without required handoff| work
    pro["/lens-pro"] -.->|Explanation support without required handoff| work
    isolated["/forge-isolated"] -->|Create isolated workspace| worktree["/worktree"]
    delegated["/forge-isolated-subagent"] -->|Create worker workspace| worktree
    isolated -->|Workspace exists and supported scope authorized| forge["/forge"]
    delegated -->|Seeded worker with supported authorized scope| forge
    forge -.->|Release preflight requested| release["/ci-pub"]
    release -->|Approved setup determines commands| publish["Publication handoff"]
```

Context endpoints are not extra skills. Wrapper edges describe composition,
not alternative recommendations; workspace creation or dispatch does not prove
worker completion. Release readiness does not establish publication. Explanation,
mapping, documentation lookup, initialization, and worktree list/delete may end
without a downstream invocation.

## Source Contracts

- Framing: [distill](../skills/distill/SKILL.md), [clarify](../skills/clarify/SKILL.md), [architect-decision](../skills/architect-decision/SKILL.md), [contract-freeze](../skills/contract-freeze/SKILL.md), [seam-plan](../skills/seam-plan/SKILL.md).
- Proof: [diagnose](../skills/diagnose/SKILL.md), [spike](../skills/spike/SKILL.md), [skeleton-spike](../skills/skeleton-spike/SKILL.md), [tdd](../skills/tdd/SKILL.md).
- Planning and implementation: [changeset](../skills/changeset/SKILL.md), [split-tasks](../skills/split-tasks/SKILL.md), [tracking-tasks](../skills/tracking-tasks/SKILL.md), [to-tasks](../skills/to-tasks/SKILL.md), [forge](../skills/forge/SKILL.md), [simplify-audit](../skills/simplify-audit/SKILL.md).
- Review: [assay](../skills/assay/SKILL.md), [harness-audit](../skills/harness-audit/SKILL.md), [perimeter](../skills/perimeter/SKILL.md), [inspect-structure](../skills/inspect-structure/SKILL.md), [inspect-role-naming](../skills/inspect-role-naming/SKILL.md), [inspect-perf](../skills/inspect-perf/SKILL.md), [inspect-security](../skills/inspect-security/SKILL.md).
- Support: [recon](../skills/recon/SKILL.md), [scout](../skills/scout/SKILL.md), [ssot](../skills/ssot/SKILL.md), [env-doctor](../skills/env-doctor/SKILL.md), [zenforge-init](../skills/zenforge-init/SKILL.md), [zenforge-soul](../skills/zenforge-soul/SKILL.md), [lens-pro](../skills/lens-pro/SKILL.md), [lens-eli5](../skills/lens-eli5/SKILL.md).
- Isolation and release: [worktree](../skills/worktree/SKILL.md), [forge-isolated](../skills/forge-isolated/SKILL.md), [forge-isolated-subagent](../skills/forge-isolated-subagent/SKILL.md), [ci-pub](../skills/ci-pub/SKILL.md).
