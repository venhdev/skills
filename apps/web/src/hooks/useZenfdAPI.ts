import { useState, useCallback } from 'react'
import type { Plan, Task, Artifact, ProjectDoc, ContextPacket } from '../types/contracts'

export function useZenfdAPI() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

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

  const fetchTasks = useCallback(async (repoPath?: string, status?: string): Promise<Task[]> => {
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
      return []
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

  return {
    loading,
    error,
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
  }
}
