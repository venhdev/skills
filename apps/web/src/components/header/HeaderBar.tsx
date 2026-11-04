import React from 'react'
import {
  Zap,
  Activity,
  GitFork,
  Cpu,
  Trash2,
  Bot,
} from 'lucide-react'
import type { Plan } from '../../types/contracts'

export type CockpitView = 'dag' | 'env'

interface HeaderBarProps {
  connected: boolean
  activePlan: Plan | null
  plans: string[]
  onSelectPlan: (slug: string) => void
  activeWorkers: number
  onDeletePlanClick?: () => void
  currentView?: CockpitView
  onChangeView?: (view: CockpitView) => void
}

export const HeaderBar: React.FC<HeaderBarProps> = ({
  connected,
  activePlan,
  plans,
  onSelectPlan,
  activeWorkers,
  onDeletePlanClick,
  currentView = 'dag',
  onChangeView,
}) => {
  return (
    <header className="flex h-14 w-full items-center justify-between border-b border-zinc-800 bg-zinc-900/90 px-4 backdrop-blur select-none">
      {/* Brand & Project / Plan Selector */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2 font-mono text-sm font-semibold tracking-wider text-zinc-100">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/30">
            <Zap className="h-4 w-4 fill-emerald-400" />
          </div>
          <span>ZenForge</span>
          <span className="rounded bg-zinc-800 px-1.5 py-0.5 text-[10px] text-zinc-400 font-sans">
            v0.1.0
          </span>
        </div>

        <div className="h-4 w-[1px] bg-zinc-800" />

        {/* Plan Selector & Actions */}
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-zinc-400">Plan:</span>
          <select
            value={activePlan?.slug || ''}
            onChange={(e) => onSelectPlan(e.target.value)}
            className="rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1 text-xs font-mono font-medium text-zinc-200 outline-none hover:border-zinc-700 focus:border-violet-500 max-w-[150px] truncate"
          >
            {activePlan && !plans.includes(activePlan.slug) && (
              <option value={activePlan.slug}>{activePlan.slug}</option>
            )}
            {plans.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
            {plans.length === 0 && !activePlan && <option value="">(No active plans)</option>}
          </select>

          {activePlan && onDeletePlanClick && (
            <button
              onClick={onDeletePlanClick}
              className="rounded border border-zinc-800 bg-zinc-950 p-1 text-zinc-500 hover:border-rose-900/50 hover:bg-rose-950/20 hover:text-rose-400"
              title={`Delete plan '${activePlan.slug}' and cascading tasks`}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          )}

          {activePlan && (
            <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-400 ring-1 ring-emerald-500/20">
              {activePlan.status}
            </span>
          )}
        </div>
      </div>

      {/* Top Cockpit View Switcher */}
      {onChangeView && (
        <div className="flex items-center rounded-lg border border-zinc-800 bg-zinc-950 p-0.5 text-xs shadow-inner">
          <button
            onClick={() => onChangeView('dag')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1 font-semibold transition-colors ${
              currentView === 'dag'
                ? 'bg-zinc-800 text-white shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <GitFork className="h-3.5 w-3.5" />
            <span>DAG Canvas</span>
          </button>
          <button
            onClick={() => onChangeView('env')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1 font-semibold transition-colors ${
              currentView === 'env'
                ? 'bg-zinc-800 text-white shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Cpu className="h-3.5 w-3.5" />
            <span>Env</span>
          </button>
        </div>
      )}

      {/* Active Ephemeral Workers Badge & SSE Health Status */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-1.5 text-xs font-mono text-zinc-400">
          <Bot className={`h-3.5 w-3.5 ${activeWorkers > 0 ? 'text-violet-400 animate-pulse' : 'text-zinc-500'}`} />
          <span>Active: {activeWorkers}</span>
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
      </div>
    </header>
  )
}
