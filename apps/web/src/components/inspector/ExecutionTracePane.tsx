import React, { useState, useRef, useEffect } from 'react'
import {
  UserCheck,
  Bot,
  Shield,
  Sparkles,
  Hash,
  Timer,
  FolderGit2,
  Copy,
  Check,
  FileText,
  AlertTriangle,
  GitMerge,
  ShieldAlert,
  Paperclip,
} from 'lucide-react'
import type { Task, Artifact } from '../../types/contracts'
import { elapsedLabel } from '../../lib/duration'
import { ArtifactLightbox } from './ArtifactLightbox'

interface ExecutionTracePaneProps {
  task: Task | null
  artifacts: Artifact[]
}

export const ExecutionTracePane: React.FC<ExecutionTracePaneProps> = ({ task, artifacts }) => {
  const [copiedWorktree, setCopiedWorktree] = useState(false)
  const timerRef = useRef<number | null>(null)

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [])

  if (!task) {
    return (
      <div className="flex h-full w-full items-center justify-center p-6 text-center select-none text-zinc-500">
        <p className="text-xs">No task selected for execution trace.</p>
      </div>
    )
  }

  const handleCopyWorktree = async () => {
    if (!task.worktree_path) return
    try {
      await navigator.clipboard.writeText(task.worktree_path)
      setCopiedWorktree(true)
      timerRef.current = window.setTimeout(() => setCopiedWorktree(false), 2000)
    } catch {
      // ignore
    }
  }

  return (
    <div className="flex h-full w-full flex-col overflow-y-auto bg-zinc-950 p-4 space-y-4 select-none font-sans">
      {/* 1. Claim Telemetry Hub */}
      {task.claimed_by ? (
        <div className="rounded-lg border border-zinc-800 bg-zinc-900/60 p-3 space-y-2.5">
          <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
            <UserCheck className="h-3.5 w-3.5 text-emerald-400" />
            <span>Worker Claim Telemetry</span>
          </div>
          <div className="grid grid-cols-2 gap-x-3 gap-y-2 text-xs">
            <div className="flex items-center gap-1.5 text-zinc-400">
              <Bot className="h-3 w-3 text-emerald-400 shrink-0" />
              <span>Worker:</span>
              <strong className="text-zinc-200 font-mono truncate">{task.claimed_by}</strong>
            </div>
            <div className="flex items-center gap-1.5 text-zinc-400">
              <Shield className="h-3 w-3 text-blue-400 shrink-0" />
              <span>Role:</span>
              <strong className="text-zinc-200 font-mono">{task.worker_role || '—'}</strong>
            </div>
            {task.orchestrator_id && (
              <div className="flex items-center gap-1.5 text-zinc-400">
                <Sparkles className="h-3 w-3 text-violet-400 shrink-0" />
                <span>Orchestrator:</span>
                <strong className="text-zinc-200 font-mono truncate">{task.orchestrator_id}</strong>
              </div>
            )}
            {task.claim_session_id && (
              <div className="flex items-center gap-1.5 text-zinc-400">
                <Hash className="h-3 w-3 text-zinc-500 shrink-0" />
                <span>Session:</span>
                <strong className="text-zinc-200 font-mono truncate">{task.claim_session_id}</strong>
              </div>
            )}
            {task.claimed_at && (
              <div className="col-span-2 flex items-center gap-1.5 text-zinc-400">
                <Timer className="h-3 w-3 text-amber-400 shrink-0" />
                <span>Claimed At:</span>
                <strong className="text-zinc-200">
                  {new Date(task.claimed_at).toLocaleTimeString()}
                </strong>
                {task.status === 'RUNNING' && (
                  <span className="text-zinc-500">· {elapsedLabel(task.claimed_at)}</span>
                )}
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="rounded-lg border border-zinc-800/80 bg-zinc-900/30 p-3 text-xs text-zinc-500 italic">
          Task has not been claimed by any worker yet.
        </div>
      )}

      {/* 2. Isolated Worktree */}
      {task.worktree_path && (
        <div className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-3 space-y-1.5">
          <div className="flex items-center justify-between text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
            <span className="flex items-center gap-1.5">
              <FolderGit2 className="h-3.5 w-3.5 text-cyan-400" />
              <span>Worktree Slice</span>
            </span>
            <button
              onClick={handleCopyWorktree}
              className="flex items-center gap-1 text-[10px] text-zinc-400 hover:text-zinc-200 transition"
            >
              {copiedWorktree ? (
                <>
                  <Check className="h-3 w-3 text-emerald-400" />
                  <span className="text-emerald-400">Copied</span>
                </>
              ) : (
                <>
                  <Copy className="h-3 w-3" />
                  <span>Copy</span>
                </>
              )}
            </button>
          </div>
          <div className="truncate rounded border border-zinc-800 bg-black/50 px-2.5 py-1.5 font-mono text-[11px] text-zinc-300 select-text">
            {task.worktree_path}
          </div>
        </div>
      )}

      {/* 3. Supplemental Operational Notes */}
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

      {/* 4. Awaiting Review Handshake */}
      {task.status === 'AWAITING_REVIEW' && (
        <div className="rounded-lg border border-orange-500/40 bg-orange-950/20 p-3.5 space-y-3">
          <div className="flex items-center gap-2 text-xs font-bold text-orange-300">
            <AlertTriangle className="h-4 w-4 text-orange-400 shrink-0" />
            <span>Architectural Review Requested</span>
          </div>

          {task.review_reason && (
            <div className="text-xs text-orange-200">
              <span className="font-semibold text-orange-300">Reason:</span> {task.review_reason}
            </div>
          )}

          {task.review_details && (
            <div className="rounded bg-black/40 p-2.5 text-[11px] font-mono text-zinc-300 break-words border border-orange-500/20 select-text">
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

          {task.review_response && (
            <div className="rounded border border-emerald-500/30 bg-emerald-950/20 p-2 text-[11px] text-emerald-300">
              <span className="font-semibold">Resolution:</span> {task.review_response}
              {task.resolved_by && (
                <span className="text-zinc-400">
                  {' '}— by <strong className="text-zinc-300 font-mono">{task.resolved_by}</strong>
                </span>
              )}
            </div>
          )}
        </div>
      )}

      {/* 5. Awaiting Merge Gate */}
      {task.status === 'AWAITING_MERGE' && (
        <div className="rounded-lg border border-cyan-500/40 bg-cyan-950/20 p-3.5 space-y-2">
          <div className="flex items-center gap-2 text-xs font-bold text-cyan-300">
            <GitMerge className="h-4 w-4 text-cyan-400 shrink-0" />
            <span>Worktree Tested & Awaiting Merge</span>
          </div>
          <p className="text-[11px] text-zinc-300 leading-relaxed">
            Slice tests passed in isolation. Merge the branch and mark with{' '}
            <code className="font-mono text-cyan-300">transition_task(merge_outcome=MERGED)</code>.
          </p>
        </div>
      )}

      {/* 6. Merge Conflict Gate */}
      {task.status === 'CONFLICT' && (
        <div className="rounded-lg border border-rose-500/40 bg-rose-950/20 p-3.5 space-y-2">
          <div className="flex items-center gap-2 text-xs font-bold text-rose-300">
            <ShieldAlert className="h-4 w-4 text-rose-400 shrink-0" />
            <span>Merge Conflict Detected</span>
          </div>
          {task.conflict_files && task.conflict_files.length > 0 && (
            <div className="space-y-1">
              {task.conflict_files.map((file, idx) => (
                <div
                  key={idx}
                  className="rounded bg-black/40 px-2 py-1 font-mono text-[11px] text-rose-300 border border-rose-500/20"
                >
                  {file}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* 7. Full Artifacts Gallery */}
      <div className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-3 space-y-2">
        <div className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
          <Paperclip className="h-3.5 w-3.5 text-violet-400" />
          <span>Task Artifacts & Media ({artifacts.length})</span>
        </div>
        <div className="h-64 rounded border border-zinc-800/80 overflow-hidden bg-zinc-950">
          <ArtifactLightbox artifacts={artifacts} compact={false} />
        </div>
      </div>
    </div>
  )
}
