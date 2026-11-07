import { useCallback } from 'react'
import type {
  Agent,
  AgentInput,
  AgentSession,
  Artifact,
  ContextPacket,
  Document,
  DocumentFilter,
  DocumentInput,
  DocumentSearchResult,
  DocumentStatus,
  EventMessage,
  Plan,
  Project,
  ProjectInput,
  ProjectStatus,
  SearchResult,
  Task,
} from '../types/contracts'

export type ApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: string; code?: string; status?: number }

async function apiRequest<T>(path: string, init?: RequestInit): Promise<ApiResult<T>> {
  try {
    const res = await fetch(path, init)
    if (!res.ok) {
      const text = await res.text()
      let errorMsg = text.trim() || res.statusText || `HTTP ${res.status}`
      let errorCode: string | undefined
      try {
        const parsed = JSON.parse(text)
        if (parsed?.error?.message) {
          errorMsg = parsed.error.message
          errorCode = parsed.error.code
        } else if (typeof parsed?.error === 'string') {
          errorMsg = parsed.error
        }
      } catch {
        // text is plain error
      }
      return {
        ok: false,
        error: errorMsg,
        code: errorCode,
        status: res.status,
      }
    }
    if (res.status === 204) {
      return { ok: true, data: undefined as T }
    }
    const text = await res.text()
    if (!text) {
      return { ok: true, data: undefined as T }
    }
    return { ok: true, data: JSON.parse(text) as T }
  } catch (err) {
    return {
      ok: false,
      error: err instanceof Error ? err.message : String(err),
    }
  }
}

function jsonInit(method: 'POST' | 'PATCH', body: unknown): RequestInit {
  return {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }
}

function queryString(params: Record<string, string | number | boolean | undefined>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === '' || value === false) continue
    search.set(key, String(value))
  }
  const encoded = search.toString()
  return encoded ? `?${encoded}` : ''
}

