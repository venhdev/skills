# Naming Rules

## Doc Files (Primary Scope)

### Filenames
- Format: **kebab-case**, noun phrase
- Examples: `routing.md`, `forms-validation.md`, `api-design-principles.md`

### Forbidden patterns

- **Dates**: `2024-migration.md`, `march-update.md` ❌
- **Version numbers**: `api-v2-spec.md`, `config-v3.md` ❌
- **Generic/non-descriptive**: `misc.md`, `temp.md`, `notes.md`, `other.md` ❌
- **Person/team names**: `johns-notes.md`, `backend-team-guide.md` ❌
- **Unclear abbreviations**: `cfg.md` (unclear), `auth-svc.md` (ambiguous) ❌

### Frontmatter `title`
- Descriptive, human-readable
- Don't encode: tech stack, version, team, environment
- Example: ✅ `title: API Design Principles` (not `API v2 Design Spec for Backend Team`)

---

## Anti-Patterns (Universal — Doc + Code)

Applied across both documentation and code scope when extended:

### Tech stack encoding
- ❌ `mysql-setup.md` → ✅ `database-setup.md`
- ❌ `react-forms.md` → ✅ `forms.md`

### Team/person naming
- ❌ `johns-notes.md` → ✅ `auth-notes.md`
- ❌ `backend-config.md` → ✅ `configuration.md`

### Environment/region encoding
- ❌ `prod-config.md` → ✅ `configuration.md`
- ❌ `us-deployment.md` → ✅ `deployment.md`

### Version encoding
- ❌ `api-v2-spec.md` → ✅ `api-spec.md`
- ❌ `form-validation-v3.md` → ✅ `form-validation.md`

### Generic/placeholder names
- ❌ `misc`, `other`, `temp`, `random`, `notes` (without context) → ✅ `[descriptive-purpose].md`

---

## Code Files (Extended Scope)

Load this section **only when user specifies code scope** (folder path, codebase, etc.).

### Classes / Types / Interfaces / Enums

**Format**: `PascalCase` — no `I` prefix for interfaces

Examples:
- ✅ `UserService.ts` (not `IUserService.ts`)
- ✅ `CreateUserDto.ts` (DTO with Dto suffix)
- ✅ `UserCreatedEvent.ts` (event with Event suffix)
- ✅ `UserNotFoundException.ts` (exception with Error/Exception suffix)

### Functions / Methods

**Format**: `camelCase` with verb prefix

| Prefix | Semantics | Example |
|---|---|---|
| `get*` | Idempotent retrieve (property-like) | `getUserName()` |
| `find*` | Search with predicate/filter | `findUsersByRole(role)` |
| `fetch*` | Async retrieval (network/I/O) | `fetchUserData()` |
| `create*` | Create new entity | `createUser(data)` |
| `update*` | Update existing entity | `updateUserEmail(id, email)` |
| `delete*` / `remove*` | Delete/remove entity | `deleteUser(id)` |
| `is*` / `has*` / `can*` / `should*` | Predicate (return boolean) | `isAdmin()`, `hasPermission(perm)` |
| `on*` / `handle*` | Event handler | `onClick()`, `handleUserSubmit()` |
| `compute*` / `calculate*` | Derive computed value | `computeTotal()` |
| `build*` | Assemble composite object | `buildUserProfile()` |
| `parse*` / `serialize*` | String ↔ structured type | `parseJSON()`, `serializeToXML()` |

### Code Files

- **Single class/type export**: `PascalCase.ts` — Example: `UserService.ts`
- **DTO/Entity/Enum**: `kebab-case.<kind>.ts` — Examples: `user.dto.ts`, `role.enum.ts`
- **Utilities**: `kebab-case.ts` — Example: `string-utils.ts`
- **Types-only**: `kebab-case.types.ts` — Example: `api.types.ts`
- **Tests**: 
  - Unit: `PascalCase.spec.ts` — Example: `UserService.spec.ts`
  - E2E: `kebab-case.e2e.ts` — Example: `user-registration.e2e.ts`

### Directories

- **Always** `kebab-case`
- Examples: `src/auth/`, `src/user-management/`, `src/api/routes/`

---

## Excluded Paths

Files matching these patterns are excluded from naming checks:

**In git repositories:**
- Gitignored paths via `git ls-files --directory --others --ignored --exclude-standard`
- Hardcoded fallback: `.claude`, `.github`, `node_modules`, `vendor`, `dist`, `build`, `.git`

**Outside git repositories:**
- Hardcoded: `.claude`, `.github`, `node_modules`, `vendor`, `dist`, `build`, `.git`

---

## Project Convention

If project consistently follows a different convention — document it and acknowledge it rather than flag as error.

Example: If all docs use `_draft-` prefix for WIP and it's team convention, don't flag it as "generic naming".
