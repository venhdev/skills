---
name: assay
description: "Evaluate changesets, task breakdowns, or code diffs for scope bloat, architectural drift, and regression risks."
disable-model-invocation: true
---

# assay — Mutation Review & Adversarial Quality Engine

## Domain Rubric

### 1. Review Dimensions

Inspect mutation artifacts against 4 structural dimensions:

- **Scope Integrity (Blast Radius & Anti-Bloat)**: Verify that every planned or modified file is strictly load-bearing. Flag accidental formatting, cosmetic churn, unrelated refactoring, or speculative files.
- **Contract & Spec Fidelity (Anti-Drift)**: Reconcile changes against governing specifications, SSOT documents, and established interfaces. Flag undocumented behavioral changes or broken contracts.
- **Structural Cohesion & Slicing**: Verify modular encapsulation without leaky abstractions. For tasks, verify true vertical slicing (end-to-end capability) over horizontal layer fragmentation.
- **Operational Hygiene & Safety**: Expose unhandled edge cases, resource leaks, lingering debug probes (`[DEBUG-<hex>]`), or unredacted secrets.

### 2. Review Verdicts

- `PASS`: Zero structural or contract defects; blast radius is minimal and strictly load-bearing; ready for execution or commit.
- `CONDITIONAL`: Minor defects observed (accidental churn, missing edge test, localized bloat); actionable remediation provided to unblock.
- `FAIL`: Critical defects detected (contract violation, unauthorized blast radius, fake vertical slice, regression risk); reject and halt.

### 3. Anti-Hallucination & Grounding Guardrails

- **Dual-Citation Rule**: Every reported `Contract & Spec Fidelity` defect must explicitly cite both the offending code pointer (`file:///path/to/code#L<N>`) and the violated specification or interface contract (`file:///path/to/spec#L<M>`). Forbid claiming drift without contract citation.
- **Multi-Layer Contract Grounding**: Ground mutations against 3 authoritative layers: explicit specifications/tasks, structural code interfaces/schemas, and baseline test invariants. Mark Contract Fidelity as `UNGROUNDED` only when no governing specs or code contracts exist across the repository; evaluate strictly on Scope Integrity and Hygiene. Forbid inventing imaginary requirements.

## Canonical Output Contract

```markdown
# Assay Report: <Target / Scope Name>

## 1. Executive Verdict
- **Verdict**: <PASS | CONDITIONAL | FAIL>
- **Target Inspected**: <Working Tree Diff | Staged Changeset | Task Breakdown | Commit <hash>>
- **Governing Contracts**: <Cited specification file pointers or 'UNGROUNDED (No governing spec found)'>
- **Summary**: <Single-sentence technical judgment>

## 2. Findings & Actionable Remediations
*(If PASS: 'No defects observed across scope integrity, contract fidelity, structural cohesion, and operational hygiene.')*

- **[<Dimension>] <Defect Title>**: <Concise technical description of defect and failure risk>
  - Offending Code: [<file>#L<N>](file:///path/to/file#L<N>)
  - Violated Contract: [<spec_or_interface>#L<M>](file:///path/to/spec#L<M>)
  - Remediation: <Exact minimal correction needed>

## 3. Pipeline Routing
- `/forge` ── Target is PASS; proceed to implementation or commit handoff.
- `/changeset` ── Blast radius bloat or structural defects require re-planning.
- `/clarify` ── Contract drift or ungrounded architectural tensions detected.
```

## Execution Protocol

### Phase 1: Scope & Contract Grounding

1. Ingest target mutation artifact (uncommitted diff, staged changeset, `.agents/tasks/`, or commit range).
2. Trace and ground governing boundaries across the codebase per Multi-Layer Contract Grounding guardrails.

### Phase 2: Scale-Adaptive Adversarial Review

1. Evaluate mutation scope scale:
   - *In-Turn Review* (localized diff, 1–5 files, or single task card): Audit mutations directly in-turn against review dimensions.
   - *Subagent Fan-Out* (multi-package changesets or broad PR diffs): Launch 2–3 `research` subagents (Role: `Mutation Reviewer (<Slice>)`). Seed each with target diff slice, discovered contracts, and Canonical Output Contract.
2. Seed each subagent with:
   - Its allocated diff slice and target file pointers.
   - Discovered governing contracts and invariant baselines.
   - The Dual-Citation Rule and Canonical Output Contract as the mandatory output contract.
3. Assign subagent role: `Mutation & Architecture Reviewer (<Target Slice/Subsystem>)`.

### Phase 3: Synthesis & Turn-Halt Gate (Fan-in)

1. Reconcile and deduplicate subagent findings into the Canonical Output Contract.
2. Compute the definitive Executive Verdict (`PASS` only if all slices pass; downgrade to `CONDITIONAL` or `FAIL` based on highest defect severity).
3. Emit the report into the conversation stream and halt turn immediately.

