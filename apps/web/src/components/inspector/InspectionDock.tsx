import React, { useState, useEffect, useCallback, useRef } from 'react'
import { ChevronRight, ChevronLeft, Layers, Activity, Zap } from 'lucide-react'
import { TaskSpecPane } from './TaskSpecPane'
import { ExecutionTracePane } from './ExecutionTracePane'
import { ClaimPacketPane } from './ClaimPacketPane'
import type { Task, Artifact, ContextPacket } from '../../types/contracts'
import type { ApiResult } from '../../hooks/useZenfAPI'

type DockLens = 'spec' | 'trace' | 'packet'

interface InspectionDockProps {
  selectedTask: Task | null
  allTasks: Task[]
  artifacts: Artifact[]
  onSelectTaskId?: (taskId: string) => void
  collapsed: boolean
  onToggleCollapse: () => void
  fetchTaskPacket?: (taskId: string) => Promise<ApiResult<ContextPacket>>
}

export const InspectionDock: React.FC<InspectionDockProps> = ({
  selectedTask,
  allTasks,
  artifacts,
  onSelectTaskId,
  collapsed,
  onToggleCollapse,
  fetchTaskPacket,
}) => {
  const [activeLens, setActiveLens] = useState<DockLens>('spec')
  const [packet, setPacket] = useState<ContextPacket | null>(null)
  const [packetLoading, setPacketLoading] = useState(false)
  const [packetTaskId, setPacketTaskId] = useState<string | null>(null)

  // Async sequence counter to prevent race conditions on fast task switching (Finding #12)
  const fetchSeqRef = useRef(0)

  const doFetchPacket = useCallback(
    async (taskId: string) => {
      if (!fetchTaskPacket) return
      const seq = ++fetchSeqRef.current
      setPacketLoading(true)
      try {
        const result = await fetchTaskPacket(taskId)
        if (seq === fetchSeqRef.current && result.ok) {
          setPacket(result.data)
          setPacketTaskId(taskId)
        }
      } catch {
        // ignore
      } finally {
        if (seq === fetchSeqRef.current) {
          setPacketLoading(false)
        }
      }
    },
    [fetchTaskPacket]
  )

  // Auto-fetch when packet tab is active and task changes
  useEffect(() => {
    if (activeLens === 'packet' && selectedTask && selectedTask.id !== packetTaskId) {
      setPacket(null)
      setPacketTaskId(null)
      doFetchPacket(selectedTask.id)
    }
  }, [activeLens, selectedTask, packetTaskId, doFetchPacket])

  // Reset packet when task changes
  useEffect(() => {
    if (selectedTask?.id !== packetTaskId) {
      setPacket(null)
      setPacketTaskId(null)
    }
  }, [selectedTask?.id, packetTaskId])

  if (collapsed) {
    return (
      <aside className="flex w-10 flex-col items-center border-l border-zinc-800 bg-zinc-900/60 py-3 select-none">
        <button
          onClick={onToggleCollapse}
          className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white transition"
          title="Expand Inspection Dock"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>
      </aside>
    )
  }

  const lenses: { key: DockLens; label: string; icon: React.ReactNode }[] = [
    { key: 'spec', label: 'Spec', icon: <Layers className="h-3 w-3" /> },
    { key: 'trace', label: 'Execution & Trace', icon: <Activity className="h-3 w-3" /> },
    { key: 'packet', label: 'Claim Packet ⚡', icon: <Zap className="h-3 w-3" /> },
  ]

  return (
    <aside className="relative flex w-[500px] flex-col border-l border-zinc-800 bg-zinc-950 select-none">
      {/* Dock Top Lens Bar */}
      <div className="flex h-9 items-center justify-between border-b border-zinc-800 bg-zinc-900/60 px-3 text-xs text-zinc-400">
        <div className="flex items-center gap-1">
          {lenses.map((lens) => (
            <button
              key={lens.key}
              onClick={() => {
                setActiveLens(lens.key)
                if (lens.key === 'packet' && selectedTask && !packet) {
                  doFetchPacket(selectedTask.id)
                }
              }}
              className={`flex items-center gap-1.5 rounded px-2 py-1 text-[11px] font-semibold transition-colors ${
                activeLens === lens.key
                  ? 'bg-zinc-800 text-white shadow-sm'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/50'
              }`}
            >
              {lens.icon}
              <span>{lens.label}</span>
            </button>
          ))}
        </div>

        <button
          onClick={onToggleCollapse}
          className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white transition"
          title="Collapse Inspector Dock"
        >
          <ChevronRight className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Pane Content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {activeLens === 'spec' && (
          <div className="h-full overflow-hidden">
            <TaskSpecPane
              task={selectedTask}
              allTasks={allTasks}
              artifacts={artifacts}
              onSelectTaskId={onSelectTaskId}
              fetchTaskPacket={fetchTaskPacket}
              onViewArtifacts={() => setActiveLens('trace')}
            />
          </div>
        )}

        {activeLens === 'trace' && (
          <div className="h-full overflow-hidden">
            <ExecutionTracePane task={selectedTask} artifacts={artifacts} />
          </div>
        )}

        {activeLens === 'packet' && (
          <div className="h-full overflow-hidden">
            <ClaimPacketPane
              packet={packet}
              loading={packetLoading}
              onFetch={() => selectedTask && doFetchPacket(selectedTask.id)}
            />
          </div>
        )}
      </div>
    </aside>
  )
}
