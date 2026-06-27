---
name: agents-validator
description: Use when validating CLAUDE.md and AGENTS.md hierarchies, checking for missing files, broken @AGENTS.md references, or ensuring content meets minimum length standards before using agent teams.
compatibility: Requires bash, find, grep, wc (standard Unix tools). Optional .gitignore for smart filtering.
---

# Agents Validator

Validates CLAUDE.md/AGENTS.md hierarchies for agent team workflows.

## Overview

Audits your project structure to ensure CLAUDE.md and AGENTS.md files are present, properly formatted, correctly referenced, and meet minimum content standards across hierarchy levels.

**Core checks:**
- File existence and location
- Valid Markdown syntax
- @AGENTS.md reference correctness (same-directory only)
- Content meets minimum word count per level
- No orphaned or disconnected files

## When to Use

✅ **Before starting agent teams** — verify structure is ready
✅ **Adding new modules** — ensure consistency
✅ **Onboarding teammates** — validate standards compliance
✅ **After restructuring** — check for consistency issues

## When NOT to Use

❌ Modifying content (use code-simplifier or writing-skills instead)
❌ Creating initial project (use `/init` command first)
❌ Auto-fixing structure (this skill warns + suggests, never modifies)

## Workflow

### Step 1: Configure Audit

Skill asks three questions:

| Question | Purpose | Options |
|----------|---------|---------|
| **Project path** | Which project to audit | Manual input (--path) or current directory |
| **Scan depth** | How deep to scan | Root only / Root+modules / Full hierarchical (buttons) |
| **Ignore patterns** | What to exclude | Respect .gitignore / Use defaults / Custom / None |

### Step 2: Run Audit

Scans project structure based on configuration.

### Minimum Content Standards

**Root Level:**
- CLAUDE.md: ✅ Required | 30-50 words | Must reference @AGENTS.md
- AGENTS.md: ✅ Required | 200-300 words | Source of truth for standards

**Per-Module Level:**
- CLAUDE.md: ✅ Required | 20-30 words | Must reference @AGENTS.md
- AGENTS.md: ✅ Required | 80-150 words | Module-specific patterns

**All files must:**
- Have valid Markdown syntax
- Start with `#` heading
- Reference @AGENTS.md (CLAUDE.md only, same directory)
- Not be empty or placeholder-only

### File Validation

| File | Check | Pass | Fail |
|------|-------|------|------|
| CLAUDE.md | Has `#` heading | ✅ | ❌ No heading |
| CLAUDE.md | Contains @AGENTS.md | ✅ On own line only | ❌ Missing or embedded in text |
| CLAUDE.md | Min word count (excl. @AGENTS.md line) | ✅ Met | ❌ Too short |
| AGENTS.md | Has content | ✅ Substantive | ❌ Placeholder only |
| AGENTS.md | Min word count | ✅ Met | ❌ Too short |
| Any file | Valid Markdown | ✅ Parses OK | ❌ Syntax error |

### Reference Syntax - STRICT RULE

**VALID (only this format):**
```
✅ @AGENTS.md            (on its own line, nothing else)
```

**INVALID (all of these):**
```
❌ Follow @AGENTS.md              (embedded in text)
❌ See @AGENTS.md for rules       (prose context)
❌ @AGENTS.md for details         (not standalone)
❌ @../AGENTS.md                  (relative paths)
❌ @root/AGENTS.md                (absolute paths)
❌ ../AGENTS.md                   (missing @ prefix)
```

**Rule:** @AGENTS.md must appear ALONE on its line. No surrounding text. This is an **import directive**, not a reference in prose.

**Correct minimal CLAUDE.md:**
```markdown
# Module Name

@AGENTS.md
```

**Incorrect:**
```markdown
# Module Name

For patterns, see @AGENTS.md for details.    ❌ (text around it)
```

## Output Format

Reports use standard symbols:

```
✅ PASS     — File/check passed
⚠️  WARN    — Fixable issue (suggestions provided)
❌ FAIL    — Blocking issue (must fix)
```

**Example output:**
```
AUDIT RESULT
Path: /home/user/project | Depth: Full | Ignore: defaults

ROOT
 ✅ CLAUDE.md (54 words, has @AGENTS.md)
 ✅ AGENTS.md (280 words)

src/modules/rides/
 ✅ CLAUDE.md (26 words)
 ⚠️  AGENTS.md missing
    → Suggestion: Create file (min 80 words)

src/modules/users/
 ❌ CLAUDE.md (15 words, min 20 needed)
    → Action: Add 5+ words

SUMMARY: 1 fail, 1 warn, 2 pass. Fix before using agent teams.
```

## Interpreting Results

**✅ PASS** — No action needed

**⚠️  WARN** — Fix recommended:
- Content is thin but present
- File exists in unexpected location
- Reference syntax is non-standard
- Suggestions provided

**❌ FAIL** — Must fix before agent teams:
- Critical file missing
- Invalid Markdown syntax
- Broken references
- Content below minimum

---

## Common Mistakes

❌ **"I'll write 'Follow @AGENTS.md' for clarity"**
→ Wrong. @AGENTS.md is import syntax, not prose. Must be alone on its line:
```markdown
@AGENTS.md        ✅ correct
Follow @AGENTS.md ❌ wrong
```

❌ **"CLAUDE.md should contain everything"**
→ No. CLAUDE.md is thin reference to AGENTS.md. Put substance in AGENTS.md.

❌ **"I'll skip per-module AGENTS.md if modules are simple"**
→ Do this anyway. Consistency matters for agent team context.

❌ **"Using relative paths like @../AGENTS.md"**
→ Only same-directory `@AGENTS.md` supported. Must be on own line.

❌ **"AGENTS.md with only 50 words is enough"**
→ Minimum standards exist (200 root, 80 module). Meet them or agent confusion increases.

❌ **"Fixing issues without re-running audit"**
→ Always re-run after fixes to verify results.

## Quick Start

```bash
# Audit current directory, interactive prompts
claude

# Audit specific path
claude --path /home/user/project

# Audit and check it worked
Run audit again in 30 days as part of onboarding checklist
```

---

## Testing Your Audit

Verify the audit correctly detected issues:

1. **Intentionally break something:**
   - Remove @AGENTS.md from CLAUDE.md
   - Delete a file
   - Rename a file
   - Shorten AGENTS.md below minimum

2. **Run audit** — Should catch each break

3. **Fix intentional breaks** — Re-run, should pass

If audit misses any issue, it's a bug → report it.

---

## Tips for Success

| Problem | Solution |
|---------|----------|
| Unsure what @AGENTS.md should contain | Start with tech stack, naming conventions, testing patterns |
| Per-module AGENTS.md feels repetitive | It's OK. Be specific to each module's gotchas |
| Structure keeps breaking after months | Re-run audit quarterly as standard practice |