import { useState, useEffect, useCallback } from 'react'
import { HeaderBar } from './components/header/HeaderBar'
import { PlanTree } from './components/sidebar/PlanTree'
import { DAGCanvas } from './components/stage/DAGCanvas'
import { InspectionDock } from './components/inspector/InspectionDock'
import { useSSE } from './hooks/useSSE'
import { useZenfdAPI } from './hooks/useZenfdAPI'
import type { Plan, Task, Artifact } from './types/contracts'

export default function App() {
  const [plans, setPlans] = useState<string[]>([])
  const [activePlan, setActivePlan] = useState<Plan | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)
  const [artifacts, setArtifacts] = useState<Artifact[]>([])

  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [dockCollapsed, setDockCollapsed] = useState(false)
  const [isPaused, setIsPaused] = useState(false)

  const { fetchPlan, fetchTasks, killRun } = useZenfdAPI()

  // Real-time synchronization with daemon SQLite state
  const refreshData = useCallback(async () => {
    const taskList = await fetchTasks()
    const allTasks = taskList || []

    if (allTasks.length === 0) {
      setTasks([])
      setPlans([])
      setActivePlan(null)
      setSelectedTaskId(null)
      setArtifacts([])
      return
    }

    // Extract unique plan slugs
    const uniquePlans = Array.from(
      new Set(allTasks.map((t) => t.plan_id))
    ).filter(Boolean)
    setPlans(uniquePlans)

    // Determine target plan
    let targetPlanSlug = activePlan?.slug
    if (!targetPlanSlug || !uniquePlans.includes(targetPlanSlug)) {
      targetPlanSlug = uniquePlans[0]
    }

    if (targetPlanSlug) {
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
      }
    }
  }, [fetchTasks, fetchPlan, activePlan?.slug])

  // Initial fetch on mount
  useEffect(() => {
    refreshData()
  }, [refreshData])

  // Real-time SSE updates
  const { connected } = useSSE({
    onEvent: (evt) => {
      // Whenever a task status or plan changes in SQLite, refresh state
      if (
        evt.type === 'task:status' ||
        evt.type === 'plan:created' ||
        evt.type === 'run:killed'
      ) {
        refreshData()
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
      setSelectedTaskId(data.tasks?.[0]?.id || null)
    }
  }

  const handleKillProcess = async (runId: string) => {
    const ok = await killRun(runId)
    if (ok) {
      refreshData()
    }
  }

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden bg-zinc-950 font-sans text-zinc-100">
      {/* 1. Header Bar */}
      <HeaderBar
        connected={connected}
        activePlan={activePlan}
        plans={plans}
        onSelectPlan={handleSelectPlan}
        activeWorkers={activeWorkerCount}
        maxWorkers={5}
        isPaused={isPaused}
        onTogglePause={() => setIsPaused(!isPaused)}
      />

      {/* 2. Main Three-Seam Cockpit Body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Task Tree Sidebar */}
        <PlanTree
          tasks={tasks}
          selectedTaskId={selectedTaskId}
          onSelectTask={(task) => setSelectedTaskId(task.id)}
          collapsed={sidebarCollapsed}
          onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)}
        />

        {/* Center: Topological DAG Flow Canvas */}
        <DAGCanvas
          tasks={tasks}
          selectedTaskId={selectedTaskId}
          onSelectTask={(task) => setSelectedTaskId(task.id)}
        />

        {/* Right: Dual Inspection Dock (Terminal & Artifacts) */}
        <InspectionDock
          selectedTask={selectedTask}
          artifacts={artifacts}
          onKillProcess={handleKillProcess}
          collapsed={dockCollapsed}
          onToggleCollapse={() => setDockCollapsed(!dockCollapsed)}
        />
      </div>
    </div>
  )
}
