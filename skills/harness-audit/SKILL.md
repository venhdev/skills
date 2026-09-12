---
name: harness-audit
description: "Audit test suites for infrastructure deficits, flakiness smells, and invariant blind spots, staging remediation changesets for /forge."
disable-model-invocation: true
---

# harness-audit — Test Suite & Harness Quality Inspector

Audit repository test suites for harness anchor compliance, infrastructure deficits, flakiness smells, and invariant blind spots without mutating code.

## Domain Engine & Standards

### 1. The Audit Hierarchy

Inspect target test suites across progressive architectural layers:

1. **Harness Anchor & Foundation Layer**:
   - Inspect the authoritative project harness anchor at `.agents/harness-anchor.md` and follow its `Governing Testing SSOTs` routes to read project testing standards.
   - If missing: Check for legacy test guides (`test/AGENTS.md`, `tests/AGENTS.md`). If found, use as temporary SSOT and advise running `/zenforge-init` to establish canonical `.agents/harness-anchor.md`. If absent, mark anchor as `UNGROUNDED` and route to `/zenforge-init`.
   - Verify existence on disk of registered hermetic primitives (Deterministic Clock, Storage Sandbox, Fixtures / Factories, Transport / Network Mock, Lifecycle / Cleanup).
2. **Smells & Flakiness Traps**:
   - Evaluate tests against anti-patterns and flakiness traps detailed in `references/test-smells.md`.
3. **Invariant Dimension Blind Spots**:
   - Audit test coverage against invariant dimensions detailed in `references/invariant-dimensions.md`.

### 2. Mandatory Reference Grounding

Before executing audit passes, agents MUST read and ground themselves in:
- `references/test-smells.md` for concrete smell definitions and remedies.
- `references/invariant-dimensions.md` for invariant testing vectors.

### 3. Review Verdicts & Citation Rule

- `PASS`: Harness anchor active; hermetic foundation verified; zero flakiness smells; balanced invariant dimension coverage.
- `WARN`: Infrastructure present but maintenance debt observed (duplicate setup, missing assertion messages).
- `CRITICAL`: Harness anchor absent, foundation missing (no virtual clock/ephemeral storage), or flakiness smells detected (clock leaks, sleeps).
- **Dual-Citation Rule**: Every defect must cite both the offending code pointer (`file:///path/to/test#L<N>`) and the violated invariant/remediation.

## Canonical Audit Report Format

```markdown
# Harness Audit Report: <Target Scope>

## 1. Executive Verdict
- **Verdict**: <PASS | WARN | CRITICAL>
- **Target Inspected**: <Directory or Test Files>
- **Harness Anchor (SSOT)**: <Cited anchor pointer or 'UNGROUNDED (Missing .agents/harness-anchor.md)'>
- **Flakiness Risk**: <LOW | MEDIUM | HIGH>
- **Summary**: <Single-sentence technical judgment>

## 2. Infrastructure & Anchor Gaps
*(If clean: 'Harness anchor verified; foundational hermetic primitives active.')*
- **[<Anchor / Primitive>] <Gap Title>**: <Deficit description>
  - Impact: <Why this causes flakiness or maintenance debt>
  - Recommendation: <Architecture fix or seed anchor pointer>

## 3. Findings & Invariant Blind Spots
- **[<Smell / Dimension>] <Title>**: <Defect description>
  - Offending Code: [<file>#L<N>](file:///path/to/test#L<N>)
  - Remediation: <Exact architectural correction required>

## 4. Remediation Changeset (Staged for /forge)
*(If defects or missing infrastructure are identified, stage a ready-to-run Changeset)*

# Changeset: Remediate Test Harness & Coverage for <Scope>

📁 <target_test_dir>/
├── 📄 <new_or_updated_helper>
│   └── [CREATE/UPDATE] <Fix, e.g., Controlled virtual clock provider or harness anchor>.
└── 📄 <remediated_test_file>
    └── [UPDATE] <Fix, e.g., Eliminate clock leaks and add D4 concurrency tests>.

## 5. Pipeline Routing
- `/zenforge-init` ── Harness anchor (.agents/harness-anchor.md) is missing; bootstrap project agent governance.
- `/forge` ── Authorize and execute the Remediation Changeset in a single unified turn.
- `/simplify` ── Test suite is verified and robust; proceed to safe code refactoring.
```

## Delegation & Synthesis Protocol

**SUB-SKILL:** zenforge-init, forge, simplify

1. **Grounding & Scope Intake**:
   - Inspect `.agents/harness-anchor.md` if present.
   - Read `references/test-smells.md` and `references/invariant-dimensions.md`.
2. **Scope Partitioning & Dispatch (Fan-out)**:
   - Launch `research` subagents partitioned by test directory or package.
   - Instruct subagents to evaluate target tests against the Audit Hierarchy and cite exact pointers.
3. **Synthesis & Turn-Halt Gate (Fan-in)**:
   - Deduplicate findings into the **Canonical Audit Report Format**.
   - If deficits exist, draft the Remediation Changeset directly in the report for `/forge`.
   - Present completed report into conversation stream and halt turn immediately. Never mutate files.
