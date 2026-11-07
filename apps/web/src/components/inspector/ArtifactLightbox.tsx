import React, { useMemo, useState } from 'react'
import {
  Download,
  FileSpreadsheet,
  Paperclip,
  Maximize2,
  X,
} from 'lucide-react'
import type { Artifact } from '../../types/contracts'

interface ArtifactLightboxProps {
  artifacts: Artifact[]
  compact?: boolean
}

export const ArtifactLightbox: React.FC<ArtifactLightboxProps> = ({ artifacts, compact = false }) => {
  const [zoomImageSrc, setZoomImageSrc] = useState<string | null>(null)

  const inputArtifacts = useMemo(
    () => artifacts.filter((a) => a.role === 'INPUT'),
    [artifacts]
  )

  const outputArtifacts = useMemo(
    () => artifacts.filter((a) => a.role === 'OUTPUT'),
    [artifacts]
  )

  // Zero-Footprint Rule: when 0 attachments and compact mode, render a slim single-line bar
  if (artifacts.length === 0 && compact) {
    return (
      <div className="flex h-9 items-center justify-between border-t border-zinc-800/80 bg-zinc-950 px-3 text-xs text-zinc-500 select-none">
        <div className="flex items-center gap-1.5 font-mono text-[11px]">
          <Paperclip className="h-3 w-3 text-zinc-600" />
          <span>Attachments: 0</span>
        </div>
        <span className="text-[10px] text-zinc-600">read-only hub</span>
      </div>
    )
  }

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-zinc-950 font-sans select-none">
      {/* Lightbox Toolbar */}
      <div className="flex h-9 shrink-0 items-center justify-between border-b border-zinc-800 bg-zinc-900/80 px-3">
        <div className="flex items-center gap-2">
          <Paperclip className="h-3.5 w-3.5 text-violet-400 shrink-0" />
          <span className="text-xs font-semibold text-zinc-200">
            Multi-Modal Hub
          </span>
          <span className="rounded bg-zinc-800 px-1 py-0.2 text-[10px] text-zinc-400 font-bold">
            {artifacts.length}
          </span>
        </div>

        <div className="flex items-center gap-2 text-[11px] font-mono text-zinc-500">
          <span>In ({inputArtifacts.length})</span>
          <span>·</span>
          <span>Out ({outputArtifacts.length})</span>
        </div>
      </div>

      {/* Content Area */}
      <div className="flex-1 overflow-y-auto bg-zinc-950 p-3 space-y-3">
        {artifacts.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center text-zinc-500">
            <Paperclip className="h-6 w-6 text-zinc-700 mb-2" />
            <p className="text-xs font-medium text-zinc-400">No attachments attached</p>
            <p className="text-[11px] text-zinc-600 mt-0.5 max-w-xs">
              Agents attach media through MCP tools during task execution.
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              {artifacts.map((art) => {
                const isImage = art.type === 'IMAGE' || art.mime_type.startsWith('image/')
                const fileUrl = `/api/v1/artifacts/${art.id}`
                return (
                  <div
                    key={art.id}
                    className="group relative flex flex-col rounded-md border border-zinc-800 bg-zinc-900/40 p-2 overflow-hidden hover:border-zinc-700 transition"
                  >
                    {isImage ? (
                      <div
                        onClick={() => setZoomImageSrc(fileUrl)}
                        className="relative h-24 w-full cursor-zoom-in overflow-hidden rounded bg-black/50"
                      >
                        <img
                          src={fileUrl}
                          alt={art.name || art.id}
                          className="h-full w-full object-contain transition-transform group-hover:scale-105"
                        />
                        <div className="absolute top-1 right-1 rounded bg-black/60 p-0.5 text-zinc-300 opacity-0 group-hover:opacity-100 transition-opacity">
                          <Maximize2 className="h-3 w-3" />
                        </div>
                      </div>
                    ) : (
                      <div className="flex h-24 w-full items-center justify-center rounded bg-zinc-900">
                        <FileSpreadsheet className="h-8 w-8 text-zinc-500" />
                      </div>
                    )}

                    <div className="mt-2 flex items-center justify-between text-xs">
                      <span className="truncate font-mono text-[11px] text-zinc-300" title={art.name}>
                        {art.name || art.id}
                      </span>
                      <a
                        href={fileUrl}
                        download={art.name || art.id}
                        className="text-zinc-500 hover:text-emerald-400 transition"
                        title="Download"
                      >
                        <Download className="h-3 w-3" />
                      </a>
                    </div>
                    <div className="flex items-center justify-between text-[10px] text-zinc-500">
                      <span className="uppercase">{art.role}</span>
                      <span>{(art.byte_size / 1024).toFixed(1)} KB</span>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </div>

      {/* Modal Lightbox for Zoom */}
      {zoomImageSrc && (
        <div
          onClick={() => setZoomImageSrc(null)}
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4"
        >
          <div className="relative max-h-full max-w-full overflow-hidden rounded-lg bg-zinc-950 p-2 border border-zinc-800">
            <button
              onClick={() => setZoomImageSrc(null)}
              className="absolute top-3 right-3 rounded-full bg-zinc-900/80 p-1 text-zinc-400 hover:text-white border border-zinc-700"
            >
              <X className="h-4 w-4" />
            </button>
            <img
              src={zoomImageSrc}
              alt="Zoomed attachment"
              className="max-h-[85vh] max-w-[85vw] object-contain rounded"
            />
          </div>
        </div>
      )}
    </div>
  )
}
