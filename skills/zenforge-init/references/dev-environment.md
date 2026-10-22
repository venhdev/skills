# Dev Environment & Host Profile — <Project Name>

Machine-local development profile. Excluded by `.gitignore`.
*(Sections 2 and 3 are project-adaptive; omit if the repository does not use them)*

## 1. Toolchain & Environment

- **runtime_wrapper**: <mise exec -- | bun | pnpm | uv run | none>
- **test_runner**: <cargo test | pnpm test | pytest | flutter test>
- **static_analysis**: <cargo clippy | ruff check | pnpm lint>
- **tracker_cli**: <gh (authenticated) | glab (authenticated) | none>
- **version_managers**: <installed version managers>
- **runtimes_and_build**: <installed compilers and runtimes>
- **mobile_and_device**: <installed mobile tools | none>
- **task_runners**: <make | just | turbo | none>
- **search_ast_data**: <rg, fd, jq, ast-grep>
- **vcs_containers_db**: <git, docker, docker-compose, psql, sqlite3>

## 2. Local Services & Datastores *(omit if none)*

| Resource | Type / Startup Command | Local Target / Endpoint |
| :--- | :--- | :--- |
| **<Name>** | `<e.g. docker compose up -d postgres>` | `<e.g. localhost:5432/app_dev (scratch: app_test)>` |

## 3. Safety & Secret Boundaries

- **Secrets**: Forbid inspecting (`read_file`, grep, cat) `.env*` (except `.env.example`), `*.pem`, `*.key`, `id_rsa`. Inject dummy mocks when running local tests.
- **Boundaries**: Strictly forbid dispatching mutations to external/production infrastructure or destructively purging primary development datastores without confirmation.
