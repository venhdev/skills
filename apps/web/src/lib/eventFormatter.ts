import type { EventMessage } from '../types/contracts'

export interface EventSummary {
  text: string
  tone: 'project' | 'plan' | 'task' | 'document' | 'agent' | 'session' | 'media' | 'muted'
}

/**
 * Safely decodes a JSON payload string from audit triggers.
 */
export function parseEventPayload(raw: string): Record<string, any> {
  try {
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

/**
 * Formats an EventMessage into a unified readable line and UI tone.
 */
export function formatEvent(evt: EventMessage): EventSummary {
  const p = parseEventPayload(evt.payload)
  switch (evt.type) {
    case 'project.created':
      return { text: `project created · ${p.name || p.slug || evt.entity_id}`, tone: 'project' }
    case 'project.updated':
      return {
        text: `project ${p.kind === 'status' ? `→ ${p.status}` : p.kind || 'updated'} · ${p.slug || evt.entity_id}`,
        tone: 'project',
      }
    case 'project.deleted':
      return { text: `project deleted · ${p.slug || evt.entity_id}`, tone: 'project' }
    case 'plan.created':
      return { text: `plan created · ${p.title || p.slug || evt.entity_id}`, tone: 'plan' }
    case 'plan.updated':
      return {
        text:
          p.kind === 'archived'
            ? `plan archived · ${p.slug || evt.entity_id}`
            : p.kind === 'unarchived'
              ? `plan restored · ${p.slug || evt.entity_id}`
              : `plan ${p.kind || 'updated'} · ${p.slug || evt.entity_id}${p.from && p.to && p.from !== p.to ? ` ${p.from}→${p.to}` : ''}`,
        tone: 'plan',
      }
    case 'plan.deleted':
      return { text: `plan deleted · ${p.slug || evt.entity_id}`, tone: 'plan' }
    case 'task.created':
      return { text: `task created · ${p.task_id || evt.entity_id} (${p.status || 'PENDING'})`, tone: 'task' }
    case 'task.updated': {
      const suffix =
        p.kind === 'status' && p.from && p.to
          ? ` ${p.from}→${p.to}`
          : p.kind === 'claim'
            ? ` claimed by ${p.claimed_by || 'worker'}`
            : ` ${p.kind || 'updated'}`
      return { text: `${p.task_id || evt.entity_id}${suffix}`, tone: 'task' }
    }
    case 'task.deleted':
      return { text: `task deleted · ${p.task_id || evt.entity_id}`, tone: 'task' }
    case 'document.created':
      return { text: `document registered · ${p.key || p.title || evt.entity_id}`, tone: 'document' }
    case 'document.updated':
      return { text: `document ${p.kind || 'updated'} · ${p.title || p.key || evt.entity_id}`, tone: 'document' }
    case 'document.deleted':
      return { text: `document removed · ${p.title || p.key || evt.entity_id}`, tone: 'document' }
    case 'document.tagged':
      return { text: `document #${p.tag} · ${p.document_id || evt.entity_id}`, tone: 'document' }
    case 'document.untagged':
      return { text: `document −#${p.tag} · ${p.document_id || evt.entity_id}`, tone: 'document' }
    case 'agent.created':
      return { text: `agent created · ${p.name || p.slug || evt.entity_id} (${p.persona || 'worker'})`, tone: 'agent' }
    case 'agent.updated':
      return { text: `agent updated · ${p.name || p.slug || evt.entity_id}`, tone: 'agent' }
    case 'agent.deleted':
      return { text: `agent deleted · ${p.name || p.slug || evt.entity_id}`, tone: 'agent' }
    case 'session.registered':
      return { text: `session registered · ${p.session_type || 'session'} ${p.agent_slug || p.id || evt.entity_id}`, tone: 'session' }
    case 'session.updated':
      return { text: `session ${p.status || 'updated'} · ${p.agent_slug || p.id || evt.entity_id}`, tone: 'session' }
    case 'session.deleted':
      return { text: `session removed · ${p.agent_slug || p.id || evt.entity_id}`, tone: 'session' }
    case 'artifact.added':
      return {
        text: `${p.role === 'INPUT' ? 'media attached' : 'artifact'} · ${p.name || p.artifact_id || 'item'} → ${p.task_id || 'task'}`,
        tone: 'media',
      }
    default:
      return { text: `${evt.type} · ${evt.task_id || evt.plan_slug || evt.entity_id || ''}`, tone: 'muted' }
  }
}

export const toneClass: Record<EventSummary['tone'], string> = {
  project: 'text-emerald-300',
  plan: 'text-sky-300',
  task: 'text-violet-300',
  document: 'text-amber-300',
  agent: 'text-rose-300',
  session: 'text-teal-300',
  media: 'text-cyan-300',
  muted: 'text-zinc-400',
}

export const toneBadgeClass: Record<EventSummary['tone'], string> = {
  project: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
  plan: 'bg-sky-500/10 text-sky-400 border-sky-500/20',
  task: 'bg-violet-500/10 text-violet-400 border-violet-500/20',
  document: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
  agent: 'bg-rose-500/10 text-rose-400 border-rose-500/20',
  session: 'bg-teal-500/10 text-teal-400 border-teal-500/20',
  media: 'bg-cyan-500/10 text-cyan-400 border-cyan-500/20',
  muted: 'bg-zinc-800 text-zinc-400 border-zinc-700',
}
