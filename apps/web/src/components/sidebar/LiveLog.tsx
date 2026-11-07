import React, { useCallback, useEffect, useRef, useState } from 'react'
import { ChevronDown, ChevronUp, ScrollText } from 'lucide-react'
import type { EventMessage } from '../../types/contracts'
import { formatEvent, toneClass } from '../../lib/eventFormatter'
import { useZenfAPI } from '../../hooks/useZenfAPI'

interface LiveLogProps {
  /** Increments on every SSE event, including the daemon's state:changed. */
  tick: number
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
  const { fetchEvents } = useZenfAPI()

  const load = useCallback(async (initial: boolean) => {
    try {
      const afterId = initial ? 0 : lastIdRef.current
      const res = await fetchEvents(afterId, 80)
      if (!res.ok) return
      const batch = res.data
      if (!Array.isArray(batch) || batch.length === 0) return

      const newest = batch[batch.length - 1].id
      if (newest > lastIdRef.current) lastIdRef.current = newest

      setEvents((prev) => (initial ? batch : [...prev, ...batch]).slice(-300))
    } catch {
      // The daemon may be restarting; the next tick retries.
    }
  }, [fetchEvents])

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
