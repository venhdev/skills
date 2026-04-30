---
name: Implementation Plan Template
description: Comprehensive plan with milestones, gate conditions, and phase-level checklists
type: reference
kind: [template]
---

# Implementation Plan: [Project Name]

## Overview

### Goal
[What are we building/fixing/improving?]

### Scope
- In scope: [What's included]
- Out of scope: [What's explicitly excluded]

### Success Criteria
[How do we know this project is complete and successful?]

---

## Gate Conditions (Entry/Exit Criteria)

These conditions determine readiness to proceed between phases. **Must verify ALL before moving forward.**

### Pre-Phase 1 (Project Start)
- [ ] [Precondition 1 — verifiable]
- [ ] [Precondition 2 — verifiable]
- [ ] [Precondition 3 — verifiable]

### After Phase 1 → Before Phase 2
- [ ] [Gate 1 — how to verify]
- [ ] [Gate 2 — how to verify]
- [ ] [Gate 3 — how to verify]

### After Phase 2 → Before Phase 3
- [ ] [Gate 1 — how to verify]
- [ ] [Gate 2 — how to verify]

### After Phase 3 → Before Phase 4
- [ ] [Gate 1 — how to verify]

### Project Complete (All Phases)
- [ ] [Final acceptance criteria 1]
- [ ] [Final acceptance criteria 2]
- [ ] [Final acceptance criteria 3]

---

## Phases

### Phase 1: [Milestone Name]
**Estimated duration**: [X hours/days]  
**Blocked by**: [Phase X, or "none"]  
**Blocks**: [Phase Y, or "none"]  

See: `phase-1-[short-name].md`

### Phase 2: [Milestone Name]
**Estimated duration**: [X hours/days]  
**Blocked by**: [Phase 1]  
**Blocks**: [Phase 3, or "none"]  

See: `phase-2-[short-name].md`

### Phase 3: [Milestone Name]
**Estimated duration**: [X hours/days]  
**Blocked by**: [Phase 2, or "none"]  
**Blocks**: [Phase 4, or "none"]  

See: `phase-3-[short-name].md`

### Phase 4: [Milestone Name]
**Estimated duration**: [X hours/days]  
**Blocked by**: [Phase 3]  
**Blocks**: [None]  

See: `phase-4-[short-name].md`

---

## Timeline

| Phase | Duration | Dependencies | Notes |
|-------|----------|--------------|-------|
| Phase 1 | [Est. time] | None | Can start immediately |
| Phase 2 | [Est. time] | Phase 1 complete | Blocked by gate conditions |
| Phase 3 | [Est. time] | Phase 2 complete | Blocked by gate conditions |
| Phase 4 | [Est. time] | Phase 3 complete | Blocked by gate conditions |
| **Total** | **[Sum]** | — | — |

---

## Risks & Fallbacks

### Risk 1: [Description]
**Impact**: [What breaks if this happens]  
**Likelihood**: [High/Medium/Low]  
**Fallback**: [Plan B if this happens]  

### Risk 2: [Description]
**Impact**: [What breaks if this happens]  
**Likelihood**: [High/Medium/Low]  
**Fallback**: [Plan B if this happens]  

---

## Navigation

```
docs/plans/[project-short-name]/
├─ implementation-plan.md (this file)
├─ phase-1-[short-name].md
├─ phase-2-[short-name].md
├─ phase-3-[short-name].md
└─ phase-4-[short-name].md
```

Start with Phase 1 checklist. Do not proceed to Phase 2 until all Phase 1 gate conditions are ✓.

---

**Created**: [Date]  
**Last updated**: [Date]  
**Status**: [Planning / In Progress / Complete]
