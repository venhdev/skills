import React, { useCallback, useEffect, useState } from 'react'
import {
  BookOpen,
  FileText,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Trash2,
  X,
} from 'lucide-react'
import { useZenfAPI } from '../../hooks/useZenfAPI'
import type {
  Document,
  DocumentScope,
  DocumentSearchResult,
  DocumentStatus,
  DocumentType,
  Project,
} from '../../types/contracts'

interface DocsViewProps {
  projects: Project[]
  activeProjectSlug: string | null
}

interface DocumentForm {
  project_slug: string
  key: string
  slug: string
  type: DocumentType
  scope: DocumentScope
  scope_ref: string
  title: string
  path: string
  summary: string
  tags: string
  content: string
  status: DocumentStatus
}

const emptyForm = (projectSlug: string): DocumentForm => ({
  project_slug: projectSlug,
  key: '',
  slug: '',
  type: 'SPEC',
  scope: projectSlug ? 'PROJECT' : 'GLOBAL',
  scope_ref: '',
  title: '',
  path: '',
  summary: '',
  tags: '',
  content: '',
  status: 'ACTIVE',
})

const docTypes: DocumentType[] = [
  'ARCHITECTURE',
  'ADR',
  'SPEC',
  'RULE',
  'GUIDE',
  'REFERENCE',
  'MEMORY',
  'OTHER',
]
const docScopes: DocumentScope[] = ['GLOBAL', 'PROJECT', 'PLAN', 'TASK']
const docStatuses: DocumentStatus[] = ['DRAFT', 'ACTIVE', 'SUPERSEDED', 'ARCHIVED']

const typeClass: Record<string, string> = {
  ARCHITECTURE: 'border-sky-500/30 bg-sky-500/10 text-sky-300',
  ADR: 'border-violet-500/30 bg-violet-500/10 text-violet-300',
  SPEC: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
  RULE: 'border-rose-500/30 bg-rose-500/10 text-rose-300',
  GUIDE: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
  REFERENCE: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-300',
  MEMORY: 'border-fuchsia-500/30 bg-fuchsia-500/10 text-fuchsia-300',
  OTHER: 'border-zinc-700 bg-zinc-800 text-zinc-300',
}

const statusClass = (status: DocumentStatus) => {
  switch (status) {
    case 'ACTIVE':
      return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300'
    case 'DRAFT':
      return 'border-sky-500/30 bg-sky-500/10 text-sky-300'
    case 'SUPERSEDED':
      return 'border-amber-500/30 bg-amber-500/10 text-amber-300'
    case 'ARCHIVED':
      return 'border-zinc-700 bg-zinc-800 text-zinc-400'
  }
}

function parseTags(raw: string): string[] {
  return raw
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean)
}

