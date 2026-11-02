import React, { useState } from 'react'
import { FileCode, Download, Image as ImageIcon, FileText } from 'lucide-react'
import type { Artifact } from '../../types/contracts'

interface ArtifactLightboxProps {
  artifacts: Artifact[]
  sampleDiff?: string
}

export const ArtifactLightbox: React.FC<ArtifactLightboxProps> = ({
  artifacts,
  sampleDiff = `--- a/internal/scheduler/scheduler.go\n+++ b/internal/scheduler/scheduler.go\n@@ -120,6 +120,12 @@\n-    return run, nil\n+    // Thread-safe snapshot copy before background reaping\n+    runCopy := *run\n+    return &runCopy, nil\n`,
}) => {
  const [activeTab, setActiveTab] = useState<'diff' | 'image' | 'logs'>('diff')
  const [selectedArtifactId, setSelectedArtifactId] = useState<string | null>(
    artifacts[0]?.id || null
  )

  const selectedArtifact = artifacts.find((a) => a.id === selectedArtifactId)

  const renderDiffLines = (diff: string) => {
    return diff.split('\n').map((line, idx) => {
      let bg = 'hover:bg-zinc-900'
      let text = 'text-zinc-300'

      if (line.startsWith('+') && !line.startsWith('+++')) {
        bg = 'bg-emerald-950/30 hover:bg-emerald-950/50'
        text = 'text-emerald-400 font-medium'
      } else if (line.startsWith('-') && !line.startsWith('---')) {
        bg = 'bg-rose-950/30 hover:bg-rose-950/50'
        text = 'text-rose-400 font-medium'
      } else if (line.startsWith('@@')) {
        bg = 'bg-violet-950/30'
        text = 'text-violet-400 font-semibold'
      }

      return (
        <div key={idx} className={`flex px-3 py-0.5 font-mono text-xs ${bg}`}>
          <span className="w-8 shrink-0 select-none text-[10px] text-zinc-600">
            {idx + 1}
          </span>
          <span className={`whitespace-pre select-text ${text}`}>{line}</span>
        </div>
      )
    })
  }

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-zinc-950 font-mono select-none">
      {/* Lightbox Toolbar */}
      <div className="flex h-9 shrink-0 items-center justify-between border-b border-zinc-800 bg-zinc-900/80 px-3">
        <div className="flex items-center gap-2">
          <FileCode className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
          <span className="text-xs font-semibold text-zinc-200">
            Claim-Check Artifacts
          </span>
          <span className="rounded bg-zinc-800 px-1 py-0.2 text-[10px] text-zinc-400">
            {artifacts.length}
          </span>
        </div>

        <div className="flex items-center gap-1">
          <button
            onClick={() => setActiveTab('diff')}
            className={`flex items-center gap-1 rounded px-2 py-0.5 text-xs transition-colors ${
              activeTab === 'diff'
                ? 'bg-zinc-800 text-white font-semibold'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <FileCode className="h-3 w-3" />
            <span>Diffs</span>
          </button>
          <button
            onClick={() => setActiveTab('image')}
            className={`flex items-center gap-1 rounded px-2 py-0.5 text-xs transition-colors ${
              activeTab === 'image'
                ? 'bg-zinc-800 text-white font-semibold'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <ImageIcon className="h-3 w-3" />
            <span>Images</span>
          </button>
          <button
            onClick={() => setActiveTab('logs')}
            className={`flex items-center gap-1 rounded px-2 py-0.5 text-xs transition-colors ${
              activeTab === 'logs'
                ? 'bg-zinc-800 text-white font-semibold'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <FileText className="h-3 w-3" />
            <span>Logs</span>
          </button>
        </div>
      </div>

      {/* Artifact selector pills if multiple exist */}
      {artifacts.length > 1 && (
        <div className="flex gap-1 border-b border-zinc-850 bg-zinc-900/30 p-1.5 overflow-x-auto text-[10px]">
          {artifacts.map((a) => (
            <button
              key={a.id}
              onClick={() => setSelectedArtifactId(a.id)}
              className={`rounded px-2 py-0.5 whitespace-nowrap ${
                selectedArtifactId === a.id
                  ? 'bg-violet-600 text-white font-bold'
                  : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200'
              }`}
            >
              {a.id} ({a.type})
            </button>
          ))}
        </div>
      )}

      {/* Content Area */}
      <div className="flex-1 overflow-y-auto bg-zinc-950 p-2">
        {activeTab === 'diff' && (
          <div className="rounded border border-zinc-850 bg-zinc-900/50 py-1">
            <div className="flex items-center justify-between border-b border-zinc-850 px-3 py-1.5 text-[11px] text-zinc-400">
              <span className="truncate">
                {selectedArtifact?.file_path || 'patch.diff'}
              </span>
              <a
                href={
                  selectedArtifact
                    ? `/api/v1/artifacts/${selectedArtifact.id}`
                    : '#'
                }
                download
                className="flex items-center gap-1 text-[10px] text-emerald-400 hover:underline"
              >
                <Download className="h-3 w-3" /> Download
              </a>
            </div>
            <div className="py-1">{renderDiffLines(sampleDiff)}</div>
          </div>
        )}

        {activeTab === 'image' && (
          <div className="flex h-full flex-col items-center justify-center py-8 text-center text-xs text-zinc-500">
            {selectedArtifact && selectedArtifact.type === 'IMAGE' ? (
              <img
                src={`/api/v1/artifacts/${selectedArtifact.id}`}
                alt="Artifact Preview"
                className="max-h-64 rounded border border-zinc-800 object-contain shadow-lg"
              />
            ) : (
              <div className="flex flex-col items-center gap-2">
                <ImageIcon className="h-8 w-8 text-zinc-700" />
                <span>No UI screenshots recorded for this step.</span>
              </div>
            )}
          </div>
        )}

        {activeTab === 'logs' && (
          <div className="rounded border border-zinc-850 bg-zinc-900/50 p-3 text-xs text-zinc-400">
            <p className="text-zinc-500">// Claim-Check Log Pointer</p>
            <p className="mt-1 font-mono text-zinc-300">
              {selectedArtifact?.file_path ||
                '~/.zenforge/logs/run-execution.log'}
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
