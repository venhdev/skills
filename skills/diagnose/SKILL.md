---
name: diagnose
description: "Isolate root causes from defect descriptions, errors, logs, or reproductions through discriminating investigation and minimal feedback loops."
disable-model-invocation: true
---

# diagnose — Root-Cause Isolation

## Operating Invariants

- **Investigation Boundary**: Read production source and existing tests; write temporary harnesses, fixtures, trace replays, and report files only inside `.agents/scratch/`. End at a root-cause report; leave production fixes, permanent tests, and fix changesets to subsequent work.
- **Working Tree Preservation**: Preserve pre-existing changes and scratch artifacts. Run probes against isolated copies or existing read-only seams; keep instrumentation out of production source. Tag temporary probes `[DEBUG-<hex>]` and remove only session-owned instrumentation after collecting evidence.
- **Sensitive Evidence & Environment**: Redact secrets before logging; never inspect secret files (`.env*`, `*.key`, `*.pem`). Ground runners in `.agents/dev-environment.md`; confine execution strictly to local scratch boundaries without touching production or host-level containers.

## Domain Rubric

### Intake and Causal Evidence

- Accept a defect description, error, log, stack trace, or reproduction. Identify expected behavior, observed behavior, relevant environment, and the execution seam; distinguish missing evidence from contradicted evidence.
- Begin investigation with available evidence even when an executable RED is unavailable. Treat a failing assertion as symptom evidence; establish cause through a discriminating probe, a conclusive execution trace, or a verified causal chain.
- Separate trigger and context from the mechanism that produces the failure. For each defensible hypothesis, state a prediction: `If <cause>, then <controlled probe> changes <observation>.`
- Test competing explanations when evidence permits; vary one causal factor at a time and record results that support or falsify each candidate. Prefer direct traces for obvious syntax errors or deterministic failures over unnecessary harnesses.

### Reproduction and Feedback Loops

Choose the tightest loop faithful to the symptom and available runtime:

| Seam | Investigation method |
|---|---|
| Existing test | Run the focused assertion at the failing call site |
| Running process | Invoke the CLI or HTTP path and compare expected/actual output |
| Isolated callable | Build a minimal scratch harness; disclose substituted I/O |
| Captured event | Replay sanitized payloads, logs, or HAR data in scratch |
| Historical regression | Compare revisions in an isolated scratch checkout |
| Interactive-only trigger | Adapt [the local loop template](scripts/hitl-loop.template.sh) in scratch; capture redacted responses |

- Minimize inputs, configuration, and steps while preserving the exact symptom; retain every factor needed for fidelity, including relevant integration behavior.
- For intermittent failures, control seeds, clocks, event ordering, or concurrency where feasible. Report attempts, failures, and intervention results; use bounded repetitions rather than treating one lucky pass as proof.
- Preserve real architectural seams. If mocks omit the suspected mechanism or no faithful seam is available, document that proof gap and continue evidence-based investigation within scope.

### Conclusion States

- `CONFIRMED`: Evidence verifies the causal mechanism, including the link from that mechanism to the reported symptom.
- `INFERRED`: Evidence supports a specific causal hypothesis, but a stated gap prevents confirmation.
- `REJECTED`: Discriminating evidence falsifies the investigated causal claim in the stated scope; identify the claim rather than implying the defect is disproven.
- `UNRESOLVED`: Available evidence supports no defensible cause, or an input, environment, or access blocker prevents a meaningful conclusion.

## Canonical Output Contract

Produce one compact report. Include commands, relevant observations, and source locations sufficient to reproduce or inspect the reasoning; keep raw logs in sanitized scratch artifacts when needed.

```markdown
# Root-Cause Report: <defect>

State: <CONFIRMED | INFERRED | REJECTED | UNRESOLVED>
Conclusion: <causal mechanism, supported hypothesis, rejected claim, or unresolved question>
Scope: <symptom, runtime/environment, and inspected seam>
Evidence: <commands or trace references; observed results; discriminating causal link>
Conditions and proof limits: <assumptions, substitutions, missing evidence, and applicability limits>

## Competing explanations *(omit if none were investigated)*
<claim -> probe/prediction -> observation -> supported or falsified>

## Reproduction *(omit if no executable or interactive loop was established)*
<scratch path or existing command; required setup; assertion and observed symptom>
```

### Routing

- An existing candidate and direction need solution proof → (via /spike).
- Root-cause findings need a solution direction → (via /architect-decision).

## Execution Protocol

1. Ground the symptom in supplied evidence and inspect the relevant call chain. Record the narrow causal question and available verification seam.
2. Establish and minimize a faithful RED loop when possible. When execution is unavailable, continue through traces and source evidence while recording the missing proof.
3. Probe ranked hypotheses until a cause is confirmed, supported with an explicit gap, falsified, or unresolved. Stop an investigation branch when further probes require unavailable input/access or merely repeat the same signal.
4. Present the root-cause report with its state and proof limits. Preserve useful sanitized reproduction artifacts, remove session-owned instrumentation, and verify production files remain unchanged.
