import React, { useMemo, useState, useEffect, useRef } from 'react'
import {
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  Background,
  Controls,
  MiniMap,
  type Node,
  type Edge,
  MarkerType,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { GitFork, Kanban } from 'lucide-react'
import { TaskNode } from './TaskNode'
import { SwimlanesView } from './SwimlanesView'
import type { Task } from '../../types/contracts'

interface DAGCanvasProps {
  tasks: Task[]
  selectedTaskId: string | null
  onSelectTask: (task: Task, source?: 'sidebar' | 'canvas' | 'dock') => void
  focusTarget?: { id: string; timestamp: number } | null
}

const nodeTypes = {
  taskNode: TaskNode,
}

// Focus handler sub-component that uses useReactFlow hook
function FlowFocusHandler({
  focusTarget,
}: {
  focusTarget?: { id: string; timestamp: number } | null
}) {
  const { fitView } = useReactFlow()

  useEffect(() => {
    if (!focusTarget?.id) return
    // Smooth auto-focus onto the selected node: 600ms, maxZoom 1.0, padding 0.3
    fitView({
      nodes: [{ id: focusTarget.id }],
      duration: 600,
      padding: 0.3,
      maxZoom: 1.0,
    })
  }, [focusTarget, fitView])

  return null
}

// Compute topological depths for left-to-right DAG layout
function computeLayout(tasks: Task[]): { nodes: Node[]; edges: Edge[] } {
  const taskMap = new Map<string, Task>()
  tasks.forEach((t) => taskMap.set(t.id, t))

  // Calculate topological depth for each task
  const depthMap = new Map<string, number>()
  function getDepth(id: string, visited = new Set<string>()): number {
    if (depthMap.has(id)) return depthMap.get(id)!
    if (visited.has(id)) return 0 // cycle protection
    visited.add(id)

    const task = taskMap.get(id)
    if (!task || !task.dependencies || task.dependencies.length === 0) {
      depthMap.set(id, 0)
      return 0
    }

    let maxPredDepth = 0
    for (const dep of task.dependencies) {
      maxPredDepth = Math.max(maxPredDepth, getDepth(dep, visited) + 1)
    }

    depthMap.set(id, maxPredDepth)
    return maxPredDepth
  }

  tasks.forEach((t) => getDepth(t.id))

  // Group tasks by depth level
  const levels = new Map<number, Task[]>()
  tasks.forEach((t) => {
    const d = depthMap.get(t.id) || 0
    if (!levels.has(d)) levels.set(d, [])
    levels.get(d)!.push(t)
  })

  // Build nodes with (x, y) coordinates
  const nodes: Node[] = []
  levels.forEach((levelTasks, depth) => {
    levelTasks.forEach((task, index) => {
      nodes.push({
        id: task.id,
        type: 'taskNode',
        position: {
          x: depth * 320 + 50,
          y: index * 160 + 50,
        },
        data: {
          task,
          isSelected: false,
          onSelect: () => {},
        },
      })
    })
  })

  // Build directed edges
  const edges: Edge[] = []
  tasks.forEach((task) => {
    if (task.dependencies) {
      task.dependencies.forEach((depId) => {
        if (taskMap.has(depId)) {
          const isRunning = task.status === 'RUNNING'
          const isDone = task.status === 'COMPLETED'

          edges.push({
            id: `edge-${depId}-${task.id}`,
            source: depId,
            target: task.id,
            animated: isRunning,
            style: {
              stroke: isRunning ? '#a855f7' : isDone ? '#10b981' : '#3f3f46',
              strokeWidth: isRunning ? 2.5 : 1.5,
            },
            markerEnd: {
              type: MarkerType.ArrowClosed,
              color: isRunning ? '#a855f7' : isDone ? '#10b981' : '#52525b',
            },
          })
        }
      })
    }
  })

  return { nodes, edges }
}

// Compute topological blast radius: set of all tasks that transitively depend on selectedId
function getDownstreamBlastRadius(selectedId: string | null, tasks: Task[]): Set<string> {
  const downstream = new Set<string>()
  if (!selectedId) return downstream
  const queue = [selectedId]
  while (queue.length > 0) {
    const current = queue.shift()!
    for (const t of tasks) {
      if (t.dependencies && t.dependencies.includes(current) && !downstream.has(t.id)) {
        downstream.add(t.id)
        queue.push(t.id)
      }
    }
  }
  return downstream
}

const DAGCanvasInner: React.FC<DAGCanvasProps> = ({
  tasks,
  selectedTaskId,
  onSelectTask,
  focusTarget,
}) => {
  const [viewMode, setViewMode] = useState<'dag' | 'swimlanes'>('dag')
  const [isMapActive, setIsMapActive] = useState(false)
  const mapTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const handleMapActivity = () => {
    setIsMapActive(true)
    if (mapTimerRef.current) {
      clearTimeout(mapTimerRef.current)
      mapTimerRef.current = null
    }
  }

  const handleMapMouseLeave = () => {
    if (mapTimerRef.current) {
      clearTimeout(mapTimerRef.current)
    }
    // After 3.5s of idle/no mouse, smoothly fade out
    mapTimerRef.current = setTimeout(() => {
      setIsMapActive(false)
    }, 3500)
  }

  useEffect(() => {
    return () => {
      if (mapTimerRef.current) {
        clearTimeout(mapTimerRef.current)
      }
    }
  }, [])

  const { nodes, edges } = useMemo(() => {
    const layout = computeLayout(tasks)
    const blastSet = getDownstreamBlastRadius(selectedTaskId, tasks)

    const updatedNodes = layout.nodes.map((node) => ({
      ...node,
      data: {
        task: (node.data as { task: Task }).task,
        isSelected: node.id === selectedTaskId,
        isBlastRadius: blastSet.has(node.id),
        onSelect: (t: Task) => onSelectTask(t, 'canvas'),
      },
    }))

    const updatedEdges = layout.edges.map((edge) => {
      const isBlastEdge =
        (edge.source === selectedTaskId && blastSet.has(edge.target)) ||
        (blastSet.has(edge.source) && blastSet.has(edge.target))

      if (isBlastEdge) {
        return {
          ...edge,
          animated: true,
          style: {
            ...edge.style,
            stroke: '#f59e0b',
            strokeWidth: 2.5,
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: '#f59e0b',
          },
        }
      }
      return edge
    })

    return { nodes: updatedNodes, edges: updatedEdges }
  }, [tasks, selectedTaskId, onSelectTask])

  return (
    <main className="relative flex flex-1 flex-col h-full w-full min-h-0 overflow-hidden bg-zinc-950">
      {/* View Switcher Bar */}
      <div className="flex h-10 items-center justify-between border-b border-zinc-800 bg-zinc-900/40 px-4 select-none">
        <div className="flex items-center gap-1">
          <button
            onClick={() => setViewMode('dag')}
            className={`flex items-center gap-1.5 rounded px-2.5 py-1 text-xs font-semibold transition-colors ${
              viewMode === 'dag'
                ? 'bg-zinc-800 text-white'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <GitFork className="h-3.5 w-3.5" />
            <span>DAG Flow Canvas</span>
          </button>
          <button
            onClick={() => setViewMode('swimlanes')}
            className={`flex items-center gap-1.5 rounded px-2.5 py-1 text-xs font-semibold transition-colors ${
              viewMode === 'swimlanes'
                ? 'bg-zinc-800 text-white'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Kanban className="h-3.5 w-3.5" />
            <span>Kanban Swimlanes</span>
          </button>
        </div>

        <div className="text-[11px] font-mono text-zinc-500">
          {tasks.length} total tasks
        </div>
      </div>

      {/* Main View Area */}
      <div className="relative flex-1 h-full w-full min-h-0">
        {tasks.length === 0 ? (
          <div className="flex h-full w-full flex-col items-center justify-center p-8 text-center select-none">
            <div className="flex h-16 w-16 items-center justify-center rounded-2xl border border-zinc-800 bg-zinc-900/80 text-zinc-500 mb-4 shadow-xl">
              <GitFork className="h-8 w-8 text-zinc-600" />
            </div>
            <h3 className="text-sm font-semibold text-zinc-200">
              No Active Tasks in Database
            </h3>
            <p className="mt-1 text-xs text-zinc-400 max-w-sm">
              The ZenForge Control Plane is connected and listening on port 1123, but the database is currently empty.
            </p>
            <div className="mt-4 flex flex-col items-center gap-1.5 rounded-lg border border-zinc-800 bg-zinc-900/90 px-4 py-2 text-xs font-mono">
              <span className="text-[10px] text-zinc-500 uppercase tracking-wider font-sans font-bold">
                Push a plan via CLI:
              </span>
              <span className="text-emerald-400">
                zenfd plan push .agents/scratch/plans/zenfd/README.md
              </span>
            </div>
          </div>
        ) : viewMode === 'dag' ? (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.25, includeHiddenNodes: false }}
            minZoom={0.1}
            maxZoom={1.5}
            proOptions={{ hideAttribution: true }}
            className="h-full w-full"
            defaultEdgeOptions={{ type: 'smoothstep' }}
          >
            <FlowFocusHandler focusTarget={focusTarget} />
            <Background color="#27272a" gap={24} size={1} />
            <Controls className="!border-zinc-800 !bg-zinc-900 !text-zinc-300" />
            <MiniMap
              pannable
              zoomable
              bgColor="#09090b"
              maskColor="rgba(24, 24, 27, 0.75)"
              maskStrokeColor="#8b5cf6"
              maskStrokeWidth={1.5}
              onMouseEnter={handleMapActivity}
              onMouseMove={handleMapActivity}
              onMouseLeave={handleMapMouseLeave}
              nodeColor={(node) => {
                const task = (node.data as { task?: Task })?.task
                if (task?.status === 'COMPLETED') return '#10b981'
                if (task?.status === 'RUNNING') return '#8b5cf6'
                if (task?.status === 'READY') return '#f59e0b'
                if (task?.status === 'FAILED') return '#f43f5e'
                return '#3f3f46'
              }}
              className={`!border !border-zinc-800 !bg-zinc-950/90 !rounded-lg shadow-xl cursor-grab active:cursor-grabbing transition-opacity duration-700 ${
                isMapActive ? 'opacity-100' : 'opacity-20 hover:opacity-100'
              }`}
            />
          </ReactFlow>
        ) : (
          <SwimlanesView
            tasks={tasks}
            selectedTaskId={selectedTaskId}
            onSelectTask={(t) => onSelectTask(t, 'canvas')}
          />
        )}
      </div>
    </main>
  )
}

export const DAGCanvas: React.FC<DAGCanvasProps> = (props) => {
  return (
    <ReactFlowProvider>
      <DAGCanvasInner {...props} />
    </ReactFlowProvider>
  )
}
