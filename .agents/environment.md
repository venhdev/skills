# Environment Specification — my-skills (ZenForge Ecosystem)

Infrastructure, toolchain manifest, and execution environment specification.

## 1. Toolchain & Environment

- **Primary Runtime (Backend)**: Go 1.24+ (`/usr/bin/go`, toolchain: go1.26+)
- **Primary Runtime (Frontend)**: Node.js v24.19.0 (`/home/rirong/.nvm/versions/node/v24.19.0/bin/node`)
- **Package & Task Wrappers**:
  - `pnpm` v11.23.0 (Frontend dependency management)
  - `GNU Make` 4.4.1 (Monorepo build automation: `apps/daemon/Makefile`)
- **Version Control & Shell**:
  - `git` 2.47.3
  - `bash` (Linux amd64 POSIX shell)
- **Container Engine**: Docker v29.7.2 (optional, not required for core daemon)

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
