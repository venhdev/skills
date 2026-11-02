import { useState, useCallback } from 'react'
import type { Plan, Task } from '../types/contracts'

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
      const data = await res.json()
      return data
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

  const killRun = useCallback(async (runId: string): Promise<boolean> => {
    try {
      const res = await fetch(`/api/v1/runs/${encodeURIComponent(runId)}/kill`, {
        method: 'POST',
      })
      return res.ok
    } catch {
      return false
    }
  }, [])

  return {
    loading,
    error,
    fetchPlan,
    fetchTasks,
    killRun,
  }
}
