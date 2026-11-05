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
import dagre from '@dagrejs/dagre'
import { GitFork, Kanban, Zap } from 'lucide-react'
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

const NODE_WIDTH = 256

// Task cards grow with their status-specific sections. Feeding an estimated
// height into Dagre keeps nodes from overlapping without needing a DOM
// measurement pass.
function estimateNodeHeight(task: Task): number {
  let height = 120
  if (task.dependencies && task.dependencies.length > 0) height += 16
  if (task.status === 'RUNNING') {
    height += task.orchestrator_id ? 36 : 20
  } else if (task.status === 'COMPLETED' && task.claimed_by) {
    height += 20
  }
  return height
}

// Focus handler sub-component that uses useReactFlow hook
function FlowFocusHandler({
  focusTarget,
  tasks,
}: {
  focusTarget?: { id: string; timestamp: number } | null
  tasks: Task[]
}) {
  const { fitView, getInternalNode } = useReactFlow()
  // Keep the latest task list without making data refreshes restart the
  // camera animation. Only an explicit focusTarget change should move it.
  const tasksRef = useRef(tasks)
  tasksRef.current = tasks

  useEffect(() => {
    if (!focusTarget?.id) return

    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | null = null

    const focus = (remainingRetries: number) => {
      if (cancelled) return

      // getNode() returns the user node (without measured); the internal node
      // is the one that carries the measured width/height.
      const selectedNode = getInternalNode(focusTarget.id)
      if (!selectedNode?.measured?.width) {
        if (remainingRetries > 0) {
          timer = setTimeout(() => focus(remainingRetries - 1), 50)
        }
        return
      }

      // Frame the selected task together with its direct prerequisites and
      // dependents so the focused relations stay in view.
      const focusNodeIds = [...getDirectNeighbors(focusTarget.id, tasksRef.current)]
        .filter((id) => getInternalNode(id))

      if (focusNodeIds.length === 0) return

      const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
      fitView({
        nodes: focusNodeIds.map((id) => ({ id })),
        duration: reduceMotion ? 0 : 450,
        padding: 0.25,
        maxZoom: 1.0,
      })
    }

    // Nodes are usually measured on the next frame after a plan/layout change.
    timer = setTimeout(() => focus(10), 50)

    return () => {
      cancelled = true
      if (timer) clearTimeout(timer)
    }
  }, [focusTarget, fitView, getInternalNode])

  return null
}

// Compute topological depths for left-to-right DAG layout
function computeLayout(tasks: Task[]): {
  nodes: Node[]
  edges: Edge[]
  depthMap: Map<string, number>
} {
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

  // Dagre performs rank assignment, crossing minimization, and variable
  // node-height spacing. Depth is still derived from dependencies because the
  // edge hierarchy must not depend on pixel positions.
  const graph = new dagre.graphlib.Graph()
  graph.setGraph({
    rankdir: 'LR',
    nodesep: 36,
    ranksep: 90,
    marginx: 24,
    marginy: 24,
    acyclicer: 'greedy',
    ranker: 'network-simplex',
  })
  graph.setDefaultEdgeLabel(() => ({}))

  tasks.forEach((task) => {
    graph.setNode(task.id, {
      width: NODE_WIDTH,
      height: estimateNodeHeight(task),
    })
  })
  tasks.forEach((task) => {
    task.dependencies?.forEach((depId) => {
      if (taskMap.has(depId)) {
        graph.setEdge(depId, task.id)
      }
    })
  })

  dagre.layout(graph)

  // Build nodes from Dagre's top-left coordinates.
  const nodes: Node[] = tasks.map((task) => {
    const position = graph.node(task.id) as
      | { x: number; y: number; width: number; height: number }
      | undefined
    const width = position?.width ?? NODE_WIDTH
    const height = position?.height ?? estimateNodeHeight(task)

    return {
      id: task.id,
      type: 'taskNode',
      position: {
        x: (position?.x ?? 0) - width / 2,
        y: (position?.y ?? 0) - height / 2,
      },
      data: {
        task,
        isSelected: false,
        onSelect: () => {},
      },
    }
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
            type: 'smoothstep',
            animated: isRunning,
            style: {
              stroke: isRunning ? '#a855f7' : isDone ? '#10b981' : '#52525b',
              strokeWidth: isRunning ? 2.5 : 2,
            },
            markerEnd: {
              type: MarkerType.ArrowClosed,
              color: isRunning ? '#a855f7' : isDone ? '#10b981' : '#71717a',
              width: 16,
              height: 16,
            },
          })
        }
      })
    }
  })

  return { nodes, edges, depthMap }
}

