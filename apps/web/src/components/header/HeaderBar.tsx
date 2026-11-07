import React from 'react'
import {
  Zap,
  Activity,
  GitFork,
  Cpu,
  Bot,
  FolderKanban,
  BookOpen,
  ScrollText,
} from 'lucide-react'
import type { Plan, Project } from '../../types/contracts'

export type CockpitView = 'dag' | 'projects' | 'docs' | 'agents' | 'activity' | 'env'

interface HeaderBarProps {
  connected: boolean
  projects: Project[]
  activeProject: Project | null
  onSelectProject: (slug: string) => void
  activePlan: Plan | null
  plans: Plan[]
  onSelectPlan: (slug: string) => void
  showArchived: boolean
  onToggleArchived: () => void
  activeWorkers: number
  currentView?: CockpitView
  onChangeView?: (view: CockpitView) => void
}

interface ViewButton {
  key: CockpitView
  label: string
  icon: React.ReactNode
}

const viewButtons: ViewButton[] = [
  { key: 'dag', label: 'Board', icon: <GitFork className="h-3.5 w-3.5" /> },
  { key: 'projects', label: 'Projects', icon: <FolderKanban className="h-3.5 w-3.5" /> },
  { key: 'docs', label: 'Docs', icon: <BookOpen className="h-3.5 w-3.5" /> },
  { key: 'agents', label: 'Agents', icon: <Bot className="h-3.5 w-3.5" /> },
  { key: 'activity', label: 'Activity', icon: <ScrollText className="h-3.5 w-3.5" /> },
  { key: 'env', label: 'Env', icon: <Cpu className="h-3.5 w-3.5" /> },
]

export const HeaderBar: React.FC<HeaderBarProps> = ({
  connected,
  projects,
  activeProject,
  onSelectProject,
  activePlan,
  plans,
  onSelectPlan,
  showArchived,
  onToggleArchived,
  activeWorkers,
  currentView = 'dag',
  onChangeView,
}) => {
  return (
    <header className="flex h-14 w-full items-center justify-between gap-3 border-b border-zinc-800 bg-zinc-900/90 px-4 backdrop-blur select-none">
      {/* Brand & Project / Plan Selector */}
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex shrink-0 items-center gap-2 font-mono text-sm font-semibold tracking-wider text-zinc-100">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/30">
            <Zap className="h-4 w-4 fill-emerald-400" />
          </div>
          <span>ZenForge</span>
          <span className="rounded bg-zinc-800 px-1.5 py-0.5 text-[10px] text-zinc-400 font-sans">
            v0.2.1
          </span>
        </div>

        <div className="h-4 w-[1px] shrink-0 bg-zinc-800" />

        {/* Project Selector */}
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="text-xs text-zinc-400">Project:</span>
          <select
            value={activeProject?.slug || ''}
            onChange={(e) => onSelectProject(e.target.value)}
            className="max-w-[170px] truncate rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1 text-xs font-mono font-medium text-zinc-200 outline-none hover:border-zinc-700 focus:border-emerald-500"
          >
            {activeProject && !projects.some((p) => p.slug === activeProject.slug) && (
              <option value={activeProject.slug}>{activeProject.name || activeProject.slug}</option>
            )}
            {projects.map((project) => (
              <option key={project.slug} value={project.slug}>
                {project.name || project.slug}
                {project.status === 'ARCHIVED' ? ' (archived)' : ''}
              </option>
            ))}
            {projects.length === 0 && <option value="">(No projects)</option>}
          </select>
        </div>

        {/* Plan Selector & Actions */}
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="text-xs text-zinc-400">Plan:</span>
          <select
            value={activePlan?.slug || ''}
            onChange={(e) => onSelectPlan(e.target.value)}
            className="max-w-[150px] truncate rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1 text-xs font-mono font-medium text-zinc-200 outline-none hover:border-zinc-700 focus:border-violet-500"
          >
            {activePlan && !plans.some((p) => p.slug === activePlan.slug) && (
              <option value={activePlan.slug}>{activePlan.title || activePlan.slug}</option>
            )}
            {plans.map((p) => (
              <option key={p.slug} value={p.slug}>
                {p.title || p.slug}
                {p.archived_at ? ' (archived)' : ''}
              </option>
            ))}
            {plans.length === 0 && !activePlan && <option value="">(No active plans)</option>}
          </select>

          <button
            onClick={onToggleArchived}
            title="Include archived plans in the selector"
            className={`rounded border px-2 py-1 text-[11px] font-medium transition-colors ${
              showArchived
                ? 'border-amber-500/40 bg-amber-500/10 text-amber-300'
                : 'border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-zinc-200'
            }`}
          >
            Archived
          </button>

          {activePlan && (
            <span className="hidden rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-400 ring-1 ring-emerald-500/20 xl:inline">
              {activePlan.status}
            </span>
          )}
        </div>
      </div>

      {/* Top Cockpit View Switcher */}
      {onChangeView && (
        <div className="flex shrink-0 items-center rounded-lg border border-zinc-800 bg-zinc-950 p-0.5 text-xs shadow-inner">
          {viewButtons.map((view) => (
            <button
              key={view.key}
              onClick={() => onChangeView(view.key)}
              title={view.label}
              className={`flex items-center gap-1.5 rounded-md px-2 py-1 font-semibold transition-colors ${
                currentView === view.key
                  ? 'bg-zinc-800 text-white shadow-sm'
                  : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              {view.icon}
              <span className="hidden lg:inline">{view.label}</span>
            </button>
          ))}
        </div>
      )}

      {/* Active Ephemeral Workers Badge & SSE Health Status */}
      <div className="flex shrink-0 items-center gap-3">
        <div className="flex items-center gap-1.5 text-xs font-mono text-zinc-400">
          <Bot className={`h-3.5 w-3.5 ${activeWorkers > 0 ? 'text-violet-400 animate-pulse' : 'text-zinc-500'}`} />
          <span>Active: {activeWorkers}</span>
        </div>

        <div className="hidden h-4 w-[1px] bg-zinc-800 sm:block" />

        {/* Real-time SSE Connection Indicator */}
        <div className="flex items-center gap-2 text-xs">
          <div
            className={`h-2 w-2 rounded-full ${
              connected
                ? 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]'
                : 'bg-rose-500 animate-pulse'
            }`}
          />
          <span className="hidden font-mono text-[11px] text-zinc-300 sm:inline">
            {connected ? 'Port 1123 Active' : 'Connecting...'}
          </span>
          <Activity className="h-3.5 w-3.5 text-zinc-500" />
        </div>
      </div>
    </header>
  )
}
