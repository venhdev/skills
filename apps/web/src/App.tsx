import { useState, useEffect, useCallback, useRef } from 'react'
import { HeaderBar, type CockpitView } from './components/header/HeaderBar'
import { PlanTree } from './components/sidebar/PlanTree'
import { DAGCanvas } from './components/stage/DAGCanvas'
import { InspectionDock } from './components/inspector/InspectionDock'
import { BlueprintStudio } from './components/blueprint/BlueprintStudio'
import { DossierExplorer } from './components/dossier/DossierExplorer'
import { ArchitectureDocsView } from './components/docs/ArchitectureDocsView'
import { GlobalEnvView } from './components/docs/GlobalEnvView'
import { AgentSwarmDrawer } from './components/swarm/AgentSwarmDrawer'
import { useSSE } from './hooks/useSSE'
import { useZenfdAPI } from './hooks/useZenfdAPI'
import type { Plan, Task, Artifact, ProjectDoc, AgentSession } from './types/contracts'

export default function App() {
  const [projects, setProjects] = useState<string[]>([])
  const [activeProject, setActiveProject] = useState<string>('')
  const [plans, setPlans] = useState<string[]>([])
  const [activePlan, setActivePlan] = useState<Plan | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])
  const [sessions, setSessions] = useState<AgentSession[]>([])
  const [docs, setDocs] = useState<ProjectDoc[]>([])
  const [currentView, setCurrentView] = useState<CockpitView>('dag')

  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)
  const [focusTarget, setFocusTarget] = useState<{ id: string; timestamp: number } | null>(null)
  const [artifacts, setArtifacts] = useState<Artifact[]>([])

  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [dockCollapsed, setDockCollapsed] = useState(false)
  const [isSwarmOpen, setIsSwarmOpen] = useState(false)

  const activePlanRef = useRef<Plan | null>(null)
  activePlanRef.current = activePlan

  const activeProjectRef = useRef<string>('')
  activeProjectRef.current = activeProject

  const {
    fetchProjects,
    fetchPlans,
    fetchPlan,
    fetchTasks,
    fetchTaskArtifacts,
    deletePlan,
    updateTaskDependencies,
    updateTaskNotes,
    reclaimTask,
    evictTask,
    fetchSessions,
    uploadAttachment,
    fetchDocs,
    saveDoc,
    exportDocs,
    resolveReview,
    mergeTask,
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

  const refreshDocs = useCallback(async () => {
    const repo = activePlanRef.current?.repo_path || activeProjectRef.current
    const list = await fetchDocs(repo)
    setDocs(list || [])
  }, [fetchDocs])

  const refreshSessions = useCallback(async () => {
    const list = await fetchSessions()
    setSessions(list || [])
  }, [fetchSessions])

  const refreshProjects = useCallback(async () => {
    const projs = await fetchProjects()
    setProjects(projs || [])
    if (projs && projs.length > 0) {
      setActiveProject((curr) => curr || projs[0])
    }
  }, [fetchProjects])

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

    if (taskList.length === 0) {
      // Check if there are any empty plans
      const availablePlans = await fetchPlans(activeProjectRef.current)
      if (availablePlans.length > 0) {
        setPlans(availablePlans)
        let targetSlug = activePlanRef.current?.slug
        if (!targetSlug || !availablePlans.includes(targetSlug)) {
          targetSlug = availablePlans[0]
        }
        const planData = await fetchPlan(targetSlug)
        if (planData?.plan) {
          setActivePlan(planData.plan)
          setTasks(planData.tasks || [])
        }
      } else {
        setTasks([])
        setPlans([])
        setActivePlan(null)
        setSelectedTaskId(null)
        setArtifacts([])
      }
      return
    }

    // Extract unique plan slugs
    const uniquePlans = Array.from(
      new Set(taskList.map((t) => t.plan_id))
    ).filter(Boolean)
    setPlans(uniquePlans)

    // Determine target plan
    let targetPlanSlug = activePlanRef.current?.slug
    if (!targetPlanSlug || !uniquePlans.includes(targetPlanSlug)) {
      targetPlanSlug = uniquePlans[0]
    }

    if (targetPlanSlug) {
      const planData = await fetchPlan(targetPlanSlug)
      if (planData?.plan) {
        setActivePlan(planData.plan)
        if (planData.plan.repo_path) {
          setActiveProject((curr) => curr || planData.plan.repo_path)
        }
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
  }, [fetchTasks, fetchPlans, fetchPlan])

  // Initial fetch on mount
  useEffect(() => {
    refreshData()
    refreshProjects()
    refreshSessions()
    refreshDocs()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Real-time SSE updates
  const { connected } = useSSE({
    onEvent: (evt) => {
      if (
        evt.type === 'task:status' ||
        evt.type === 'plan:created' ||
        evt.type === 'run:killed'
      ) {
        refreshData()
        refreshSessions()
      }
      if (evt.type === 'session:registered' || evt.type === 'session:heartbeat') {
        refreshSessions()
      }
      if (evt.type === 'task:artifact_added') {
        refreshArtifacts(selectedTaskId)
      }
    },
  })

  const selectedTask = tasks.find((t) => t.id === selectedTaskId) || null
  const activeWorkerCount = tasks.filter((t) => t.status === 'RUNNING').length

  const handleSelectProject = async (repo: string) => {
    setActiveProject(repo)
    const projsPlans = await fetchPlans(repo)
    setPlans(projsPlans)
    if (projsPlans.length > 0) {
      handleSelectPlan(projsPlans[0])
    } else {
      setActivePlan(null)
      setTasks([])
      setSelectedTaskId(null)
    }
  }

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

  const handleDeletePlan = async () => {
    if (!activePlan) return
    if (
      window.confirm(
        `Are you sure you want to delete plan '${activePlan.slug}' and all its tasks? This action deletes from SQLite SSOT.`
      )
    ) {
      const ok = await deletePlan(activePlan.slug)
      if (ok) {
        await refreshProjects()
        await refreshData()
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

  const handleResolveReview = async (taskId: string, response: string) => {
    await resolveReview(taskId, response)
    refreshData()
  }

  const handleMergeTask = async (taskId: string) => {
    await mergeTask(taskId)
    refreshData()
  }

  const handleUploadAttachment = async (taskId: string, file: File, name?: string) => {
    const res = await uploadAttachment(taskId, file, name)
    if (res) {
      refreshArtifacts(taskId)
    }
    return res
  }

  const handleUpdateDependencies = async (taskId: string, dependencies: string[]) => {
    const ok = await updateTaskDependencies(taskId, dependencies)
    if (ok) {
      refreshData()
    }
    return ok
  }

  const handleUpdateNotes = async (taskId: string, notes: string) => {
    const ok = await updateTaskNotes(taskId, notes)
    if (ok) {
      refreshData()
    }
    return ok
  }

  const handleReclaimTask = async (taskId: string) => {
    const res = await reclaimTask(taskId)
    if (res) {
      refreshData()
      refreshSessions()
    }
    return res
  }

  const handleEvictTask = async (taskId: string, reason?: string, action?: 'RESET_TO_READY' | 'CANCEL') => {
    const ok = await evictTask(taskId, reason, action)
    if (ok) {
      refreshData()
      refreshSessions()
    }
    return ok
  }

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden bg-zinc-950 font-sans text-zinc-100">
      {/* 1. Header Bar with Project Switcher, Plan CRUD & Active Worker Count */}
      <HeaderBar
        connected={connected}
        activePlan={activePlan}
        plans={plans}
        onSelectPlan={handleSelectPlan}
        activeWorkers={activeWorkerCount}
        projects={projects}
        activeProject={activeProject}
        onSelectProject={handleSelectProject}
        onDeletePlanClick={handleDeletePlan}
        onToggleSwarmDrawer={() => setIsSwarmOpen(!isSwarmOpen)}
        isSwarmOpen={isSwarmOpen}
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

          {/* Center: Topological DAG Flow Canvas with Interactive Connection & Blast Radius */}
          <DAGCanvas
            tasks={tasks}
            selectedTaskId={selectedTaskId}
            onSelectTask={handleSelectTask}
            focusTarget={focusTarget}
            onUpdateDependencies={handleUpdateDependencies}
          />

          {/* Right: Inspection Dock with Seeding Hub, Review Handshake & Suspended Recovery */}
          <InspectionDock
            selectedTask={selectedTask}
            allTasks={tasks}
            artifacts={artifacts}
            onSelectTaskId={handleSelectTaskId}
            onResolveReview={handleResolveReview}
            onMergeTask={handleMergeTask}
            onUploadAttachment={handleUploadAttachment}
            onUpdateNotes={handleUpdateNotes}
            onReclaimTask={handleReclaimTask}
            onEvictTask={handleEvictTask}
            collapsed={dockCollapsed}
            onToggleCollapse={() => setDockCollapsed(!dockCollapsed)}
          />
        </div>
      )}

      {currentView === 'blueprint' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <BlueprintStudio
            tasks={tasks}
            onSelectTask={(task) => {
              handleSelectTask(task, 'sidebar')
              setCurrentView('dag')
            }}
          />
        </div>
      )}

      {currentView === 'dossier' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <DossierExplorer planSlug={activePlan?.slug || ''} />
        </div>
      )}

      {currentView === 'docs' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <ArchitectureDocsView
            repoPath={activePlan?.repo_path || activeProject}
            docs={docs}
            onSaveDoc={saveDoc}
            onExportDocs={exportDocs}
            onRefresh={refreshDocs}
          />
        </div>
      )}

      {currentView === 'env' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <GlobalEnvView onFetchEnv={fetchGlobalEnv} />
        </div>
      )}

      {/* 3. Ephemeral Swarm Hierarchy Drawer */}
      <AgentSwarmDrawer
        isOpen={isSwarmOpen}
        onClose={() => setIsSwarmOpen(false)}
        sessions={sessions}
        tasks={tasks}
        onSelectTask={(task) => handleSelectTask(task, 'dock')}
        onEvictTask={handleEvictTask}
      />
    </div>
  )
}
