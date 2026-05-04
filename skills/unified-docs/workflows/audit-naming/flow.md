# Audit Naming Flow

## Load

- `workflows/audit-naming/naming-rules.md`
- `templates/reports/health-report.md`

---

## Scope Determination

### If triggered from `--audit-codebase`

Scope is fixed:
- **Doc filenames** (kebab-case, noun phrase, no generic/date/version/person names)
- **Frontmatter `title`** (descriptive, no tech stack/version/team encoding)
- **Skip asking** — proceed directly to checks

### If triggered from `--audit-naming` directly

- **User provided scope** (files/folder path attached): Use that scope
- **No scope specified**: STOP and ask user:

```
Where would you like to check naming?

(a) Doc filenames + titles (quick, less context)
(b) A specific folder (e.g. src/, docs/)
(c) Entire codebase (slower, needs confirmation)
```

Wait for user selection before proceeding.

---

## Checks

Apply rules from `naming-rules.md` according to scope:

- **Doc scope (a)**: Filenames + titles, anti-patterns table
- **Folder scope (b)**: Doc + code rules for that folder
- **Full codebase (c)**: All doc + code rules, with confirmation for performance

---

## Output

- **Severity**: Warning (affects findability) or Info (style suggestion). Never Critical.
- **Format**:
  ```
  Current: johns-notes.md
  Suggest: auth-notes.md
  Reason: Anti-pattern — person name encoding (see naming-rules.md)
  ```
- **User approval**: Ask user to approve each rename before applying. No auto-rename.
- **Project convention**: If consistent convention found, acknowledge it instead of flagging

---

## Mutation Policy

- **Suggest only** — report findings + recommendations
- **Ask user** before any renames
- **No execution** unless user explicitly approves each item
