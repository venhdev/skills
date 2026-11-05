import React, { useMemo, useState } from 'react'
import {
  Download,
  Image as ImageIcon,
  FileSpreadsheet,
  Paperclip,
  Maximize2,
  X,
  Lock,
} from 'lucide-react'
import type { Artifact } from '../../types/contracts'

interface ArtifactLightboxProps {
  artifacts: Artifact[]
}

export const ArtifactLightbox: React.FC<ArtifactLightboxProps> = ({ artifacts }) => {
  const [zoomImageSrc, setZoomImageSrc] = useState<string | null>(null)

  const inputArtifacts = useMemo(
    () => artifacts.filter((a) => a.role === 'INPUT'),
    [artifacts]
  )

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

        <div className="flex items-center gap-1.5 text-[11px] font-mono text-zinc-500">
          <ImageIcon className="h-3 w-3" />
          <span>Inputs ({inputArtifacts.length})</span>
        </div>
      </div>

      {/* Content Area */}
      <div className="flex-1 overflow-y-auto bg-zinc-950 p-3">
        <div className="space-y-3">
            {/* Read-only notice: artifacts are seeded by agents/MCP/CLI */}
            <div className="flex items-center gap-2.5 rounded-lg border border-zinc-800 bg-zinc-900/40 p-3 text-xs text-zinc-400">
              <Lock className="h-4 w-4 text-zinc-500 shrink-0" />
              <span>
                Read-only hub. Agents attach media through MCP while the task is READY or
                PENDING; this cockpit only displays and downloads it.
              </span>
            </div>

            {/* List of Attached Input Artifacts */}
            {inputArtifacts.length === 0 ? (
              <div className="py-6 text-center text-xs text-zinc-500">
                No input attachments attached to this task yet.
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-2">
                {inputArtifacts.map((art) => {
                  const isImage = art.type === 'IMAGE' || art.mime_type.startsWith('image/')
                  const fileUrl = `/api/v1/artifacts/${art.id}`
                  return (
                    <div
                      key={art.id}
                      className="group relative flex flex-col rounded-md border border-zinc-800 bg-zinc-900/40 p-2 overflow-hidden hover:border-zinc-700"
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
                          className="text-zinc-500 hover:text-emerald-400"
                          title="Download"
                        >
                          <Download className="h-3.5 w-3.5" />
                        </a>
                      </div>
                      <span className="text-[9px] text-zinc-500 font-mono">
                        {(art.byte_size / 1024).toFixed(1)} KB
                      </span>
                    </div>
                  )
                })}
              </div>
            )}
        </div>
      </div>

      {/* Image Fullscreen Lightbox Modal */}
      {zoomImageSrc && (
        <div
          onClick={() => setZoomImageSrc(null)}
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4 cursor-zoom-out"
        >
          <button
            onClick={() => setZoomImageSrc(null)}
            className="absolute top-4 right-4 rounded-full bg-zinc-800 p-2 text-zinc-300 hover:bg-zinc-700 hover:text-white"
          >
            <X className="h-5 w-5" />
          </button>
          <img
            src={zoomImageSrc}
            alt="Expanded View"
            className="max-h-[90vh] max-w-[90vw] rounded-lg object-contain shadow-2xl border border-zinc-800"
          />
        </div>
      )}
    </div>
  )
}
