import React, { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { CheckCircle2, PlayCircle, Clock, AlertCircle, Terminal } from 'lucide-react'
import type { Task, TaskStatus } from '../../types/contracts'

export interface TaskNodeData {
  task: Task
  isSelected: boolean
  onSelect: (task: Task) => void
}

const statusStyles: Record<
  TaskStatus,
  {
    border: string
    bg: string
    text: string
    glow: string
    icon: React.ReactNode
  }
> = {
  COMPLETED: {
    border: 'border-emerald-500/40',
    bg: 'bg-emerald-950/20',
    text: 'text-emerald-400',
    glow: 'hover:shadow-[0_0_12px_rgba(16,185,129,0.2)]',
    icon: <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />,
  },
  RUNNING: {
    border: 'border-violet-500/80',
    bg: 'bg-violet-950/30',
    text: 'text-violet-400',
    glow: 'shadow-[0_0_16px_rgba(139,92,246,0.3)] animate-pulse',
    icon: <PlayCircle className="h-3.5 w-3.5 text-violet-400 animate-spin" />,
  },
  READY: {
    border: 'border-amber-500/40',
    bg: 'bg-amber-950/20',
    text: 'text-amber-400',
    glow: 'hover:shadow-[0_0_12px_rgba(245,158,11,0.2)]',
    icon: <Clock className="h-3.5 w-3.5 text-amber-400" />,
  },
  FAILED: {
    border: 'border-rose-500/60',
    bg: 'bg-rose-950/30',
    text: 'text-rose-400',
    glow: 'shadow-[0_0_14px_rgba(244,63,94,0.3)]',
    icon: <AlertCircle className="h-3.5 w-3.5 text-rose-400" />,
  },
  PENDING: {
    border: 'border-zinc-800',
    bg: 'bg-zinc-900/40',
    text: 'text-zinc-500',
    glow: '',
    icon: <div className="h-2 w-2 rounded-full bg-zinc-600" />,
  },
  BLOCKED: {
    border: 'border-zinc-800',
    bg: 'bg-zinc-900/40',
    text: 'text-zinc-500',
    glow: '',
    icon: <div className="h-2 w-2 rounded-full bg-zinc-600" />,
  },
  CANCELLED: {
    border: 'border-zinc-800',
    bg: 'bg-zinc-900/40',
    text: 'text-zinc-600',
    glow: '',
    icon: <div className="h-2 w-2 rounded-full bg-zinc-700" />,
  },
}

export const TaskNode = memo(({ data }: { data: TaskNodeData }) => {
  const { task, isSelected, onSelect } = data
  const style = statusStyles[task.status] || statusStyles.PENDING

  return (
    <div
      onClick={() => onSelect(task)}
      className={`w-64 rounded-lg border p-3 transition-all cursor-pointer select-none backdrop-blur-sm ${
        style.border
      } ${style.bg} ${style.glow} ${
        isSelected
          ? 'ring-2 ring-violet-500 shadow-[0_0_18px_rgba(139,92,246,0.4)]'
          : ''
      }`}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="!h-2.5 !w-2.5 !border-2 !border-zinc-900 !bg-zinc-500"
      />

      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5 font-mono text-xs font-bold text-zinc-100">
          {style.icon}
          <span>{task.id}</span>
        </div>
        <span
          className={`rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider ${style.text}`}
        >
          {task.status}
        </span>
      </div>

      {/* Title */}
      <div className="mt-1.5 line-clamp-2 text-xs font-medium text-zinc-200">
        {task.title}
      </div>

      {/* Skill Command & Harness */}
      <div className="mt-2.5 flex items-center justify-between border-t border-zinc-800/80 pt-2 text-[10px] text-zinc-400 font-mono">
        <div className="flex items-center gap-1 truncate max-w-[150px]">
          <Terminal className="h-3 w-3 text-zinc-500 shrink-0" />
          <span className="truncate text-zinc-300">{task.skill_command}</span>
        </div>
        <span className="text-zinc-500 uppercase">{task.harness}</span>
      </div>

      {/* Dependencies */}
      {task.dependencies && task.dependencies.length > 0 && (
        <div className="mt-1 flex items-center gap-1 text-[9px] text-zinc-500 font-mono truncate">
          <span>prereq:</span>
          <span className="text-zinc-400 truncate">
            {task.dependencies.join(', ')}
          </span>
        </div>
      )}

      <Handle
        type="source"
        position={Position.Right}
        className="!h-2.5 !w-2.5 !border-2 !border-zinc-900 !bg-zinc-500"
      />
    </div>
  )
})

TaskNode.displayName = 'TaskNode'
