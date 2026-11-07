# ZenForge Skills

## Install

```bash
npx skills add venhdev/skills
```

## zenf

`zenf` is the single-binary control plane and multi-agent tracking daemon for ZenForge.

### Install

Install via **ZenCLI**:
```bash
zen i zenforge
```
Or download pre-compiled binaries directly from [GitHub Releases](https://github.com/venhdev/zenforge/releases).

### Build from source

```bash
cd apps/daemon && make build
# Binary is produced at apps/daemon/bin/zenf (and symlinked as zenfd)
```

### CLI Quickstart

```bash
zenf                  # Show quickstart overview and active daemon state
zenf daemon           # Display daemon management commands
zenf daemon start     # Start daemon in foreground
zenf daemon start -d  # Start daemon in background (detached mode)
zenf daemon stop      # Gracefully stop running background daemon
zenf daemon logs      # Stream live logs from background daemon
zenf status           # Comprehensive status report (daemon runtime + all AI MCP clients)
zenf mcp              # Audit MCP setups & view integration guide (zero hang)
```

Client setup, tool reference, write surfaces, and verification: [MCP.md](MCP.md).

Agent-facing cards:

- [zenf-tools](skills/zenf-tools/SKILL.md) — MCP tool surface and task lifecycle.
- [zenf-project](skills/zenf-project/SKILL.md) — project/plan/task hierarchy front door.
- [zenf-docs](skills/zenf-docs/SKILL.md) — SSOT document registry front door.
- [zenf-agent](skills/zenf-agent/SKILL.md) — worker personas and observed sessions.
