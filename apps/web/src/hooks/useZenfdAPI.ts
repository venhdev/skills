import { useState, useCallback } from 'react'
import type {
  Plan,
  Task,
  Artifact,
  ProjectDoc,
  ContextPacket,
  AgentSession,
  TechSpec,
  PlanRevision,
  SnapshotFile,
  FileDiff,
} from '../types/contracts'

export function useZenfdAPI() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchProjects = useCallback(async (): Promise<string[]> => {
    try {
      const res = await fetch('/api/v1/projects')
      if (!res.ok) return []
      const data = await res.json()
      return data.projects || []
    } catch {
      return []
    }
  }, [])

  const fetchPlans = useCallback(async (repoPath?: string): Promise<string[]> => {
    try {
      const url = repoPath ? `/api/v1/plans?repo_path=${encodeURIComponent(repoPath)}` : '/api/v1/plans'
      const res = await fetch(url)
      if (!res.ok) return []
      const data = await res.json()
      return data.plans || []
    } catch {
      return []
    }
  }, [])

  const createPlan = useCallback(async (slug: string, title: string, repoPath?: string): Promise<boolean> => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/v1/plans', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ slug, title, repo_path: repoPath || '', tasks: [] }),
      })
      if (!res.ok) throw new Error(`Failed to create plan: ${res.statusText}`)
      return true
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      return false
    } finally {
      setLoading(false)
    }
  }, [])

  const deletePlan = useCallback(async (slug: string): Promise<boolean> => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/v1/plans/${encodeURIComponent(slug)}`, {
        method: 'DELETE',
      })
      if (!res.ok) throw new Error(`Failed to delete plan: ${res.statusText}`)
      return true
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      return false
    } finally {
      setLoading(false)
    }
  }, [])

  const createTask = useCallback(async (task: Partial<Task>): Promise<Task | null> => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/v1/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(task),
      })
      if (!res.ok) throw new Error(`Failed to create task: ${res.statusText}`)
      return await res.json()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const updateTaskDependencies = useCallback(async (taskId: string, dependencies: string[]): Promise<boolean> => {
    try {
      const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/dependencies`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dependencies }),
      })
      return res.ok
    } catch {
      return false
    }
  }, [])

  const updateTaskNotes = useCallback(async (taskId: string, notes: string): Promise<boolean> => {
    try {
      const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/context`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notes }),
      })
      return res.ok
    } catch {
      return false
    }
  }, [])

  const reclaimTask = useCallback(async (taskId: string, workerId?: string, orchestratorId?: string): Promise<ContextPacket | null> => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/reclaim`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ worker_id: workerId, orchestrator_id: orchestratorId }),
      })
      if (!res.ok) throw new Error(`Failed to reclaim task: ${res.statusText}`)
      return await res.json()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const evictTask = useCallback(async (taskId: string, reason?: string, action?: 'RESET_TO_READY' | 'CANCEL'): Promise<boolean> => {
    try {
      const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/evict`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: reason || 'Evicted via Cockpit', action: action || 'RESET_TO_READY' }),
      })
      return res.ok
    } catch {
      return false
    }
  }, [])

  const fetchSessions = useCallback(async (): Promise<AgentSession[]> => {
    try {
      const res = await fetch('/api/v1/sessions')
      if (!res.ok) return []
      const data = await res.json()
      return data.sessions || []
    } catch {
      return []
    }
  }, [])

  const fetchPlan = useCallback(async (slug: string): Promise<{ plan: Plan; tasks: Task[] } | null> => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/v1/plans/${encodeURIComponent(slug)}`)
      if (!res.ok) {
        throw new Error(`Failed to fetch plan ${slug}: ${res.statusText}`)
      }
      return await res.json()
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchTasks = useCallback(async (repoPath?: string, status?: string): Promise<Task[] | null> => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams()
      if (repoPath) params.set('repo_path', repoPath)
      if (status) params.set('status', status)
      const res = await fetch(`/api/v1/tasks?${params.toString()}`)
      if (!res.ok) {
        throw new Error(`Failed to fetch tasks: ${res.statusText}`)
      }
      const data = await res.json()
      return data || []
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const uploadAttachment = useCallback(
    async (taskId: string, file: File, name?: string): Promise<Artifact | null> => {
      setLoading(true)
      setError(null)
      try {
        const formData = new FormData()
        formData.append('file', file)
        if (name) formData.append('name', name)

        const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/attachments`, {
          method: 'POST',
          body: formData,
        })
        if (!res.ok) {
          throw new Error(`Upload failed: ${res.statusText}`)
        }
        return await res.json()
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg)
        return null
      } finally {
        setLoading(false)
      }
    },
    []
  )

  const fetchTaskArtifacts = useCallback(
    async (taskId: string, role?: string): Promise<Artifact[]> => {
      try {
        const url = role
          ? `/api/v1/tasks/${encodeURIComponent(taskId)}/artifacts?role=${encodeURIComponent(role)}`
          : `/api/v1/tasks/${encodeURIComponent(taskId)}/artifacts`
        const res = await fetch(url)
        if (!res.ok) return []
        return await res.json()
      } catch {
        return []
      }
    },
    []
  )

  const updateTaskStatus = useCallback(
    async (taskId: string, status: string): Promise<boolean> => {
      setLoading(true)
      setError(null)
      try {
        const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/status`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ status }),
        })
        if (!res.ok) {
          throw new Error(`Failed to update task status: ${res.statusText}`)
        }
        return true
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg)
        return false
      } finally {
        setLoading(false)
      }
    },
    []
  )

  const resolveReview = useCallback(
    async (taskId: string, response: string): Promise<boolean> => {
      setLoading(true)
      setError(null)
      try {
        const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/resolve`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ response }),
        })
        if (!res.ok) {
          throw new Error(`Failed to resolve review: ${res.statusText}`)
        }
        return true
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg)
        return false
      } finally {
        setLoading(false)
      }
    },
    []
  )

  const mergeTask = useCallback(
    async (taskId: string): Promise<{ merged: boolean; conflict_files?: string[]; status: string } | null> => {
      setLoading(true)
      setError(null)
      try {
        const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/merge`, {
          method: 'POST',
        })
        if (!res.ok) {
          throw new Error(`Failed to merge task: ${res.statusText}`)
        }
        return await res.json()
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg)
        return null
      } finally {
        setLoading(false)
      }
    },
    []
  )

  const fetchTaskLogs = useCallback(
    async (taskId: string): Promise<string> => {
      try {
        const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/logs`)
        if (!res.ok) return ''
        return await res.text()
      } catch {
        return ''
      }
    },
    []
  )

  const fetchDocs = useCallback(
    async (repoPath?: string, category?: string): Promise<ProjectDoc[]> => {
      setLoading(true)
      setError(null)
      try {
        const params = new URLSearchParams()
        if (repoPath) params.set('repo_path', repoPath)
        if (category) params.set('category', category)
        const res = await fetch(`/api/v1/docs?${params.toString()}`)
        if (!res.ok) return []
        return await res.json()
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg)
        return []
      } finally {
        setLoading(false)
      }
    },
    []
  )

  const saveDoc = useCallback(
    async (doc: Partial<ProjectDoc>): Promise<ProjectDoc | null> => {
      setLoading(true)
      setError(null)
      try {
        const res = await fetch('/api/v1/docs', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(doc),
        })
        if (!res.ok) throw new Error(`Failed to save doc: ${res.statusText}`)
        return await res.json()
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg)
        return null
      } finally {
        setLoading(false)
      }
    },
    []
  )

  const exportDocs = useCallback(
    async (repoPath?: string, targetDir = 'docs'): Promise<{ exported_count: number; target_dir: string } | null> => {
      setLoading(true)
      setError(null)
      try {
        const res = await fetch('/api/v1/docs/export', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ repo_path: repoPath, target_dir: targetDir }),
        })
        if (!res.ok) throw new Error(`Export failed: ${res.statusText}`)
        return await res.json()
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg)
        return null
      } finally {
        setLoading(false)
      }
    },
    []
  )

  const fetchGlobalEnv = useCallback(async (): Promise<Record<string, any> | null> => {
    try {
      const res = await fetch('/api/v1/env/global')
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  }, [])

  const fetchTaskContext = useCallback(async (taskId: string): Promise<ContextPacket | null> => {
    try {
      const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/context`)
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  }, [])

  const fetchSpecs = useCallback(async (repoPath?: string, tier?: number): Promise<TechSpec[]> => {
    try {
      let url = '/api/v1/specs?'
      if (repoPath) url += `repo_path=${encodeURIComponent(repoPath)}&`
      if (tier) url += `tier=${tier}`
      const res = await fetch(url)
      if (!res.ok) return []
      const data = await res.json()
      return data.specs || []
    } catch {
      return []
    }
  }, [])

  const fetchSpec = useCallback(async (slug: string): Promise<TechSpec | null> => {
    try {
      const res = await fetch(`/api/v1/specs/${encodeURIComponent(slug)}`)
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  }, [])

  const searchSpecs = useCallback(async (q: string): Promise<TechSpec[]> => {
    try {
      const res = await fetch(`/api/v1/specs/search?q=${encodeURIComponent(q)}`)
      if (!res.ok) return []
      const data = await res.json()
      return data.results || []
    } catch {
      return []
    }
  }, [])

  const upsertSpec = useCallback(async (spec: TechSpec): Promise<TechSpec | null> => {
    try {
      const res = await fetch('/api/v1/specs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(spec),
      })
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  }, [])

  const fetchPlanRevisions = useCallback(async (slug: string): Promise<PlanRevision[]> => {
    try {
      const res = await fetch(`/api/v1/plans/${encodeURIComponent(slug)}/revisions`)
      if (!res.ok) return []
      const data = await res.json()
      return data.revisions || []
    } catch {
      return []
    }
  }, [])

  const fetchPlanSnapshot = useCallback(async (slug: string, rev?: string): Promise<SnapshotFile[]> => {
    try {
      const url = rev ? `/api/v1/plans/${encodeURIComponent(slug)}/snapshot?rev=${encodeURIComponent(rev)}` : `/api/v1/plans/${encodeURIComponent(slug)}/snapshot`
      const res = await fetch(url)
      if (!res.ok) return []
      const data = await res.json()
      return data.files || []
    } catch {
      return []
    }
  }, [])

  const fetchPlanDiff = useCallback(async (slug: string, fromRev: string, toRev: string): Promise<FileDiff[]> => {
    try {
      const res = await fetch(`/api/v1/plans/${encodeURIComponent(slug)}/diff?from=${encodeURIComponent(fromRev)}&to=${encodeURIComponent(toRev)}`)
      if (!res.ok) return []
      const data = await res.json()
      return data.diffs || []
    } catch {
      return []
    }
  }, [])

  const savePlanSnapshot = useCallback(async (slug: string, payload: any): Promise<boolean> => {
    try {
      const res = await fetch(`/api/v1/plans/${encodeURIComponent(slug)}/snapshot`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      return res.ok
    } catch {
      return false
    }
  }, [])

  return {
    loading,
    error,
    fetchProjects,
    fetchPlans,
    createPlan,
    deletePlan,
    createTask,
    updateTaskDependencies,
    updateTaskNotes,
    reclaimTask,
    evictTask,
    fetchSessions,
    fetchPlan,
    fetchTasks,
    uploadAttachment,
    fetchTaskArtifacts,
    fetchTaskLogs,
    updateTaskStatus,
    resolveReview,
    mergeTask,
    fetchDocs,
    saveDoc,
    exportDocs,
    fetchGlobalEnv,
    fetchTaskContext,
    fetchSpecs,
    fetchSpec,
    searchSpecs,
    upsertSpec,
    fetchPlanRevisions,
    fetchPlanSnapshot,
    fetchPlanDiff,
    savePlanSnapshot,
  }
}
