# Environment Specification — my-skills (ZenForge Ecosystem)

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
- **Single-Binary Monorepo Build**: `cd apps/daemon && make build` (tự động build web/dist -> embed vào Go binary)
- **Plumbing Spike Suite**: `cd .agents/scratch/spike-zenfd && go test -v -race ./...`

## 4. Safety, Secrets & Ephemeral Storage Boundary

- **Secrets Boundary**: Strictly forbid reading `.env*` (except `.env.example`), private keys (`*.pem`, `*.key`, `id_rsa`).
- **Ephemeral Scratch Directory**: `.agents/scratch/` (strictly git-ignored).
- **Persistent State Directory**: `~/.zenforge/` (local machine daemon state and database).
- **Claim-Check Disk Cache**: `.agents/scratch/logs/` and `.agents/scratch/artifacts/`.

## 5. Development Mode & Live-Reload Workflow

#### Terminal 1: Backend Daemon (Go Auto-Reload qua air)

```bash
make -C apps/daemon dev
```

(Bất cứ khi nào bạn sửa bất kỳ file `.go` nào trong `apps/daemon`, `air` sẽ tự compile lại và khởi động lại `zenfd` tức thì mà không cần bạn bấm `Ctrl+C`).

#### Terminal 2: Frontend Dashboard (Vite HMR)

```bash
cd apps/web && pnpm dev
```

(Khởi chạy Vite dev server tại `http://localhost:5173` với Instant HMR và reverse proxy tự động chuyển tiếp API/SSE/WebSocket về daemon cổng 1123).

