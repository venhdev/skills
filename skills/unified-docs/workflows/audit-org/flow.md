# Audit Organization Flow

## Load

- `workflows/audit-org/organization-patterns.md`
- `templates/reports/health-report.md`

---

## Instructions (read-only — suggestion only)

When `--audit-org` arg is active OR reorganization is triggered from Maintain mode:

1. **Inventory current doc structure**
   - List all folders, subfolders, and files (depth ≤ 4)
   - Note any patterns: naming, grouping, hierarchy

2. **Check red flags** from `organization-patterns.md`
   - Related docs scattered across folders
   - Generic filenames (misc, temp, other)
   - Person/team-based organization
   - No entry point / README
   - Excessive depth (5+ levels)
   - Team can't find placement

3. **Recommend pattern** from decision matrix based on:
   - Team size
   - Doc count
   - Audiences (new devs vs architects)

4. **Report findings**
   - Severity: Warning (organization clarity declining)
   - Do NOT move files
   - Do NOT mutate structure
   - Frame recommendations as "Consider Pattern X because Y"

---

## Reorganization mode (from Maintain)

When user says "reorganize docs" in Maintain mode context:

1. Follow steps 1–3 above
2. **Present target structure** — Show what folder hierarchy would look like
3. **Ask for confirmation** before proceeding
4. **If confirmed**: Follow Migration Checklist in `organization-patterns.md`
5. **Report all mutations** — List every file moved, renamed, or created
