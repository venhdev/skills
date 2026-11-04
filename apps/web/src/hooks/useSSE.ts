import { useEffect, useState, useRef, useCallback } from 'react'
import type { EventMessage } from '../types/contracts'

interface UseSSEOptions {
  url?: string
  onEvent?: (event: EventMessage) => void
  enabled?: boolean
}

export function useSSE({
  url = '/api/v1/events/stream',
  onEvent,
  enabled = true,
}: UseSSEOptions = {}) {
  const [connected, setConnected] = useState(false)
  const [lastEvent, setLastEvent] = useState<EventMessage | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
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

    // Generic message handler
    es.onmessage = (e) => {
      try {
        const parsed: EventMessage = JSON.parse(e.data)
        setLastEvent(parsed)
        onEventRef.current?.(parsed)
      } catch {
        // Raw string event
        const rawEvent: EventMessage = {
          type: e.type || 'message',
          payload: e.data,
          timestamp: new Date().toISOString(),
        }
        setLastEvent(rawEvent)
        onEventRef.current?.(rawEvent)
      }
    }

    // Specific domain events
    const eventTypes = [
      'plan:created',
      'plan:deleted',
      'task:created',
      'task:status',
      'task:dependencies_updated',
      'task:notes_updated',
      'task:artifact_added',
      'run:killed',
    ]

    eventTypes.forEach((type) => {
      es.addEventListener(type, (e: MessageEvent) => {
        try {
          const parsed: EventMessage = JSON.parse(e.data)
          setLastEvent(parsed)
          onEventRef.current?.(parsed)
        } catch {
          const raw: EventMessage = {
            type,
            payload: e.data,
            timestamp: new Date().toISOString(),
          }
          setLastEvent(raw)
          onEventRef.current?.(raw)
        }
      })
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
    lastEvent,
    reconnect: connect,
  }
}