export const DocsView: React.FC<DocsViewProps> = ({ projects, activeProjectSlug }) => {
  const {
    fetchDocuments,
    searchDocuments,
    createDocument,
    updateDocument,
    refreshDocument,
    deleteDocument,
  } = useZenfAPI()

  const [projectFilter, setProjectFilter] = useState(activeProjectSlug || '')
  const [includeGlobal, setIncludeGlobal] = useState(true)
  const [typeFilter, setTypeFilter] = useState<DocumentType | ''>('')
  const [statusFilter, setStatusFilter] = useState<DocumentStatus | ''>('')
  const [documents, setDocuments] = useState<Document[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const [searching, setSearching] = useState(false)
  const [searchResults, setSearchResults] = useState<DocumentSearchResult[] | null>(null)
  const [mode, setMode] = useState<'create' | 'edit'>('create')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [form, setForm] = useState<DocumentForm>(emptyForm(activeProjectSlug || ''))

  useEffect(() => {
    if (activeProjectSlug) {
      setProjectFilter(activeProjectSlug)
      setForm((current) =>
        current.project_slug ? current : { ...current, project_slug: activeProjectSlug, scope: 'PROJECT' }
      )
    }
  }, [activeProjectSlug])

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    const result = await fetchDocuments({
      projectSlug: projectFilter,
      includeGlobal,
      type: typeFilter,
      status: statusFilter,
      limit: 100,
    })
    setLoading(false)
    if (!result.ok) {
      setError(result.error)
      return
    }
    setDocuments(result.data)
  }, [fetchDocuments, includeGlobal, projectFilter, statusFilter, typeFilter])

  useEffect(() => {
    void load()
  }, [load])

  const resetForm = () => {
    setMode('create')
    setSelectedId(null)
    setForm(emptyForm(projectFilter || activeProjectSlug || ''))
  }

  const startCreate = () => {
    resetForm()
  }

  const startEdit = (document: Document) => {
    setMode('edit')
    setSelectedId(document.id)
    setForm({
      project_slug: document.project_slug || '',
      key: document.key,
      slug: document.slug,
      type: document.type,
      scope: document.scope,
      scope_ref: document.scope_ref || '',
      title: document.title,
      path: document.path,
      summary: document.summary || '',
      tags: (document.tags || []).join(', '),
      content: '',
      status: document.status,
    })
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    const tags = parseTags(form.tags)
    if (mode === 'create') {
      if (!form.key.trim() || !form.title.trim() || !form.path.trim()) {
        setError('key, title, and path are required')
        return
      }
      setBusy('create')
      const result = await createDocument({
        project_slug: form.scope === 'GLOBAL' ? '' : form.project_slug,
        key: form.key.trim(),
        slug: form.slug.trim() || undefined,
        type: form.type,
        scope: form.scope,
        scope_ref: form.scope_ref.trim() || undefined,
        title: form.title.trim(),
        path: form.path.trim(),
        summary: form.summary.trim(),
        content: form.content,
        status: form.status,
        tags,
      })
      setBusy('')
      if (!result.ok) {
        setError(result.error)
        return
      }
      resetForm()
      await load()
      return
    }

    if (!selectedId) return
    setBusy('update')
    const result = await updateDocument(selectedId, {
      title: form.title.trim(),
      summary: form.summary.trim(),
      path: form.path.trim(),
      status: form.status,
      tags,
    })
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    await load()
  }

  const handleRefresh = async (document: Document) => {
    setBusy(`refresh:${document.id}`)
    setError('')
    const result = await refreshDocument(document.id)
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    await load()
  }

  const handleDelete = async (document: Document) => {
    if (!window.confirm(`Delete document registry entry "${document.title}"? The canonical file is not removed.`)) {
      return
    }
    setBusy(`delete:${document.id}`)
    setError('')
    const result = await deleteDocument(document.id)
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    if (selectedId === document.id) resetForm()
    await load()
  }

  const handleSearch = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!query.trim()) {
      setSearchResults(null)
      return
    }
    setSearching(true)
    setError('')
    const result = await searchDocuments(query.trim(), projectFilter)
    setSearching(false)
    if (!result.ok) {
      setError(result.error)
      return
    }
    setSearchResults(result.data)
  }

  const visibleDocuments = searchResults
    ? searchResults.map((hit) => hit.document)
    : documents
  const snippetById = new Map(searchResults?.map((hit) => [hit.document.id, hit.snippet]) || [])

  return (
    <div className="flex h-full w-full overflow-hidden bg-zinc-950">
      {/* Document list */}
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden border-r border-zinc-800">
        <div className="shrink-0 space-y-3 border-b border-zinc-800 p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-sky-500/10 text-sky-400 ring-1 ring-sky-500/30">
                <BookOpen className="h-4.5 w-4.5" />
              </div>
              <div>
                <h2 className="text-sm font-bold text-zinc-100">SSOT Documents</h2>
                <p className="text-[11px] text-zinc-500">
                  Files are canonical. The registry keeps identity, tags, hash, and a searchable snapshot.
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => void load()}
                disabled={loading}
                className="flex items-center gap-1.5 rounded border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 disabled:opacity-50"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
                Reload
              </button>
              <button
                onClick={startCreate}
                className="flex items-center gap-1.5 rounded bg-emerald-500/15 px-2.5 py-1.5 text-xs font-semibold text-emerald-300 ring-1 ring-emerald-500/30 hover:bg-emerald-500/25"
              >
                <Plus className="h-3.5 w-3.5" /> Register
              </button>
            </div>
          </div>

          <form onSubmit={handleSearch} className="flex flex-wrap items-center gap-2">
            <div className="relative min-w-[240px] flex-1">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-500" />
              <input
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value)
                  if (!e.target.value.trim()) setSearchResults(null)
                }}
                placeholder="Full-text search across document snapshots…"
                className="w-full rounded border border-zinc-800 bg-zinc-950 py-1.5 pl-8 pr-2.5 text-xs text-zinc-200 outline-none focus:border-sky-500"
              />
            </div>
            <button
              type="submit"
              disabled={searching}
              className="rounded border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs font-semibold text-zinc-300 hover:bg-zinc-800 disabled:opacity-50"
            >
              {searching ? 'Searching…' : 'Search'}
            </button>
            {searchResults && (
              <button
                type="button"
                onClick={() => {
                  setQuery('')
                  setSearchResults(null)
                }}
                className="flex items-center gap-1 rounded border border-zinc-800 px-2.5 py-1.5 text-xs text-zinc-400 hover:text-zinc-200"
              >
                <X className="h-3 w-3" /> Clear
              </button>
            )}
          </form>

          <div className="flex flex-wrap items-center gap-2 text-xs">
            <select
              value={projectFilter}
              onChange={(e) => setProjectFilter(e.target.value)}
              className="rounded border border-zinc-800 bg-zinc-950 px-2 py-1 text-xs text-zinc-300 outline-none focus:border-sky-500"
            >
              <option value="">All projects</option>
              {projects.map((project) => (
                <option key={project.slug} value={project.slug}>
                  {project.name}
                </option>
              ))}
            </select>
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value as DocumentType | '')}
              className="rounded border border-zinc-800 bg-zinc-950 px-2 py-1 text-xs text-zinc-300 outline-none focus:border-sky-500"
            >
              <option value="">All types</option>
              {docTypes.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as DocumentStatus | '')}
              className="rounded border border-zinc-800 bg-zinc-950 px-2 py-1 text-xs text-zinc-300 outline-none focus:border-sky-500"
            >
              <option value="">All statuses</option>
              {docStatuses.map((status) => (
                <option key={status} value={status}>
                  {status}
                </option>
              ))}
            </select>
            <label className="flex items-center gap-1.5 text-[11px] text-zinc-400">
              <input
                type="checkbox"
                checked={includeGlobal}
                onChange={(e) => setIncludeGlobal(e.target.checked)}
                className="accent-sky-500"
              />
              include global
            </label>
            <span className="ml-auto rounded bg-zinc-800 px-2 py-0.5 font-mono text-[10px] text-zinc-400">
              {searchResults ? `${searchResults.length} hits` : `${documents.length} documents`}
            </span>
          </div>
        </div>

        {error && (
          <div className="mx-5 mt-3 rounded border border-rose-500/30 bg-rose-500/10 px-3 py-2 font-mono text-xs text-rose-300">
            {error}
          </div>
        )}

        <div className="flex-1 space-y-2 overflow-y-auto p-5">
          {loading ? (
            <div className="py-12 text-center font-mono text-xs text-zinc-500">loading documents…</div>
          ) : visibleDocuments.length === 0 ? (
            <div className="rounded-xl border border-dashed border-zinc-800 py-12 text-center font-mono text-xs text-zinc-500">
              {searchResults ? 'No document matched the query.' : 'No documents registered for this scope.'}
            </div>
          ) : (
            visibleDocuments.map((document) => {
              const snippet = snippetById.get(document.id)
              return (
                <article
                  key={document.id}
                  className={`rounded-lg border bg-zinc-900/40 p-3 transition ${
                    selectedId === document.id ? 'border-sky-500/40' : 'border-zinc-800 hover:border-zinc-700'
                  }`}
                >
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className={`rounded border px-1.5 py-0.5 text-[9.5px] font-bold ${typeClass[document.type] || typeClass.OTHER}`}>
                          {document.type}
                        </span>
                        <span className="rounded border border-zinc-700 bg-zinc-800 px-1.5 py-0.5 text-[9.5px] font-mono text-zinc-300">
                          {document.scope}
                        </span>
                        <span className={`rounded border px-1.5 py-0.5 text-[9.5px] font-semibold ${statusClass(document.status)}`}>
                          {document.status}
                        </span>
                        {document.project_slug ? (
                          <span className="rounded bg-zinc-800 px-1.5 py-0.5 font-mono text-[9.5px] text-zinc-400">
                            {document.project_slug}
                          </span>
                        ) : (
                          <span className="rounded bg-zinc-800 px-1.5 py-0.5 font-mono text-[9.5px] text-zinc-400">
                            global
                          </span>
                        )}
                      </div>
                      <h3 className="mt-1.5 truncate text-sm font-semibold text-zinc-100">
                        {document.title}
                        <span className="ml-2 font-mono text-[10.5px] font-normal text-zinc-500">{document.key}</span>
                      </h3>
                    </div>
                    <div className="flex shrink-0 items-center gap-1">
                      <button
                        onClick={() => {
                          startEdit(document)
                          setSelectedId(document.id)
                        }}
                        title="Edit registry metadata"
                        className="rounded border border-zinc-800 p-1 text-zinc-400 hover:text-sky-300"
                      >
                        <Pencil className="h-3 w-3" />
                      </button>
                      <button
                        onClick={() => void handleRefresh(document)}
                        disabled={busy === `refresh:${document.id}`}
                        title="Refresh snapshot from the canonical file"
                        className="rounded border border-zinc-800 p-1 text-zinc-400 hover:text-emerald-300 disabled:opacity-40"
                      >
                        <RefreshCw className={`h-3 w-3 ${busy === `refresh:${document.id}` ? 'animate-spin' : ''}`} />
                      </button>
                      <button
                        onClick={() => void handleDelete(document)}
                        disabled={busy === `delete:${document.id}`}
                        title="Delete registry entry (file untouched)"
                        className="rounded border border-zinc-800 p-1 text-zinc-500 hover:text-rose-300 disabled:opacity-40"
                      >
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </div>
                  </div>

                  {snippet ? (
                    <p className="mt-2 font-mono text-[11px] text-sky-200/90">{snippet}</p>
                  ) : (
                    document.summary && <p className="mt-2 line-clamp-2 text-xs text-zinc-400">{document.summary}</p>
                  )}

                  <div className="mt-2 flex flex-wrap items-center gap-3 font-mono text-[10px] text-zinc-600">
                    <span className="truncate">{document.path}</span>
                    <span>v{document.version}</span>
                    {document.content_hash && <span>sha256:{document.content_hash.slice(0, 10)}…</span>}
                    <span>{document.byte_size} B</span>
                  </div>
                  {(document.tags || []).length > 0 && (
                    <div className="mt-1.5 flex flex-wrap gap-1">
                      {(document.tags || []).map((tag) => (
                        <span key={tag} className="rounded bg-zinc-800/80 px-1.5 py-0.5 font-mono text-[9.5px] text-zinc-400">
                          #{tag}
                        </span>
                      ))}
                    </div>
                  )}
                </article>
              )
            })
          )}
        </div>
      </div>

      {/* Registry editor */}
      <aside className="flex w-[390px] shrink-0 flex-col overflow-y-auto bg-zinc-950 p-4">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wider text-zinc-300">
            <FileText className="h-4 w-4 text-sky-400" />
            {mode === 'create' ? 'Register document' : 'Edit metadata'}
          </h3>
          {mode === 'edit' && (
            <button onClick={resetForm} className="text-[11px] text-zinc-500 hover:text-zinc-200">
              + New
            </button>
          )}
        </div>

        <form onSubmit={handleSubmit} className="space-y-2.5">
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Project</label>
              <select
                value={form.project_slug}
                onChange={(e) =>
                  setForm({
                    ...form,
                    project_slug: e.target.value,
                    scope: form.scope === 'GLOBAL' ? 'PROJECT' : form.scope,
                  })
                }
                disabled={form.scope === 'GLOBAL' || mode === 'edit'}
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-300 outline-none focus:border-sky-500 disabled:opacity-50"
              >
                <option value="">Global</option>
                {projects.map((project) => (
                  <option key={project.slug} value={project.slug}>
                    {project.slug}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Scope</label>
              <select
                value={form.scope}
                onChange={(e) => setForm({ ...form, scope: e.target.value as DocumentScope })}
                disabled={mode === 'edit'}
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-300 outline-none focus:border-sky-500 disabled:opacity-50"
              >
                {docScopes.map((scope) => (
                  <option key={scope} value={scope}>
                    {scope}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Key</label>
              <input
                value={form.key}
                onChange={(e) => setForm({ ...form, key: e.target.value })}
                disabled={mode === 'edit'}
                placeholder="ARCH"
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-sky-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Slug</label>
              <input
                value={form.slug}
                onChange={(e) => setForm({ ...form, slug: e.target.value })}
                disabled={mode === 'edit'}
                placeholder="architecture"
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-sky-500 disabled:opacity-50"
              />
            </div>
          </div>

          {(form.scope === 'PLAN' || form.scope === 'TASK') && (
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Scope reference</label>
              <input
                value={form.scope_ref}
                onChange={(e) => setForm({ ...form, scope_ref: e.target.value })}
                disabled={mode === 'edit'}
                placeholder={form.scope === 'PLAN' ? 'plan slug' : 'task id'}
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-sky-500 disabled:opacity-50"
              />
            </div>
          )}

          <div>
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Title</label>
            <input
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              placeholder="Architecture overview"
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-200 outline-none focus:border-sky-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Type</label>
              <select
                value={form.type}
                onChange={(e) => setForm({ ...form, type: e.target.value as DocumentType })}
                disabled={mode === 'edit'}
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-300 outline-none focus:border-sky-500 disabled:opacity-50"
              >
                {docTypes.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Status</label>
              <select
                value={form.status}
                onChange={(e) => setForm({ ...form, status: e.target.value as DocumentStatus })}
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-300 outline-none focus:border-sky-500"
              >
                {docStatuses.map((status) => (
                  <option key={status} value={status}>
                    {status}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Canonical path</label>
            <input
              value={form.path}
              onChange={(e) => setForm({ ...form, path: e.target.value })}
              placeholder="/home/dev/src/owner/zenforge/docs/ARCHITECTURE.md"
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-[11px] text-zinc-200 outline-none focus:border-sky-500"
            />
          </div>

          <div>
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Summary</label>
            <textarea
              value={form.summary}
              onChange={(e) => setForm({ ...form, summary: e.target.value })}
              rows={2}
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-200 outline-none focus:border-sky-500"
            />
          </div>

          <div>
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Tags</label>
            <input
              value={form.tags}
              onChange={(e) => setForm({ ...form, tags: e.target.value })}
              placeholder="architecture, core"
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-200 outline-none focus:border-sky-500"
            />
          </div>

          {mode === 'create' && (
            <div>
              <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">
                Content snapshot (optional)
              </label>
              <textarea
                value={form.content}
                onChange={(e) => setForm({ ...form, content: e.target.value })}
                rows={6}
                placeholder="Leave empty to register the path; refresh later to snapshot the file."
                className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-[11px] text-zinc-300 outline-none focus:border-sky-500"
              />
            </div>
          )}

          <div className="flex gap-2 pt-1">
            <button
              type="submit"
              disabled={busy === 'create' || busy === 'update'}
              className="flex-1 rounded bg-sky-500/15 px-3 py-1.5 text-xs font-bold text-sky-300 ring-1 ring-sky-500/30 hover:bg-sky-500/25 disabled:opacity-50"
            >
              {mode === 'create' ? 'Register document' : 'Save metadata'}
            </button>
            {mode === 'edit' && (
              <button
                type="button"
                onClick={resetForm}
                className="rounded border border-zinc-800 px-3 py-1.5 text-xs text-zinc-400 hover:text-zinc-200"
              >
                Cancel
              </button>
            )}
          </div>
        </form>
      </aside>
    </div>
  )
}
