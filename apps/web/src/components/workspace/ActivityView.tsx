import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Activity,
  Bot,
  BookOpen,
  FolderKanban,
  GitFork,
  ListTree,
  RefreshCw,
  Search,
  Sparkles,
} from 'lucide-react'
import { useZenfdAPI } from '../../hooks/useZenfdAPI'
import type { EventMessage, SearchResult } from '../../types/contracts'
import { formatEvent } from '../../lib/eventFormatter'

interface ActivityViewProps {
  tick: number
  projectSlug: string | null
}

type EntityFilter = 'all' | 'project' | 'plan' | 'task' | 'document' | 'agent' | 'session' | 'artifact'

const entityFilters: { key: EntityFilter; label: string }[] = [
  { key: 'all', label: 'Everything' },
  { key: 'project', label: 'Projects' },
  { key: 'plan', label: 'Plans' },
  { key: 'task', label: 'Tasks' },
  { key: 'document', label: 'Documents' },
  { key: 'agent', label: 'Agents' },
  { key: 'session', label: 'Sessions' },
  { key: 'artifact', label: 'Artifacts' },
]

const entityClass: Record<string, string> = {
  project: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
  plan: 'border-sky-500/30 bg-sky-500/10 text-sky-300',
  task: 'border-violet-500/30 bg-violet-500/10 text-violet-300',
  document: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-300',
  agent: 'border-fuchsia-500/30 bg-fuchsia-500/10 text-fuchsia-300',
  session: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
  artifact: 'border-teal-500/30 bg-teal-500/10 text-teal-300',
}

const searchIcon: Record<string, React.ReactNode> = {
  project: <FolderKanban className="h-3.5 w-3.5" />,
  plan: <GitFork className="h-3.5 w-3.5" />,
  task: <ListTree className="h-3.5 w-3.5" />,
  document: <BookOpen className="h-3.5 w-3.5" />,
  agent: <Bot className="h-3.5 w-3.5" />,
  session: <Activity className="h-3.5 w-3.5" />,
}

