import React, { useState, useMemo } from 'react'
import {
  ChevronLeft,
  ChevronRight,
  Search,
  CheckCircle2,
  Clock,
  PlayCircle,
  AlertCircle,
  AlertTriangle,
  GitMerge,
  FolderGit2,
} from 'lucide-react'
import type { Task, TaskStatus } from '../../types/contracts'

interface PlanTreeProps {
  tasks: Task[]
  selectedTaskId: string | null
  onSelectTask: (task: Task, source?: 'sidebar') => void
  collapsed: boolean
  onToggleCollapse: () => void
}

export const PlanTree: React.FC<PlanTreeProps> = ({
  tasks,
  selectedTaskId,
  onSelectTask,
  collapsed,
  onToggleCollapse,
}) => {
  const [searchTerm, setSearchTerm] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('ALL')

  const statusCounts = useMemo(() => {
    const counts: Record<string, number> = {
      ALL: tasks.length,
      RUNNING: 0,
      READY: 0,
      AWAITING_REVIEW: 0,
      AWAITING_MERGE: 0,
      CONFLICT: 0,
      COMPLETED: 0,
      FAILED: 0,
      PENDING: 0,
    }
    tasks.forEach((t) => {
      counts[t.status] = (counts[t.status] || 0) + 1
    })
    return counts
  }, [tasks])

  const filteredTasks = useMemo(() => {
    return tasks.filter((t) => {
      const matchesSearch =
        t.id.toLowerCase().includes(searchTerm.toLowerCase()) ||
        t.title.toLowerCase().includes(searchTerm.toLowerCase())
      const matchesStatus =
        statusFilter === 'ALL' || t.status === statusFilter
      return matchesSearch && matchesStatus
    })
  }, [tasks, searchTerm, statusFilter])

  const renderStatusIcon = (status: TaskStatus) => {
    switch (status) {
      case 'COMPLETED':
        return <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />
      case 'RUNNING':
        return (
          <PlayCircle className="h-3.5 w-3.5 text-violet-400 animate-spin" />
        )
      case 'READY':
        return <Clock className="h-3.5 w-3.5 text-amber-400" />
      case 'AWAITING_REVIEW':
        return <AlertTriangle className="h-3.5 w-3.5 text-orange-400" />
      case 'AWAITING_MERGE':
        return <GitMerge className="h-3.5 w-3.5 text-cyan-400" />
      case 'CONFLICT':
        return <AlertCircle className="h-3.5 w-3.5 text-rose-400" />
      case 'FAILED':
        return <AlertCircle className="h-3.5 w-3.5 text-rose-400" />
      default:
        return <div className="h-2 w-2 rounded-full bg-zinc-600" />
    }
  }

  if (collapsed) {
    return (
      <aside className="flex w-12 flex-col items-center border-r border-zinc-800 bg-zinc-900/60 py-3 select-none">
        <button
          onClick={onToggleCollapse}
          className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white"
          title="Expand Task Explorer"
        >
          <ChevronRight className="h-4 w-4" />
        </button>
        <div className="mt-4 flex flex-col gap-2">
          {tasks.slice(0, 10).map((t) => (
            <button
              key={t.id}
              onClick={() => onSelectTask(t, 'sidebar')}
              className={`flex h-7 w-7 items-center justify-center rounded text-xs transition-colors ${
                t.id === selectedTaskId
                  ? 'bg-violet-600 text-white'
                  : 'hover:bg-zinc-800 text-zinc-400'
              }`}
              title={`${t.id}: ${t.title}`}
            >
              {renderStatusIcon(t.status)}
            </button>
          ))}
        </div>
      </aside>
    )
  }

  return (
    <aside className="flex w-72 flex-col border-r border-zinc-800 bg-zinc-900/60 select-none">
      {/* Sidebar Header */}
      <div className="flex h-10 items-center justify-between border-b border-zinc-800 px-3">
        <div className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-zinc-400">
          <FolderGit2 className="h-3.5 w-3.5 text-zinc-400" />
          <span>Task Explorer</span>
          <span className="rounded bg-zinc-800 px-1 py-0.2 text-[10px] text-zinc-400">
            {tasks.length}
          </span>
        </div>
        <button
          onClick={onToggleCollapse}
          className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white"
          title="Collapse Panel"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>
      </div>

      {/* Search Input */}
      <div className="border-b border-zinc-800 p-2">
        <div className="flex items-center gap-1.5 rounded border border-zinc-800 bg-zinc-950 px-2 py-1 text-xs text-zinc-300 focus-within:border-violet-500">
          <Search className="h-3.5 w-3.5 text-zinc-500" />
          <input
            type="text"
            placeholder="Filter tasks by ID or name..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full bg-transparent outline-none placeholder:text-zinc-600 font-mono text-xs"
          />
        </div>
      </div>

      {/* Filter Tabs */}
      <div className="flex gap-1 border-b border-zinc-800 p-1.5 text-[10px] font-medium overflow-x-auto">
        {['ALL', 'RUNNING', 'READY', 'COMPLETED', 'FAILED'].map((st) => (
          <button
            key={st}
            onClick={() => setStatusFilter(st)}
            className={`rounded px-2 py-0.5 whitespace-nowrap transition-colors ${
              statusFilter === st
                ? 'bg-zinc-800 text-white font-semibold'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            {st} ({statusCounts[st] || 0})
          </button>
        ))}
      </div>

      {/* Task List */}
      <div className="flex-1 overflow-y-auto p-1.5 space-y-1">
        {filteredTasks.map((t) => {
          const isSelected = t.id === selectedTaskId
          return (
            <div
              key={t.id}
              onClick={() => onSelectTask(t, 'sidebar')}
              className={`group flex cursor-pointer flex-col rounded-md border p-2 transition-all ${
                isSelected
                  ? 'border-violet-500/60 bg-violet-500/10 shadow-[0_0_12px_rgba(139,92,246,0.15)]'
                  : 'border-zinc-800/80 bg-zinc-900/40 hover:border-zinc-750 hover:bg-zinc-850/50'
              }`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  {renderStatusIcon(t.status)}
                  <span className="font-mono text-xs font-semibold text-zinc-200">
                    {t.id}
                  </span>
                </div>
                <span
                  className={`text-[9px] font-bold uppercase tracking-wider ${
                    t.status === 'COMPLETED'
                      ? 'text-emerald-400'
                      : t.status === 'RUNNING'
                      ? 'text-violet-400'
                      : t.status === 'READY'
                      ? 'text-amber-400'
                      : t.status === 'FAILED'
                      ? 'text-rose-400'
                      : 'text-zinc-500'
                  }`}
                >
                  {t.status}
                </span>
              </div>
              <p className="mt-1 line-clamp-1 text-xs text-zinc-400 group-hover:text-zinc-300">
                {t.title}
              </p>
              {t.dependencies && t.dependencies.length > 0 && (
                <div className="mt-1.5 flex items-center gap-1 text-[10px] text-zinc-500 font-mono">
                  <span>deps:</span>
                  <span className="truncate">
                    {t.dependencies.join(', ')}
                  </span>
                </div>
              )}
            </div>
          )
        })}

        {filteredTasks.length === 0 && (
          <div className="py-8 text-center text-xs text-zinc-500">
            No matching tasks found.
          </div>
        )}
      </div>
    </aside>
  )
}
