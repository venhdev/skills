# Create Plan: Two-Tier Planning Workflow

## Overview

When creating a plan doc, users choose how deep to go:

1. **Tier 1 — Milestone Plan** (always): High-level "what and when" — goal, scope, milestones
2. **Tier 2 — Detailed Implementation** (optional): "How and with what gates" — per-phase checklists with architecture decisions, action steps, and acceptance criteria

This file documents the flow and when each tier is triggered.

---

## Tier 1: Milestone Plan

When the user asks to create a plan, unified-docs Create mode:
1. Runs discovery until the project's goal, scope, milestones, and timeline are clear
2. Creates a plan document using the milestone plan template (`implementation-plan-template.md`)
3. Plan includes goal, scope, milestones (with measurable outcomes), and optional gate conditions

---

## Tier 2: Post-creation Auto-Ask (Plan docs only)

After a milestone plan is created, offer the user an optional follow-up: a detailed implementation plan with per-phase action checklists and gate conditions.

**Trigger**: Only for plan docs, immediately after the authored milestone plan is emitted.

**Ask the user**:

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

**If user says yes**:
1. Check if `docs/plans/[project-short-name]/` exists. If so, ask user to choose a different folder name.
2. Create folder: `docs/plans/[folder-name]/`
3. Copy `implementation-plan-template.md` from this folder → `implementation-plan.md` in the project folder
4. Populate implementation-plan.md with: project name, goal, scope, milestones (from plan), gate conditions (if present)
5. Generate phase files (one per milestone): `phase-1-[short-name].md`, `phase-2-[short-name].md`, etc.
   - Copy from `phase-checklist-template.md` in this folder
   - Populate phase name and number
   - Leave `[TODO: Fill in after discovery]` for Architecture Decision, Action Checklist, Acceptance Criteria
6. Create `README.md` with:
   - Brief "How to use this folder" inline guide (do not link to external docs)
   - List of files: implementation-plan.md, phase-1-*.md, phase-2-*.md, etc.
   - Instructions: start with implementation-plan.md, fill in each phase file, verify all gate conditions before proceeding to next phase

**If user says no**:
- Done. User can create a detailed plan later by making a new Create request.

---

## Gate Conditions Pattern

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

## Phase File Structure

Each phase file (phase-1-*.md, phase-2-*.md, etc.) follows this structure:

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

## Folder Structure (When Detailed Plan Created)

```
docs/plans/[project-short-name]/
├─ implementation-plan.md        (overview, gate conditions, phase timeline)
├─ phase-1-[short-name].md       (one per milestone)
├─ phase-2-[short-name].md
├─ phase-N-[short-name].md
└─ README.md                      (how to use this folder)
```

---

## Key Principles

1. **Milestone first, details second** — High level before low level
2. **User chooses depth** — Not everyone needs detailed plans
3. **Gate conditions prevent drift** — Can't proceed without verification
4. **Folder per project** — All related plans together
5. **Verifiable gates** — Each criterion is checkable, not subjective
