import React, { useState } from 'react'
import { ChevronRight, ChevronLeft } from 'lucide-react'
import { TerminalPane } from './TerminalPane'
import { ArtifactLightbox } from './ArtifactLightbox'
import type { Task, Artifact } from '../../types/contracts'

interface InspectionDockProps {
  selectedTask: Task | null
  artifacts: Artifact[]
  onKillProcess?: (runId: string) => void
  collapsed: boolean
  onToggleCollapse: () => void
}

export const InspectionDock: React.FC<InspectionDockProps> = ({
  selectedTask,
  artifacts,
  onKillProcess,
  collapsed,
  onToggleCollapse,
}) => {
  const [activePane, setActivePane] = useState<'split' | 'terminal' | 'artifacts'>('split')

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

  const runId = selectedTask ? `run-${selectedTask.id}` : undefined

  return (
    <aside className="relative flex w-[480px] flex-col border-l border-zinc-800 bg-zinc-950 select-none">
      {/* Dock Top Toggle Bar */}
      <div className="flex h-8 items-center justify-between border-b border-zinc-800 bg-zinc-900/60 px-3 text-xs text-zinc-400">
        <div className="flex items-center gap-1">
          <button
            onClick={() => setActivePane('split')}
            className={`rounded px-1.5 py-0.5 text-[10px] uppercase font-bold transition-colors ${
              activePane === 'split' ? 'bg-zinc-800 text-white' : 'hover:text-zinc-200'
            }`}
          >
            Split (50/50)
          </button>
          <button
            onClick={() => setActivePane('terminal')}
            className={`rounded px-1.5 py-0.5 text-[10px] uppercase font-bold transition-colors ${
              activePane === 'terminal' ? 'bg-zinc-800 text-white' : 'hover:text-zinc-200'
            }`}
          >
            Terminal Only
          </button>
          <button
            onClick={() => setActivePane('artifacts')}
            className={`rounded px-1.5 py-0.5 text-[10px] uppercase font-bold transition-colors ${
              activePane === 'artifacts' ? 'bg-zinc-800 text-white' : 'hover:text-zinc-200'
            }`}
          >
            Artifacts Only
          </button>
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
        {(activePane === 'split' || activePane === 'terminal') && (
          <div className={activePane === 'split' ? 'h-1/2 border-b border-zinc-800' : 'h-full'}>
            <TerminalPane
              runId={runId}
              taskTitle={selectedTask?.title}
              onKillProcess={onKillProcess}
              isMaximized={activePane === 'terminal'}
              onToggleMaximize={() =>
                setActivePane(activePane === 'terminal' ? 'split' : 'terminal')
              }
            />
          </div>
        )}

        {(activePane === 'split' || activePane === 'artifacts') && (
          <div className={activePane === 'split' ? 'h-1/2' : 'h-full'}>
            <ArtifactLightbox artifacts={artifacts} />
          </div>
        )}
      </div>
    </aside>
  )
}
