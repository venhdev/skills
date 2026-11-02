import { useState, useEffect, useCallback } from 'react'
import { HeaderBar } from './components/header/HeaderBar'
import { PlanTree } from './components/sidebar/PlanTree'
import { DAGCanvas } from './components/stage/DAGCanvas'
import { InspectionDock } from './components/inspector/InspectionDock'
import { useSSE } from './hooks/useSSE'
import { useZenfdAPI } from './hooks/useZenfdAPI'
import type { Plan, Task, Artifact } from './types/contracts'

// Sane default starter tasks for offline / standalone preview
const defaultTasks: Task[] = [
  {
    id: 'P-00',
    plan_id: 'zenfd',
    repo_path: '/my-skills',
    title: 'Walking Skeleton Plumbing Proof (REST -> SQLite -> PTY -> SSE)',
    dependencies: [],
    harness: 'antigravity',
    skill_command: '/forge P-00',
    worktree_path: '',
    status: 'COMPLETED',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'P-01',
    plan_id: 'zenfd',
    repo_path: '/my-skills',
    title: 'SQLite WAL Storage Engine (ncruces/go-sqlite3 CGO-free)',
    dependencies: ['P-00'],
    harness: 'antigravity',
    skill_command: '/forge P-01',
    worktree_path: '',
    status: 'COMPLETED',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'P-02',
    plan_id: 'zenfd',
    repo_path: '/my-skills',
    title: 'Process Group Supervisor & Unix PTY Runner (creack/pty)',
    dependencies: ['P-01'],
    harness: 'antigravity',
    skill_command: '/forge P-02',
    worktree_path: '',
    status: 'COMPLETED',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'P-03',
    plan_id: 'zenfd',
    repo_path: '/my-skills',
    title: 'DAG Dependency Scheduler & Crash Reaper',
    dependencies: ['P-01', 'P-02'],
    harness: 'antigravity',
    skill_command: '/forge P-03',
    worktree_path: '',
    status: 'COMPLETED',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'P-04',
    plan_id: 'zenfd',
    repo_path: '/my-skills',
    title: 'Go 1.22+ Gateway Server, SSE Hub & coder/websocket Bridge',
    dependencies: ['P-01', 'P-03'],
    harness: 'antigravity',
    skill_command: '/forge P-04',
    worktree_path: '',
    status: 'COMPLETED',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'P-05',
    plan_id: 'zenfd',
    repo_path: '/my-skills',
    title: 'Cobra CLI (zenfd) & Markdown Living Plan Ingestion Bridge',
    dependencies: ['P-04'],
    harness: 'antigravity',
    skill_command: '/forge P-05',
    worktree_path: '',
    status: 'COMPLETED',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'P-06',
    plan_id: 'zenfd',
    repo_path: '/my-skills',
    title: 'Web Dashboard UI (Vite + React 19 + Tailwind v4 + @xyflow)',
    dependencies: ['P-04', 'P-05'],
    harness: 'antigravity',
    skill_command: '/forge P-06',
    worktree_path: '',
    status: 'RUNNING',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'P-07',
    plan_id: 'zenfd',
    repo_path: '/my-skills',
    title: 'Multi-Modal Artifact Inspection Hub (Diffs, Screenshots)',
    dependencies: ['P-06'],
    harness: 'antigravity',
    skill_command: '/forge P-07',
    worktree_path: '',
    status: 'READY',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'P-08',
    plan_id: 'zenfd',
    repo_path: '/my-skills',
    title: 'Ecosystem Integration & zenf-cli Bootstrap',
    dependencies: ['P-05', 'P-07'],
    harness: 'antigravity',
    skill_command: '/forge P-08',
    worktree_path: '',
    status: 'PENDING',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
]

export default function App() {
  const [plans, setPlans] = useState<string[]>(['zenfd'])
  const [activePlan, setActivePlan] = useState<Plan | null>({
    slug: 'zenfd',
    title: 'ZenForge Daemon & Web Control Plane',
    repo_path: '/home/rirong/src/owner/my-skills',
    status: 'ACTIVE',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  })
  const [tasks, setTasks] = useState<Task[]>(defaultTasks)
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>('P-06')
  const [artifacts] = useState<Artifact[]>([
    {
      id: 'art-patch-p06',
      run_id: 'run-P-06-current',
      task_id: 'P-06',
      type: 'DIFF',
      file_path: 'apps/web/src/App.tsx',
      mime_type: 'text/x-diff',
      byte_size: 4096,
      sha256: '9a8b7c6d5e4f3a2b1c0d',
      created_at: new Date().toISOString(),
    },
  ])

  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [dockCollapsed, setDockCollapsed] = useState(false)
  const [isPaused, setIsPaused] = useState(false)

  const { fetchPlan, fetchTasks, killRun } = useZenfdAPI()

  // Refresh tasks and plans from daemon
  const refreshData = useCallback(async () => {
    const taskList = await fetchTasks()
    if (taskList && taskList.length > 0) {
      setTasks(taskList)
      const uniquePlans = Array.from(new Set(taskList.map((t) => t.plan_id)))
      if (uniquePlans.length > 0) {
        setPlans(uniquePlans)
      }
    }
    if (activePlan?.slug) {
      const planData = await fetchPlan(activePlan.slug)
      if (planData?.plan) {
        setActivePlan(planData.plan)
        if (planData.tasks && planData.tasks.length > 0) {
          setTasks(planData.tasks)
        }
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
      // Whenever a task status or plan changes, refresh state
      if (evt.type === 'task:status' || evt.type === 'plan:created') {
        refreshData()
      }
    },
  })

  const selectedTask = tasks.find((t) => t.id === selectedTaskId) || null

  const activeWorkerCount = tasks.filter((t) => t.status === 'RUNNING').length

  const handleSelectPlan = async (slug: string) => {
    const data = await fetchPlan(slug)
    if (data) {
      setActivePlan(data.plan)
      setTasks(data.tasks)
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