function formatTime(raw: string): string {
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString(undefined, {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

export const ActivityView: React.FC<ActivityViewProps> = ({ tick, projectSlug }) => {
  const { fetchEvents, searchAll } = useZenfdAPI()
  const [events, setEvents] = useState<EventMessage[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [entityFilter, setEntityFilter] = useState<EntityFilter>('all')

  const [query, setQuery] = useState('')
  const [searching, setSearching] = useState(false)
  const [results, setResults] = useState<SearchResult[] | null>(null)
  const [searchError, setSearchError] = useState('')
  const [scopeToProject, setScopeToProject] = useState(true)

  const loadEvents = useCallback(async () => {
    const result = await fetchEvents(0, 200)
    setLoading(false)
    if (!result.ok) {
      setError(result.error)
      return
    }
    setError('')
    setEvents(result.data)
  }, [fetchEvents])

  useEffect(() => {
    void loadEvents()
  }, [loadEvents])

  // SSE tick means the daemon wrote something. Debounce bursts so a plan push
  // does not trigger a fetch per row.
  useEffect(() => {
    if (tick === 0) return
    const timer = window.setTimeout(() => {
      void loadEvents()
    }, 250)
    return () => window.clearTimeout(timer)
  }, [tick, loadEvents])

  const visibleEvents = useMemo(() => {
    const filtered =
      entityFilter === 'all' ? events : events.filter((evt) => evt.entity_type === entityFilter)
    return [...filtered].reverse()
  }, [events, entityFilter])

  const handleSearch = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!query.trim()) {
      setResults(null)
      return
    }
    setSearching(true)
    setSearchError('')
    const result = await searchAll(query.trim(), scopeToProject ? projectSlug || '' : '')
    setSearching(false)
    if (!result.ok) {
      setSearchError(result.error)
      return
    }
    setResults(result.data)
  }

  return (
    <div className="flex h-full w-full overflow-hidden bg-zinc-950">
      {/* Activity stream */}
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden border-r border-zinc-800">
        <div className="shrink-0 space-y-3 border-b border-zinc-800 p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-amber-500/10 text-amber-400 ring-1 ring-amber-500/30">
                <Activity className="h-4.5 w-4.5" />
              </div>
              <div>
                <h2 className="text-sm font-bold text-zinc-100">Activity Ledger</h2>
                <p className="text-[11px] text-zinc-500">
                  Every writer — CLI, MCP, or web — lands here through SQLite audit triggers.
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded bg-zinc-800 px-2 py-1 font-mono text-[10px] text-zinc-400">
                {visibleEvents.length} / {events.length}
              </span>
              <button
                onClick={() => void loadEvents()}
                disabled={loading}
                className="flex items-center gap-1.5 rounded border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 disabled:opacity-50"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            </div>
          </div>

          <div className="flex flex-wrap gap-1">
            {entityFilters.map((filter) => (
              <button
                key={filter.key}
                onClick={() => setEntityFilter(filter.key)}
                className={`rounded-full border px-2.5 py-1 text-[11px] font-semibold transition ${
                  entityFilter === filter.key
                    ? 'border-amber-500/40 bg-amber-500/10 text-amber-300'
                    : 'border-zinc-800 bg-zinc-900 text-zinc-400 hover:text-zinc-200'
                }`}
              >
                {filter.label}
              </button>
            ))}
          </div>
        </div>

        {error && (
          <div className="mx-5 mt-3 rounded border border-rose-500/30 bg-rose-500/10 px-3 py-2 font-mono text-xs text-rose-300">
            {error}
          </div>
        )}

        <div className="flex-1 space-y-1 overflow-y-auto p-3">
          {loading ? (
            <div className="py-12 text-center font-mono text-xs text-zinc-500">loading events…</div>
          ) : visibleEvents.length === 0 ? (
            <div className="rounded-xl border border-dashed border-zinc-800 py-12 text-center font-mono text-xs text-zinc-500">
              No events for this filter yet.
            </div>
          ) : (
            visibleEvents.map((evt) => (
              <div
                key={evt.id}
                className="flex flex-wrap items-center gap-2 rounded border border-zinc-800/60 bg-zinc-900/30 px-2.5 py-1.5 font-mono text-[11px]"
              >
                  <span className="shrink-0 text-zinc-600">#{evt.id}</span>
                  <span className="shrink-0 text-zinc-600">{formatTime(evt.timestamp)}</span>
                  <span
                    className={`shrink-0 rounded border px-1.5 py-0.5 text-[9px] font-bold uppercase ${
                      entityClass[evt.entity_type] || 'border-zinc-700 bg-zinc-800 text-zinc-300'
                    }`}
                  >
                    {evt.entity_type}
                  </span>
                  <span className="min-w-0 flex-1 truncate text-zinc-300">{formatEvent(evt).text}</span>
                  {evt.project_slug && <span className="shrink-0 text-zinc-600">{evt.project_slug}</span>}
                  {evt.plan_slug && <span className="shrink-0 text-zinc-600">{evt.plan_slug}</span>}
                </div>
            ))
          )}
        </div>
      </div>

      {/* Cross-entity search */}
      <aside className="flex w-[390px] shrink-0 flex-col overflow-hidden bg-zinc-950">
        <div className="space-y-3 border-b border-zinc-800 p-4">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-sky-500/10 text-sky-400 ring-1 ring-sky-500/30">
              <Sparkles className="h-4 w-4" />
            </div>
            <div>
              <h3 className="text-xs font-bold uppercase tracking-wider text-zinc-300">Cross-entity search</h3>
              <p className="text-[10.5px] text-zinc-500">Projects, plans, tasks, documents, agents, sessions.</p>
            </div>
          </div>

          <form onSubmit={handleSearch} className="space-y-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-500" />
              <input
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value)
                  if (!e.target.value.trim()) setResults(null)
                }}
                placeholder="Search identifiers and titles…"
                className="w-full rounded border border-zinc-800 bg-zinc-950 py-1.5 pl-8 pr-2.5 text-xs text-zinc-200 outline-none focus:border-sky-500"
              />
            </div>
            <div className="flex items-center justify-between gap-2">
              <label className="flex items-center gap-1.5 text-[10.5px] text-zinc-400">
                <input
                  type="checkbox"
                  checked={scopeToProject}
                  onChange={(e) => setScopeToProject(e.target.checked)}
                  className="accent-sky-500"
                />
                scope to active project
              </label>
              <button
                type="submit"
                disabled={searching || !query.trim()}
                className="rounded bg-sky-500/15 px-3 py-1.5 text-xs font-semibold text-sky-300 ring-1 ring-sky-500/30 hover:bg-sky-500/25 disabled:opacity-40"
              >
                {searching ? 'Searching…' : 'Search'}
              </button>
            </div>
          </form>

          {searchError && (
            <div className="rounded border border-rose-500/30 bg-rose-500/10 px-2 py-1.5 font-mono text-[10.5px] text-rose-300">
              {searchError}
            </div>
          )}
        </div>

        <div className="flex-1 space-y-1.5 overflow-y-auto p-4">
          {results === null ? (
            <div className="rounded-xl border border-dashed border-zinc-800 py-10 text-center font-mono text-[10.5px] text-zinc-600">
              Run a query to search across entity identities.
            </div>
          ) : results.length === 0 ? (
            <div className="rounded-xl border border-dashed border-zinc-800 py-10 text-center font-mono text-[10.5px] text-zinc-600">
              No matching entity.
            </div>
          ) : (
            results.map((hit, index) => (
              <article
                key={`${hit.entity_type}:${hit.entity_id}:${index}`}
                className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-2.5"
              >
                <div className="flex items-center gap-1.5">
                  <span
                    className={`flex items-center gap-1 rounded border px-1.5 py-0.5 text-[9px] font-bold uppercase ${
                      entityClass[hit.entity_type] || 'border-zinc-700 bg-zinc-800 text-zinc-300'
                    }`}
                  >
                    {searchIcon[hit.entity_type] || <Search className="h-3 w-3" />}
                    {hit.entity_type}
                  </span>
                  {hit.project_slug && (
                    <span className="rounded bg-zinc-800 px-1.5 py-0.5 font-mono text-[9px] text-zinc-400">
                      {hit.project_slug}
                    </span>
                  )}
                </div>
                <h4 className="mt-1.5 truncate text-xs font-semibold text-zinc-100">{hit.title || hit.entity_id}</h4>
                <div className="truncate font-mono text-[10px] text-zinc-500">{hit.entity_id}</div>
                {hit.snippet && <p className="mt-1 line-clamp-2 text-[10.5px] text-zinc-400">{hit.snippet}</p>}
              </article>
            ))
          )}
        </div>
      </aside>
    </div>
  )
}
