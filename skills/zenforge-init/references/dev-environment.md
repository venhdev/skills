# Dev Environment & Host Profile — <Project Name>

Machine-local development environment, runtime toolchains, and service contracts.

> [!NOTE]
> 1. This profile is machine-local and MUST be excluded by `.gitignore`.
> 2. **Selective & Omissible**: Sections 2, 3, and 5 are project-adaptive. If the repository does not use background containers, emulators, or databases (e.g., pure CLI, library, frontend mock), omit those sections or mark them `none`; never manufacture synthetic filler.

## 1. Toolchain & Environment

- **runtime_wrapper**: <active wrapper, e.g., mise exec -- | bun | pnpm | uv run | none>
- **test_runner**: <canonical test command, e.g., cargo test | pnpm test | pytest | flutter test>
- **static_analysis**: <canonical lint command, e.g., cargo clippy | ruff check | pnpm lint>
- **tracker_cli**: <installed tracker auth, e.g., gh (authenticated) | glab (authenticated) | none>
- **version_managers**: <installed version managers, e.g., mise, asdf, nvm, fvm, uv>
- **runtimes_and_build**: <installed compilers and build runtimes>
- **mobile_and_device**: <installed mobile tools, e.g., adb, xcodebuild, flutterfire | none>
- **task_runners**: <task runners, e.g., make, just, turbo | none>
- **search_ast_data**: <search utilities, e.g., rg, fd, jq, ast-grep>
- **vcs_containers_db**: <vcs, container and db clients, e.g., git, docker, docker-compose, psql, sqlite3>

## 2. Dev Services & Emulators *(omit if not applicable)*

| Service / Target | Type | Startup / Connect Command | Health Check |
| :--- | :--- | :--- | :--- |
| **<Name>** | <Container \| Native \| Simulator \| Mock> | `<command>` | `<check command>` |

## 3. Local Datastores & State Isolation *(omit if not applicable)*

- **Development Store**: `<local connection URI, e.g., postgresql://... | sqlite path | local storage>`
  - *Usage*: Manual testing and local execution runtime.
- **Scratch / Isolated Test Store**: `<isolated URI, e.g., test db | :memory: | temp directory>`
  - *Usage*: Mandatory for test suites, `/diagnose`, and `/spike` to preserve main dev state.

## 4. Environment Variables & Secret Policies

- **Permitted Template**: `<e.g., .env.example | .env.test | none>` (schema and default keys reference only).
- **Hard Prohibitions**:
  - STRICTLY FORBIDDEN to read (`read_file`, `cat`, `head`, `grep`) secret files: `.env`, `.env.*` (except `.env.example`), `*.pem`, `*.key`, `id_rsa`, `*credentials*`.
  - STRICTLY FORBIDDEN to commit or write real tokens, passwords, or production secrets into `.agents/scratch/` or git history.
- **Mock Fallback**: Inject dummy values (e.g., `API_KEY=dummy-test-key`) when running local test probes.

## 5. Execution Boundaries & Safety *(omit if not applicable)*

- **Allowed Local Actions**: `<safe local actions: e.g., run migrations on local dev DB, reset test cache, rebuild container>`
- **Forbidden Actions**:
  - Dropping or destructively purging development datastores without explicit user confirmation.
  - Dispatching network commands or mutations to external, staging, or production cloud infrastructure.
