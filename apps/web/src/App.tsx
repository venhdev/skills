import { useState, useEffect, useCallback, useRef } from 'react'
import { HeaderBar, type CockpitView } from './components/header/HeaderBar'
import { PlanTree } from './components/sidebar/PlanTree'
import { DAGCanvas } from './components/stage/DAGCanvas'
import { InspectionDock } from './components/inspector/InspectionDock'
import { GlobalEnvView } from './components/env/GlobalEnvView'
import { useSSE } from './hooks/useSSE'
import { useZenfdAPI } from './hooks/useZenfdAPI'
import type { Plan, Task, Artifact } from './types/contracts'

export default function App() {
  const [plans, setPlans] = useState<string[]>([])
  const [activePlan, setActivePlan] = useState<Plan | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])
  const [currentView, setCurrentView] = useState<CockpitView>('dag')

  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)
  const [focusTarget, setFocusTarget] = useState<{ id: string; timestamp: number } | null>(null)
  const [artifacts, setArtifacts] = useState<Artifact[]>([])

  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [dockCollapsed, setDockCollapsed] = useState(false)

  const activePlanRef = useRef<Plan | null>(null)
  activePlanRef.current = activePlan

  const {
    fetchPlans,
    fetchPlan,
    fetchTasks,
    fetchTaskArtifacts,
    fetchGlobalEnv,
  } = useZenfdAPI()

  const refreshArtifacts = useCallback(
    async (taskId: string | null) => {
      if (!taskId) {
        setArtifacts([])
        return
      }
      const arts = await fetchTaskArtifacts(taskId)
      setArtifacts(arts || [])
    },
    [fetchTaskArtifacts]
  )

  // Refresh artifacts whenever selectedTaskId changes
  useEffect(() => {
    refreshArtifacts(selectedTaskId)
  }, [selectedTaskId, refreshArtifacts])

  // Real-time synchronization with daemon SQLite state
  const refreshData = useCallback(async () => {
    const taskList = await fetchTasks()
    if (taskList === null) {
      // Network error occurred. Retain current state safely without wiping.
      return
    }

    // The plans endpoint is the source of truth for the selector and includes
    // plans that do not have any tasks yet.
    const availablePlans = await fetchPlans()
    setPlans(availablePlans)

    // Determine target plan. When nothing is selected, prefer a plan that has
    // tasks so the canvas is not empty on first load, while still exposing
    // plans without tasks through the selector.
    let targetPlanSlug = activePlanRef.current?.slug
    if (!targetPlanSlug || !availablePlans.includes(targetPlanSlug)) {
      const planWithTasks = taskList.find((t) => availablePlans.includes(t.plan_id))?.plan_id
      targetPlanSlug = planWithTasks || availablePlans[0]
    }

    if (!targetPlanSlug) {
      setTasks([])
      setActivePlan(null)
      setSelectedTaskId(null)
      setArtifacts([])
      return
    }

    const planData = await fetchPlan(targetPlanSlug)
    if (planData?.plan) {
      setActivePlan(planData.plan)
      const planTasks = planData.tasks || []
      setTasks(planTasks)

      // Select task
      setSelectedTaskId((prev) => {
        if (prev && planTasks.some((t) => t.id === prev)) {
          return prev
        }
        return planTasks[0]?.id || null
      })
    } else {
      // The selected plan disappeared (e.g. deleted externally).
      setActivePlan(null)
      setTasks([])
      setSelectedTaskId(null)
      setArtifacts([])
    }
  }, [fetchTasks, fetchPlans, fetchPlan])

  // Initial fetch on mount
  useEffect(() => {
    refreshData()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Safety-net reconcile when the tab regains focus. Primary updates arrive
  // through the daemon's state:changed event (which covers MCP/CLI writers).
  useEffect(() => {
    const reconcile = () => {
      if (document.visibilityState === 'visible') {
        refreshData()
      }
    }
    document.addEventListener('visibilitychange', reconcile)
    window.addEventListener('focus', reconcile)
    const interval = window.setInterval(reconcile, 30000)
    return () => {
      document.removeEventListener('visibilitychange', reconcile)
      window.removeEventListener('focus', reconcile)
      window.clearInterval(interval)
    }
  }, [refreshData])

  // Real-time SSE updates
  const { connected } = useSSE({
    onEvent: (evt) => {
      if (
        evt.type === 'task:status' ||
        evt.type === 'task:created' ||
        evt.type === 'task:dependencies_updated' ||
        evt.type === 'task:notes_updated' ||
        evt.type === 'plan:created' ||
        evt.type === 'plan:deleted' ||
        evt.type === 'state:changed' ||
        evt.type === 'run:killed'
      ) {
        refreshData()
      }
      if (evt.type === 'task:artifact_added') {
        refreshArtifacts(selectedTaskId)
      }
    },
  })

  const selectedTask = tasks.find((t) => t.id === selectedTaskId) || null
  const activeWorkerCount = tasks.filter((t) => t.status === 'RUNNING').length

  const handleSelectPlan = async (slug: string) => {
    const data = await fetchPlan(slug)
    if (data?.plan) {
      setActivePlan(data.plan)
      setTasks(data.tasks || [])
      const firstTask = data.tasks?.[0]
      if (firstTask) {
        setSelectedTaskId(firstTask.id)
        setFocusTarget({ id: firstTask.id, timestamp: Date.now() })
      } else {
        setSelectedTaskId(null)
      }
    }
  }

  const handleSelectTask = (task: Task, source: 'sidebar' | 'canvas' | 'dock' = 'sidebar') => {
    setSelectedTaskId(task.id)
    if (source !== 'canvas') {
      setFocusTarget({ id: task.id, timestamp: Date.now() })
    }
  }

  const handleSelectTaskId = (taskId: string) => {
    const t = tasks.find((x) => x.id === taskId)
    if (t) {
      handleSelectTask(t, 'dock')
    } else {
      setSelectedTaskId(taskId)
      setFocusTarget({ id: taskId, timestamp: Date.now() })
    }
  }

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden bg-zinc-950 font-sans text-zinc-100">
      {/* 1. Header Bar with Plan CRUD & Active Worker Count */}
      <HeaderBar
        connected={connected}
        activePlan={activePlan}
        plans={plans}
        onSelectPlan={handleSelectPlan}
        activeWorkers={activeWorkerCount}
        currentView={currentView}
        onChangeView={(v) => setCurrentView(v)}
      />

      {/* 2. Cockpit Body */}
      {currentView === 'dag' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          {/* Left: Task Tree Sidebar */}
          <PlanTree
            tasks={tasks}
            selectedTaskId={selectedTaskId}
            onSelectTask={(task) => handleSelectTask(task, 'sidebar')}
            collapsed={sidebarCollapsed}
            onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)}
          />

          {/* Center: Topological DAG Flow Canvas (read-only view) */}
          <DAGCanvas
            tasks={tasks}
            selectedTaskId={selectedTaskId}
            onSelectTask={handleSelectTask}
            focusTarget={focusTarget}
          />

          {/* Right: Read-only Inspection Dock (spec, artifacts & logs) */}
          <InspectionDock
            selectedTask={selectedTask}
            allTasks={tasks}
            artifacts={artifacts}
            onSelectTaskId={handleSelectTaskId}
            collapsed={dockCollapsed}
            onToggleCollapse={() => setDockCollapsed(!dockCollapsed)}
          />
        </div>
      )}

      {currentView === 'env' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <GlobalEnvView onFetchEnv={fetchGlobalEnv} />
        </div>
      )}
    </div>
  )
}