// Direct neighborhood of the selected task: the task itself, its prerequisites,
// and the tasks that depend on it.
function getDirectNeighbors(selectedId: string | null, tasks: Task[]): Set<string> {
  const neighbors = new Set<string>()
  if (!selectedId) return neighbors

  neighbors.add(selectedId)
  const selected = tasks.find((t) => t.id === selectedId)
  selected?.dependencies?.forEach((depId) => neighbors.add(depId))

  for (const task of tasks) {
    if (task.dependencies?.includes(selectedId)) {
      neighbors.add(task.id)
    }
  }
  return neighbors
}

// BFS depth map for every task downstream of the selected task.
// Depth 1 = direct dependent, depth 2 = its dependents, and so on.
function getDownstreamDepth(selectedId: string | null, tasks: Task[]): Map<string, number> {
  const depths = new Map<string, number>()
  if (!selectedId) return depths

  let frontier = [selectedId]
  let depth = 0

  while (frontier.length > 0) {
    depth += 1
    const next: string[] = []
    for (const current of frontier) {
      for (const task of tasks) {
        if (
          task.dependencies?.includes(current) &&
          task.id !== selectedId &&
          !depths.has(task.id)
        ) {
          depths.set(task.id, depth)
          next.push(task.id)
        }
      }
    }
    frontier = next
  }

  return depths
}

// Blast edges are edges that stay inside the downstream subgraph so the
// selected task and every task it can impact remain connected.
function isDownstreamEdge(
  edge: Edge,
  selectedId: string | null,
  depths: Map<string, number>
): boolean {
  if (!selectedId) return false
  if (edge.source === selectedId) return depths.has(edge.target)
  return depths.has(edge.source) && depths.has(edge.target)
}

