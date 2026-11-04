import React, { useState, useEffect, useRef } from 'react'
import {
  Download,
  Image as ImageIcon,
  FileText,
  UploadCloud,
  FileSpreadsheet,
  Paperclip,
  Maximize2,
  X,
  Lock,
} from 'lucide-react'
import type { Artifact, TaskStatus } from '../../types/contracts'

interface ArtifactLightboxProps {
  taskId?: string
  taskStatus?: TaskStatus
  artifacts: Artifact[]
  onUploadAttachment?: (file: File) => Promise<void>
}

export const ArtifactLightbox: React.FC<ArtifactLightboxProps> = ({
  taskId,
  taskStatus,
  artifacts,
  onUploadAttachment,
}) => {
  const [activeTab, setActiveTab] = useState<'inputs' | 'logs'>('inputs')
  const [selectedArtifactId, setSelectedArtifactId] = useState<string | null>(null)
  const [taskLogs, setTaskLogs] = useState<string>('')
  const [isUploading, setIsUploading] = useState(false)
  const [zoomImageSrc, setZoomImageSrc] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const inputArtifacts = artifacts.filter((a) => a.role === 'INPUT')

  // Set default selected artifact when tab changes or artifacts list updates
  useEffect(() => {
    if (activeTab === 'inputs') {
      if (inputArtifacts.length > 0 && !inputArtifacts.some((a) => a.id === selectedArtifactId)) {
        setSelectedArtifactId(inputArtifacts[0].id)
      }
    }
  }, [activeTab, inputArtifacts, selectedArtifactId])

  // Fetch raw task logs when logs tab is active
  useEffect(() => {
    if (activeTab === 'logs' && taskId) {
      const controller = new AbortController()
      fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/logs`, { signal: controller.signal })
        .then((res) => (res.ok ? res.text() : ''))
        .then((txt) => setTaskLogs(txt))
        .catch((err) => {
          if (err.name !== 'AbortError') {
            setTaskLogs('// Failed to load task logs')
          }
        })
      return () => controller.abort()
    }
  }, [activeTab, taskId])

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files || e.target.files.length === 0 || !onUploadAttachment) return
    const file = e.target.files[0]
    setIsUploading(true)
    try {
      await onUploadAttachment(file)
    } finally {
      setIsUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault()
    if (!onUploadAttachment) return
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      const file = e.dataTransfer.files[0]
      setIsUploading(true)
      try {
        await onUploadAttachment(file)
      } finally {
        setIsUploading(false)
      }
    }
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

        <div className="flex items-center gap-1">
          <button
            onClick={() => setActiveTab('inputs')}
            className={`flex items-center gap-1 rounded px-2 py-0.5 text-xs transition-colors ${
              activeTab === 'inputs'
                ? 'bg-zinc-800 text-white font-semibold'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <ImageIcon className="h-3 w-3" />
            <span>Inputs ({inputArtifacts.length})</span>
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

      {/* Content Area */}
      <div className="flex-1 overflow-y-auto bg-zinc-950 p-3">
        {/* TAB 1: INPUTS (Mockups, Specs & Attachments) */}
        {activeTab === 'inputs' && (
          <div className="space-y-3">
            {/* Drag & Drop Upload Zone (Writable only when READY or PENDING) */}
            {taskStatus && taskStatus !== 'READY' && taskStatus !== 'PENDING' ? (
              <div className="flex items-center gap-2.5 rounded-lg border border-zinc-800 bg-zinc-900/40 p-3 text-xs text-zinc-400">
                <Lock className="h-4 w-4 text-zinc-500 shrink-0" />
                <span>
                  Media attachments locked (Task is <strong className="text-zinc-200 font-mono">{taskStatus}</strong>). Uploads permitted only when READY or PENDING.
                </span>
              </div>
            ) : (
              <div
                onDragOver={(e) => e.preventDefault()}
                onDrop={handleDrop}
                onClick={() => fileInputRef.current?.click()}
                className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-zinc-800 bg-zinc-900/30 p-4 transition-colors hover:border-violet-500/50 hover:bg-zinc-900/60"
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  className="hidden"
                  onChange={handleFileChange}
                />
                <UploadCloud className="h-6 w-6 text-zinc-500 mb-1" />
                <p className="text-xs font-medium text-zinc-300">
                  {isUploading ? 'Uploading file...' : 'Drop design mockups / specs here, or click to upload'}
                </p>
                <p className="text-[10px] text-zinc-500 mt-0.5">
                  PNG, JPG, WebP, PDF, Markdown up to 25MB (Agent context)
                </p>
              </div>
            )}

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
        )}

        {/* TAB 2: LOGS (Raw terminal logs) */}
        {activeTab === 'logs' && (
          <div className="h-full rounded border border-zinc-850 bg-zinc-900/60 p-3 font-mono text-xs text-zinc-300 overflow-x-auto">
            <pre className="whitespace-pre-wrap select-text leading-relaxed">
              {taskLogs || '// No logs recorded yet for this task.'}
            </pre>
          </div>
        )}
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
