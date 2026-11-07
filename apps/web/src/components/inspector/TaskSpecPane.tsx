import React, { useState, useMemo, useRef, useEffect } from 'react'
import {
  Terminal,
  Copy,
  Check,
  CheckCircle2,
  GitBranch,
  Clock,
  Layers,
  ExternalLink,
  FileCode,
  FileCheck,
  Clipboard,
  Loader2,
  Wrench,
  BookOpen,
  Sparkles,
  Bot,
  Shield,
  CalendarDays,
  PlayCircle,
  AlertCircle,
  Paperclip,
  ChevronRight,
  Lock,
  FileSpreadsheet,
} from 'lucide-react'
import type { Task, ContextPacket, Artifact } from '../../types/contracts'
import type { ApiResult } from '../../hooks/useZenfAPI'
import { getTaskStatusStyle } from '../stage/taskStatusTheme'
import { parseTaskDescription } from '../../lib/taskDescriptionParser'

interface TaskSpecPaneProps {
  task: Task | null
  allTasks: Task[]
  artifacts?: Artifact[]
  onSelectTaskId?: (taskId: string) => void
  isMaximized?: boolean
  onToggleMaximize?: () => void
  fetchTaskPacket?: (taskId: string) => Promise<ApiResult<ContextPacket>>
  onViewArtifacts?: () => void
}

