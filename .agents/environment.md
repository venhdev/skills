# Environment Specification & Architecture Framework

This document governs the execution environment for developer agents and automated runners. It defines the universal architectural invariants, an open discovery framework, and an adaptive template for deriving project-specific environment specifications.

---

## 1. Governing Invariants

- **Stack & Host Agnostic**: Never assume a specific language runtime, framework, or container engine. Adapt to pure libraries, distributed microservices, CLI binaries, or embedded targets.
- **Zero-Secret Exposure**: Never store real credentials, production connection strings, or private keys. Confine test authentication strictly to mock fixtures or synthetic `.env.test` templates.
- **Zero Business Coupling**: Confine all content strictly to runtime infrastructure, execution toolchains, test datastores, and verification commands. Forbid feature specifications or domain logic.
- **Profile Portability**: Project-wide invariants belong in version control; host-specific ephemeral paths or machine overrides must be isolated in `.gitignore` or local user profiles.

---

## 2. Agent Discovery Questions (Open Heuristic)

When an agent initializes or reconciles the environment layout, it evaluates these 5 open questions against the target codebase:

1. **Host & Platform Primitives**: What system binaries (`git`, `curl`, `make`) or container runtimes (Docker, Podman) must be present on the host?
2. **Project Toolchains & Execution Wrappers**: Where are language versions pinned (`.nvmrc`, `Cargo.toml`, `go.mod`, `pyproject.toml`)? What execution wrapper manages dependencies (`cargo`, `pnpm`, `uv`, `mise`, `poetry`)?
3. **External Services & Datastores *(omit if none)* **: Does the project depend on external datastores, queues, or emulators? How are local test instances provisioned and probed for readiness?
4. **Verification Harness & Command Tiers**: What is the canonical fast command for unit/static checks? What command runs hermetic integration tests? What command formats/lints?
5. **Observability & Scratch Storage *(omit if none)* **: Where do temporary debug artifacts and logs reside (`.agents/scratch/logs/`)? What local ports or health endpoints expose status?

---

## 3. Universal Adaptive Template

The agent derives the concrete layout below, populating only the dimensions relevant to this repository and omitting non-applicable sections:

```markdown
# Project Environment Specification

## 1. Host & Platform Prerequisites
- Platform: <OS / Architecture constraints, e.g. Linux x86_64, macOS arm64, POSIX>
- Required Host Tools: <List of essential system CLIs, e.g. git, curl, jq>
- Container Engine: <Docker Engine / Podman / None (pure host execution)>

## 2. Runtime & Toolchain Matrix
- Primary Runtime: <Language and version constraint, e.g. Rust 1.78+, Node 20.x, Python 3.11+>
- Version Manifest: <Repo-relative manifest path, e.g. Cargo.toml, .nvmrc, go.mod>
- Package & Execution Wrapper: <Canonical runner, e.g. cargo, pnpm, uv run, ./mvnw>

## 3. Backing Services & Datastores *(omit for standalone libraries / CLIs)*
- Provisioning Mechanism: <docker-compose.yml / testcontainers / local daemon / embedded>
- Service Inventory:
  - `<Service Name>`: Port `<Port>` | Health Probe: `<Command or socket check>`
- Test Datastore Isolation: <Target test database name or in-memory instance, e.g. *_test, :memory:>

## 4. Canonical Verification Commands
- Fast / Static Check: `<Command for typecheck, linter, or fast unit tests>`
- Hermetic Integration Check: `<Command for end-to-end or container-backed test suite>`
- Dependency Sync: `<Command to install locked dependencies, e.g. pnpm install --frozen-lockfile>`

## 5. Ephemeral Storage & Secrets Boundary
- Test Configuration: `<Repo-relative path for test env, e.g. .env.test or mock fixtures>`
- Ephemeral Scratch Directory: `.agents/scratch/` (strictly git-ignored)
- Local Log / Trace Output: `.agents/scratch/logs/` *(omit if none)*
```
