import React from 'react'
import { Terminal, CheckCircle2, PlayCircle, Clock, AlertCircle } from 'lucide-react'
import type { Task, TaskStatus } from '../../types/contracts'

interface SwimlanesViewProps {
  tasks: Task[]
  selectedTaskId: string | null
  onSelectTask: (task: Task) => void
}

const columns: { status: TaskStatus; label: string; color: string }[] = [
  { status: 'READY', label: 'Ready for Dispatch', color: 'text-amber-400' },
  { status: 'RUNNING', label: 'Actively Running', color: 'text-violet-400' },
  { status: 'COMPLETED', label: 'Verified & Done', color: 'text-emerald-400' },
  { status: 'PENDING', label: 'Prerequisites Pending', color: 'text-zinc-500' },
  { status: 'FAILED', label: 'Failed / Drifted', color: 'text-rose-400' },
]

export const SwimlanesView: React.FC<SwimlanesViewProps> = ({
  tasks,
  selectedTaskId,
  onSelectTask,
}) => {
  const getTasksByStatus = (status: TaskStatus) =>
    tasks.filter((t) => t.status === status)

  const renderIcon = (status: TaskStatus) => {
    switch (status) {
      case 'COMPLETED':
        return <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />
      case 'RUNNING':
        return <PlayCircle className="h-3.5 w-3.5 text-violet-400 animate-spin" />
      case 'READY':
        return <Clock className="h-3.5 w-3.5 text-amber-400" />
      case 'FAILED':
        return <AlertCircle className="h-3.5 w-3.5 text-rose-400" />
      default:
        return <div className="h-2 w-2 rounded-full bg-zinc-600" />
    }
  }

  return (
    <div className="flex h-full w-full gap-4 overflow-x-auto p-4 select-none">
      {columns.map((col) => {
        const columnTasks = getTasksByStatus(col.status)
        return (
          <div
            key={col.status}
            className="flex w-72 flex-col rounded-lg border border-zinc-800 bg-zinc-900/40 shrink-0"
          >
            {/* Column Header */}
            <div className="flex items-center justify-between border-b border-zinc-800 px-3 py-2.5">
              <span className={`text-xs font-bold uppercase tracking-wider ${col.color}`}>
                {col.label}
              </span>
              <span className="rounded bg-zinc-800 px-1.5 py-0.5 text-[10px] font-mono text-zinc-400 font-bold">
                {columnTasks.length}
              </span>
            </div>

            {/* Task Cards */}
            <div className="flex-1 space-y-2 overflow-y-auto p-2">
              {columnTasks.map((t) => {
                const isSelected = t.id === selectedTaskId
                return (
                  <div
                    key={t.id}
                    onClick={() => onSelectTask(t)}
                    className={`cursor-pointer rounded-md border p-3 transition-all ${
                      isSelected
                        ? 'border-violet-500 bg-violet-500/10 shadow-[0_0_12px_rgba(139,92,246,0.2)]'
                        : 'border-zinc-800 bg-zinc-950/70 hover:border-zinc-700 hover:bg-zinc-900'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-1.5 font-mono text-xs font-semibold text-zinc-100">
                        {renderIcon(t.status)}
                        <span>{t.id}</span>
                      </div>
                      <span className="text-[10px] uppercase text-zinc-500">
                        {t.harness}
                      </span>
                    </div>

                    <p className="mt-1.5 text-xs text-zinc-300 font-medium line-clamp-2">
                      {t.title}
                    </p>

                    <div className="mt-2.5 flex items-center justify-between border-t border-zinc-850 pt-1.5 text-[10px] text-zinc-400 font-mono">
                      <div className="flex items-center gap-1 truncate max-w-[170px]">
                        <Terminal className="h-3 w-3 text-zinc-500 shrink-0" />
                        <span className="truncate">{t.skill_command}</span>
                      </div>
                    </div>
                  </div>
                )
              })}

              {columnTasks.length === 0 && (
                <div className="py-8 text-center text-xs text-zinc-600 italic">
                  No tasks in this lane
                </div>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}
