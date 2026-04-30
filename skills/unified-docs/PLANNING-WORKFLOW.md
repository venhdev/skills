# Planning Workflow for unified-docs Skill

## Overview

This document describes the two-tier planning workflow:
1. **Milestone Plan** — High-level "what and when"
2. **Detailed Implementation Plan** — "How and with what acceptance criteria"

---

## Workflow Process

### Tier 1: Create Milestone Plan

When user asks to create a plan for a project:

1. Brainstorming skill enters Plan mode
2. Creates a milestone-focused plan document
3. Example: `INTAKE-PLAN-V2.md` with phases/milestones
4. Plan includes:
   - Project goal and scope
   - List of milestones (Phase 1, Phase 2, etc.)
   - High-level success criteria
   - Timeline estimate

### Tier 2: Auto-Ask for Detailed Plan

**After milestone plan is complete, skill automatically asks**:

```
Plan created. Would you like me to create a detailed implementation plan too?

Detailed plan includes:
- Architecture decisions per phase
- Action checklists (step-by-step)
- Acceptance criteria and gate conditions
- Dependencies between phases
- Risk mitigation strategies

This helps prevent scope creep and ensures clear completion criteria
before moving between phases.
```

### Tier 3: Create Detailed Implementation Plan (If User Agrees)

If user says **yes**:

1. Create a new folder: `docs/plans/[project-short-name]/`
2. Create `implementation-plan.md` (overview with gate conditions)
3. Create `phase-1-[short-name].md`, `phase-2-...md`, etc.
4. Each phase file contains:
   - Architecture Decision (what/why/impact)
   - Action Checklist (concrete HOW steps)
   - Acceptance Criteria (gate conditions)
   - Dependencies and risks

If user says **no**:
- Stop. Milestone plan is sufficient.
- User can create detailed plan later if needed.

---

## Folder Structure

When detailed plan is created:

```
docs/plans/[project-short-name]/
├─ implementation-plan.md
│  (Overview, gate conditions, phase timeline)
│
├─ phase-1-[short-name].md
│  (Architecture Decision + Action Checklist + Acceptance Criteria)
│
├─ phase-2-[short-name].md
│  (Architecture Decision + Action Checklist + Acceptance Criteria)
│
├─ phase-3-[short-name].md
│  (Architecture Decision + Action Checklist + Acceptance Criteria)
│
└─ phase-4-[short-name].md
│  (Architecture Decision + Action Checklist + Acceptance Criteria)
```

Example:
```
docs/plans/enhance-unified-docs/
├─ implementation-plan.md
├─ phase-1-severity-fix.md
├─ phase-2-fixture-rebuild.md
├─ phase-3-evaluate.md
└─ phase-4-create-discovery-gate.md
```

---

## Gate Conditions (Key Pattern)

**Gate conditions are the prevention mechanism.**

Each phase has **acceptance criteria** that must be met before proceeding to the next phase.

Example gate flow:

```
Phase 1 Complete
    ↓
[GATE CONDITIONS CHECK]
  - Is severity fix actually working? (EV1 re-run shows D7 improved)
  - Are all findings still found? (18/18 issues)
  - No regressions? (Overall score improved)
    ↓
IF all gates pass: → Proceed to Phase 2
IF gate fails: → Phase 1 not done, must fix before proceeding
```

**This prevents "sunk cost" situation** where bad work continues because we're already invested.

---

## Phase Checklist Structure

Each phase file follows this structure:

### 1. Architecture Decision
- **What**: Describe what changes
- **Why**: Explain context and constraints
- **Impact**: Show downstream effects

### 2. Action Checklist
- Step 1: Concrete HOW with specific file/location
- Step 2: Concrete HOW with specific file/location
- Step 3: Concrete HOW with specific file/location
- Each step has verification point

### 3. Acceptance Criteria (Gate Conditions)
- AC-1: [Verifiable]
- AC-2: [Verifiable]
- AC-3: [Verifiable]

**All must be ✓ before phase is done.**

### 4. Dependencies & Risks
- What blocks this phase
- What this phase blocks
- Risks and fallback plans

---

## Using the Templates

### For Milestone Plan
Use: `templates/planning/implementation-plan-template.md`

Copy structure and fill in:
- Goal and scope
- Phase names and durations
- Gate conditions between phases
- Timeline and risks

### For Phase Checklists
Use: `templates/planning/phase-checklist-template.md`

For each phase:
- Fill in architecture decision (what/why/impact)
- List concrete action steps with locations
- Define acceptance criteria (gates)
- Document dependencies and risks

---

## Key Principles

1. **Milestone first, details second** — High level before low level
2. **User chooses depth** — Not everyone needs detailed plans
3. **Gate conditions prevent drift** — Can't proceed without verification
4. **Folder per project** — All related plans together
5. **No "who"** — Plans focus on HOW and WHEN, not WHO (context-agnostic)
6. **Verifiable gates** — Each criterion is checkable, not subjective

---

## Example: Severity Fix Phase

**Milestone plan says**:
> Phase 1: Fix severity classification in Audit mode

**Detailed plan (phase-1-severity-fix.md) says**:

**Architecture Decision**:
- What: Change modes/audit.md lines 57-59 from ambiguous guidance to two-class model
- Why: D7 dimension scored 4/10 due to inversion; Critical vs Warning unclear
- Impact: EV1 re-run will score D7 9-10/10; overall 91 → 98/100

**Action Checklist**:
- [ ] Read current modes/audit.md L57-59
- [ ] Draft new severity guidance (3 sections: Critical/Warning/Info)
- [ ] Add 5+ examples per section
- [ ] Verify contract alignment (frontmatter.md, lifecycle.md)
- [ ] Commit: "Fix severity classification in Audit mode"
- [ ] Run EV1 re-run

**Acceptance Criteria**:
- [ ] modes/audit.md has clear two-class model with examples
- [ ] EV1 re-run D7 score ≥ 9/10 (was 4/10)
- [ ] EV1 overall score ≥ 95/100 (was 91/100)
- [ ] All 18 test issues still found (no regression)

**Gate before Phase 2**:
If D7 not improved or overall score < 95, phase 1 NOT DONE. Do not proceed to phase 2.

---

## When to Use This Workflow

✅ **Use detailed planning when**:
- Project has 3+ distinct phases
- Each phase has dependencies on previous
- Multiple people involved (or future you needs to understand)
- Complex decisions need documentation
- Want clear "done" criteria before starting

❌ **Skip detailed planning when**:
- Single simple task
- No dependencies between steps
- Quick exploratory work
- "Spike" or throwaway code

---

**Version**: 1.0  
**Part of**: unified-docs skill  
**Related**: brainstorming skill (Plan mode)
