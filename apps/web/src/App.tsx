import { useState, useEffect, useCallback, useRef } from 'react'
import { HeaderBar, type CockpitView } from './components/header/HeaderBar'
import { PlanTree } from './components/sidebar/PlanTree'
import { DAGCanvas } from './components/stage/DAGCanvas'
import { InspectionDock } from './components/inspector/InspectionDock'
import { GlobalEnvView } from './components/env/GlobalEnvView'
import { ProjectsView } from './components/workspace/ProjectsView'
import { DocsView } from './components/workspace/DocsView'
import { AgentsView } from './components/workspace/AgentsView'
import { ActivityView } from './components/workspace/ActivityView'
import { useSSE } from './hooks/useSSE'
import { useZenfdAPI } from './hooks/useZenfdAPI'
import type { Plan, Project, Task, Artifact } from './types/contracts'

export default function App() {
  const [projects, setProjects] = useState<Project[]>([])
  const [projectsReady, setProjectsReady] = useState(false)
  const [activeProjectSlug, setActiveProjectSlug] = useState<string | null>(null)
  const [plans, setPlans] = useState<Plan[]>([])
  const [showArchived, setShowArchived] = useState(false)
  const [logTick, setLogTick] = useState(0)
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
  const refreshSeqRef = useRef(0)
  const refreshTimerRef = useRef<number | null>(null)

  const {
    fetchProjects,
    fetchPlans,
    fetchPlan,
    fetchTasks,
    fetchTaskArtifacts,
    fetchTaskPacket,
    fetchGlobalEnv,
  } = useZenfdAPI()

  const refreshArtifacts = useCallback(
    async (taskId: string | null) => {
      if (!taskId) {
        setArtifacts([])
        return
      }
      const artsRes = await fetchTaskArtifacts(taskId)
      setArtifacts(artsRes.ok ? artsRes.data : [])
    },
    [fetchTaskArtifacts]
  )

  // Refresh artifacts whenever selectedTaskId changes
  useEffect(() => {
    refreshArtifacts(selectedTaskId)
  }, [selectedTaskId, refreshArtifacts])

  const refreshProjects = useCallback(async () => {
    const result = await fetchProjects(false)
    if (!result.ok) return
    setProjects(result.data)
    setActiveProjectSlug((current) => {
      if (current && result.data.some((project) => project.slug === current)) {
        return current
      }
      return result.data.find((project) => project.status === 'ACTIVE')?.slug || result.data[0]?.slug || null
    })
    setProjectsReady(true)
  }, [fetchProjects])

  // Real-time synchronization with daemon SQLite state
  const refreshData = useCallback(async () => {
    if (!projectsReady) return

    // Coalesce overlapping refreshes (SSE + focus + interval). Only the most
    // recent generation is allowed to apply its results.
    const seq = ++refreshSeqRef.current

    // The plans endpoint is the source of truth for the selector and includes
    // plans that do not have any tasks yet. A failed request must not be
    // interpreted as "no plans".
    const plansResult = await fetchPlans(activeProjectSlug || '', showArchived)
    if (!plansResult.ok) return
    if (seq !== refreshSeqRef.current) return

    const availablePlans = plansResult.data
    setPlans(availablePlans)
    const availableSlugs = new Set(availablePlans.map((p) => p.slug))

    // Determine target plan. When nothing is selected, the project task list is
    // needed once to prefer a plan that has tasks so the canvas is not empty on
    // first load. Once a plan is active, skip that request entirely.
    let targetPlanSlug = activePlanRef.current?.slug
    if (!targetPlanSlug || !availableSlugs.has(targetPlanSlug)) {
      const taskRes = await fetchTasks(activeProjectSlug ? { projectSlug: activeProjectSlug } : undefined)
      if (!taskRes.ok) {
        // Network error occurred. Retain current state safely without wiping.
        return
      }
      if (seq !== refreshSeqRef.current) return
      const taskList = taskRes.data
      targetPlanSlug = taskList.find((t) => t.plan_slug && availableSlugs.has(t.plan_slug))?.plan_slug
      targetPlanSlug = targetPlanSlug || availablePlans[0]?.slug
    }

    if (!targetPlanSlug) {
      // Successful response with a genuinely empty plan list.
      setTasks([])
      setActivePlan(null)
      setSelectedTaskId(null)
      setArtifacts([])
      return
    }

    const planResult = await fetchPlan(targetPlanSlug)
    if (seq !== refreshSeqRef.current) return

    if (!planResult.ok) {
      // A transient failure keeps the current canvas intact. Only a 404 means
      // the plan is definitively gone.
      if (planResult.status === 404) {
        setActivePlan(null)
        setTasks([])
        setSelectedTaskId(null)
        setArtifacts([])
      }
      return
    }

    const planData = planResult.data
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
  }, [activeProjectSlug, fetchTasks, fetchPlans, fetchPlan, projectsReady, showArchived])

  // SSE bursts (for example plan:created followed by state:changed) are
  // collapsed into a single refresh instead of one request triple per event.
  const scheduleRefresh = useCallback(() => {
    if (refreshTimerRef.current !== null) {
      window.clearTimeout(refreshTimerRef.current)
    }
    refreshTimerRef.current = window.setTimeout(() => {
      refreshTimerRef.current = null
      void refreshData()
    }, 120)
  }, [refreshData])

  useEffect(() => {
    return () => {
      if (refreshTimerRef.current !== null) {
        window.clearTimeout(refreshTimerRef.current)
      }
    }
  }, [])

  // Initial load: projects first, then plans scoped to the selected project.
  useEffect(() => {
    void refreshProjects()
  }, [refreshProjects])

  // A project switch invalidates every in-flight or scheduled refresh from the
  // previous project before the scoped reload starts.
  useEffect(() => {
    refreshSeqRef.current++
    if (refreshTimerRef.current !== null) {
      window.clearTimeout(refreshTimerRef.current)
      refreshTimerRef.current = null
    }
  }, [activeProjectSlug])

  // Reload whenever the project or archive filter changes. The initial run is
  // gated on projectsReady so plans are always scoped to a real project.
  useEffect(() => {
    if (!projectsReady) return
    void refreshData()
  }, [projectsReady, refreshData])

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
      setLogTick((value) => value + 1)
      // The activity log covers every writer, so any event here is a concrete
      // refresh signal. Artifact-only events do not change task state.
      if (evt.type === 'artifact.added') {
        refreshArtifacts(selectedTaskId)
        return
      }
      if (evt.type.startsWith('project.')) {
        void refreshProjects()
        scheduleRefresh()
        return
      }
      if (evt.type.startsWith('plan.') || evt.type.startsWith('task.') || evt.type === 'state.changed') {
        scheduleRefresh()
        // An external writer may also have attached media to the selected task.
        if (evt.type === 'state.changed') {
          refreshArtifacts(selectedTaskId)
        }
      }
    },
  })

  const selectedTask = tasks.find((t) => t.id === selectedTaskId) || null
  const activeWorkerCount = tasks.filter((t) => t.status === 'RUNNING').length
  const activeProject = projects.find((project) => project.slug === activeProjectSlug) || null

  const handleSelectProject = useCallback((slug: string) => {
    refreshSeqRef.current++
    setActiveProjectSlug(slug || null)
    setActivePlan(null)
    setPlans([])
    setTasks([])
    setSelectedTaskId(null)
    setArtifacts([])
  }, [])

  const handleOpenProject = useCallback(
    (slug: string) => {
      handleSelectProject(slug)
      setCurrentView('dag')
    },
    [handleSelectProject]
  )

  const handleProjectsChanged = useCallback(() => {
    void refreshProjects()
  }, [refreshProjects])

  const handleSelectPlan = useCallback(
    async (slug: string) => {
      // Invalidate any in-flight refresh so it cannot overwrite this selection.
      refreshSeqRef.current++
      const result = await fetchPlan(slug)
      if (!result.ok) return
      const data = result.data
      setActivePlan(data.plan)
      setTasks(data.tasks || [])
      const firstTask = data.tasks?.[0]
      if (firstTask) {
        setSelectedTaskId(firstTask.id)
        setFocusTarget({ id: firstTask.id, timestamp: Date.now() })
      } else {
        setSelectedTaskId(null)
      }
    },
    [fetchPlan]
  )

  const handleSelectTask = useCallback(
    (task: Task, source: 'sidebar' | 'canvas' | 'dock' = 'sidebar') => {
      setSelectedTaskId(task.id)
      if (source !== 'canvas') {
        setFocusTarget({ id: task.id, timestamp: Date.now() })
      }
    },
    []
  )

  const handleSelectTaskId = useCallback(
    (taskId: string) => {
      const t = tasks.find((x) => x.id === taskId)
      if (t) {
        handleSelectTask(t, 'dock')
      } else {
        setSelectedTaskId(taskId)
        setFocusTarget({ id: taskId, timestamp: Date.now() })
      }
    },
    [tasks, handleSelectTask]
  )

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden bg-zinc-950 font-sans text-zinc-100">
      {/* 1. Header Bar with Plan CRUD & Active Worker Count */}
      <HeaderBar
        connected={connected}
        projects={projects}
        activeProject={activeProject}
        onSelectProject={handleSelectProject}
        activePlan={activePlan}
        plans={plans}
        onSelectPlan={handleSelectPlan}
        showArchived={showArchived}
        onToggleArchived={() => setShowArchived((value) => !value)}
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
            logTick={logTick}
          />

          {/* Center: Topological DAG Flow Canvas (read-only view) */}
          <DAGCanvas
            tasks={tasks}
            selectedTaskId={selectedTaskId}
            onSelectTask={handleSelectTask}
            focusTarget={focusTarget}
          />

          {/* Right: Read-only Inspection Dock (spec & artifacts) */}
          <InspectionDock
            selectedTask={selectedTask}
            allTasks={tasks}
            artifacts={artifacts}
            onSelectTaskId={handleSelectTaskId}
            collapsed={dockCollapsed}
            onToggleCollapse={() => setDockCollapsed(!dockCollapsed)}
            fetchTaskPacket={fetchTaskPacket}
          />
        </div>
      )}

      {currentView === 'projects' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <ProjectsView
            activeProjectSlug={activeProjectSlug}
            onOpenProject={handleOpenProject}
            onProjectsChanged={handleProjectsChanged}
          />
        </div>
      )}

      {currentView === 'docs' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <DocsView projects={projects} activeProjectSlug={activeProjectSlug} />
        </div>
      )}

      {currentView === 'agents' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <AgentsView tick={logTick} />
        </div>
      )}

      {currentView === 'activity' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <ActivityView tick={logTick} projectSlug={activeProjectSlug} />
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
