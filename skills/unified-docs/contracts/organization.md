# Organization Contract

This contract describes folder structure patterns for documentation and when to consult them.

## When to load this contract

Load **only** when one of these triggers is active:

- User explicitly requests reorganization ("reorganize docs", "fix doc structure", "restructure folders")
- Audit or Maintain mode detects structural issues: no clear folder hierarchy, docs mixed at root, no README at major sections
- Create mode: no existing docs folder or convention to follow (from-scratch project)

This contract is **lazy-loaded** and not consulted during routine docs creation, maintenance, or audits with clear structure.

---

## 4 Structure Patterns

### Pattern 1: By Documentation Type

**Structure:**
```
docs/
├── getting-started/       ← Onboarding & setup
├── guides/                ← How-to tutorials
├── architecture/          ← Design & decisions
├── reference/             ← Lookup (API, specs)
├── concepts/              ← Theory & explanations
└── troubleshooting/       ← Common issues
```

**Best for:**
- Small-to-medium teams (1–10 people)
- Users who search by **task** ("How do I login?")
- Educational focus

**Tradeoffs:**
- ✅ Users find docs easily by what they need
- ✅ Clear learning progression
- ✅ Easy to build navigation trees
- ❌ Related docs scattered across folders
- ❌ Technical dependencies hard to see

---

### Pattern 2: By Technical Layer

**Structure:**
```
docs/
├── core/              ← Foundation & overview
├── state/             ← State, data, storage
├── api/               ← Server integration
├── ui/                ← Components, styling
├── features/          ← Feature-specific docs
├── testing/           ← QA & testing
└── deployment/        ← Build & release
```

**Best for:**
- Engineers thinking in **layers** (data → UI → deploy)
- Technical decision-making focused
- Backend-heavy documentation

**Tradeoffs:**
- ✅ Clear technical separation
- ✅ Good for architects/tech leads
- ✅ Easy to trace dependencies
- ❌ Hard for new developers to onboard
- ❌ Scattered guides for one feature

---

### Pattern 3: Hybrid

**Structure:**
```
docs/
├── getting-started/       ← Onboarding entry point
│   ├── setup.md
│   ├── first-feature.md
│   └── checklist.md
│
├── architecture/          ← Why designed this way
│   ├── overview.md
│   ├── decisions/
│   └── diagrams/
│
├── guides/                ← How-to by task
│   ├── routing/
│   ├── state-management/
│   └── ...
│
├── reference/             ← Specific lookup
│   ├── api-reference.md
│   ├── troubleshooting.md
│   └── faq.md
│
└── concepts/              ← Deep dives (optional)
```

**Best for:**
- Growing teams (10–50+ people)
- Mixed audiences (new devs + architects)
- Long-term products
- Need **both** onboarding **and** technical depth

**Tradeoffs:**
- ✅ Best UX for all audiences
- ✅ Clear onboarding path
- ✅ Technical clarity for architects
- ✅ Scales well with team growth
- ⚠️ Some redundancy (linked from multiple places)
- ⚠️ More structure to maintain

---

### Pattern 4: Scalable/Enterprise

**Structure:**
```
docs/
├── 00-getting-started/
├── 01-architecture/
├── 02-development/
├── 03-testing/
├── 04-deployment/
└── 05-appendix/
```

**Best for:**
- Large teams (50+ people)
- Complex products with many features
- Need strict governance & structure

**Tradeoffs:**
- ✅ Maximum organization
- ✅ Scales to 100+ docs
- ✅ Numbered folders ensure sort order
- ❌ Can feel bureaucratic
- ❌ Overkill for small teams

---

## Decision Matrix

| Criterion | Pattern 1 | Pattern 2 | Pattern 3 | Pattern 4 |
|---|---|---|---|---|
| **Team size** | 1–10 | 5–30 | 10–50+ | 50+ |
| **Docs count** | < 30 | 20–60 | 30–100 | 100+ |
| **Onboarding focus** | ⭐⭐⭐ | ⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| **For growing teams** | Outgrows quickly | Limited growth | Yes | Overkill |

---

## Red Flags (Reorganization Signals)

Report these when found during audit or maintain operations. They signal structural clarity is declining:

- **Related docs scattered** — Docs about same topic across unrelated folders (e.g., auth in both `security/` and `guides/`)
- **Team can't find placement** — Developers unsure where to add new docs; convention unclear
- **Generic filenames** — Files named `misc.md`, `other.md`, `temp.md`, `random.md`
- **Person/team organization** — Folders named after people: `docs/alice/`, `docs/backend-team/`
- **Alphabetical order** — Only organizing principle is A–Z; no semantic meaning
- **No entry point** — No README or index; new contributors don't know where to start
- **Excessive depth** — Folders nested 5+ levels deep
- **AI agent cost signal** — LLM-based tools require too many reads to understand the structure and find the right place for changes

---

## Anti-Patterns (Avoid)

- ❌ Docs organized by person or team name
- ❌ Alphabetical organization as the only principle
- ❌ Generic names (`misc.md`, `other.md`)
- ❌ Duplicated content in multiple files (link instead)
- ❌ No clear entry point
- ❌ Deeply nested folders (aim for max 3–4 levels)

---

## Best Practices (All Patterns)

### File Organization

- ✅ One concept per file
- ✅ Kebab-case filenames: `routing.md`, `forms-validation.md`
- ✅ Subfolders for related docs: `guides/routing/`, `architecture/decisions/`
- ✅ Avoid deeply nested folders (max 3–4 levels)

### Navigation

- ✅ One README/index per major section
- ✅ Breadcrumbs or navigation aids in docs
- ✅ "Related docs" section at end of each doc
- ✅ Clear progression (foundations → advanced)

### Linking

- ✅ Relative links within docs tree: `../guides/routing/routing.md`
- ✅ Descriptive link text (avoid "click here")
- ✅ Update links when reorganizing
- ✅ Test for broken links before and after reorganization

---

## Migration Checklist

When a user confirms a restructuring decision:

1. **Plan target structure** — Decide on one of the 4 patterns and sketch final folder layout
2. **Map each doc** — List where each current doc moves in the new structure
3. **Update frontmatter** — Correct `depends-on` and `updates` links before moving files
4. **Move files** — Reorganize folders and files to match target structure
5. **Check broken links** — Search for and fix any relative paths that now point to wrong locations
6. **Create README files** — Add index/README at each major section
7. **Update cross-references** — Search codebase for hardcoded doc references; update if needed

---

## When Not to Reorganize

- Small projects (< 30 docs) with clear structure already in place
- Routine doc updates that do not require folder changes
- Audit findings about individual doc quality (frontmatter, content) — those are separate from organization structure

Use Audit and Maintain modes for those cases; they do not load this contract.
