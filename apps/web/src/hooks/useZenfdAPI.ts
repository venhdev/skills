import { useState, useCallback } from 'react'
import type { Plan, Task, Artifact } from '../types/contracts'

export function useZenfdAPI() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchPlans = useCallback(async (): Promise<string[]> => {
    try {
      const res = await fetch('/api/v1/plans')
      if (!res.ok) return []
      const data = await res.json()
      return data.plans || []
    } catch {
      return []
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

  const fetchTasks = useCallback(async (status?: string): Promise<Task[] | null> => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams()
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

  const fetchGlobalEnv = useCallback(async (): Promise<Record<string, any> | null> => {
    try {
      const res = await fetch('/api/v1/env/global')
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  }, [])

  return {
    loading,
    error,
    fetchPlans,
    deletePlan,
    updateTaskDependencies,
    updateTaskNotes,
    evictTask,
    fetchPlan,
    fetchTasks,
    uploadAttachment,
    fetchTaskArtifacts,
    fetchGlobalEnv,
  }
}
