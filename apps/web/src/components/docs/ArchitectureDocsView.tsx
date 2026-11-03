import React, { useState, useEffect, useRef } from 'react'
import {
  BookOpen,
  Plus,
  Download,
  FileText,
  Save,
  Check,
  Code,
  Eye,
  GitBranch,
  Layers,
  Sparkles,
} from 'lucide-react'
import mermaid from 'mermaid'
import type { ProjectDoc } from '../../types/contracts'

mermaid.initialize({
  startOnLoad: false,
  theme: 'dark',
  securityLevel: 'loose',
  fontFamily: 'monospace',
})

const MermaidRenderer: React.FC<{ chart: string }> = ({ chart }) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const [svg, setSvg] = useState<string>('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let isMounted = true
    const id = `mermaid-${Math.random().toString(36).substring(2, 9)}`
    mermaid
      .render(id, chart)
      .then(({ svg }) => {
        if (isMounted) {
          setSvg(svg)
          setError(null)
        }
      })
      .catch((err) => {
        if (isMounted) {
          setError(String(err))
        }
      })
    return () => {
      isMounted = false
    }
  }, [chart])

  if (error) {
    return (
      <div className="rounded border border-rose-500/30 bg-rose-950/20 p-3 font-mono text-xs text-rose-300">
        <div className="font-bold mb-1">Mermaid Syntax Note:</div>
        <pre className="whitespace-pre-wrap">{chart}</pre>
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      className="flex justify-center rounded-lg border border-zinc-800 bg-zinc-900/40 p-4 overflow-x-auto my-3"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  )
}

interface ArchitectureDocsViewProps {
  repoPath?: string
  docs: ProjectDoc[]
  onSaveDoc: (doc: Partial<ProjectDoc>) => Promise<ProjectDoc | null>
  onExportDocs: (repoPath?: string, targetDir?: string) => Promise<{ exported_count: number; target_dir: string } | null>
  onRefresh: () => void
}

