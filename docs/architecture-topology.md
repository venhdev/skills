# ZenForge Daemon (zenfd) Architecture & Seams

> Category: ARCHITECTURE | Version: 1

# ZenForge Daemon (zenfd) Architecture

ZenForge Daemon is an agent-native operating environment combining SQLite WAL as Single Source of Truth with Model Context Protocol (MCP) and POSIX process group supervision.

```mermaid
graph LR
  subgraph Host [Developer Host / Agent Runner]
    A[Agent CLI / IDE] -->|JSON-RPC 2.0 stdio| B(zenfd mcp)
    C[Web Cockpit :5173] -->|HTTP / SSE :1123| D[Gateway Server]
  end
  subgraph ControlPlane [Daemon Engine]
    B --> E[(SQLite WAL SSOT)]
    D --> E
    D --> F[DAG Scheduler]
    F --> G[POSIX Supervisor]
  end
```

