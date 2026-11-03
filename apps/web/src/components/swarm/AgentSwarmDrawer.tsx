import React from 'react'
import {
  Bot,
  Crown,
  ChevronDown,
  Terminal,
  Unplug,
  Clock,
  ArrowRight,
} from 'lucide-react'
import type { Task, AgentSession } from '../../types/contracts'

interface AgentSwarmDrawerProps {
  isOpen: boolean
  onClose: () => void
  sessions: AgentSession[]
  tasks: Task[]
  onSelectTask?: (task: Task) => void
  onEvictTask?: (taskId: string, reason?: string) => Promise<any>
}

function formatElapsed(claimedAt?: string): string {
  if (!claimedAt) return '< 1m'
  const start = new Date(claimedAt).getTime()
  const now = Date.now()
  const diffSec = Math.max(0, Math.floor((now - start) / 1000))
  const mins = Math.floor(diffSec / 60)
  const secs = diffSec % 60
  if (mins === 0) return `${secs}s`
  return `${mins}m ${secs}s`
}

export const AgentSwarmDrawer: React.FC<AgentSwarmDrawerProps> = ({
  isOpen,
  onClose,
  sessions,
  tasks,
  onSelectTask,
  onEvictTask,
}) => {
  if (!isOpen) return null

  const runningTasks = tasks.filter((t) => t.status === 'RUNNING')
  const suspendedTasks = tasks.filter((t) => t.status === 'SUSPENDED')

  // Group running tasks by orchestrator_id
  const orchMap = new Map<string, Task[]>()
  runningTasks.forEach((t) => {
    const orchId = t.orchestrator_id || 'unassigned-orchestrator'
    if (!orchMap.has(orchId)) {
      orchMap.set(orchId, [])
    }
    orchMap.get(orchId)!.push(t)
  })

  // Map session info by session_id and name
  const sessionMap = new Map<string, AgentSession>()
  sessions.forEach((s) => {
    sessionMap.set(s.session_id, s)
    if (s.name) {
      sessionMap.set(s.name, s)
    }
  })

  return (
    <div className="fixed bottom-0 left-0 right-0 z-40 max-h-[380px] flex flex-col border-t border-zinc-800 bg-zinc-950/95 shadow-2xl backdrop-blur-md transition-all duration-300">
      {/* Top Header Bar */}
      <div className="flex h-10 items-center justify-between border-b border-zinc-800/80 bg-zinc-900/80 px-4 select-none">
        <div className="flex items-center gap-2">
          <Bot className="h-4 w-4 text-violet-400" />
          <span className="font-mono text-xs font-bold text-zinc-100">
            Active Agent Swarm Hierarchy
          </span>
          <span className="rounded-full bg-violet-500/10 px-2 py-0.5 font-mono text-[10px] font-semibold text-violet-400 ring-1 ring-violet-500/30">
            {runningTasks.length} Ephemeral Worker{runningTasks.length === 1 ? '' : 's'} Active
          </span>
          {suspendedTasks.length > 0 && (
            <span className="rounded-full bg-amber-500/10 px-2 py-0.5 font-mono text-[10px] font-semibold text-amber-400 ring-1 ring-amber-500/30">
              {suspendedTasks.length} Suspended
            </span>
          )}
        </div>

        <button
          onClick={onClose}
          className="flex items-center gap-1 rounded px-2 py-1 text-xs text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200"
          title="Minimize Swarm Drawer"
        >
          <ChevronDown className="h-4 w-4" />
          <span>Minimize</span>
        </button>
      </div>

      {/* Drawer Body Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {runningTasks.length === 0 && suspendedTasks.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-6 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-zinc-800 bg-zinc-900 text-zinc-500 mb-2">
              <Bot className="h-6 w-6 text-zinc-600" />
            </div>
            <h4 className="text-xs font-semibold text-zinc-300">
              No Active Ephemeral Workers
            </h4>
            <p className="mt-1 max-w-md text-[11px] text-zinc-500">
              ZenForge strictly governs subagents as short-lived ephemeral workers (
              <span className="font-mono text-zinc-400">
                SPAWN ➔ CLAIM ➔ EXECUTE ➔ REPORT ➔ TERMINATE
              </span>
              ). There is no idle runner pool waiting. Workers appear here while actively claiming tasks.
            </p>
          </div>
        ) : (
          <>
            {/* Orchestrator Sections */}
            {Array.from(orchMap.entries()).map(([orchId, orchTasks]) => {
              const session = sessionMap.get(orchId)
              return (
                <div
                  key={orchId}
                  className="rounded-lg border border-violet-950/60 bg-zinc-900/40 p-3"
                >
                  {/* Orchestrator Badge */}
                  <div className="flex items-center justify-between border-b border-zinc-800/80 pb-2 mb-2.5">
                    <div className="flex items-center gap-2">
                      <div className="flex h-6 w-6 items-center justify-center rounded bg-violet-500/20 text-violet-400">
                        <Crown className="h-3.5 w-3.5" />
                      </div>
                      <span className="font-mono text-xs font-bold text-violet-200">
                        {session?.name || orchId}
                      </span>
                      {(session?.pid || orchTasks[0]?.orchestrator_pid) && (
                        <span className="rounded bg-zinc-800 px-1.5 py-0.5 font-mono text-[10px] text-zinc-400">
                          PID: {session?.pid || orchTasks[0]?.orchestrator_pid}
                        </span>
                      )}
                      {session?.session_id && (
                        <span className="font-mono text-[10px] text-zinc-500">
                          session: {session.session_id.slice(0, 8)}
                        </span>
                      )}
                    </div>
                    <span className="text-[10px] text-zinc-500 font-mono">
                      {orchTasks.length} task{orchTasks.length === 1 ? '' : 's'} in flight
                    </span>
                  </div>

                  {/* Claimed Tasks Tree */}
                  <div className="space-y-1.5 pl-3 border-l-2 border-violet-900/40 ml-2">
                    {orchTasks.map((task) => (
                      <div
                        key={task.id}
                        className="flex items-center justify-between rounded border border-zinc-800 bg-zinc-950/80 px-3 py-1.5 hover:border-violet-700/50 transition-colors"
                      >
                        <div className="flex items-center gap-2.5 min-w-0">
                          <Bot className="h-3.5 w-3.5 text-violet-400 animate-pulse shrink-0" />
                          <span className="font-mono text-xs font-semibold text-zinc-200">
                            {task.claimed_by || 'subagent'}
                          </span>
                          {task.worker_role && (
                            <span className="rounded bg-violet-950/80 border border-violet-800/50 px-1.5 py-0.2 text-[9px] font-mono text-violet-300">
                              {task.worker_role}
                            </span>
                          )}
                          {task.pid && (
                            <span className="rounded bg-zinc-800 px-1 py-0.2 text-[9px] font-mono text-zinc-400">
                              PID {task.pid}
                            </span>
                          )}

                          <ArrowRight className="h-3 w-3 text-zinc-600 shrink-0" />

                          <button
                            onClick={() => onSelectTask?.(task)}
                            className="font-mono text-xs font-bold text-amber-400 hover:underline truncate"
                            title="Inspect Task in Dock"
                          >
                            [{task.id}]
                          </button>
                          <span className="text-xs text-zinc-300 truncate max-w-[280px]">
                            {task.title}
                          </span>
                        </div>

                        <div className="flex items-center gap-3 shrink-0 ml-2">
                          {task.worktree_path && (
                            <span className="flex items-center gap-1 font-mono text-[10px] text-zinc-400">
                              <Terminal className="h-2.5 w-2.5 text-zinc-500" />
                              <span>{task.worktree_path}</span>
                            </span>
                          )}
                          <span className="flex items-center gap-1 font-mono text-[10px] text-zinc-400">
                            <Clock className="h-2.5 w-2.5 text-zinc-500" />
                            <span>{formatElapsed(task.claimed_at)}</span>
                          </span>

                          {onEvictTask && (
                            <button
                              onClick={() => {
                                if (
                                  window.confirm(
                                    `Are you sure you want to evict worker on task '${task.id}'? The task will reset to READY.`
                                  )
                                ) {
                                  onEvictTask(task.id, 'Evicted from Swarm Drawer')
                                }
                              }}
                              className="rounded border border-rose-900/60 bg-rose-950/20 px-2 py-0.5 text-[10px] font-semibold text-rose-400 hover:bg-rose-900/40"
                              title="Evict active worker and reset task to READY"
                            >
                              Evict
                            </button>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )
            })}

            {/* Suspended Tasks Recovery Warning Section */}
            {suspendedTasks.length > 0 && (
              <div className="rounded-lg border border-amber-900/50 bg-amber-950/20 p-3">
                <div className="flex items-center gap-2 mb-2">
                  <Unplug className="h-4 w-4 text-amber-500" />
                  <span className="font-mono text-xs font-bold text-amber-400">
                    Suspended Tasks ({suspendedTasks.length}) — Worker Process Terminated
                  </span>
                </div>
                <div className="space-y-1.5 pl-3 border-l-2 border-amber-800/40 ml-2">
                  {suspendedTasks.map((task) => (
                    <div
                      key={task.id}
                      className="flex items-center justify-between rounded border border-amber-900/40 bg-zinc-950/80 px-3 py-1.5"
                    >
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => onSelectTask?.(task)}
                          className="font-mono text-xs font-bold text-amber-400 hover:underline"
                        >
                          [{task.id}]
                        </button>
                        <span className="text-xs text-zinc-300 truncate max-w-[300px]">
                          {task.title}
                        </span>
                        <span className="rounded bg-amber-950/60 px-1.5 py-0.2 text-[9px] font-mono text-amber-400">
                          Last worker: {task.claimed_by || 'unknown'} (PID {task.pid || 'dead'})
                        </span>
                      </div>
                      <button
                        onClick={() => onSelectTask?.(task)}
                        className="rounded border border-amber-700/60 bg-amber-950/40 px-2 py-0.5 text-[10px] font-semibold text-amber-300 hover:bg-amber-900/50"
                      >
                        Inspect & Recover
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
