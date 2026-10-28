---
name: env-doctor
description: "Audit runtime toolchains, verify container service health, seed hermetic test primitives, and execute safe local remediation."
disable-model-invocation: true
---

# env-doctor — Execution Environment Diagnostics & Remediation

## Operating Invariants

- **Idempotent Local Boundary**: Confine remediation strictly to project-scoped actions (declared dependency sync, local service boot, test datastore migration, synthetic test secrets); forbid system-wide mutations (`sudo apt`, `brew`), host PATH tampering, or remote secret fetching.
- **Zero-Secret Exposure**: Never inspect or copy production secret files (`.env*` except `.env.example`); inject deterministic mocks into `.env.test` or environment stubs.
- **Manifest SSOT Grounding**: Read `.agents/environment.md` as the authoritative specification. Treat omissible sections (services for pure libraries, external datastores) as `PASS` when absent.
- **Adaptive Invocation Output**: Render the full Canonical Output Contract when invoked by a human; output only a single-line Micro-Summary token (`READY` or `UNRESOLVED: <blocker>`) when called by an autonomous runner.

## Domain Rubric

### 1. Host & Toolchain Integrity
Cross-reference declared manifests (`package.json`, `Cargo.toml`, `go.mod`, `pyproject.toml`, `.nvmrc`) against host runtimes. Verify declared execution wrappers (`pnpm`, `cargo`, `uv`, `mise`, `mix`) before falling back to bare binaries.

### 2. Service Dependencies & Deep Probes
For projects declaring backing services, execute declared readiness probes (socket connection, HTTP `/healthz`, database ping) with exponential backoff (max 15s). Forbid treating process state (`Up`) as readiness. Omit probe evaluation for standalone libraries or CLI targets lacking external services.

### 3. Hermetic Test Primitives & Datastores
Verify declared test datastores are provisioned, schema migrations applied, and deterministic seeds loaded. Verify existence of declared test configuration files (`.env.test`); synthesize from `.env.example` with dummy values if absent. Ensure `.agents/scratch/` has write permissions.

### 4. Safe Auto-Remediation Matrix
- *Toolchain*: Run project-declared dependency sync commands when manifests are newer than installed modules.
- *Services*: Boot declared local services (`docker compose up -d <service>`) when offline.
- *Datastore*: Run migrations and seeds targeted exclusively at test datastores.
- *Escalations*: Stop and report `UNRESOLVED` for port collisions, missing host daemons, or destructive operations (volume wipes, table drops).

## Canonical Output Contract

```markdown
# Environment Health Report: <Scope> · <READY | STAGED | UNRESOLVED>

Basis: .agents/environment.md

| Dimension | Declared / Target | Active / Detected | State |
| :--- | :--- | :--- | :--- |
| **Toolchain** | <Target runtime version> | <Active version / wrapper> | <PASS | FAIL> |
| **Services** | <Declared services or N/A> | <Probe outcome or Standalone>| <PASS | FAIL> |
| **Datastore** | <Target test datastore> | <Migration & seed status> | <PASS | FAIL> |
| **Test Stubs** | <Test env / secrets> | <Stubs presence (.env.test)> | <PASS | FAIL> |

*(omit if clean)* ## Applied Remediation
- **[<Auto | Staged>] <Action>**: <Command and outcome or pending approval>

*(omit if READY)* Gap: <Root cause blocker and required human action.>
```

*Autonomous Micro-Token Contract*:
```text
READY | runtime: <version> | services: <healthy|standalone> | fixtures: verified
```
*(On failure, output `UNRESOLVED: <single-sentence blocker>`)*.

### Routing

- Environment is READY, proceed to task decomposition → (via /split-tasks).
- Environment is READY, proceed to code implementation → (via /forge).
- Environment is READY, proceed to root-cause diagnosis → (via /diagnose).
- Missing service architecture requires design decision → (via /architect-decision).

## Execution Protocol

1. Read `.agents/environment.md` and probe declared host tools, runtimes, services, and fixtures.
2. Short-circuit to `READY` if all declared dimensions pass.
3. Execute safe auto-remediations (boot declared services, sync dependencies, provision test datastore).
4. Re-probe health. If probes pass, output `READY`. If blocked by missing host runtimes, port collisions, or destructive actions, halt turn with `UNRESOLVED`.
