import React, { useCallback, useEffect, useRef, useState } from 'react'
import { ChevronDown, ChevronUp, ScrollText } from 'lucide-react'
import type { EventMessage } from '../../types/contracts'

interface LiveLogProps {
  /** Increments on every SSE event, including the daemon's state:changed. */
  tick: number
}

interface EventSummary {
  text: string
  tone: 'plan' | 'task' | 'media' | 'muted'
}

function parsePayload(raw: string): Record<string, any> {
  try {
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

// formatEvent turns a trigger-produced row into one readable line. The payload
// shape is owned by schema.go's audit triggers.
function formatEvent(evt: EventMessage): EventSummary {
  const p = parsePayload(evt.payload)
  switch (evt.type) {
    case 'plan.created':
      return { text: `plan created · ${p.slug}`, tone: 'plan' }
    case 'plan.updated':
      return {
        text:
          p.kind === 'archived'
            ? `plan archived · ${p.slug}`
            : p.kind === 'unarchived'
              ? `plan restored · ${p.slug}`
              : `plan ${p.kind} · ${p.slug}${p.from && p.to && p.from !== p.to ? ` ${p.from}→${p.to}` : ''}`,
        tone: 'plan',
      }
    case 'plan.deleted':
      return { text: `plan deleted · ${p.slug}`, tone: 'plan' }
    case 'task.created':
      return { text: `task created · ${p.task_id} (${p.status})`, tone: 'task' }
    case 'task.updated': {
      const suffix =
        p.kind === 'status' && p.from && p.to
          ? ` ${p.from}→${p.to}`
          : p.kind === 'claim'
            ? ` claimed by ${p.claimed_by || 'worker'}`
            : ` ${p.kind}`
      return { text: `${p.task_id}${suffix}`, tone: 'task' }
    }
    case 'task.deleted':
      return { text: `task deleted · ${p.task_id}`, tone: 'task' }
    case 'artifact.added':
      return {
        text: `${p.role === 'INPUT' ? 'media attached' : 'artifact'} · ${p.name || p.artifact_id} → ${p.task_id}`,
        tone: 'media',
      }
    default:
      return { text: `${evt.type} · ${evt.task_id || evt.plan_id || ''}`, tone: 'muted' }
  }
}

const toneClass: Record<EventSummary['tone'], string> = {
  plan: 'text-sky-300',
  task: 'text-violet-300',
  media: 'text-emerald-300',
  muted: 'text-zinc-400',
}

function formatTime(raw: string): string {
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return '--:--:--'
  return date.toLocaleTimeString(undefined, { hour12: false })
}

export const LiveLog: React.FC<LiveLogProps> = ({ tick }) => {
  const [events, setEvents] = useState<EventMessage[]>([])
  const [collapsed, setCollapsed] = useState(false)
  const lastIdRef = useRef(0)
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const pinnedRef = useRef(true)

  const load = useCallback(async (initial: boolean) => {
    const query = initial ? '?limit=80' : `?after_id=${lastIdRef.current}&limit=80`
    try {
      const res = await fetch(`/api/v1/events${query}`)
      if (!res.ok) return
      const batch: EventMessage[] = await res.json()
      if (!Array.isArray(batch) || batch.length === 0) return

      const newest = batch[batch.length - 1].id
      if (newest > lastIdRef.current) lastIdRef.current = newest

      setEvents((prev) => (initial ? batch : [...prev, ...batch]).slice(-300))
    } catch {
      // The daemon may be restarting; the next tick retries.
    }
  }, [])

  useEffect(() => {
    void load(true)
  }, [load])

  useEffect(() => {
    if (tick > 0) void load(false)
  }, [tick, load])

  useEffect(() => {
    const node = scrollRef.current
    if (node && pinnedRef.current) {
      node.scrollTop = node.scrollHeight
    }
  }, [events])

  const handleScroll = () => {
    const node = scrollRef.current
    if (!node) return
    pinnedRef.current = node.scrollHeight - node.scrollTop - node.clientHeight < 24
  }

  return (
    <section className="flex h-1/4 min-h-[120px] flex-col border-t border-zinc-800 bg-zinc-950/80">
      <div className="flex h-8 shrink-0 items-center justify-between px-3">
        <div className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wider text-zinc-400">
          <ScrollText className="h-3.5 w-3.5 text-emerald-400" />
          <span>Live Log</span>
          <span className="rounded bg-zinc-800 px-1 text-[10px] text-zinc-400">
            {events.length}
          </span>
        </div>
        <button
          onClick={() => setCollapsed((value) => !value)}
          className="rounded p-1 text-zinc-500 hover:bg-zinc-800 hover:text-zinc-200"
          title={collapsed ? 'Expand live log' : 'Collapse live log'}
        >
          {collapsed ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
        </button>
      </div>

      {!collapsed && (
        <div
          ref={scrollRef}
          onScroll={handleScroll}
          className="flex-1 overflow-y-auto px-2 pb-2 font-mono text-[10.5px] leading-relaxed"
        >
          {events.length === 0 ? (
            <div className="px-1 py-2 text-zinc-600">// waiting for activity</div>
          ) : (
            events.map((evt) => {
              const summary = formatEvent(evt)
              return (
                <div key={evt.id} className="flex gap-2 whitespace-nowrap">
                  <span className="shrink-0 text-zinc-600">{formatTime(evt.timestamp)}</span>
                  <span className={`truncate ${toneClass[summary.tone]}`} title={summary.text}>
                    {summary.text}
                  </span>
                </div>
              )
            })
          )}
        </div>
      )}
    </section>
  )
}
