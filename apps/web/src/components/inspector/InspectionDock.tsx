import React, { useState, useEffect, useCallback } from 'react'
import { ChevronRight, ChevronLeft } from 'lucide-react'
import { TaskSpecPane } from './TaskSpecPane'
import { ClaimPacketPane } from './ClaimPacketPane'
import { ArtifactLightbox } from './ArtifactLightbox'
import type { Task, Artifact, ContextPacket } from '../../types/contracts'
import type { ApiResult } from '../../hooks/useZenfdAPI'

type DockPane = 'split' | 'spec' | 'packet' | 'artifacts'

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
  const [activePane, setActivePane] = useState<DockPane>('split')
  const [packet, setPacket] = useState<ContextPacket | null>(null)
  const [packetLoading, setPacketLoading] = useState(false)
  const [packetTaskId, setPacketTaskId] = useState<string | null>(null)

  // Fetch packet when switching to packet tab or task changes while on packet tab
  const doFetchPacket = useCallback(async () => {
    if (!selectedTask || !fetchTaskPacket) return
    if (packetTaskId === selectedTask.id && packet) return // already loaded
    setPacketLoading(true)
    try {
      const result = await fetchTaskPacket(selectedTask.id)
      if (result.ok) {
        setPacket(result.data)
        setPacketTaskId(selectedTask.id)
      }
    } catch {
      // ignore
    } finally {
      setPacketLoading(false)
    }
  }, [selectedTask, fetchTaskPacket, packetTaskId, packet])

  // Auto-fetch when packet tab is active and task changes
  useEffect(() => {
    if (activePane === 'packet' && selectedTask && selectedTask.id !== packetTaskId) {
      setPacket(null)
      setPacketTaskId(null)
      doFetchPacket()
    }
  }, [activePane, selectedTask, packetTaskId, doFetchPacket])

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
          className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white"
          title="Expand Inspection Dock"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>
      </aside>
    )
  }

  const tabs: { key: DockPane; label: string }[] = [
    { key: 'split', label: 'Split (Card / Hub)' },
    { key: 'spec', label: 'Spec Card' },
    { key: 'packet', label: 'Claim Packet ⚡' },
    { key: 'artifacts', label: 'Multi-Modal Hub' },
  ]

  // Dynamic split: give more space to spec when no artifacts
  const hasArtifacts = artifacts.length > 0
  const specSplitClass = hasArtifacts ? 'h-1/2' : 'h-3/4'
  const artSplitClass = hasArtifacts ? 'h-1/2' : 'h-1/4'

  return (
    <aside className="relative flex w-[500px] flex-col border-l border-zinc-800 bg-zinc-950 select-none">
      {/* Dock Top Toggle Bar */}
      <div className="flex h-8 items-center justify-between border-b border-zinc-800 bg-zinc-900/60 px-3 text-xs text-zinc-400">
        <div className="flex items-center gap-1">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => {
                setActivePane(tab.key)
                if (tab.key === 'packet' && selectedTask && !packet) {
                  doFetchPacket()
                }
              }}
              className={`rounded px-1.5 py-0.5 text-[10px] uppercase font-bold transition-colors ${
                activePane === tab.key ? 'bg-zinc-800 text-white' : 'hover:text-zinc-200'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <button
          onClick={onToggleCollapse}
          className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white"
          title="Collapse Inspector Dock"
        >
          <ChevronRight className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Pane Content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {(activePane === 'split' || activePane === 'spec') && (
          <div className={activePane === 'split' ? `${specSplitClass} border-b border-zinc-800 overflow-hidden` : 'h-full overflow-hidden'}>
            <TaskSpecPane
              task={selectedTask}
              allTasks={allTasks}
              onSelectTaskId={onSelectTaskId}
              isMaximized={activePane === 'spec'}
              onToggleMaximize={() =>
                setActivePane(activePane === 'spec' ? 'split' : 'spec')
              }
              fetchTaskPacket={fetchTaskPacket}
            />
          </div>
        )}

        {activePane === 'packet' && (
          <div className="h-full overflow-hidden">
            <ClaimPacketPane
              packet={packet}
              loading={packetLoading}
              onFetch={doFetchPacket}
            />
          </div>
        )}

        {(activePane === 'split' || activePane === 'artifacts') && (
          <div className={activePane === 'split' ? `${artSplitClass} overflow-hidden` : 'h-full overflow-hidden'}>
            <ArtifactLightbox artifacts={artifacts} />
          </div>
        )}
      </div>
    </aside>
  )
}
