# Environment Specification — zenforge (ZenForge Ecosystem)

Infrastructure, toolchain manifest, and execution environment specification.

## Toolchain & Environment

- **tracker_cli**: gh (authenticated)
- **version_managers**: fvm, nvm, corepack, uv, uvx
- **runtimes_and_build**: flutter, dart, pnpm, yarn, npm, node, python3, go, java
- **mobile_and_device**: adb, bundletool, shorebird, flutterfire, fluttercraft
- **task_runners**: make
- **search_ast_data**: rg, fzf, jq, tree, curl
- **vcs_containers_db**: git, gh, docker, kubectl, psql, sqlite3

## 2. Local Services & Control Plane Endpoints

| Resource | Type / Startup Command | Local Target / Endpoint | Health Probe |
| :--- | :--- | :--- | :--- |
| **zenfd Control Plane** | Go Daemon binary (`apps/daemon/bin/zenfd`) | `127.0.0.1:1123` | `curl -f http://127.0.0.1:1123/api/v1/plans` |
| **zenfd Web UI** | Embedded SPA or Vite Dev Server (`apps/web`) | `http://127.0.0.1:1123` (Prod) / `http://localhost:5173` (Dev) | Browser HTTP GET |
| **SQLite State DB** | Embedded WASM SQLite WAL (`ncruces/go-sqlite3`) | `~/.zenforge/state.db` | `PRAGMA journal_mode;` -> `wal` |

## 3. Canonical Verification Commands

- **Backend Daemon Tests**: `cd apps/daemon && go test -v -race ./...`
- **Frontend Dashboard Build**: `cd apps/web && pnpm build`
- **Single-Binary Monorepo Build**: `cd apps/daemon && make build` (automatically builds web/dist -> embeds into Go binary)
- **Plumbing Spike Suite**: `cd .agents/scratch/spike-zenfd && go test -v -race ./...`

## 4. Safety, Secrets & Ephemeral Storage Boundary

- **Secrets Boundary**: Strictly forbid reading `.env*` (except `.env.example`), private keys (`*.pem`, `*.key`, `id_rsa`).
- **Ephemeral Scratch Directory**: `.agents/scratch/` (strictly git-ignored).
- **Persistent State Directory**: `~/.zenforge/` (local machine daemon state and database).
- **Attachment Store**: `~/.zenforge/attachments/<task>/` (INPUT media copied by MCP; 25 MB per file).
- **Activity Log**: `events` table in `~/.zenforge/state.db`, appended by DB triggers, pruned after 30 days.

## 5. Development Mode & Live-Reload Workflow

#### Terminal 1: Backend Daemon (Go Auto-Reload via air)

```bash
make -C apps/daemon dev
```

(Whenever any `.go` file in `apps/daemon` is modified, `air` automatically recompiles and restarts `zenfd` without requiring `Ctrl+C`).

#### Terminal 2: Frontend Dashboard (Vite HMR)

```bash
cd apps/web && pnpm dev
```

(Starts Vite dev server at `http://localhost:5173` with Instant HMR and automated reverse proxy forwarding API/SSE traffic to daemon port 1123).

