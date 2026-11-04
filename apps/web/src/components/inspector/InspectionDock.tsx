import React, { useState } from 'react'
import { ChevronRight, ChevronLeft } from 'lucide-react'
import { TaskSpecPane } from './TaskSpecPane'
import { ArtifactLightbox } from './ArtifactLightbox'
import type { Task, Artifact } from '../../types/contracts'

interface InspectionDockProps {
  selectedTask: Task | null
  allTasks: Task[]
  artifacts: Artifact[]
  onSelectTaskId?: (taskId: string) => void
  collapsed: boolean
  onToggleCollapse: () => void
}

export const InspectionDock: React.FC<InspectionDockProps> = ({
  selectedTask,
  allTasks,
  artifacts,
  onSelectTaskId,
  collapsed,
  onToggleCollapse,
}) => {
  const [activePane, setActivePane] = useState<'split' | 'spec' | 'artifacts'>('split')

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

  return (
    <aside className="relative flex w-[500px] flex-col border-l border-zinc-800 bg-zinc-950 select-none">
      {/* Dock Top Toggle Bar */}
      <div className="flex h-8 items-center justify-between border-b border-zinc-800 bg-zinc-900/60 px-3 text-xs text-zinc-400">
        <div className="flex items-center gap-1">
          <button
            onClick={() => setActivePane('split')}
            className={`rounded px-1.5 py-0.5 text-[10px] uppercase font-bold transition-colors ${
              activePane === 'split' ? 'bg-zinc-800 text-white' : 'hover:text-zinc-200'
            }`}
          >
            Split (Card / Hub)
          </button>
          <button
            onClick={() => setActivePane('spec')}
            className={`rounded px-1.5 py-0.5 text-[10px] uppercase font-bold transition-colors ${
              activePane === 'spec' ? 'bg-zinc-800 text-white' : 'hover:text-zinc-200'
            }`}
          >
            Spec Card
          </button>
          <button
            onClick={() => setActivePane('artifacts')}
            className={`rounded px-1.5 py-0.5 text-[10px] uppercase font-bold transition-colors ${
              activePane === 'artifacts' ? 'bg-zinc-800 text-white' : 'hover:text-zinc-200'
            }`}
          >
            Multi-Modal Hub
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
        {(activePane === 'split' || activePane === 'spec') && (
          <div className={activePane === 'split' ? 'h-1/2 border-b border-zinc-800 overflow-hidden' : 'h-full overflow-hidden'}>
            <TaskSpecPane
              task={selectedTask}
              allTasks={allTasks}
              onSelectTaskId={onSelectTaskId}
              isMaximized={activePane === 'spec'}
              onToggleMaximize={() =>
                setActivePane(activePane === 'spec' ? 'split' : 'spec')
              }
            />
          </div>
        )}

        {(activePane === 'split' || activePane === 'artifacts') && (
          <div className={activePane === 'split' ? 'h-1/2 overflow-hidden' : 'h-full overflow-hidden'}>
            <ArtifactLightbox
              taskId={selectedTask?.id}
              artifacts={artifacts}
            />
          </div>
        )}
      </div>
    </aside>
  )
}
