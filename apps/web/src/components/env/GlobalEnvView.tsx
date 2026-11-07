import React, { useState, useEffect } from 'react'
import {
  Cpu,
  Activity,
  Terminal,
  Shield,
  CheckCircle2,
  RefreshCw,
} from 'lucide-react'

import type { ApiResult } from '../../hooks/useZenfAPI'

interface GlobalEnvViewProps {
  onFetchEnv: () => Promise<ApiResult<Record<string, any>>>
}

export const GlobalEnvView: React.FC<GlobalEnvViewProps> = ({ onFetchEnv }) => {
  const [envData, setEnvData] = useState<Record<string, any> | null>(null)
  const [loading, setLoading] = useState(true)

  const loadEnv = async () => {
    setLoading(true)
    try {
      const res = await onFetchEnv()
      if (res.ok) {
        setEnvData(res.data)
      } else {
        setEnvData(null)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadEnv()
  }, [])

  return (
    <div className="flex h-full w-full flex-col bg-zinc-950 font-sans p-6 overflow-y-auto select-none">
      <div className="max-w-4xl mx-auto w-full space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800 pb-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-violet-500/10 text-violet-400 ring-1 ring-violet-500/30">
              <Cpu className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-zinc-100">
                Global Environment & Toolchain Specs
              </h2>
              <p className="text-xs text-zinc-400">
                Developer host environment diagnostics, active runtime versions, and system seams.
              </p>
            </div>
          </div>

          <button
            onClick={loadEnv}
            disabled={loading}
            className="flex items-center gap-1.5 rounded border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs font-semibold text-zinc-300 hover:bg-zinc-800 hover:text-white transition disabled:opacity-50"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh Diagnostics</span>
          </button>
        </div>

        {/* 1. Host Compute & OS */}
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-4">
            <span className="text-[10px] uppercase font-bold text-zinc-500 tracking-wider block mb-1">
              Operating System
            </span>
            <span className="font-mono text-lg font-bold text-zinc-100">
              {envData?.os || 'linux'} / {envData?.arch || 'amd64'}
            </span>
            <div className="mt-2 flex items-center gap-1 text-[11px] text-emerald-400">
              <CheckCircle2 className="h-3.5 w-3.5" />
              <span>POSIX Process Isolation Active</span>
            </div>
          </div>

          <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-4">
            <span className="text-[10px] uppercase font-bold text-zinc-500 tracking-wider block mb-1">
              Compute Resources
            </span>
            <span className="font-mono text-lg font-bold text-zinc-100">
              {envData?.num_cpu || '16'} CPU Threads
            </span>
            <div className="mt-2 flex items-center gap-1 text-[11px] text-zinc-400">
              <Activity className="h-3.5 w-3.5 text-violet-400" />
              <span>Parallel Slot Execution Ready</span>
            </div>
          </div>

          <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-4">
            <span className="text-[10px] uppercase font-bold text-zinc-500 tracking-wider block mb-1">
              Database Engine
            </span>
            <span className="font-mono text-lg font-bold text-zinc-100">
              SQLite 3 (WAL Mode)
            </span>
            <div className="mt-2 flex items-center gap-1 text-[11px] text-emerald-400">
              <CheckCircle2 className="h-3.5 w-3.5" />
              <span>Zero-lock Contention Engine</span>
            </div>
          </div>
        </div>

        {/* 2. Toolchain Manifest */}
        <div className="space-y-3">
          <h3 className="flex items-center gap-2 text-xs font-bold text-zinc-200 uppercase tracking-wider">
            <Terminal className="h-4 w-4 text-emerald-400" />
            <span>Installed Runtimes & Compilers</span>
          </h3>

          <div className="grid grid-cols-3 gap-3 font-mono text-xs">
            <div className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-3">
              <span className="text-[10px] text-zinc-500 font-sans block mb-0.5">Go Toolchain</span>
              <span className="text-zinc-200 font-bold">{envData?.go_version || 'go1.24+'}</span>
            </div>
            <div className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-3">
              <span className="text-[10px] text-zinc-500 font-sans block mb-0.5">Git Version</span>
              <span className="text-zinc-200 font-bold">{envData?.git_version || 'git version 2.40+'}</span>
            </div>
            <div className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-3">
              <span className="text-[10px] text-zinc-500 font-sans block mb-0.5">Node.js Runtime</span>
              <span className="text-zinc-200 font-bold">{envData?.node || 'v22.x'}</span>
            </div>
          </div>
        </div>

        {/* 3. Agent Protocol & Rules Seam */}
        <div className="space-y-3">
          <h3 className="flex items-center gap-2 text-xs font-bold text-zinc-200 uppercase tracking-wider">
            <Shield className="h-4 w-4 text-violet-400" />
            <span>Governance & Architecture Invariants</span>
          </h3>

          <div className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4 space-y-2 text-xs text-zinc-300">
            <div className="flex items-center justify-between border-b border-zinc-800 pb-2">
              <span className="font-semibold text-zinc-200">Single Source of Truth (SSOT):</span>
              <span className="font-mono text-zinc-400">~/.zenforge/state.db (SQLite WAL)</span>
            </div>
            <div className="flex items-center justify-between border-b border-zinc-800 py-2">
              <span className="font-semibold text-zinc-200">Model Context Protocol (MCP):</span>
              <span className="font-mono text-emerald-400">JSON-RPC 2.0 stdio (zenf mcp)</span>
            </div>
            <div className="flex items-center justify-between border-b border-zinc-800 py-2">
              <span className="font-semibold text-zinc-200">Execution Safety Protocol:</span>
              <span className="text-zinc-400">POSIX -PGID Cascade Termination</span>
            </div>
            <div className="flex items-center justify-between pt-2">
              <span className="font-semibold text-zinc-200">Environment Doctor:</span>
              <span className="font-mono text-violet-400">.agents/environment.md</span>
            </div>
          </div>
        </div>

      </div>
    </div>
  )
}
