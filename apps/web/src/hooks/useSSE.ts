import { useEffect, useState, useRef, useCallback } from 'react'
import type { EventMessage } from '../types/contracts'

interface UseSSEOptions {
  url?: string
  onEvent?: (event: EventMessage) => void
  enabled?: boolean
}

// Event types the daemon streams. Keeping the list here documents the contract
// and avoids re-subscribing on every render.
const EVENT_TYPES = [
  'plan.created',
  'plan.deleted',
  'plan.archived',
  'plan.unarchived',
  'task.created',
  'task.status',
  'task.dependencies_updated',
  'task.notes_updated',
  'artifact.added',
  'state.changed',
] as const

export function useSSE({
  url = '/api/v1/events/stream',
  onEvent,
  enabled = true,
}: UseSSEOptions = {}) {
  const [connected, setConnected] = useState(false)
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  // The callback is read through a ref so a new closure never reconnects.
  const onEventRef = useRef(onEvent)
  onEventRef.current = onEvent

  const connect = useCallback(() => {
    if (!enabled) return

    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }

    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }

    const es = new EventSource(url)
    eventSourceRef.current = es

    es.onopen = () => {
      setConnected(true)
    }

    es.onerror = () => {
      setConnected(false)
      es.close()
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
      }
      // Auto-reconnect after 3 seconds
      reconnectTimerRef.current = setTimeout(() => {
        if (enabled) {
          connect()
        }
      }, 3000)
    }

    // Standard ping event
    es.addEventListener('ping', () => {
      setConnected(true)
    })

    const dispatch = (type: string, raw: string) => {
      let parsed: EventMessage
      try {
        parsed = JSON.parse(raw)
      } catch {
        parsed = { id: 0, type, payload: raw, timestamp: new Date().toISOString() }
      }
      onEventRef.current?.(parsed)
    }

    // Generic handler covers any event the daemon adds without a frontend
    // release. It only updates the connection state, never React data.
    es.onmessage = (e) => dispatch(e.type || 'message', e.data)

    EVENT_TYPES.forEach((type) => {
      es.addEventListener(type, (e: MessageEvent) => dispatch(type, e.data))
    })
  }, [url, enabled])

  useEffect(() => {
    connect()
    return () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
        reconnectTimerRef.current = null
      }
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
    }
  }, [connect])

  return {
    connected,
  }
}