export function useZenfdAPI() {
  // -------------------------------------------------------------------------
  // Projects
  // -------------------------------------------------------------------------
  const fetchProjects = useCallback(async (includeArchived = false): Promise<ApiResult<Project[]>> => {
    const result = await apiRequest<{ projects?: Project[] }>(
      `/api/v1/projects${includeArchived ? '?include_archived=true' : ''}`
    )
    return result.ok ? { ok: true, data: result.data?.projects || [] } : result
  }, [])

  const createProject = useCallback(async (input: ProjectInput): Promise<ApiResult<Project>> => {
    return apiRequest<Project>('/api/v1/projects', jsonInit('POST', input))
  }, [])

  const updateProject = useCallback(
    async (
      slug: string,
      patch: Partial<ProjectInput> & { status?: ProjectStatus }
    ): Promise<ApiResult<Project>> => {
      return apiRequest<Project>(`/api/v1/projects/${encodeURIComponent(slug)}`, jsonInit('PATCH', patch))
    },
    []
  )

  const deleteProject = useCallback(async (slug: string): Promise<ApiResult<void>> => {
    return apiRequest<void>(`/api/v1/projects/${encodeURIComponent(slug)}`, { method: 'DELETE' })
  }, [])

  // -------------------------------------------------------------------------
  // Plans and tasks
  // -------------------------------------------------------------------------
  const fetchPlans = useCallback(
    async (projectSlug = '', includeArchived = false): Promise<ApiResult<Plan[]>> => {
      const query = queryString({
        project_slug: projectSlug,
        include_archived: includeArchived,
      })
      const result = await apiRequest<{ plans?: Plan[] }>(`/api/v1/plans${query}`)
      return result.ok ? { ok: true, data: result.data?.plans || [] } : result
    },
    []
  )

  const fetchPlan = useCallback(
    async (slug: string): Promise<ApiResult<{ plan: Plan; tasks: Task[] }>> => {
      return apiRequest<{ plan: Plan; tasks: Task[] }>(`/api/v1/plans/${encodeURIComponent(slug)}`)
    },
    []
  )

  const fetchTasks = useCallback(
    async (filter?: {
      projectSlug?: string
      planSlug?: string
      status?: string
      agentSlug?: string
      orphan?: boolean
    }): Promise<ApiResult<Task[]>> => {
      const query = queryString({
        project_slug: filter?.projectSlug,
        plan_slug: filter?.planSlug,
        status: filter?.status,
        agent_slug: filter?.agentSlug,
        orphan: filter?.orphan,
      })
      const result = await apiRequest<{ tasks?: Task[] } | Task[]>(`/api/v1/tasks${query}`)
      if (!result.ok) return result
      const tasks = Array.isArray(result.data) ? result.data : result.data?.tasks || []
      return { ok: true, data: tasks }
    },
    []
  )

  const fetchTaskArtifacts = useCallback(
    async (taskId: string, role?: string): Promise<ApiResult<Artifact[]>> => {
      const query = role ? `?role=${encodeURIComponent(role)}` : ''
      const result = await apiRequest<{ artifacts?: Artifact[] } | Artifact[]>(
        `/api/v1/tasks/${encodeURIComponent(taskId)}/artifacts${query}`
      )
      if (!result.ok) return result
      const artifacts = Array.isArray(result.data) ? result.data : result.data?.artifacts || []
      return { ok: true, data: artifacts }
    },
    []
  )

  const fetchTaskPacket = useCallback(
    async (taskId: string): Promise<ApiResult<ContextPacket>> => {
      return apiRequest<ContextPacket>(`/api/v1/tasks/${encodeURIComponent(taskId)}?include_packet=true`)
    },
    []
  )

  // -------------------------------------------------------------------------
  // Documents
  // -------------------------------------------------------------------------
  const fetchDocuments = useCallback(async (filter?: DocumentFilter): Promise<ApiResult<Document[]>> => {
    const query = queryString({
      project_slug: filter?.projectSlug,
      include_global: filter?.includeGlobal,
      type: filter?.type,
      scope: filter?.scope,
      status: filter?.status,
      tag: filter?.tag,
      limit: filter?.limit,
    })
    const result = await apiRequest<{ documents?: Document[] }>(`/api/v1/docs${query}`)
    return result.ok ? { ok: true, data: result.data?.documents || [] } : result
  }, [])

  const searchDocuments = useCallback(
    async (query: string, projectSlug = ''): Promise<ApiResult<DocumentSearchResult[]>> => {
      const params = queryString({ project_slug: projectSlug, include_global: true, limit: 50 })
      const result = await apiRequest<{ results?: DocumentSearchResult[] } | DocumentSearchResult[]>(
        `/api/v1/docs/search${params}${params ? '&' : '?'}query=${encodeURIComponent(query)}`
      )
      if (!result.ok) return result
      const list = Array.isArray(result.data) ? result.data : (result.data?.results || [])
      return { ok: true, data: list }
    },
    []
  )

  const createDocument = useCallback(async (input: DocumentInput): Promise<ApiResult<Document>> => {
    return apiRequest<Document>('/api/v1/docs', jsonInit('POST', input))
  }, [])

  const updateDocument = useCallback(
    async (
      id: string,
      patch: { title?: string; summary?: string; status?: DocumentStatus; path?: string; tags?: string[] }
    ): Promise<ApiResult<Document>> => {
      return apiRequest<Document>(`/api/v1/docs/${encodeURIComponent(id)}`, jsonInit('PATCH', patch))
    },
    []
  )

  const refreshDocument = useCallback(async (id: string): Promise<ApiResult<Document>> => {
    return apiRequest<Document>(`/api/v1/docs/${encodeURIComponent(id)}/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
    })
  }, [])

  const deleteDocument = useCallback(async (id: string): Promise<ApiResult<void>> => {
    return apiRequest<void>(`/api/v1/docs/${encodeURIComponent(id)}`, { method: 'DELETE' })
  }, [])

  // -------------------------------------------------------------------------
  // Agents and observed sessions
  // -------------------------------------------------------------------------
  const fetchAgents = useCallback(async (includeDisabled = false): Promise<ApiResult<Agent[]>> => {
    const query = includeDisabled ? '?include_disabled=true' : ''
    const result = await apiRequest<{ agents?: Agent[] }>(`/api/v1/agents${query}`)
    return result.ok ? { ok: true, data: result.data?.agents || [] } : result
  }, [])

  const createAgent = useCallback(async (input: AgentInput): Promise<ApiResult<Agent>> => {
    return apiRequest<Agent>('/api/v1/agents', jsonInit('POST', input))
  }, [])

  const updateAgent = useCallback(
    async (slug: string, patch: Partial<AgentInput>): Promise<ApiResult<Agent>> => {
      return apiRequest<Agent>(`/api/v1/agents/${encodeURIComponent(slug)}`, jsonInit('PATCH', patch))
    },
    []
  )

  const deleteAgent = useCallback(async (slug: string): Promise<ApiResult<void>> => {
    return apiRequest<void>(`/api/v1/agents/${encodeURIComponent(slug)}`, { method: 'DELETE' })
  }, [])

  const fetchSessions = useCallback(
    async (filter?: {
      type?: string
      agentSlug?: string
      status?: string
      activeOnly?: boolean
      limit?: number
    }): Promise<ApiResult<AgentSession[]>> => {
      const query = queryString({
        session_type: filter?.type,
        agent_slug: filter?.agentSlug,
        status: filter?.status,
        active_only: filter?.activeOnly,
        limit: filter?.limit || 50,
      })
      const result = await apiRequest<{ sessions?: AgentSession[] }>(`/api/v1/sessions${query}`)
      return result.ok ? { ok: true, data: result.data?.sessions || [] } : result
    },
    []
  )

  // -------------------------------------------------------------------------
  // Activity and search
  // -------------------------------------------------------------------------
  const fetchEvents = useCallback(
    async (afterId = 0, limit = 200): Promise<ApiResult<EventMessage[]>> => {
      const query = queryString({ after_id: afterId || undefined, limit })
      const result = await apiRequest<EventMessage[]>(`/api/v1/events${query}`)
      return result.ok ? { ok: true, data: result.data || [] } : result
    },
    []
  )

  const searchAll = useCallback(async (query: string, projectSlug = ''): Promise<ApiResult<SearchResult[]>> => {
    const params = queryString({ q: query, project_slug: projectSlug, limit: 50 })
    const result = await apiRequest<{ results?: SearchResult[] }>(`/api/v1/search${params}`)
    return result.ok ? { ok: true, data: result.data?.results || [] } : result
  }, [])

  const fetchGlobalEnv = useCallback(async (): Promise<ApiResult<Record<string, any>>> => {
    return apiRequest<Record<string, any>>('/api/v1/env/global')
  }, [])

  return {
    fetchProjects,
    createProject,
    updateProject,
    deleteProject,
    fetchPlans,
    fetchPlan,
    fetchTasks,
    fetchTaskArtifacts,
    fetchTaskPacket,
    fetchDocuments,
    searchDocuments,
    createDocument,
    updateDocument,
    refreshDocument,
    deleteDocument,
    fetchAgents,
    createAgent,
    updateAgent,
    deleteAgent,
    fetchSessions,
    fetchEvents,
    searchAll,
    fetchGlobalEnv,
  }
}
