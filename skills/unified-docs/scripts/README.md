# Scripts

Utility scripts for the unified-docs skill. Each script is structure-independent — it works on any docs directory.

## cascade_targets.py

Maps a document's cascade graph: who it depends on, who it updates, and who depends on it or updates it.

**Usage:**
```bash
python scripts/cascade_targets.py <file> [root]
```

**Arguments:**
- `file`: Path to the target Markdown file (absolute or relative to cwd)
- `root`: Root directory to scan for incoming references. Defaults to cwd.

**Output:** JSON with metadata, outgoing (depends-on, updates), and incoming references.

**Example:**
```bash
python scripts/cascade_targets.py docs/specs/api-contract-spec.md
# Scans from cwd, finds all docs that depend on or update api-contract-spec.md

python scripts/cascade_targets.py docs/specs/api-contract-spec.md /path/to/project
# Scans from /path/to/project instead
```

**Fallback:** If Python unavailable, see "When scripts are unavailable" in `contracts/cascade.md`.

---

## check_frontmatter.py

Validates a single Markdown file's frontmatter against the unified-docs schema.

**Usage:**
```bash
python scripts/check_frontmatter.py <file> [root]
```

**Arguments:**
- `file`: Path to the Markdown file to validate
- `root`: Project root used to resolve `replacedBy` targets. Optional.

**Exit codes:**
- `0`: Validation passed (`OK: <path>`)
- `1`: Validation failed (error details on stderr)

**Example:**
```bash
python scripts/check_frontmatter.py docs/specs/api-contract-spec.md
# Validates required fields, types, kinds, and ADR/plan/spec coupling rules

python scripts/check_frontmatter.py docs/plans/migration-v2-plan.md /path/to/project
# Also validates that replacedBy target resolves to an accepted spec
```

**Fallback:** If Python unavailable, apply schema rules from `contracts/frontmatter.md` directly. See `modes/maintain.md` step 9 for inline procedure.

---

## Independence & Fallback

Both scripts are designed to work correctly regardless of project structure:

- **cascade_targets.py** now defaults to scanning from `cwd`, not the target file's parent folder
- **check_frontmatter.py** validates against schema, not file location
- Both explicitly exclude tool/vendor directories (`.claude/`, `.github/`, `node_modules/`, etc.)

If either script cannot run:
1. Check the relevant contract file for the manual fallback procedure
2. Apply the same logic by hand (read frontmatter, search docs, validate fields)
3. Ask the user if you need additional tools

The fallback procedures are equivalent to the script output and ensure the skill is always usable.
