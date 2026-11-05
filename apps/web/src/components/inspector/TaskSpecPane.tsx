import React, { useState } from 'react'
import {
  Terminal,
  Copy,
  Check,
  CheckCircle2,
  AlertTriangle,
  GitMerge,
  GitBranch,
  FolderGit2,
  Clock,
  Layers,
  ExternalLink,
  UserCheck,
  ShieldAlert,
  FileCode,
  FileCheck,
  FileText,
} from 'lucide-react'
import type { Task } from '../../types/contracts'
import { elapsedLabel } from '../../lib/duration'
import { getTaskStatusStyle } from '../stage/taskStatusTheme'

interface TaskSpecPaneProps {
  task: Task | null
  allTasks: Task[]
  onSelectTaskId?: (taskId: string) => void
  isMaximized?: boolean
  onToggleMaximize?: () => void
}

export const TaskSpecPane: React.FC<TaskSpecPaneProps> = ({
  task,
  allTasks,
  onSelectTaskId,
  isMaximized = false,
  onToggleMaximize,
}) => {
  const [copiedCmd, setCopiedCmd] = useState(false)
  const [copiedWorktree, setCopiedWorktree] = useState(false)

  if (!task) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center p-6 text-center select-none text-zinc-500">
        <Terminal className="h-8 w-8 text-zinc-600 mb-2" />
        <p className="text-xs font-medium text-zinc-400">No task selected</p>
        <p className="text-[11px] text-zinc-500 mt-0.5">
          Select a task from the DAG canvas or sidebar to inspect its specification and status.
        </p>
      </div>
    )
  }

  const commandToRun = `zenfd task claim ${task.id}`

  const handleCopyCommand = async () => {
    try {
      await navigator.clipboard.writeText(commandToRun)
      setCopiedCmd(true)
      setTimeout(() => setCopiedCmd(false), 2000)
    } catch {
      // ignore clipboard error
    }
  }

  const handleCopyWorktree = async () => {
    if (!task.worktree_path) return
    try {
      await navigator.clipboard.writeText(task.worktree_path)
      setCopiedWorktree(true)
      setTimeout(() => setCopiedWorktree(false), 2000)
    } catch {
      // ignore
    }
  }

  const taskMap = new Map<string, Task>()
  allTasks.forEach((t) => taskMap.set(t.id, t))

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-zinc-950 font-sans select-none">
      {/* 1. Header Toolbar */}
      <div className="flex h-9 shrink-0 items-center justify-between border-b border-zinc-800 bg-zinc-900/80 px-3">
        <div className="flex items-center gap-2 overflow-hidden text-xs">
          <Layers className="h-3.5 w-3.5 text-violet-400 shrink-0" />
          <span className="font-semibold text-zinc-200 truncate">
            Task Specification Hub
          </span>
          <span className="text-[11px] font-mono text-zinc-500 truncate">
            [{task.plan_id}]
          </span>
        </div>

        {onToggleMaximize && (
          <button
            onClick={onToggleMaximize}
            className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white"
            title={isMaximized ? 'Split View' : 'Maximize Spec'}
          >
            <ExternalLink className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      {/* 2. Scrollable Body */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Title, Status & Worker Guard */}
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-2">
            <span className="font-mono text-sm font-bold text-zinc-100">
              {task.id}
            </span>
            <div className="flex items-center gap-1.5">
              <span
                className={`rounded-full border px-2.5 py-0.5 text-[10px] font-bold tracking-wider uppercase ${getTaskStatusStyle(
                  task.status
                )}`}
              >
                {task.status}
              </span>
            </div>
          </div>

          <h2 className="text-sm font-semibold text-zinc-200 leading-snug">
            {task.title}
          </h2>

          {/* Worker Badge */}
          {task.claimed_by && (
            <div className="flex items-center justify-between gap-2 rounded border border-zinc-800 bg-zinc-900/60 px-2.5 py-1 text-[11px] text-zinc-300">
              <div className="flex items-center gap-1.5 truncate">
                <UserCheck className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
                <span>
                  Claimed by: <strong className="text-zinc-200 font-mono">{task.claimed_by}</strong>
                  {task.status === 'RUNNING' && task.claimed_at && (
                    <span className="text-zinc-500"> · {elapsedLabel(task.claimed_at)}</span>
                  )}
                </span>
              </div>
            </div>
          )}
        </div>

        {/* READ-ONLY: Supplemental notes supplied by the agent / operator via MCP or CLI */}
        {task.notes && (
          <div className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-3 space-y-1.5">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <FileText className="h-3 w-3 text-zinc-400" />
              <span>Supplemental Notes</span>
            </span>
            <p className="whitespace-pre-wrap text-xs text-zinc-300 select-text leading-relaxed">
              {task.notes}
            </p>
          </div>
        )}

        {/* OBSERVATION GATE: Awaiting Review Handshake (read-only) */}
        {task.status === 'AWAITING_REVIEW' && (
          <div className="rounded-lg border border-orange-500/40 bg-orange-950/20 p-3.5 space-y-3">
            <div className="flex items-center gap-2 text-xs font-bold text-orange-300">
              <AlertTriangle className="h-4 w-4 text-orange-400 shrink-0" />
              <span>Architectural Review Requested by Worker</span>
            </div>

            {task.review_reason && (
              <div className="text-xs text-orange-200">
                <span className="font-semibold text-orange-300">Reason:</span> {task.review_reason}
              </div>
            )}

            {task.review_details && (
              <div className="rounded bg-black/40 p-2.5 text-[11px] font-mono text-zinc-300 break-words border border-orange-500/20">
                {task.review_details}
              </div>
            )}

            {task.review_options && task.review_options.length > 0 && (
              <div className="space-y-1.5">
                <span className="text-[10px] font-semibold text-zinc-400 uppercase tracking-wider block">
                  Suggested Options:
                </span>
                <div className="space-y-1">
                  {task.review_options.map((opt, i) => (
                    <div
                      key={i}
                      className="rounded border border-orange-500/20 bg-orange-500/5 px-2 py-1 text-[11px] text-orange-200"
                    >
                      {opt}
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="rounded border border-orange-500/30 bg-black/30 p-2.5 text-[11px] leading-relaxed text-zinc-300">
              This gate is resolved by your{' '}
              <strong className="text-orange-200">main orchestrator agent</strong>, not from the Web Cockpit.
              Reply with your decision in the agent session; the main agent records it with{' '}
              <code className="font-mono text-orange-200">zenfd_review(action=resolve)</code> and the task returns to{' '}
              <code className="font-mono text-orange-200">READY</code>.
            </div>
          </div>
        )}

        {/* INTERACTIVE GATE: Awaiting Merge Gate */}
        {task.status === 'AWAITING_MERGE' && (
          <div className="rounded-lg border border-cyan-500/40 bg-cyan-950/20 p-3.5 space-y-3">
            <div className="flex items-center gap-2 text-xs font-bold text-cyan-300">
              <GitMerge className="h-4 w-4 text-cyan-400 shrink-0" />
              <span>Worktree Slice Tested & Ready for Merge</span>
            </div>
            <p className="text-[11px] text-zinc-300">
              Agent execution verified with passing tests on isolated worktree. Merge the worktree into the target
              branch outside MCP, then mark the outcome with <code className="font-mono">zenfd_transition_task</code>{' '}
              (<code className="font-mono">merge_outcome=MERGED</code> or <code className="font-mono">CONFLICT</code>).
            </p>
          </div>
        )}

        {/* INTERACTIVE GATE: Merge Conflict Detected */}
        {task.status === 'CONFLICT' && (
          <div className="rounded-lg border border-rose-500/40 bg-rose-950/20 p-3.5 space-y-2">
            <div className="flex items-center gap-2 text-xs font-bold text-rose-300">
              <ShieldAlert className="h-4 w-4 text-rose-400 shrink-0" />
              <span>Git Merge Conflicts Detected</span>
            </div>
            <p className="text-[11px] text-zinc-300">
              Resolve conflicts manually outside MCP, then mark the task merged via{' '}
              <code className="font-mono">zenfd_transition_task</code> with{' '}
              <code className="font-mono">merge_outcome=MERGED</code>. Conflicting files:
            </p>
            {task.conflict_files && task.conflict_files.length > 0 && (
              <div className="space-y-1">
                {task.conflict_files.map((file, idx) => (
                  <div key={idx} className="rounded bg-black/40 px-2 py-1 font-mono text-[11px] text-rose-300 border border-rose-500/20">
                    {file}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Triage Badges: Priority, Complexity, Risk */}
        {(task.priority || task.complexity || task.risk_level) && (
          <div className="flex flex-wrap items-center gap-2 text-[10px] font-semibold">
            {task.priority && (
              <span className="rounded border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-amber-300">
                Priority: {task.priority}
              </span>
            )}
            {task.complexity && (
              <span className="rounded border border-blue-500/30 bg-blue-500/10 px-2 py-0.5 text-blue-300">
                Complexity: {task.complexity}
              </span>
            )}
            {task.risk_level && (
              <span className="rounded border border-rose-500/30 bg-rose-500/10 px-2 py-0.5 text-rose-300">
                Risk: {task.risk_level}
              </span>
            )}
          </div>
        )}

        {/* Acceptance Criteria */}
        {task.acceptance_criteria && task.acceptance_criteria.length > 0 && (
          <div className="space-y-1.5">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <FileCheck className="h-3 w-3 text-emerald-400" />
              <span>Acceptance Criteria ({task.acceptance_criteria.length})</span>
            </span>
            <div className="space-y-1 rounded border border-zinc-800 bg-zinc-900/40 p-2.5">
              {task.acceptance_criteria.map((ac, idx) => (
                <div key={idx} className="flex items-start gap-2 text-xs text-zinc-300">
                  <span className="text-emerald-400 font-bold shrink-0">✓</span>
                  <span>{ac}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Target Files */}
        {task.target_files && task.target_files.length > 0 && (
          <div className="space-y-1.5">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <FileCode className="h-3 w-3 text-cyan-400" />
              <span>Target Files ({task.target_files.length})</span>
            </span>
            <div className="space-y-1 font-mono text-[11px]">
              {task.target_files.map((tf, idx) => (
                <div key={idx} className="truncate rounded border border-zinc-800 bg-zinc-900/30 px-2 py-1 text-cyan-300">
                  {tf}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Executable Command Snippet Card */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <Terminal className="h-3 w-3 text-zinc-400" />
              <span>Executable Command</span>
            </span>
            <button
              onClick={handleCopyCommand}
              className="flex items-center gap-1 rounded bg-zinc-800 px-2 py-0.5 text-[11px] font-medium text-zinc-300 transition hover:bg-zinc-700 hover:text-white"
            >
              {copiedCmd ? (
                <>
                  <Check className="h-3 w-3 text-emerald-400" />
                  <span className="text-emerald-400 font-semibold">Copied!</span>
                </>
              ) : (
                <>
                  <Copy className="h-3 w-3" />
                  <span>Copy</span>
                </>
              )}
            </button>
          </div>

          <div className="rounded-lg border border-zinc-800 bg-zinc-950 p-2.5 font-mono text-xs text-violet-300 break-all select-text">
            <code>{commandToRun}</code>
          </div>
        </div>

        {/* Dependencies Seam */}
        <div className="space-y-1.5">
          <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
            <GitBranch className="h-3 w-3 text-zinc-400" />
            <span>Dependencies ({task.dependencies?.length || 0})</span>
          </span>

          {task.dependencies && task.dependencies.length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {task.dependencies.map((depId) => {
                const depTask = taskMap.get(depId)
                const isDepDone = depTask?.status === 'COMPLETED'
                return (
                  <button
                    key={depId}
                    onClick={() => onSelectTaskId && onSelectTaskId(depId)}
                    className={`flex items-center gap-1.5 rounded border px-2.5 py-1 text-xs font-mono transition-colors ${
                      isDepDone
                        ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20'
                        : 'border-zinc-700 bg-zinc-850 text-zinc-300 hover:border-zinc-600 hover:text-white'
                    }`}
                    title={`Jump and focus on ${depId}`}
                  >
                    <span>{depId}</span>
                    {isDepDone ? (
                      <CheckCircle2 className="h-3 w-3 text-emerald-400" />
                    ) : (
                      <span className="text-[10px] text-zinc-400">({depTask?.status || '?'})</span>
                    )}
                  </button>
                )
              })}
            </div>
          ) : (
            <div className="rounded border border-zinc-850 bg-zinc-900/30 px-3 py-2 text-xs text-zinc-500 italic">
              Root task — ready for immediate execution without dependencies.
            </div>
          )}
        </div>

        {/* Metadata Seams */}
        <div className="space-y-2 border-t border-zinc-850 pt-3 text-xs">
          <div className="flex items-center justify-between text-zinc-400">
            <span className="flex items-center gap-1 text-[11px] text-zinc-400">
              <FolderGit2 className="h-3 w-3" />
              <span>Worktree Path</span>
            </span>
            {task.worktree_path && (
              <button
                onClick={handleCopyWorktree}
                className="text-[10px] text-zinc-400 hover:text-zinc-200"
              >
                {copiedWorktree ? 'Copied!' : 'Copy Path'}
              </button>
            )}
          </div>
          <div className="truncate rounded border border-zinc-850 bg-zinc-900/50 px-2 py-1 font-mono text-[11px] text-zinc-300 select-text">
            {task.worktree_path || 'Not set'}
          </div>

          <div className="grid grid-cols-2 gap-2 text-[11px] text-zinc-400 pt-1">
            <div className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              <span>Harness: <strong className="text-zinc-300">{task.harness || 'shell'}</strong></span>
            </div>
            <div className="flex items-center gap-1">
              <span>Updated: <strong className="text-zinc-300">{task.updated_at ? new Date(task.updated_at).toLocaleTimeString() : '-'}</strong></span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
