import { useState, useCallback } from 'react'
import type { Plan, Task, Artifact } from '../types/contracts'

type ApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: string; status?: number }

export function useZenfdAPI() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchPlans = useCallback(async (): Promise<ApiResult<string[]>> => {
    try {
      const res = await fetch('/api/v1/plans')
      if (!res.ok) {
        return {
          ok: false,
          error: `Failed to fetch plans: ${res.statusText}`,
          status: res.status,
        }
      }
      const data = await res.json()
      return { ok: true, data: data.plans || [] }
    } catch (err) {
      return {
        ok: false,
        error: err instanceof Error ? err.message : String(err),
      }
    }
  }, [])

  const fetchPlan = useCallback(
    async (slug: string): Promise<ApiResult<{ plan: Plan; tasks: Task[] }>> => {
      setLoading(true)
      setError(null)
      try {
        const res = await fetch(`/api/v1/plans/${encodeURIComponent(slug)}`)
        if (!res.ok) {
          const msg = `Failed to fetch plan ${slug}: ${res.statusText}`
          setError(msg)
          return { ok: false, error: msg, status: res.status }
        }
        return { ok: true, data: await res.json() }
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err)
        setError(msg)
        return { ok: false, error: msg }
      } finally {
        setLoading(false)
      }
    },
    []
  )

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
    fetchPlan,
    fetchTasks,
    fetchTaskArtifacts,
    fetchGlobalEnv,
  }
}
