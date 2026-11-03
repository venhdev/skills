import { useState, useEffect, useCallback } from 'react'
import { HeaderBar } from './components/header/HeaderBar'
import { PlanTree } from './components/sidebar/PlanTree'
import { DAGCanvas } from './components/stage/DAGCanvas'
import { InspectionDock } from './components/inspector/InspectionDock'
import { ArchitectureDocsView } from './components/docs/ArchitectureDocsView'
import { GlobalEnvView } from './components/docs/GlobalEnvView'
import { useSSE } from './hooks/useSSE'
import { useZenfdAPI } from './hooks/useZenfdAPI'
import type { Plan, Task, Artifact, ProjectDoc } from './types/contracts'

export default function App() {
  const [plans, setPlans] = useState<string[]>([])
  const [activePlan, setActivePlan] = useState<Plan | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])
  const [docs, setDocs] = useState<ProjectDoc[]>([])
  const [currentView, setCurrentView] = useState<'dag' | 'docs' | 'env'>('dag')

  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)
  const [focusTarget, setFocusTarget] = useState<{ id: string; timestamp: number } | null>(null)
  const [artifacts, setArtifacts] = useState<Artifact[]>([])

  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [dockCollapsed, setDockCollapsed] = useState(false)
  const [isPaused, setIsPaused] = useState(false)

  const {
    fetchPlan,
    fetchTasks,
    fetchTaskArtifacts,
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
    const list = await fetchDocs(activePlan?.repo_path)
    setDocs(list || [])
  }, [fetchDocs, activePlan?.repo_path])

  // Refresh artifacts whenever selectedTaskId changes
  useEffect(() => {
    refreshArtifacts(selectedTaskId)
  }, [selectedTaskId, refreshArtifacts])

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
    refreshDocs()
  }, [refreshData, refreshDocs])

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

  const handleResolveReview = async (taskId: string, response: string) => {
    await resolveReview(taskId, response)
    refreshData()
  }

  const handleMergeTask = async (taskId: string) => {
    await mergeTask(taskId)
    refreshData()
  }

  const handleUploadAttachment = async (file: File) => {
    if (!selectedTaskId) return
    const res = await uploadAttachment(selectedTaskId, file)
    if (res) {
      refreshArtifacts(selectedTaskId)
    }
  }

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden bg-zinc-950 font-sans text-zinc-100">
      {/* 1. Header Bar with 3-Tab Cockpit View Switcher */}
      <HeaderBar
        connected={connected}
        activePlan={activePlan}
        plans={plans}
        onSelectPlan={handleSelectPlan}
        activeWorkers={activeWorkerCount}
        maxWorkers={5}
        isPaused={isPaused}
        onTogglePause={() => setIsPaused(!isPaused)}
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

          {/* Center: Topological DAG Flow Canvas with Blast Radius Glow */}
          <DAGCanvas
            tasks={tasks}
            selectedTaskId={selectedTaskId}
            onSelectTask={handleSelectTask}
            focusTarget={focusTarget}
          />

          {/* Right: Inspection Dock with Read-Only Guard & Review/Merge Gates */}
          <InspectionDock
            selectedTask={selectedTask}
            allTasks={tasks}
            artifacts={artifacts}
            onSelectTaskId={handleSelectTaskId}
            onResolveReview={handleResolveReview}
            onMergeTask={handleMergeTask}
            onUploadAttachment={handleUploadAttachment}
            collapsed={dockCollapsed}
            onToggleCollapse={() => setDockCollapsed(!dockCollapsed)}
          />
        </div>
      )}

      {currentView === 'docs' && (
        <div className="flex flex-1 min-h-0 overflow-hidden">
          <ArchitectureDocsView
            repoPath={activePlan?.repo_path}
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
    </div>
  )
}
