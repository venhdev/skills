# Maintain Plan Flow

## Load

- `contracts/frontmatter.md`
- `contracts/doctypes/plan.md`
- `templates/reports/mutation-report.md`

---

## Trigger

- **Only triggered by** `--maintain-plan` arg
- **Not triggered by** natural language requests like "update plan status" (those use Layer 2: modes/maintain.md + contracts/doctypes/plan.md)

---

## Instructions

### Scan Mode (no specific plan)

When user runs `--maintain-plan` without a plan name:

1. **Find all plans** with status `draft` or `in-progress`
   - Exclude: `completed` and `archived`
   - List files found

2. **For each plan**, report:
   - Current status
   - Milestones (if present)
   - Progress indicators (if documented)
   - Last updated date
   - replacedBy target (if status is completed/superseded/archived)

3. **Offer updates**:
   - Update status (draft → in-progress, in-progress → completed)
   - Add/update milestones
   - Update progress notes
   - Do NOT mutate without explicit user confirmation

4. **Constraints**:
   - Cannot archive or mark completed without explicit user confirmation
   - Cannot mark completed without `replacedBy` pointing to a resolvable doc
   - If user wants to complete: guide through lifecycle rules in contracts/doctypes/plan.md

### Completion → Spec Promotion

When a plan reaches `status: completed`, extract durable outcomes into a spec:

1. Identify durable decisions/outcomes from the plan body
2. Create a new doc with `type: reference` and `kind: [spec]` (or update an existing spec)
3. Set the new spec to `status: draft`, offer to accept it after review
4. Set `replacedBy` on the plan to point to the new spec
5. Do NOT add the plan to the spec's `updates` (plan is temporary, spec is durable)

### Superseded Transition

When a plan is abandoned mid-flight:

1. Set `status: superseded`
2. Record `replacedBy` pointing to the chosen alternative doc (spec, TIL, explanation, etc.)
3. Add a note in the plan body explaining why it was abandoned and what replaced it
4. Do NOT promote remaining incomplete work — superseded means never implement the rest

### Single Plan Mode

When user provides plan name with `--maintain-plan` (e.g., `--maintain-plan project-roadmap.md`):

1. **Read that plan**
   - Verify it exists and is a plan doc
   - Report current state

2. **Report state**:
   - Current status
   - Milestones
   - Dependencies
   - replacedBy target
   - Last updated

3. **Offer update options** according to lifecycle rules:
   - If draft: transition to in-progress
   - If in-progress: update progress, or transition to completed (with conditions), or mark superseded
   - If completed: show path to archive (must have replacedBy)
   - If superseded: note the replacedBy target
   - If archived: offer un-archive or replace with new plan

4. **Apply updates** only with explicit user confirmation
