---
name: Phase Checklist Template
description: Per-phase detailed actions, architecture decisions, and acceptance criteria
type: reference
kind: [template]
---

# Phase X: [Phase Name]

## Architecture Decision

### What (Changes)
[Describe what will change as a result of this phase]
- [Change 1]
- [Change 2]
- [Change 3]

### Why (Context & Constraints)
[Why these changes matter, what problem they solve, what constraints apply]

### Impact (Side Effects)
[What else changes as a result? What becomes dependent on this?]
- [Downstream change 1]
- [Downstream change 2]

---

## Action Checklist

**Complete in order. Do not skip steps.**

### Step 1: [Concrete Action]
- [ ] Specific task 1 — [Location: file/line or path]
- [ ] Specific task 2 — [Location: file/path]
- [ ] Verify: [How to check this step worked]

### Step 2: [Concrete Action]
- [ ] Specific task 1 — [Location]
- [ ] Specific task 2 — [Location]
- [ ] Verify: [How to check]

### Step 3: [Concrete Action]
- [ ] Specific task 1 — [Location]
- [ ] Specific task 2 — [Location]
- [ ] Verify: [How to check]

---

## Acceptance Criteria (Gate Conditions)

**All criteria MUST be met before Phase X is considered complete.**

- [ ] **AC-1**: [Verifiable condition 1] — Verify by: [How to check]
- [ ] **AC-2**: [Verifiable condition 2] — Verify by: [How to check]
- [ ] **AC-3**: [Verifiable condition 3] — Verify by: [How to check]

If any criterion fails, phase is **not complete**. Do not proceed to next phase.

---

## Dependencies

### Blocked by
- [Phase Y must complete first]
- [External condition must be true]

### Blocks
- [Phase Z cannot start until this phase completes]

---

## Risks & Fallbacks

### Risk 1: [What could go wrong]
**Detection**: How you'll notice this failed  
**Fallback**: [Plan B — how to proceed if this happens]  

### Risk 2: [What could go wrong]
**Detection**: How you'll notice this failed  
**Fallback**: [Plan B — how to proceed if this happens]  

---

## Completion Checklist

When done with all action steps, verify:

- [ ] All action items marked ✓
- [ ] All acceptance criteria verified ✓
- [ ] All gate conditions from `implementation-plan.md` met ✓
- [ ] Changes committed to git with clear message
- [ ] No blockers for next phase

**Phase X is DONE when all boxes above are checked.**

---

**Phase**: X of [Total phases]  
**Estimated duration**: [X hours]  
**Actual duration**: [Fill in when complete]  
**Status**: [Not started / In progress / Complete / Blocked]
