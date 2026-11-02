import React from 'react'
import { Zap, Activity, AlertOctagon, Pause, Play } from 'lucide-react'
import type { Plan } from '../../types/contracts'

interface HeaderBarProps {
  connected: boolean
  activePlan: Plan | null
  plans: string[]
  onSelectPlan: (slug: string) => void
  activeWorkers: number
  maxWorkers: number
  onKillAll?: () => void
  isPaused?: boolean
  onTogglePause?: () => void
}

export const HeaderBar: React.FC<HeaderBarProps> = ({
  connected,
  activePlan,
  plans,
  onSelectPlan,
  activeWorkers,
  maxWorkers,
  onKillAll,
  isPaused = false,
  onTogglePause,
}) => {
  return (
    <header className="flex h-14 w-full items-center justify-between border-b border-zinc-800 bg-zinc-900/90 px-4 backdrop-blur select-none">
      {/* Brand & Plan Selector */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2 font-mono text-sm font-semibold tracking-wider text-zinc-100">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/30">
            <Zap className="h-4 w-4 fill-emerald-400" />
          </div>
          <span>ZenForge</span>
          <span className="rounded bg-zinc-800 px-1.5 py-0.5 text-[10px] text-zinc-400 font-sans">
            CONTROL PLANE
          </span>
        </div>

        <div className="h-4 w-[1px] bg-zinc-800" />

        {/* Plan Selector */}
        <div className="flex items-center gap-2">
          <span className="text-xs text-zinc-400">Plan:</span>
          <select
            value={activePlan?.slug || ''}
            onChange={(e) => onSelectPlan(e.target.value)}
            className="rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1 text-xs font-mono font-medium text-zinc-200 outline-none hover:border-zinc-700 focus:border-violet-500"
          >
            {plans.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
            {plans.length === 0 && <option value="">(No active plans)</option>}
          </select>
          {activePlan && (
            <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-400 ring-1 ring-emerald-500/20">
              {activePlan.status}
            </span>
          )}
        </div>
      </div>

      {/* Worker Slots, SSE Status & Controls */}
      <div className="flex items-center gap-5">
        {/* Worker Pool Utilization Gauge */}
        <div className="flex items-center gap-2 text-xs">
          <span className="text-zinc-400">Worker Pool:</span>
          <div className="flex items-center gap-1">
            {Array.from({ length: maxWorkers }).map((_, i) => (
              <div
                key={i}
                className={`h-2.5 w-2 rounded-sm transition-colors ${
                  i < activeWorkers
                    ? 'bg-violet-500 shadow-[0_0_8px_rgba(139,92,246,0.6)]'
                    : 'bg-zinc-800'
                }`}
                title={`Slot ${i + 1}: ${i < activeWorkers ? 'Occupied' : 'Idle'}`}
              />
            ))}
          </div>
          <span className="font-mono text-xs font-medium text-zinc-300">
            {activeWorkers}/{maxWorkers}
          </span>
        </div>

        <div className="h-4 w-[1px] bg-zinc-800" />

        {/* Real-time SSE Connection Indicator */}
        <div className="flex items-center gap-2 text-xs">
          <div
            className={`h-2 w-2 rounded-full ${
              connected
                ? 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]'
                : 'bg-rose-500 animate-pulse'
            }`}
          />
          <span className="font-mono text-[11px] text-zinc-300">
            {connected ? 'Port 1123 Active' : 'Connecting...'}
          </span>
          <Activity className="h-3.5 w-3.5 text-zinc-500" />
        </div>

        <div className="h-4 w-[1px] bg-zinc-800" />

        {/* Controls */}
        <div className="flex items-center gap-2">
          {onTogglePause && (
            <button
              onClick={onTogglePause}
              className="flex items-center gap-1 rounded bg-zinc-800 px-2 py-1 text-xs text-zinc-300 hover:bg-zinc-700 hover:text-white"
            >
              {isPaused ? (
                <>
                  <Play className="h-3 w-3 text-emerald-400" /> Resume
                </>
              ) : (
                <>
                  <Pause className="h-3 w-3 text-amber-400" /> Pause
                </>
              )}
            </button>
          )}

          {onKillAll && (
            <button
              onClick={onKillAll}
              className="flex items-center gap-1 rounded bg-rose-500/10 px-2.5 py-1 text-xs font-medium text-rose-400 ring-1 ring-rose-500/20 hover:bg-rose-500 hover:text-white transition-all"
              title="Send cascade SIGKILL to all active runner process groups"
            >
              <AlertOctagon className="h-3 w-3" />
              Kill All
            </button>
          )}
        </div>
      </div>
    </header>
  )
}
