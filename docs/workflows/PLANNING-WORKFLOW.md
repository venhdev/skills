# Planning Workflow for unified-docs Skill

## Overview

This document describes the two-tier planning workflow:
1. **Milestone Plan** — High-level "what and when"
2. **Detailed Implementation Plan** — "How and with what acceptance criteria"

---

## Workflow Process

### Tier 1: Create Milestone Plan

When the user asks to create a plan, unified-docs Create mode:
1. Runs discovery until the project's goal, scope, milestones, and timeline are clear
2. Creates a plan document using `templates/authoring/plan.md`
3. Plan includes goal, scope, milestones (with measurable outcomes), and optional gate conditions

### Tier 2: Auto-Ask for Detailed Plan

**After milestone plan is created, the skill asks**:

```
Would you like me to create a detailed implementation plan too?

Detailed plan includes:
- Folder: docs/plans/[project-short-name]/
- implementation-plan.md (overview, gate conditions, phase timeline)
- phase-1-[short-name].md, phase-2-[short-name].md, etc.
- Each phase has: Architecture Decision, Action Checklist, Acceptance Criteria, Dependencies, Risks

This prevents scope creep and ensures verifiable done criteria
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

When detailed plan is created (one phase file per milestone):

```
docs/plans/[project-short-name]/
├─ implementation-plan.md        (overview, gate conditions, phase timeline)
├─ phase-1-[short-name].md
├─ phase-2-[short-name].md
└─ phase-N-[short-name].md       (one file per milestone)
```

Example (4-phase project):
```
docs/plans/enhance-unified-docs/
├─ implementation-plan.md
├─ phase-1-severity-fix.md
├─ phase-2-fixture-rebuild.md
└─ phase-4-create-discovery-gate.md
```

---

## Gate Conditions

Each phase has acceptance criteria that must be verified before proceeding:

```
Phase N Complete
    ↓
[GATE CONDITIONS CHECK]
  - AC-1 verified?
  - AC-2 verified?
  - AC-3 verified?
    ↓
All pass → Proceed to Phase N+1
Any fail → Phase N incomplete. Fix before proceeding.
```

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

## Key Principles

1. **Milestone first, details second** — High level before low level
2. **User chooses depth** — Not everyone needs detailed plans
3. **Gate conditions prevent drift** — Can't proceed without verification
4. **Folder per project** — All related plans together
5. **No "who"** — Plans focus on HOW and WHEN, not WHO (context-agnostic)
6. **Verifiable gates** — Each criterion is checkable, not subjective

---

**Part of**: unified-docs skill  
**Templates**: `templates/planning/implementation-plan-template.md`, `templates/planning/phase-checklist-template.md`