const DAGCanvasInner: React.FC<DAGCanvasProps> = ({
  tasks,
  selectedTaskId,
  onSelectTask,
  focusTarget,
}) => {
  const [viewMode, setViewMode] = useState<'dag' | 'swimlanes'>('dag')
  const [showBlast, setShowBlast] = useState(false)
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

  // Dagre only depends on topology and node height, not on titles, notes, or
  // the current selection. Keying the layout on that signature keeps a data
  // refresh from re-running layout when the graph is unchanged.
  const layoutKey = useMemo(
    () =>
      tasks
        .map((t) => `${t.id}:${t.status}:${(t.dependencies || []).join(',')}`)
        .join('|'),
    [tasks]
  )
  const layout = useMemo(() => computeLayout(tasks), [layoutKey])
  const tasksById = useMemo(() => new Map(tasks.map((t) => [t.id, t])), [tasks])

  const { nodes, edges } = useMemo(() => {
    const directNeighbors = getDirectNeighbors(selectedTaskId, tasks)
    const blastDepth = showBlast
      ? getDownstreamDepth(selectedTaskId, tasks)
      : new Map<string, number>()

    const updatedNodes = layout.nodes.map((node) => {
      const isSelected = node.id === selectedTaskId
      const isDirectNeighbor = directNeighbors.has(node.id)
      const blastDepthValue = blastDepth.get(node.id)
      // Depth 1 nodes are immediate dependents and already part of the direct
      // neighborhood; only deeper descendants get the blast badge/ring.
      const isBlastRadius =
        !!selectedTaskId &&
        !isSelected &&
        blastDepthValue !== undefined &&
        blastDepthValue > 1
      // Direct neighbors stay readable; only nodes outside the focused
      // neighborhood (and blast radius when enabled) are dimmed.
      const isDimmed =
        !!selectedTaskId && !isSelected && !isDirectNeighbor && !isBlastRadius

      return {
        ...node,
        data: {
          task: tasksById.get(node.id) ?? (node.data as { task: Task }).task,
          isSelected,
          isFocusNeighbor: !!selectedTaskId && !isSelected && isDirectNeighbor,
          isBlastRadius,
          isDimmed,
          blastDepth: blastDepthValue,
          onSelect: (t: Task) => onSelectTask(t, 'canvas'),
        },
      }
    })

    const updatedEdges = layout.edges.map((edge) => {
      const touchesSelected =
        edge.source === selectedTaskId || edge.target === selectedTaskId
      const isBlastEdge = isDownstreamEdge(edge, selectedTaskId, blastDepth)
      const isRelated = touchesSelected || isBlastEdge
      const sourceDepth = layout.depthMap.get(edge.source)
      const targetDepth = layout.depthMap.get(edge.target)
      // A level edge connects adjacent topological columns. Direct edges that
      // skip columns are visually redundant (the intermediate path already
      // explains the flow) and cause long lines across unrelated nodes.
      const isLevelEdge =
        sourceDepth !== undefined &&
        targetDepth !== undefined &&
        targetDepth === sourceDepth + 1
      const isShortcutEdge = touchesSelected && !isLevelEdge

      // Fade every edge outside the focused neighborhood. Without this, a
      // root selection leaves nearly every dependency line equally bright.
      if (selectedTaskId && !isRelated) {
        return {
          ...edge,
          animated: false,
          zIndex: 0,
          style: {
            ...edge.style,
            stroke: '#3f3f46',
            strokeWidth: 1,
            opacity: 0.06,
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: '#3f3f46',
            width: 10,
            height: 10,
          },
        }
      }

      // Skip-level direct edges are real dependencies, but drawing them as
      // strongly as the adjacent-column path creates long lines across the
      // graph. Keep them visible as a quiet secondary cue.
      if (selectedTaskId && isShortcutEdge) {
        return {
          ...edge,
          animated: false,
          zIndex: 0,
          style: {
            ...edge.style,
            stroke: '#8b5cf6',
            strokeWidth: 1,
            opacity: 0.35,
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: '#8b5cf6',
            width: 10,
            height: 10,
          },
        }
      }

      // Nodes carry the focus; direct edges stay as quiet, solid guides.
      if (selectedTaskId && touchesSelected) {
        return {
          ...edge,
          animated: false,
          zIndex: 1,
          style: {
            ...edge.style,
            stroke: '#8b5cf6',
            strokeWidth: 1.75,
            opacity: 0.55,
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: '#8b5cf6',
            width: 13,
            height: 13,
          },
        }
      }

      if (selectedTaskId && isBlastEdge) {
        const depth = Math.max(
          2,
          blastDepth.get(edge.target) ?? blastDepth.get(edge.source) ?? 2
        )
        const opacity = depth <= 2 ? 0.4 : depth === 3 ? 0.25 : 0.15
        return {
          ...edge,
          animated: false,
          zIndex: 1,
          style: {
            ...edge.style,
            stroke: '#f59e0b',
            strokeWidth: 1.5,
            opacity,
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: '#f59e0b',
            width: 12,
            height: 12,
          },
        }
      }

      return edge
    })

    return { nodes: updatedNodes, edges: updatedEdges }
  }, [layout, tasksById, tasks, selectedTaskId, showBlast, onSelectTask])

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

        <div className="flex items-center gap-3">
          {viewMode === 'dag' && selectedTaskId && (
            <button
              onClick={() => setShowBlast((value) => !value)}
              className={`flex items-center gap-1.5 rounded border px-2.5 py-1 text-xs font-semibold transition-colors ${
                showBlast
                  ? 'border-amber-500/40 bg-amber-500/10 text-amber-300'
                  : 'border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-zinc-200'
              }`}
              title="Show depth-graded downstream blast radius"
            >
              <Zap
                className={`h-3.5 w-3.5 ${
                  showBlast ? 'fill-amber-400 text-amber-400' : 'text-zinc-500'
                }`}
              />
              <span>Blast Radius {showBlast ? 'On' : 'Off'}</span>
            </button>
          )}

          <div className="text-[11px] font-mono text-zinc-500">
            {tasks.length} total tasks
          </div>
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
            nodesConnectable={false}
            deleteKeyCode={null}
            fitView
            fitViewOptions={{ padding: 0.25, includeHiddenNodes: false }}
            minZoom={0.1}
            maxZoom={1.5}
            proOptions={{ hideAttribution: true }}
            className="h-full w-full"
            defaultEdgeOptions={{ type: 'smoothstep' }}
          >
            <FlowFocusHandler focusTarget={focusTarget} tasks={tasks} />
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
