import React, { useState } from 'react'
import {
  Copy,
  Check,
  FileText,
  BookOpen,
  FolderOpen,
  Terminal,
  Loader2,
  Zap,
  Paperclip,
} from 'lucide-react'
import type { ContextPacket } from '../../types/contracts'

interface ClaimPacketPaneProps {
  packet: ContextPacket | null
  loading: boolean
  onFetch: () => void
}

export const ClaimPacketPane: React.FC<ClaimPacketPaneProps> = ({
  packet,
  loading,
  onFetch,
}) => {
  const [copiedPrompt, setCopiedPrompt] = useState(false)
  const [expandedDocs, setExpandedDocs] = useState<Set<string>>(new Set())

  const handleCopyPrompt = async () => {
    if (!packet?.prompt_markdown) return
    try {
      await navigator.clipboard.writeText(packet.prompt_markdown)
      setCopiedPrompt(true)
      setTimeout(() => setCopiedPrompt(false), 2500)
    } catch {
      // ignore
    }
  }

  const toggleDocExpand = (docId: string) => {
    setExpandedDocs((prev) => {
      const next = new Set(prev)
      if (next.has(docId)) next.delete(docId)
      else next.add(docId)
      return next
    })
  }

  // Empty state
  if (!packet && !loading) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center p-6 text-center select-none text-zinc-500">
        <Zap className="h-8 w-8 text-zinc-600 mb-2" />
        <p className="text-xs font-medium text-zinc-400">No Claim Packet Loaded</p>
        <p className="text-[11px] text-zinc-500 mt-0.5 mb-3">
          Select a task to inspect its compiled agent context packet.
        </p>
        <button
          onClick={onFetch}
          className="rounded bg-violet-600/20 border border-violet-500/30 px-3 py-1.5 text-xs font-semibold text-violet-300 hover:bg-violet-600/30 transition"
        >
          Fetch Packet
        </button>
      </div>
    )
  }

  // Loading state
  if (loading) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center p-6 text-center select-none">
        <Loader2 className="h-6 w-6 text-violet-400 animate-spin mb-2" />
        <p className="text-xs text-zinc-400">Compiling context packet…</p>
      </div>
    )
  }

  if (!packet) return null

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-zinc-950 font-sans select-none">
      {/* Header */}
      <div className="flex h-9 shrink-0 items-center justify-between border-b border-zinc-800 bg-zinc-900/80 px-3">
        <div className="flex items-center gap-2 text-xs">
          <Zap className="h-3.5 w-3.5 text-amber-400 shrink-0" />
          <span className="font-semibold text-zinc-200">Claim Packet Inspector</span>
          <span className="text-[11px] font-mono text-zinc-500">{packet.task?.id}</span>
        </div>
        <button
          onClick={handleCopyPrompt}
          className="flex items-center gap-1 rounded bg-violet-600/20 border border-violet-500/30 px-2 py-0.5 text-[10px] font-semibold text-violet-300 transition hover:bg-violet-600/30 hover:text-violet-200"
        >
          {copiedPrompt ? (
            <>
              <Check className="h-3 w-3 text-emerald-400" />
              <span className="text-emerald-400">Copied!</span>
            </>
          ) : (
            <>
              <Copy className="h-3 w-3" />
              <span>Copy Full Prompt</span>
            </>
          )}
        </button>
      </div>

      {/* Scrollable body */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Skill Paths */}
        {packet.skill_paths && packet.skill_paths.length > 0 && (
          <div className="space-y-1.5">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <FolderOpen className="h-3 w-3 text-violet-400" />
              <span>Resolved Skill Paths ({packet.skill_paths.length})</span>
            </span>
            <div className="space-y-1 font-mono text-[11px]">
              {packet.skill_paths.map((sp, idx) => (
                <div key={idx} className="truncate rounded border border-zinc-800 bg-zinc-900/30 px-2 py-1 text-violet-300 select-text">
                  {sp}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Input Attachments */}
        {packet.input_attachments && packet.input_attachments.length > 0 && (
          <div className="space-y-1.5">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <Paperclip className="h-3 w-3 text-cyan-400" />
              <span>Input Attachments ({packet.input_attachments.length})</span>
            </span>
            <div className="space-y-1 text-[11px]">
              {packet.input_attachments.map((att) => (
                <div key={att.id} className="flex items-center gap-2 rounded border border-zinc-800 bg-zinc-900/30 px-2 py-1 text-zinc-300">
                  <span className="font-mono truncate flex-1">{att.name}</span>
                  <span className="text-zinc-500 text-[10px]">{att.mime_type}</span>
                  <span className="text-zinc-600 text-[10px]">{(att.byte_size / 1024).toFixed(1)}KB</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Linked SSOT Documents */}
        {packet.linked_documents && packet.linked_documents.length > 0 && (
          <div className="space-y-1.5">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <BookOpen className="h-3 w-3 text-amber-400" />
              <span>Injected SSOT Documents ({packet.linked_documents.length})</span>
            </span>
            <div className="space-y-1.5">
              {packet.linked_documents.map((doc) => {
                const isExpanded = expandedDocs.has(doc.id)
                return (
                  <div key={doc.id} className="rounded border border-zinc-800 bg-zinc-900/40 overflow-hidden">
                    <button
                      onClick={() => toggleDocExpand(doc.id)}
                      className="flex items-center gap-2 w-full px-2.5 py-1.5 text-left text-[11px] hover:bg-zinc-800/50 transition"
                    >
                      <FileText className="h-3 w-3 text-amber-400 shrink-0" />
                      <span className="font-semibold text-zinc-200 truncate flex-1">{doc.title}</span>
                      <span className="text-[10px] text-zinc-500 font-mono shrink-0">{doc.type} · {doc.scope}</span>
                      <span className="text-zinc-500 text-[10px]">{isExpanded ? '▾' : '▸'}</span>
                    </button>
                    {isExpanded && (
                      <div className="border-t border-zinc-800 px-2.5 py-2 space-y-1.5">
                        {doc.summary && (
                          <p className="text-[11px] text-zinc-400 italic">{doc.summary}</p>
                        )}
                        {doc.content && (
                          <pre className="whitespace-pre-wrap text-[11px] text-zinc-300 select-text max-h-48 overflow-y-auto bg-black/30 rounded p-2 border border-zinc-800">
                            {doc.content}
                          </pre>
                        )}
                        {doc.tags && doc.tags.length > 0 && (
                          <div className="flex flex-wrap gap-1">
                            {doc.tags.map((tag, ti) => (
                              <span key={ti} className="rounded bg-zinc-800 px-1.5 py-0.5 text-[9px] text-zinc-400 font-mono">
                                #{tag}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Verification Commands */}
        {packet.verification_commands && packet.verification_commands.length > 0 && (
          <div className="space-y-1.5">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <Terminal className="h-3 w-3 text-emerald-400" />
              <span>Verification Commands ({packet.verification_commands.length})</span>
            </span>
            <div className="space-y-1 font-mono text-[11px]">
              {packet.verification_commands.map((cmd, idx) => (
                <div key={idx} className="rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1.5 text-emerald-300 select-text break-all">
                  <code>$ {cmd}</code>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Compiled Prompt Markdown */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <span className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              <FileText className="h-3 w-3 text-violet-400" />
              <span>Compiled Prompt Markdown</span>
            </span>
            <button
              onClick={handleCopyPrompt}
              className="flex items-center gap-1 rounded bg-zinc-800 px-2 py-0.5 text-[10px] font-medium text-zinc-300 transition hover:bg-zinc-700 hover:text-white"
            >
              {copiedPrompt ? (
                <>
                  <Check className="h-3 w-3 text-emerald-400" />
                  <span className="text-emerald-400">Copied!</span>
                </>
              ) : (
                <>
                  <Copy className="h-3 w-3" />
                  <span>Copy</span>
                </>
              )}
            </button>
          </div>
          <pre className="whitespace-pre-wrap rounded-lg border border-zinc-800 bg-zinc-950 p-3 text-[11px] text-zinc-300 select-text leading-relaxed max-h-[60vh] overflow-y-auto font-mono">
            {packet.prompt_markdown || '(empty)'}
          </pre>
        </div>
      </div>
    </div>
  )
}
