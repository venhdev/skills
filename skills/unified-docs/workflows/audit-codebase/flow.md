# Audit Codebase Flow

## Load

- `contracts/frontmatter.md`
- `contracts/classification.md`
- `contracts/doctypes/how-to.md`
- `contracts/doctypes/explanation.md`
- `contracts/doctypes/til.md`
- `contracts/doctypes/plan.md`
- `contracts/doctypes/adr.md`
- `contracts/doctypes/spec.md`
- `contracts/cascade.md`
- `templates/reports/health-report.md`

---

## Instructions

Full corpus audit. Execute sequentially:

### 1. Organization check

Load and execute `workflows/audit-org/flow.md` in **read-only mode**:
- Inventory current doc structure
- Check red flags
- Recommend pattern
- Report as Warning severity

### 2. Naming check (doc scope only)

Load and execute `workflows/audit-naming/flow.md` with **fixed scope** (no asking):
- Scope = doc filenames + frontmatter `title`
- Apply anti-patterns and doc-file rules
- Report findings

### 3. Full frontmatter/lifecycle/cascade audit

Run all checks from `modes/audit.md` on entire docs corpus:
- Missing required frontmatter fields
- Invalid `type` or `kind` values
- Lifecycle state violations (plan → spec → archive, ADR supersession, etc.)
- Cascade relationship issues (depends-on, updates, inverse links)
- Current docs depending on archived/superseded docs
- Duplicate SSOT claims

### 4. Aggregate report

Combine all findings (org + naming + frontmatter/lifecycle/cascade) into **one health-report**:
- Group by severity: Critical, Warning, Info
- Include exact file paths and replacement links where known
- Provide recommendations for each category
