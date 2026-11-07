import React from 'react'
import type { Task, TaskStatus } from '../../types/contracts'
import { TASK_STATUS_CONFIG } from './taskStatusTheme'

interface SwimlanesViewProps {
  tasks: Task[]
  selectedTaskId: string | null
  onSelectTask: (task: Task) => void
}

const columnStatuses: { status: TaskStatus; label: string }[] = [
  { status: 'READY', label: 'Ready for Dispatch' },
  { status: 'RUNNING', label: 'Actively Running' },
  { status: 'AWAITING_REVIEW', label: 'Awaiting Review' },
  { status: 'AWAITING_MERGE', label: 'Awaiting Merge' },
  { status: 'CONFLICT', label: 'Merge Conflict' },
  { status: 'COMPLETED', label: 'Verified & Done' },
  { status: 'BLOCKED', label: 'Blocked' },
  { status: 'PENDING', label: 'Prerequisites Pending' },
  { status: 'FAILED', label: 'Failed / Drifted' },
  { status: 'CANCELLED', label: 'Cancelled' },
]

export const SwimlanesView: React.FC<SwimlanesViewProps> = ({
  tasks,
  selectedTaskId,
  onSelectTask,
}) => {
  const getTasksByStatus = (status: TaskStatus) =>
    tasks.filter((t) => t.status === status)

  return (
    <div className="flex h-full w-full gap-4 overflow-x-auto p-4 select-none snap-x">
      {columnStatuses.map(({ status, label }) => {
        const columnTasks = getTasksByStatus(status)
        const theme = TASK_STATUS_CONFIG[status]

        return (
          <div
            key={status}
            data-testid={`lane-${status}`}
            className="flex w-72 flex-col rounded-lg border border-zinc-800 bg-zinc-900/40 shrink-0 snap-start"
          >
            {/* Column Header */}
            <div className="flex items-center justify-between border-b border-zinc-800 px-3 py-2.5">
              <span className={`text-xs font-bold uppercase tracking-wider ${theme.text}`}>
                {label}
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
                        {theme.icon}
                        <span>{t.id}</span>
                      </div>
                      <span className="text-[10px] uppercase text-zinc-500">
                        {t.harness}
                      </span>
                    </div>

                    <p className="mt-1.5 text-xs text-zinc-300 font-medium line-clamp-2">
                      {t.title}
                    </p>
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
