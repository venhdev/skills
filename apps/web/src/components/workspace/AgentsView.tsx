import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Bot,
  CircleSlash,
  Clock3,
  Pencil,
  PlayCircle,
  Plus,
  RefreshCw,
  Search,
  Trash2,
  UserCog,
} from 'lucide-react'
import { useZenfAPI } from '../../hooks/useZenfAPI'
import type {
  Agent,
  AgentMode,
  AgentPersona,
  AgentSession,
  SessionStatus,
  SessionType,
} from '../../types/contracts'

interface AgentsViewProps {
  tick: number
}

interface AgentForm {
  slug: string
  name: string
  persona: AgentPersona
  mode: AgentMode
  goal: string
  start_conditions: string
  stop_conditions: string
  schedule: string
  skills: string
  model: string
  runner: string
  enabled: boolean
}

const emptyForm: AgentForm = {
  slug: '',
  name: '',
  persona: 'CODER',
  mode: 'HEADLESS',
  goal: '',
  start_conditions: '',
  stop_conditions: '',
  schedule: '',
  skills: '',
  model: '',
  runner: '',
  enabled: true,
}

const personas: AgentPersona[] = ['EXPLORER', 'CODER', 'REVIEWER', 'CUSTOM']
const modes: AgentMode[] = ['HEADLESS', 'INTERACTIVE']

const personaClass: Record<AgentPersona, string> = {
  EXPLORER: 'border-sky-500/30 bg-sky-500/10 text-sky-300',
  CODER: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
  REVIEWER: 'border-violet-500/30 bg-violet-500/10 text-violet-300',
  CUSTOM: 'border-zinc-700 bg-zinc-800 text-zinc-300',
}

const sessionStatusClass = (status: SessionStatus) => {
  switch (status) {
    case 'ACTIVE':
      return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300'
    case 'IDLE':
      return 'border-sky-500/30 bg-sky-500/10 text-sky-300'
    case 'DISCONNECTED':
      return 'border-amber-500/30 bg-amber-500/10 text-amber-300'
    case 'TERMINATED':
      return 'border-zinc-700 bg-zinc-800 text-zinc-400'
  }
}

const sessionTypeClass: Record<SessionType, string> = {
  MAIN: 'border-rose-500/30 bg-rose-500/10 text-rose-300',
  SUBAGENT: 'border-fuchsia-500/30 bg-fuchsia-500/10 text-fuchsia-300',
  WORKER: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
}