export const ArchitectureDocsView: React.FC<ArchitectureDocsViewProps> = ({
  repoPath = '',
  docs,
  onSaveDoc,
  onExportDocs,
  onRefresh,
}) => {
  const [selectedSlug, setSelectedSlug] = useState<string>('')
  const [isEditing, setIsEditing] = useState(false)
  const [editTitle, setEditTitle] = useState('')
  const [editCategory, setEditCategory] = useState('ARCHITECTURE')
  const [editContent, setEditContent] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const [exportNotice, setExportNotice] = useState<string | null>(null)

  const selectedDoc = docs.find((d) => d.slug === selectedSlug) || docs[0]

  useEffect(() => {
    if (selectedDoc) {
      setSelectedSlug(selectedDoc.slug)
      setEditTitle(selectedDoc.title)
      setEditCategory(selectedDoc.category)
      setEditContent(selectedDoc.content_markdown)
    }
  }, [selectedDoc])

  const handleStartNewDoc = () => {
    setSelectedSlug('new-doc-' + Date.now().toString().slice(-4))
    setEditTitle('New Architectural Document')
    setEditCategory('ARCHITECTURE')
    setEditContent(
      '# Architectural Overview\n\n## Seam Topology\n\n```mermaid\ngraph TD\n  A[Agent Runner] --> B[zenfd SQLite WAL]\n  B --> C[Web Cockpit]\n```\n'
    )
    setIsEditing(true)
  }

  const handleSave = async () => {
    if (!editTitle.trim()) return
    setIsSaving(true)
    try {
      const slug = selectedSlug.startsWith('new-doc-')
        ? editTitle.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '')
        : selectedSlug

      // Extract any mermaid code blocks
      const mermaidMatches: string[] = []
      const regex = /```mermaid\n([\s\S]*?)```/g
      let match
      while ((match = regex.exec(editContent)) !== null) {
        mermaidMatches.push(match[1].trim())
      }

      await onSaveDoc({
        slug,
        repo_path: repoPath,
        git_branch: 'main',
        title: editTitle,
        category: editCategory,
        content_markdown: editContent,
        mermaid_diagrams: mermaidMatches,
      })
      setIsEditing(false)
      onRefresh()
    } finally {
      setIsSaving(false)
    }
  }

  const handleExport = async () => {
    const res = await onExportDocs(repoPath, 'docs')
    if (res) {
      setExportNotice(`Exported ${res.exported_count} docs to ${res.target_dir}/`)
      setTimeout(() => setExportNotice(null), 4000)
    }
  }

  // Parse mermaid blocks from doc content
  const renderContentWithMermaid = (markdown: string) => {
    const parts = markdown.split(/```mermaid\n([\s\S]*?)```/g)
    if (parts.length === 1) {
      return <pre className="whitespace-pre-wrap font-sans text-sm text-zinc-300 leading-relaxed">{markdown}</pre>
    }

    return (
      <div className="space-y-3 text-sm text-zinc-300 leading-relaxed font-sans">
        {parts.map((part, i) => {
          if (i % 2 === 1) {
            // Mermaid chart block
            return <MermaidRenderer key={i} chart={part.trim()} />
          }
          return (
            <div key={i} className="whitespace-pre-wrap">
              {part}
            </div>
          )
        })}
      </div>
    )
  }

  return (
    <div className="flex h-full w-full bg-zinc-950 font-sans select-none overflow-hidden">
      {/* 1. Left Nav Sidebar */}
      <aside className="flex w-64 flex-col border-r border-zinc-800 bg-zinc-900/50">
        <div className="flex h-12 items-center justify-between border-b border-zinc-800 px-3">
          <div className="flex items-center gap-2 text-xs font-semibold text-zinc-200">
            <BookOpen className="h-4 w-4 text-emerald-400" />
            <span>Living Architecture Wiki</span>
          </div>
          <button
            onClick={handleStartNewDoc}
            className="flex items-center gap-1 rounded bg-zinc-800 px-2 py-1 text-[11px] font-medium text-zinc-300 hover:bg-zinc-700 hover:text-white transition"
            title="Create New Document"
          >
            <Plus className="h-3 w-3" />
            <span>New</span>
          </button>
        </div>

        {/* Export to Git Action */}
        <div className="p-2 border-b border-zinc-800 bg-zinc-950/40">
          <button
            onClick={handleExport}
            className="flex w-full items-center justify-center gap-1.5 rounded border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs font-semibold text-zinc-300 hover:border-emerald-500/50 hover:text-emerald-400 hover:bg-emerald-950/20 transition"
          >
            <Download className="h-3.5 w-3.5" />
            <span>Export to Git (docs/)</span>
          </button>
          {exportNotice && (
            <div className="mt-1.5 flex items-center gap-1 text-[10px] text-emerald-400 font-medium">
              <Check className="h-3 w-3" />
              <span>{exportNotice}</span>
            </div>
          )}
        </div>

        {/* Document List */}
        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          {docs.length === 0 ? (
            <div className="p-4 text-center text-xs text-zinc-500">
              No docs saved yet. Create a document or push with zenfd.
            </div>
          ) : (
            docs.map((d) => (
              <button
                key={d.slug}
                onClick={() => {
                  setSelectedSlug(d.slug)
                  setIsEditing(false)
                }}
                className={`flex w-full flex-col items-start rounded-md px-2.5 py-2 text-left transition-colors ${
                  d.slug === selectedDoc?.slug
                    ? 'bg-zinc-800 text-white border-l-2 border-emerald-500'
                    : 'text-zinc-400 hover:bg-zinc-800/50 hover:text-zinc-200'
                }`}
              >
                <div className="flex w-full items-center justify-between text-xs font-medium">
                  <span className="truncate">{d.title}</span>
                  <span className="rounded bg-zinc-900 px-1.5 py-0.2 text-[9px] font-mono text-zinc-500">
                    v{d.version}
                  </span>
                </div>
                <div className="flex items-center gap-1.5 text-[10px] text-zinc-500 mt-0.5">
                  <span className="uppercase text-emerald-500 font-semibold">{d.category}</span>
                  <span>•</span>
                  <span>{new Date(d.updated_at).toLocaleDateString()}</span>
                </div>
              </button>
            ))
          )}
        </div>
      </aside>

      {/* 2. Main Doc Viewer & Editor Pane */}
      <main className="flex flex-1 flex-col overflow-hidden">
        {selectedDoc ? (
          <>
            {/* Document Header Bar */}
            <div className="flex h-12 items-center justify-between border-b border-zinc-800 bg-zinc-900/60 px-5">
              <div className="flex items-center gap-3">
                <FileText className="h-4 w-4 text-emerald-400" />
                <span className="text-sm font-semibold text-zinc-100">
                  {selectedDoc.title}
                </span>
                <span className="rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-0.5 text-[10px] font-bold text-emerald-400 uppercase">
                  {selectedDoc.category}
                </span>
                <span className="flex items-center gap-1 text-[11px] font-mono text-zinc-500">
                  <GitBranch className="h-3 w-3" />
                  <span>{selectedDoc.git_branch || 'main'}</span>
                </span>
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={() => setIsEditing(!isEditing)}
                  className="flex items-center gap-1 rounded border border-zinc-700 bg-zinc-800 px-2.5 py-1 text-xs font-medium text-zinc-200 hover:bg-zinc-700 transition"
                >
                  {isEditing ? <Eye className="h-3.5 w-3.5" /> : <Code className="h-3.5 w-3.5" />}
                  <span>{isEditing ? 'Preview' : 'Edit Spec'}</span>
                </button>
                {isEditing && (
                  <button
                    onClick={handleSave}
                    disabled={isSaving}
                    className="flex items-center gap-1 rounded bg-emerald-600 hover:bg-emerald-500 px-3 py-1 text-xs font-semibold text-white transition disabled:opacity-50"
                  >
                    <Save className="h-3.5 w-3.5" />
                    <span>{isSaving ? 'Saving...' : 'Save Changes'}</span>
                  </button>
                )}
              </div>
            </div>

            {/* Document Content View */}
            <div className="flex-1 overflow-y-auto p-6">
              {isEditing ? (
                <div className="space-y-4 max-w-4xl mx-auto">
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="text-xs font-semibold text-zinc-400 block mb-1">
                        Document Title
                      </label>
                      <input
                        type="text"
                        value={editTitle}
                        onChange={(e) => setEditTitle(e.target.value)}
                        className="w-full rounded border border-zinc-700 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-200 focus:border-emerald-500 focus:outline-none"
                      />
                    </div>
                    <div>
                      <label className="text-xs font-semibold text-zinc-400 block mb-1">
                        Category
                      </label>
                      <select
                        value={editCategory}
                        onChange={(e) => setEditCategory(e.target.value)}
                        className="w-full rounded border border-zinc-700 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-200 focus:border-emerald-500 focus:outline-none font-mono"
                      >
                        <option value="ARCHITECTURE">ARCHITECTURE</option>
                        <option value="SCHEMA">SCHEMA</option>
                        <option value="ADR">ADR</option>
                        <option value="GUIDE">GUIDE</option>
                        <option value="MEMORY">MEMORY</option>
                      </select>
                    </div>
                  </div>

                  <div>
                    <label className="text-xs font-semibold text-zinc-400 block mb-1">
                      Markdown & Mermaid Content
                    </label>
                    <textarea
                      value={editContent}
                      onChange={(e) => setEditContent(e.target.value)}
                      rows={22}
                      className="w-full rounded border border-zinc-700 bg-zinc-900 p-3 font-mono text-xs text-zinc-200 focus:border-emerald-500 focus:outline-none"
                    />
                  </div>
                </div>
              ) : (
                <div className="max-w-4xl mx-auto space-y-6">
                  {/* Markdown with embedded Mermaid */}
                  {renderContentWithMermaid(selectedDoc.content_markdown)}

                  {/* Explicit Mermaid Diagrams array if present */}
                  {selectedDoc.mermaid_diagrams && selectedDoc.mermaid_diagrams.length > 0 && (
                    <div className="space-y-3 border-t border-zinc-800 pt-6">
                      <h4 className="flex items-center gap-1.5 text-xs font-bold text-zinc-300 uppercase tracking-wider">
                        <Sparkles className="h-3.5 w-3.5 text-emerald-400" />
                        <span>Interactive Mermaid Diagrams</span>
                      </h4>
                      {selectedDoc.mermaid_diagrams.map((chart, idx) => (
                        <MermaidRenderer key={idx} chart={chart} />
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex h-full w-full flex-col items-center justify-center p-8 text-center text-zinc-500">
            <Layers className="h-10 w-10 text-zinc-700 mb-3" />
            <h3 className="text-sm font-semibold text-zinc-300">
              No Document Selected
            </h3>
            <p className="mt-1 text-xs text-zinc-500 max-w-sm">
              Select an architectural document from the sidebar or click "New" to compose a specification.
            </p>
          </div>
        )}
      </main>
    </div>
  )
}
