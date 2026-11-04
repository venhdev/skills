import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import {
  Terminal,
  Zap,
  Bot,
  Crown,
  Unplug,
} from 'lucide-react'
import type { Task } from '../../types/contracts'
import { TASK_STATUS_CONFIG } from './taskStatusTheme'

export interface TaskNodeData {
  task: Task
  isSelected: boolean
  isFocusNeighbor?: boolean
  isBlastRadius?: boolean
  isDimmed?: boolean
  blastDepth?: number
  onSelect: (task: Task) => void
}

export const TaskNode = memo(({ data }: { data: TaskNodeData }) => {
  const {
    task,
    isSelected,
    isFocusNeighbor,
    isBlastRadius,
    isDimmed,
    blastDepth,
    onSelect,
  } = data
  const style = TASK_STATUS_CONFIG[task.status] || TASK_STATUS_CONFIG.PENDING
  const blastOpacity =
    isBlastRadius && blastDepth
      ? blastDepth <= 2
        ? 'opacity-80'
        : blastDepth === 3
        ? 'opacity-60'
        : 'opacity-40'
      : ''

  return (
    <div
      onClick={() => onSelect(task)}
      className={`w-64 rounded-lg border p-3 transition-all cursor-pointer select-none backdrop-blur-sm ${
        style.border
      } ${style.bg} ${style.glow} ${
        isSelected
          ? 'ring-2 ring-violet-500 shadow-[0_0_18px_rgba(139,92,246,0.4)]'
          : isFocusNeighbor
          ? 'ring-1 ring-violet-500/45 shadow-[0_0_14px_rgba(139,92,246,0.2)]'
          : isBlastRadius
          ? 'ring-1 ring-amber-500/80 shadow-[0_0_16px_rgba(245,158,11,0.25)] border-amber-500/50'
          : ''
      } ${isDimmed ? 'opacity-30' : blastOpacity}`}
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
        <div className="flex items-center gap-1">
          {isBlastRadius && !isSelected && (
            <span
              className="flex items-center gap-0.5 rounded bg-amber-500/20 px-1 py-0.2 text-[9px] font-bold text-amber-300"
              title="Downstream Blast Radius (affected if upstream changes)"
            >
              <Zap className="h-2.5 w-2.5 text-amber-400" />
              <span>RADIUS</span>
            </span>
          )}
          <span
            className={`rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider ${style.text}`}
          >
            {task.status}
          </span>
        </div>
      </div>

      {/* Title */}
      <div className="mt-1.5 line-clamp-2 text-xs font-medium text-zinc-200">
        {task.title}
      </div>

      {/* Ephemeral Worker & Orchestrator Badges */}
      {task.status === 'RUNNING' && (
        <div className="mt-2 space-y-1 border-t border-zinc-800/80 pt-1.5">
          <div className="flex items-center gap-1 text-[10px] text-violet-300 font-mono truncate">
            <Bot className="h-3 w-3 text-violet-400 shrink-0" />
            <span className="truncate">{task.claimed_by || 'Worker'}</span>
            {task.pid ? <span className="text-[9px] text-zinc-500">({task.pid})</span> : null}
          </div>
          {task.orchestrator_id && (
            <div className="flex items-center gap-1 text-[9px] text-amber-300/80 font-mono truncate">
              <Crown className="h-2.5 w-2.5 text-amber-400 shrink-0" />
              <span className="truncate">{task.orchestrator_id}</span>
            </div>
          )}
        </div>
      )}

      {task.status === 'SUSPENDED' && (
        <div className="mt-2 flex items-center gap-1 rounded bg-amber-500/10 px-1.5 py-0.5 text-[9px] font-mono text-amber-400 border border-amber-500/20">
          <Unplug className="h-2.5 w-2.5 shrink-0" />
          <span className="truncate">PID Disconnected</span>
        </div>
      )}

      {task.status === 'COMPLETED' && task.claimed_by && (
        <div className="mt-2 flex items-center gap-1 text-[9px] text-emerald-400/80 font-mono truncate border-t border-zinc-800/80 pt-1">
          <span>✓ by {task.claimed_by}</span>
        </div>
      )}

      {/* Skill Command & Harness */}
      <div className="mt-2 flex items-center justify-between border-t border-zinc-800/80 pt-1.5 text-[10px] text-zinc-400 font-mono">
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