function splitLines(raw: string): string[] {
  return raw
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

function splitTags(raw: string): string[] {
  return raw
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function formatRelative(raw: string): string {
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  const delta = Date.now() - date.getTime()
  const seconds = Math.round(delta / 1000)
  if (seconds < 60) return `${Math.max(seconds, 0)}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.round(hours / 24)}d ago`
}

export const AgentsView: React.FC<AgentsViewProps> = ({ tick }) => {
  const {
    fetchAgents,
    createAgent,
    updateAgent,
    deleteAgent,
    fetchSessions,
  } = useZenfAPI()

  const [agents, setAgents] = useState<Agent[]>([])
  const [sessions, setSessions] = useState<AgentSession[]>([])
  const [includeDisabled, setIncludeDisabled] = useState(true)
  const [sessionAgentFilter, setSessionAgentFilter] = useState('')
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [mode, setMode] = useState<'create' | 'edit'>('create')
  const [selectedSlug, setSelectedSlug] = useState<string | null>(null)
  const [form, setForm] = useState<AgentForm>(emptyForm)

  const loadAgents = useCallback(async () => {
    setLoading(true)
    const result = await fetchAgents(includeDisabled)
    setLoading(false)
    if (!result.ok) {
      setError(result.error)
      return
    }
    setError('')
    setAgents(result.data)
  }, [fetchAgents, includeDisabled])

  const loadSessions = useCallback(async () => {
    const result = await fetchSessions({
      agentSlug: sessionAgentFilter,
      limit: 50,
    })
    if (result.ok) setSessions(result.data)
  }, [fetchSessions, sessionAgentFilter])

  useEffect(() => {
    void loadAgents()
  }, [loadAgents])

  useEffect(() => {
    void loadSessions()
  }, [loadSessions, tick])

  const sessionCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const session of sessions) {
      if (!session.agent_slug) continue
      if (session.status === 'TERMINATED') continue
      counts.set(session.agent_slug, (counts.get(session.agent_slug) || 0) + 1)
    }
    return counts
  }, [sessions])

  const resetForm = () => {
    setMode('create')
    setSelectedSlug(null)
    setForm(emptyForm)
  }

  const startEdit = (agent: Agent) => {
    setMode('edit')
    setSelectedSlug(agent.slug)
    setForm({
      slug: agent.slug,
      name: agent.name,
      persona: agent.persona,
      mode: agent.mode,
      goal: agent.goal || '',
      start_conditions: (agent.start_conditions || []).join('\n'),
      stop_conditions: (agent.stop_conditions || []).join('\n'),
      schedule: agent.schedule || '',
      skills: (agent.skills || []).join(', '),
      model: agent.model || '',
      runner: agent.runner || '',
      enabled: agent.enabled,
    })
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')

    if (mode === 'create') {
      if (!form.slug.trim() || !form.name.trim()) {
        setError('slug and name are required')
        return
      }
      setBusy('create')
      const result = await createAgent({
        slug: form.slug.trim(),
        name: form.name.trim(),
        persona: form.persona,
        mode: form.mode,
        goal: form.goal.trim(),
        start_conditions: splitLines(form.start_conditions),
        stop_conditions: splitLines(form.stop_conditions),
        schedule: form.schedule.trim(),
        skills: splitTags(form.skills),
        model: form.model.trim(),
        runner: form.runner.trim(),
        enabled: form.enabled,
      })
      setBusy('')
      if (!result.ok) {
        setError(result.error)
        return
      }
      resetForm()
      await loadAgents()
      return
    }

    if (!selectedSlug) return
    setBusy('update')
    const result = await updateAgent(selectedSlug, {
      name: form.name.trim(),
      persona: form.persona,
      mode: form.mode,
      goal: form.goal.trim(),
      start_conditions: splitLines(form.start_conditions),
      stop_conditions: splitLines(form.stop_conditions),
      schedule: form.schedule.trim(),
      skills: splitTags(form.skills),
      model: form.model.trim(),
      runner: form.runner.trim(),
      enabled: form.enabled,
    })
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    await loadAgents()
  }

  const handleToggleEnabled = async (agent: Agent) => {
    setBusy(`toggle:${agent.slug}`)
    setError('')
    const result = await updateAgent(agent.slug, { enabled: !agent.enabled })
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    await loadAgents()
  }

  const handleDelete = async (agent: Agent) => {
    if (!window.confirm(`Delete agent definition "${agent.name}"? Observed sessions are kept.`)) return
    setBusy(`delete:${agent.slug}`)
    setError('')
    const result = await deleteAgent(agent.slug)
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    if (selectedSlug === agent.slug) resetForm()
    await loadAgents()
  }

  return (
    <div className="flex h-full w-full overflow-hidden bg-zinc-950">
      {/* Agent registry */}
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden border-r border-zinc-800">
        <div className="shrink-0 space-y-3 border-b border-zinc-800 p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-violet-500/10 text-violet-400 ring-1 ring-violet-500/30">
                <Bot className="h-4.5 w-4.5" />
              </div>
              <div>
                <h2 className="text-sm font-bold text-zinc-100">Agent Definitions</h2>
                <p className="text-[11px] text-zinc-500">
                  WORKER personas with goals, start/stop conditions, and a runner. Sessions are observed, never driven
                  from the web.
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setIncludeDisabled((value) => !value)}
                className={`rounded border px-2.5 py-1.5 text-xs font-semibold transition ${
                  includeDisabled
                    ? 'border-zinc-700 bg-zinc-800 text-zinc-200'
                    : 'border-zinc-800 bg-zinc-900 text-zinc-400 hover:text-zinc-200'
                }`}
              >
                {includeDisabled ? 'All agents' : 'Enabled only'}
              </button>
              <button
                onClick={() => void loadAgents()}
                disabled={loading}
                className="flex items-center gap-1.5 rounded border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 disabled:opacity-50"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
                Reload
              </button>
              <button
                onClick={resetForm}
                className="flex items-center gap-1.5 rounded bg-emerald-500/15 px-2.5 py-1.5 text-xs font-semibold text-emerald-300 ring-1 ring-emerald-500/30 hover:bg-emerald-500/25"
              >
                <Plus className="h-3.5 w-3.5" /> New agent
              </button>
            </div>
          </div>
        </div>

        {error && (
          <div className="mx-5 mt-3 rounded border border-rose-500/30 bg-rose-500/10 px-3 py-2 font-mono text-xs text-rose-300">
            {error}
          </div>
        )}

        <div className="flex-1 space-y-2 overflow-y-auto p-5">
          {loading ? (
            <div className="py-12 text-center font-mono text-xs text-zinc-500">loading agents…</div>
          ) : agents.length === 0 ? (
            <div className="rounded-xl border border-dashed border-zinc-800 py-12 text-center font-mono text-xs text-zinc-500">
              No agent definitions yet. Create one, or call the <span className="text-emerald-400">agents</span> MCP
              tool.
            </div>
          ) : (
            agents.map((agent) => (
              <article
                key={agent.slug}
                className={`rounded-lg border bg-zinc-900/40 p-3 transition ${
                  selectedSlug === agent.slug ? 'border-violet-500/40' : 'border-zinc-800 hover:border-zinc-700'
                }`}
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <span className={`rounded border px-1.5 py-0.5 text-[9.5px] font-bold ${personaClass[agent.persona]}`}>
                        {agent.persona}
                      </span>
                      <span className="rounded border border-zinc-700 bg-zinc-800 px-1.5 py-0.5 text-[9.5px] font-mono text-zinc-300">
                        {agent.mode}
                      </span>
                      <span
                        className={`rounded border px-1.5 py-0.5 text-[9.5px] font-semibold ${
                          agent.enabled
                            ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300'
                            : 'border-zinc-700 bg-zinc-800 text-zinc-400'
                        }`}
                      >
                        {agent.enabled ? 'ENABLED' : 'DISABLED'}
                      </span>
                      {(sessionCounts.get(agent.slug) || 0) > 0 && (
                        <span className="flex items-center gap-1 rounded bg-zinc-800 px-1.5 py-0.5 text-[9.5px] text-sky-300">
                          <PlayCircle className="h-2.5 w-2.5" />
                          {sessionCounts.get(agent.slug)} live
                        </span>
                      )}
                    </div>
                    <h3 className="mt-1.5 truncate text-sm font-semibold text-zinc-100">
                      {agent.name}
                      <span className="ml-2 font-mono text-[10.5px] font-normal text-zinc-500">{agent.slug}</span>
                    </h3>
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <button
                      onClick={() => void handleToggleEnabled(agent)}
                      disabled={busy === `toggle:${agent.slug}`}
                      title={agent.enabled ? 'Disable agent' : 'Enable agent'}
                      className="rounded border border-zinc-800 p-1 text-zinc-400 hover:text-emerald-300 disabled:opacity-40"
                    >
                      {agent.enabled ? <CircleSlash className="h-3 w-3" /> : <PlayCircle className="h-3 w-3" />}
                    </button>
                    <button
                      onClick={() => startEdit(agent)}
                      className="rounded border border-zinc-800 p-1 text-zinc-400 hover:text-violet-300"
                    >
                      <Pencil className="h-3 w-3" />
                    </button>
                    <button
                      onClick={() => void handleDelete(agent)}
                      disabled={busy === `delete:${agent.slug}`}
                      className="rounded border border-zinc-800 p-1 text-zinc-500 hover:text-rose-300 disabled:opacity-40"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                </div>

                {agent.goal && <p className="mt-2 line-clamp-2 text-xs text-zinc-400">{agent.goal}</p>}

                <div className="mt-2 flex flex-wrap items-center gap-3 font-mono text-[10px] text-zinc-600">
                  {agent.schedule && (
                    <span className="flex items-center gap-1">
                      <Clock3 className="h-2.5 w-2.5" /> {agent.schedule}
                    </span>
                  )}
                  {agent.model && <span>model:{agent.model}</span>}
                  {agent.runner && <span>runner:{agent.runner}</span>}
                  <span>
                    {agent.start_conditions?.length || 0} start · {agent.stop_conditions?.length || 0} stop
                  </span>
                </div>

                {(agent.skills || []).length > 0 && (
                  <div className="mt-1.5 flex flex-wrap gap-1">
                    {(agent.skills || []).map((skill) => (
                      <span
                        key={skill}
                        className="rounded bg-zinc-800/80 px-1.5 py-0.5 font-mono text-[9.5px] text-zinc-400"
                      >
                        {skill}
                      </span>
                    ))}
                  </div>
                )}
              </article>
            ))
          )}
        </div>

        {/* Observed sessions */}
        <div className="shrink-0 border-t border-zinc-800 bg-zinc-950/80 p-4">
          <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
            <h3 className="flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-400">
              <Search className="h-3.5 w-3.5 text-sky-400" />
              Observed sessions
              <span className="rounded bg-zinc-800 px-1.5 py-0.5 font-mono text-[9.5px] text-zinc-400">
                {sessions.length}
              </span>
            </h3>
            <select
              value={sessionAgentFilter}
              onChange={(e) => setSessionAgentFilter(e.target.value)}
              className="rounded border border-zinc-800 bg-zinc-950 px-2 py-1 text-[11px] text-zinc-300 outline-none focus:border-sky-500"
            >
              <option value="">All agents</option>
              {agents.map((agent) => (
                <option key={agent.slug} value={agent.slug}>
                  {agent.name}
                </option>
              ))}
            </select>
          </div>
          <div className="max-h-44 space-y-1 overflow-y-auto">
            {sessions.length === 0 ? (
              <div className="py-4 text-center font-mono text-[10.5px] text-zinc-600">
                No sessions observed. External runners register through the sessions MCP tool.
              </div>
            ) : (
              sessions.map((session) => (
                <div
                  key={session.id}
                  className="flex flex-wrap items-center gap-2 rounded border border-zinc-800/70 bg-zinc-900/40 px-2 py-1 font-mono text-[10px]"
                >
                  <span className={`rounded border px-1.5 py-0.5 text-[9px] font-bold ${sessionTypeClass[session.session_type]}`}>
                    {session.session_type}
                  </span>
                  <span className={`rounded border px-1.5 py-0.5 text-[9px] font-semibold ${sessionStatusClass(session.status)}`}>
                    {session.status}
                  </span>
                  <span className="truncate text-zinc-300">{session.agent_slug || 'unbound'}</span>
                  <span className="truncate text-zinc-600">{session.id}</span>
                  {session.parent_id && <span className="truncate text-zinc-600">parent:{session.parent_id}</span>}
                  <span className="ml-auto shrink-0 text-zinc-600">{formatRelative(session.last_seen_at)}</span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Persona editor */}
      <aside className="flex w-[390px] shrink-0 flex-col overflow-y-auto bg-zinc-950 p-4">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wider text-zinc-300">
            <UserCog className="h-4 w-4 text-violet-400" />
            {mode === 'create' ? 'Create agent' : 'Edit agent'}
          </h3>
          {mode === 'edit' && (
            <button onClick={resetForm} className="text-[11px] text-zinc-500 hover:text-zinc-200">
              + New
            </button>
          )}
        </div>

        <form onSubmit={handleSubmit} className="space-y-2.5">
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Slug</label>
              <input
                value={form.slug}
                onChange={(e) => setForm({ ...form, slug: e.target.value })}
                disabled={mode === 'edit'}
                placeholder="coder-alpha"
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-violet-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Name</label>
              <input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="Coder Alpha"
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-200 outline-none focus:border-violet-500"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Persona</label>
              <select
                value={form.persona}
                onChange={(e) => setForm({ ...form, persona: e.target.value as AgentPersona })}
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-300 outline-none focus:border-violet-500"
              >
                {personas.map((persona) => (
                  <option key={persona} value={persona}>
                    {persona}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Mode</label>
              <select
                value={form.mode}
                onChange={(e) => setForm({ ...form, mode: e.target.value as AgentMode })}
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-300 outline-none focus:border-violet-500"
              >
                {modes.map((agentMode) => (
                  <option key={agentMode} value={agentMode}>
                    {agentMode}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Goal</label>
            <textarea
              value={form.goal}
              onChange={(e) => setForm({ ...form, goal: e.target.value })}
              rows={3}
              placeholder="Implement READY tasks within the assigned plan scope."
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-200 outline-none focus:border-violet-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">
                Start conditions
              </label>
              <textarea
                value={form.start_conditions}
                onChange={(e) => setForm({ ...form, start_conditions: e.target.value })}
                rows={4}
                placeholder={'task is READY\nno active claim'}
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-[11px] text-zinc-300 outline-none focus:border-violet-500"
              />
            </div>
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">
                Stop conditions
              </label>
              <textarea
                value={form.stop_conditions}
                onChange={(e) => setForm({ ...form, stop_conditions: e.target.value })}
                rows={4}
                placeholder={'task AWAITING_REVIEW\nverification failed'}
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-[11px] text-zinc-300 outline-none focus:border-violet-500"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Schedule</label>
              <input
                value={form.schedule}
                onChange={(e) => setForm({ ...form, schedule: e.target.value })}
                placeholder="*/15 * * * *"
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-violet-500"
              />
            </div>
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Model</label>
              <input
                value={form.model}
                onChange={(e) => setForm({ ...form, model: e.target.value })}
                placeholder="provider/model"
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-violet-500"
              />
            </div>
          </div>

          <div>
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Runner</label>
            <input
              value={form.runner}
              onChange={(e) => setForm({ ...form, runner: e.target.value })}
              placeholder="external runner command or label"
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-violet-500"
            />
          </div>

          <div>
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">
              Skills (comma separated)
            </label>
            <input
              value={form.skills}
              onChange={(e) => setForm({ ...form, skills: e.target.value })}
              placeholder="tdd, purify, review"
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-200 outline-none focus:border-violet-500"
            />
          </div>

          <label className="flex items-center gap-2 text-[11px] text-zinc-400">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
              className="accent-violet-500"
            />
            enabled for dispatch
          </label>

          <div className="flex gap-2 pt-1">
            <button
              type="submit"
              disabled={busy === 'create' || busy === 'update'}
              className="flex-1 rounded bg-violet-500/15 px-3 py-1.5 text-xs font-bold text-violet-300 ring-1 ring-violet-500/30 hover:bg-violet-500/25 disabled:opacity-50"
            >
              {mode === 'create' ? 'Create agent' : 'Save agent'}
            </button>
            {mode === 'edit' && (
              <button
                type="button"
                onClick={resetForm}
                className="rounded border border-zinc-800 px-3 py-1.5 text-xs text-zinc-400 hover:text-zinc-200"
              >
                Cancel
              </button>
            )}
          </div>
        </form>
      </aside>
    </div>
  )
}
