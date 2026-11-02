import React, { useEffect, useRef } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import {
  Square,
  Trash2,
  Maximize2,
  Terminal as TerminalIcon,
  Radio,
} from 'lucide-react'
import { useTerminalWS } from '../../hooks/useTerminalWS'

interface TerminalPaneProps {
  runId?: string
  taskTitle?: string
  onKillProcess?: (runId: string) => void
  isMaximized?: boolean
  onToggleMaximize?: () => void
}

export const TerminalPane: React.FC<TerminalPaneProps> = ({
  runId,
  taskTitle,
  onKillProcess,
  isMaximized = false,
  onToggleMaximize,
}) => {
  const terminalRef = useRef<HTMLDivElement>(null)
  const termInstanceRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)

  const { status, sendInput, sendResize } = useTerminalWS({
    runId,
    onData: (data) => {
      if (termInstanceRef.current) {
        if (typeof data === 'string') {
          termInstanceRef.current.write(data)
        } else if (data instanceof ArrayBuffer) {
          termInstanceRef.current.write(new Uint8Array(data))
        }
      }
    },
    enabled: true,
  })

  // Initialize xterm instance
  useEffect(() => {
    if (!terminalRef.current) return

    const term = new Terminal({
      cursorBlink: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      fontSize: 12,
      lineHeight: 1.25,
      theme: {
        background: '#09090b',
        foreground: '#f4f4f5',
        cursor: '#a855f7',
        selectionBackground: 'rgba(168, 85, 247, 0.3)',
        black: '#18181b',
        red: '#f43f5e',
        green: '#10b981',
        yellow: '#f59e0b',
        blue: '#3b82f6',
        magenta: '#d946ef',
        cyan: '#06b6d4',
        white: '#fafafa',
        brightBlack: '#71717a',
        brightRed: '#fb7185',
        brightGreen: '#34d399',
        brightYellow: '#fbbf24',
        brightBlue: '#60a5fa',
        brightMagenta: '#e879f9',
        brightCyan: '#22d3ee',
        brightWhite: '#ffffff',
      },
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(terminalRef.current)

    termInstanceRef.current = term
    fitAddonRef.current = fitAddon

    if (!runId) {
      term.writeln('\x1b[38;5;242m[zenfd@control-plane] ~ No active agent run selected.\x1b[0m')
      term.writeln('\x1b[38;5;242mSelect a task from the DAG or Tree to inspect live PTY stream.\x1b[0m\r\n')
    }

    // Initial fit
    setTimeout(() => {
      try {
        fitAddon.fit()
        sendResize(term.cols, term.rows)
      } catch {
        // ignore layout race
      }
    }, 50)

    // Handle user input in terminal
    term.onData((data) => {
      sendInput(data)
    })

    // Resize observer
    const resizeObserver = new ResizeObserver(() => {
      try {
        fitAddon.fit()
        sendResize(term.cols, term.rows)
      } catch {
        // ignore resize error
      }
    })
    resizeObserver.observe(terminalRef.current)

    return () => {
      resizeObserver.disconnect()
      term.dispose()
      termInstanceRef.current = null
      fitAddonRef.current = null
    }
  }, [sendInput, sendResize])

  const handleClear = () => {
    termInstanceRef.current?.clear()
  }

  const handleKill = () => {
    if (runId && onKillProcess) {
      onKillProcess(runId)
    }
  }

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-zinc-950 font-mono select-none">
      {/* Terminal Toolbar */}
      <div className="flex h-9 shrink-0 items-center justify-between border-b border-zinc-800 bg-zinc-900/80 px-3">
        <div className="flex items-center gap-2 overflow-hidden text-xs">
          <TerminalIcon className="h-3.5 w-3.5 text-violet-400 shrink-0" />
          <span className="font-semibold text-zinc-200 truncate">
            {runId ? `PTY: ${runId}` : 'Interactive Shell'}
          </span>
          {taskTitle && (
            <span className="text-[11px] text-zinc-500 truncate max-w-[150px]">
              ({taskTitle})
            </span>
          )}
        </div>

        <div className="flex items-center gap-2">
          {/* Connection Status */}
          <div className="flex items-center gap-1 text-[10px]">
            <Radio
              className={`h-3 w-3 ${
                status === 'connected'
                  ? 'text-emerald-400 animate-pulse'
                  : 'text-zinc-500'
              }`}
            />
            <span
              className={
                status === 'connected' ? 'text-emerald-400' : 'text-zinc-500'
              }
            >
              {status}
            </span>
          </div>

          <div className="h-3 w-[1px] bg-zinc-800" />

          {/* Action buttons */}
          <button
            onClick={handleClear}
            className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white"
            title="Clear Terminal Output"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>

          {runId && onKillProcess && (
            <button
              onClick={handleKill}
              className="flex items-center gap-1 rounded bg-rose-500/10 px-2 py-0.5 text-[10px] font-semibold text-rose-400 ring-1 ring-rose-500/20 hover:bg-rose-500 hover:text-white transition-all"
              title="Terminate Process Group (-PGID)"
            >
              <Square className="h-2.5 w-2.5 fill-rose-400" />
              Kill
            </button>
          )}

          {onToggleMaximize && (
            <button
              onClick={onToggleMaximize}
              className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-white"
              title={isMaximized ? 'Restore View' : 'Maximize Terminal'}
            >
              <Maximize2 className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Terminal Canvas Container */}
      <div
        ref={terminalRef}
        className="flex-1 overflow-hidden p-2 bg-zinc-950"
      />
    </div>
  )
}