export const TaskSpecPane: React.FC<TaskSpecPaneProps> = ({
  task,
  allTasks,
  artifacts = [],
  onSelectTaskId,
  isMaximized = false,
  onToggleMaximize,
  fetchTaskPacket,
  onViewArtifacts,
}) => {
  const [copiedPrompt, setCopiedPrompt] = useState(false)
  const [promptLoading, setPromptLoading] = useState(false)
  const [copiedVerify, setCopiedVerify] = useState(false)

  // Timer cleanup on unmount (Resolving Finding #12)
  const timersRef = useRef<number[]>([])
  useEffect(() => {
    return () => {
      timersRef.current.forEach((id) => clearTimeout(id))
    }
  }, [])

  const setSafeTimeout = (fn: () => void, delay: number) => {
    const id = window.setTimeout(fn, delay)
    timersRef.current.push(id)
  }

  const parsedDesc = useMemo(() => parseTaskDescription(task?.description), [task?.description])

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

  const handleCopyAgentPrompt = async () => {
    if (!fetchTaskPacket || promptLoading) return
    setPromptLoading(true)
    try {
      const result = await fetchTaskPacket(task.id)
      if (result.ok && result.data?.prompt_markdown) {
        await navigator.clipboard.writeText(result.data.prompt_markdown)
        setCopiedPrompt(true)
        setSafeTimeout(() => setCopiedPrompt(false), 2500)
      }
    } catch {
      // ignore
    } finally {
      setPromptLoading(false)
    }
  }

  const handleCopyVerification = async () => {
    if (!task.verification_commands?.length) return
    try {
      await navigator.clipboard.writeText(task.verification_commands.join('\n'))
      setCopiedVerify(true)
      setSafeTimeout(() => setCopiedVerify(false), 2000)
    } catch {
      // ignore
    }
  }

  const taskMap = new Map<string, Task>()
  allTasks.forEach((t) => taskMap.set(t.id, t))

  const isCompleted = task.status === 'COMPLETED'

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
            [{task.project_slug ? `${task.project_slug} / ` : ''}
            {task.plan_slug || 'orphan'}]
          </span>
        </div>

        <div className="flex items-center gap-1">
          {/* Copy Agent Prompt — primary action */}
          {fetchTaskPacket && (
            <button
              onClick={handleCopyAgentPrompt}
              disabled={promptLoading}
              className="flex items-center gap-1 rounded bg-violet-600/20 border border-violet-500/30 px-2 py-0.5 text-[10px] font-semibold text-violet-300 transition hover:bg-violet-600/30 hover:text-violet-200 disabled:opacity-50"
              title="Fetch compiled agent prompt and copy to clipboard"
            >
              {promptLoading ? (
                <>
                  <Loader2 className="h-3 w-3 animate-spin" />
                  <span>Loading…</span>
                </>
              ) : copiedPrompt ? (
                <>
                  <Check className="h-3 w-3 text-emerald-400" />
                  <span className="text-emerald-400">Copied!</span>
                </>
              ) : (
                <>
                  <Clipboard className="h-3 w-3" />
                  <span>Copy Agent Prompt</span>
                </>
              )}
            </button>
          )}

          {onToggleMaximize && (
            <button
              onClick={onToggleMaximize}
              className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white"
              title={isMaximized ? 'Restore View' : 'Maximize Spec'}
            >
              <ExternalLink className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* 2. Scrollable Body */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Title, Status & ID */}
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

          {/* Phase Stepper Bar (Polymorphic: only rendered if phases exist) */}
          {task.phases && task.phases.length > 0 && (
            <div className="rounded-lg border border-zinc-800 bg-zinc-900/60 p-2.5 space-y-2">
              <div className="flex items-center justify-between text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
                <div className="flex items-center gap-1.5">
                  <Sparkles className="h-3 w-3 text-cyan-400" />
                  <span>
                    Execution Phases ({task.phases.filter((p) => p.status === 'COMPLETED').length}/
                    {task.phases.length})
                  </span>
                </div>
                {task.current_phase && (
                  <span className="font-mono text-cyan-400 text-[10px] lowercase">
                    active: {task.current_phase}
                  </span>
                )}
              </div>
              <div className="flex items-center gap-1.5 overflow-x-auto py-0.5">
                {task.phases.map((phase, idx) => {
                  const isCurrent = task.current_phase === phase.id
                  const isDone = phase.status === 'COMPLETED'
                  const isFailed = phase.status === 'FAILED'
                  const isSkipped = phase.status === 'SKIPPED'

                  return (
                    <React.Fragment key={phase.id}>
                      <div
                        className={`flex items-center gap-1.5 rounded border px-2 py-1 text-[11px] font-mono shrink-0 transition-colors ${
                          isDone
                            ? 'border-emerald-500/40 bg-emerald-950/30 text-emerald-300'
                            : isCurrent
                              ? 'border-cyan-500/70 bg-cyan-950/30 text-cyan-300 ring-1 ring-cyan-500/50 shadow-[0_0_10px_rgba(6,182,212,0.2)]'
                              : isFailed
                                ? 'border-rose-500/50 bg-rose-950/30 text-rose-300'
                                : isSkipped
                                  ? 'border-zinc-800 bg-zinc-900/30 text-zinc-500 line-through'
                                  : 'border-zinc-800 bg-zinc-900/40 text-zinc-500'
                        }`}
                        title={phase.notes ? `${phase.title}: ${phase.notes}` : phase.title}
                      >
                        {isDone ? (
                          <CheckCircle2 className="h-3 w-3 text-emerald-400 shrink-0" />
                        ) : isCurrent ? (
                          <PlayCircle className="h-3 w-3 text-cyan-400 animate-spin shrink-0" />
                        ) : isFailed ? (
                          <AlertCircle className="h-3 w-3 text-rose-400 shrink-0" />
                        ) : (
                          <Clock className="h-3 w-3 text-zinc-600 shrink-0" />
                        )}
                        <span className="font-sans font-medium text-[11px] truncate max-w-[130px]">
                          {phase.title}
                        </span>
                      </div>
                      {idx < (task.phases?.length || 0) - 1 && (
                        <ChevronRight className="h-3 w-3 text-zinc-700 shrink-0" />
                      )}
                    </React.Fragment>
                  )
                })}
              </div>
            </div>
          )}

          {/* Structured Description Blocks */}
          {parsedDesc.hasStructuredBlocks ? (
            <div className="space-y-3 pt-1">
              {/* Context Card */}
              {parsedDesc.context && (
                <div className="rounded-lg border border-violet-500/20 bg-violet-950/10 p-3 space-y-1">
                  <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-violet-400">
                    <Layers className="h-3 w-3" />
                    <span>Context</span>
                  </div>
                  <p className="text-xs text-zinc-300 leading-relaxed select-text whitespace-pre-wrap">
                    {parsedDesc.context}
                  </p>
                </div>
              )}

              {/* Input / Output 2-Column Grid */}
              {(parsedDesc.inputs?.length || parsedDesc.outputs?.length) ? (
                <div className="grid grid-cols-2 gap-2.5">
                  {/* Inputs */}
                  <div className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-2.5 space-y-1.5">
                    <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
                      <FileCode className="h-3 w-3 text-cyan-400" />
                      <span>Inputs ({parsedDesc.inputs?.length || 0})</span>
                    </span>
                    <div className="space-y-1">
                      {parsedDesc.inputs && parsedDesc.inputs.length > 0 ? (
                        parsedDesc.inputs.map((inItem, idx) => (
                          <div
                            key={idx}
                            className="rounded bg-black/40 px-2 py-1 font-mono text-[11px] text-cyan-300 border border-zinc-800/80 break-all select-text"
                          >
                            {inItem}
                          </div>
                        ))
                      ) : (
                        <span className="text-[11px] text-zinc-500 italic">None specified</span>
                      )}
                    </div>
                  </div>

                  {/* Outputs */}
                  <div className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-2.5 space-y-1.5">
                    <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
                      <CheckCircle2 className="h-3 w-3 text-emerald-400" />
                      <span>Outputs ({parsedDesc.outputs?.length || 0})</span>
                    </span>
                    <div className="space-y-1">
                      {parsedDesc.outputs && parsedDesc.outputs.length > 0 ? (
                        parsedDesc.outputs.map((outItem, idx) => (
                          <div
                            key={idx}
                            className="rounded bg-black/40 px-2 py-1 font-mono text-[11px] text-emerald-300 border border-zinc-800/80 break-all select-text"
                          >
                            {outItem}
                          </div>
                        ))
                      ) : (
                        <span className="text-[11px] text-zinc-500 italic">None specified</span>
                      )}
                    </div>
                  </div>
                </div>
              ) : null}

              {/* Condition Guards */}
              {parsedDesc.conditionGuards && parsedDesc.conditionGuards.length > 0 && (
                <div className="rounded-lg border border-amber-500/20 bg-amber-950/10 p-2.5 space-y-1.5">
                  <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-amber-400">
                    <Shield className="h-3 w-3" />
                    <span>Condition Guards</span>
                  </span>
                  <div className="space-y-1 font-mono text-[11px]">
                    {parsedDesc.conditionGuards.map((guard, idx) => (
                      <div
                        key={idx}
                        className="rounded bg-black/60 px-2.5 py-1.5 text-amber-300/90 border border-amber-500/20 select-text break-all"
                      >
                        <code>{guard}</code>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Constraints */}
              {parsedDesc.constraints && parsedDesc.constraints.length > 0 && (
                <div className="rounded-lg border border-rose-500/20 bg-rose-950/10 p-2.5 space-y-1.5">
                  <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-rose-400">
                    <Lock className="h-3 w-3" />
                    <span>Constraints</span>
                  </span>
                  <div className="space-y-1">
                    {parsedDesc.constraints.map((c, idx) => (
                      <div key={idx} className="text-xs text-rose-200">
                        • {c}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Custom Blocks */}
              {parsedDesc.customBlocks.map((block) => (
                <div key={block.key} className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-2.5 space-y-1.5">
                  <span className="text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
                    {block.label}
                  </span>
                  {block.items && block.items.length > 0 ? (
                    <div className="space-y-1">
                      {block.items.map((it, idx) => (
                        <div key={idx} className="text-xs text-zinc-300">
                          • {it}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-zinc-300 whitespace-pre-wrap select-text">
                      {block.content}
                    </p>
                  )}
                </div>
              ))}

              {/* Preamble / General Notes */}
              {parsedDesc.generalNotes && (
                <p className="whitespace-pre-wrap text-xs text-zinc-400 leading-relaxed select-text">
                  {parsedDesc.generalNotes}
                </p>
              )}
            </div>
          ) : (
            /* Fallback to standard unstructured description */
            task.description && (
              <p className="whitespace-pre-wrap text-xs text-zinc-400 leading-relaxed select-text pt-1">
                {task.description}
              </p>
            )
          )}
        </div>

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

        {/* Agent Capabilities — required_skills, required_docs, required_tools, agent_slug */}
        {(task.required_skills?.length || task.required_docs?.length || task.required_tools?.length || task.agent_slug) && (
          <div className="space-y-1.5">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <Sparkles className="h-3 w-3 text-violet-400" />
              <span>Agent Capabilities</span>
            </span>
            <div className="flex flex-wrap gap-1.5">
              {task.agent_slug && (
                <span className="rounded border border-zinc-700 bg-zinc-800 px-2 py-0.5 text-[10px] text-zinc-300 font-mono">
                  <Bot className="h-2.5 w-2.5 inline mr-1 text-zinc-400" />
                  {task.agent_slug}
                </span>
              )}
              {task.required_skills?.map((skill, idx) => (
                <span key={`skill-${idx}`} className="rounded border border-violet-500/30 bg-violet-500/10 px-2 py-0.5 text-[10px] text-violet-300">
                  {skill}
                </span>
              ))}
              {task.required_docs?.map((doc, idx) => (
                <span key={`doc-${idx}`} className="rounded border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[10px] text-amber-300">
                  <BookOpen className="h-2.5 w-2.5 inline mr-1" />
                  {doc}
                </span>
              ))}
              {task.required_tools?.map((tool, idx) => (
                <span key={`tool-${idx}`} className="rounded border border-cyan-500/30 bg-cyan-500/10 px-2 py-0.5 text-[10px] text-cyan-300">
                  <Wrench className="h-2.5 w-2.5 inline mr-1" />
                  {tool}
                </span>
              ))}
            </div>
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
                  {isCompleted ? (
                    <span className="text-emerald-400 font-bold shrink-0">☑</span>
                  ) : (
                    <span className="text-zinc-500 font-mono shrink-0 text-[11px]">◻ {String(idx + 1).padStart(2, '0')}.</span>
                  )}
                  <span>{ac}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Verification Commands */}
        {task.verification_commands && task.verification_commands.length > 0 && (
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
                <Terminal className="h-3 w-3 text-emerald-400" />
                <span>Verification Commands ({task.verification_commands.length})</span>
              </span>
              <button
                onClick={handleCopyVerification}
                className="flex items-center gap-1 rounded bg-zinc-800 px-2 py-0.5 text-[10px] font-medium text-zinc-300 transition hover:bg-zinc-700 hover:text-white"
              >
                {copiedVerify ? (
                  <>
                    <Check className="h-3 w-3 text-emerald-400" />
                    <span className="text-emerald-400">Copied!</span>
                  </>
                ) : (
                  <>
                    <Copy className="h-3 w-3" />
                    <span>Copy All</span>
                  </>
                )}
              </button>
            </div>
            <div className="space-y-1 font-mono text-[11px]">
              {task.verification_commands.map((cmd, idx) => (
                <div key={idx} className="rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1.5 text-emerald-300 select-text break-all">
                  <code>$ {cmd}</code>
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

        {/* Multi-Modal Attachments Row (Zero-Footprint empty state rule) */}
        {artifacts.length > 0 && (
          <div className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-2.5 space-y-2">
            <div className="flex items-center justify-between text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <div className="flex items-center gap-1.5">
                <Paperclip className="h-3 w-3 text-violet-400" />
                <span>Media Attachments ({artifacts.length})</span>
              </div>
              {onViewArtifacts && (
                <button
                  onClick={onViewArtifacts}
                  className="text-violet-400 hover:text-violet-300 font-sans normal-case text-[11px] transition"
                >
                  View Full Gallery →
                </button>
              )}
            </div>
            <div className="flex items-center gap-2 overflow-x-auto py-1">
              {artifacts.slice(0, 4).map((art) => (
                <div
                  key={art.id}
                  className="relative h-14 w-14 rounded border border-zinc-800 bg-black/40 overflow-hidden shrink-0 group"
                  title={art.name}
                >
                  {art.type === 'IMAGE' || art.mime_type.startsWith('image/') ? (
                    <img
                      src={`/api/v1/artifacts/${art.id}`}
                      alt={art.name}
                      className="h-full w-full object-cover transition-transform group-hover:scale-105"
                    />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center">
                      <FileSpreadsheet className="h-5 w-5 text-zinc-500" />
                    </div>
                  )}
                </div>
              ))}
              {artifacts.length > 4 && (
                <button
                  onClick={onViewArtifacts}
                  className="flex h-14 w-14 items-center justify-center rounded border border-zinc-800 bg-zinc-900/80 text-xs font-mono text-zinc-400 hover:text-white shrink-0 transition"
                >
                  +{artifacts.length - 4}
                </button>
              )}
            </div>
          </div>
        )}

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

        {/* Metadata Footer */}
        <div className="space-y-2 border-t border-zinc-850 pt-3 text-xs">
          <div className="grid grid-cols-2 gap-2 text-[11px] text-zinc-400 pt-1">
            <div className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              <span>Harness: <strong className="text-zinc-300">{task.harness || 'shell'}</strong></span>
            </div>
            <div className="flex items-center gap-1">
              <CalendarDays className="h-3 w-3" />
              <span>Created: <strong className="text-zinc-300">{task.created_at ? new Date(task.created_at).toLocaleString() : '-'}</strong></span>
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
