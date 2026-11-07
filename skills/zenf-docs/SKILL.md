---
name: zenf-docs
description: "Register, classify, refresh, and search file-canonical SSOT documents in the ZenForge registry, keeping identity and metadata separate from Git content."
disable-model-invocation: true
---

# zenf-docs — SSOT Document Front Door

Use this card when the caller wants to register a long-lived document, promote a
document to authority, resolve duplicate sources, or search the registry. The
file is canonical; the registry stores identity, classification, tags, hash, and
a searchable snapshot.

## Prerequisites

- Read `docs/README.md` to find the existing authority for the topic.
- Decide whether the document is **global** (cross-repository rule/reference) or
  **project-scoped**.
- Choose a write surface:
  - **MCP** (preferred when connected): `docs` and `search`.
  - **HTTP** (MCP unavailable): `/api/v1/docs` endpoints.
  - **Files only** (daemon unreachable): update the Markdown and the
    `docs/README.md` placement matrix; report that registration is pending.
- Media (images, PDF, DOCX, XML) is recorded by absolute path only; never inline
  binary content.

## Step Protocol

1. **Search before writing.** Run `docs(action=search, query=...)` or
   `GET /api/v1/docs/search?query=`. If an authority exists, amend it instead of
   creating a rival document.
2. **Classify.**
   - `type`: `ARCHITECTURE | ADR | SPEC | RULE | GUIDE | REFERENCE | MEMORY | OTHER`
   - `scope`: `GLOBAL | PROJECT | PLAN | TASK`
   - `scope_ref`: plan slug or task id when scope is `PLAN`/`TASK`
   - `status`: `DRAFT | ACTIVE | SUPERSEDED | ARCHIVED`
3. **Choose identity.**
   - `key` is the short stable handle (`ARCH`, `ADR-004`, `MCP`).
   - `slug` is the human-readable unique name; it defaults to `key`.
   - `project_slug` empty means global. Keys and slugs are unique per project.
4. **Confirm the path.** The path must point at the canonical file (absolute or
   repository-relative). Verify it exists and is readable before registering.
5. **Register.**
   - MCP: `docs(action=create, key=..., type=..., scope=..., title=..., path=..., tags=[...])`.
   - HTTP: `POST /api/v1/docs` with the same fields.
   - Omit `content` to let the daemon read the file; supply it only when the file
     is intentionally unavailable. Files above 4 MiB register without a snapshot.
6. **Refresh after edits.** `docs(action=refresh, id=...)` or
   `POST /api/v1/docs/{id}/refresh` recomputes hash, size, version, and FTS.
7. **Retire cleanly.** Set `status=SUPERSEDED` and link the replacement before
   archiving. Archive hides a record without deleting the file. Delete removes
   the registry row only; the file stays in Git.

## Metadata Rubric

| Field | Rule |
| :--- | :--- |
| `title` | One sentence, no slug duplication |
| `summary` | One or two lines used by search results |
| `tags` | Lowercase, stable vocabulary (`architecture`, `security`, `mcp`) |
| `path` | Canonical file only; no scratch or generated build output |
| `content` | Snapshot for FTS; never the authority |
| `status` | `ACTIVE` only after the content is reviewed |

## MCP-Optional Matrix

| Need | MCP | HTTP fallback |
| :--- | :--- | :--- |
| Register | `docs(action=create)` | `POST /api/v1/docs` |
| Read | `docs(action=get, id/key/slug)` | `GET /api/v1/docs/{id}` |
| List/filter | `docs(action=list)` | `GET /api/v1/docs?project_slug=&type=&scope=&status=&tag=` |
| Update metadata | `docs(action=update)` | `PATCH /api/v1/docs/{id}` |
| Refresh snapshot | `docs(action=refresh)` | `POST /api/v1/docs/{id}/refresh` |
| Search | `docs(action=search)` | `GET /api/v1/docs/search?query=` |
| Cross-entity search | `search(query=...)` | `GET /api/v1/search?q=` |

## Guardrails

- **One authority per topic.** A new file must supersede or narrow an existing
  one; never create a silent parallel source.
- **Do not copy long content into chat.** Reference the path and key.
- **Do not register generated or transient files** (build output, scratch,
  `.agents/scratch/`) as SSOT.
- **Do not edit the snapshot to fix content.** Edit the file, then refresh.
- **Global documents must be genuinely cross-project.** Project rules stay
  project-scoped.
- **Tag vocabulary drift** makes search useless; reuse existing tags first.

## Output

```markdown
## Document
- key / slug / type / scope / status
- project: <slug or global>
- path: /absolute/path
- hash: sha256:<prefix> · bytes: <n> · version: <n>

## Verification
- file exists: yes/no
- FTS hit for "<query>": yes/no
- supersedes: <key or none>
```

## Handoffs

- Need exact registry payloads or schema fields → (via /zenf-tools).
- Need to reconcile conflicting authorities → (via /ssot).
- Need to attach the document as task input context → (via /zenf-project).
- Need to bind the document to a worker persona → (via /zenf-agent).
