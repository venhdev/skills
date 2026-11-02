import { useEffect, useRef, useState, useCallback } from 'react'

interface UseTerminalWSOptions {
  runId?: string
  onData?: (data: string | ArrayBuffer) => void
  enabled?: boolean
}

export type WSConnectionStatus = 'connecting' | 'connected' | 'disconnected'

export function useTerminalWS({
  runId,
  onData,
  enabled = true,
}: UseTerminalWSOptions = {}) {
  const [status, setStatus] = useState<WSConnectionStatus>('disconnected')
  const wsRef = useRef<WebSocket | null>(null)
  const onDataRef = useRef(onData)
  onDataRef.current = onData

  const connect = useCallback(() => {
    if (!enabled) return

    if (wsRef.current) {
      wsRef.current.close()
    }

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const path = runId
      ? `/api/v1/terminal/ws/${encodeURIComponent(runId)}`
      : '/api/v1/terminal/ws'
    const url = `${proto}//${host}${path}`

    setStatus('connecting')
    const ws = new WebSocket(url)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => {
      setStatus('connected')
    }

    ws.onclose = () => {
      setStatus('disconnected')
    }

    ws.onerror = () => {
      setStatus('disconnected')
    }

    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        onDataRef.current?.(event.data)
      } else if (event.data instanceof ArrayBuffer) {
        onDataRef.current?.(event.data)
      }
    }
  }, [runId, enabled])

  useEffect(() => {
    connect()
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [connect])

  const sendInput = useCallback((data: string | Uint8Array) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(data)
    }
  }, [])

  const sendResize = useCallback((cols: number, rows: number) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      const payload = JSON.stringify({ type: 'resize', cols, rows })
      wsRef.current.send(payload)
    }
  }, [])

  return {
    status,
    sendInput,
    sendResize,
    reconnect: connect,
  }
}
