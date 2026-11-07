import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Archive,
  ArchiveRestore,
  FolderKanban,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
  ArrowRight,
} from 'lucide-react'
import { useZenfAPI } from '../../hooks/useZenfAPI'
import type { Plan, Project } from '../../types/contracts'

interface ProjectsViewProps {
  activeProjectSlug: string | null
  onOpenProject: (slug: string) => void
  onProjectsChanged: () => void
}

interface ProjectForm {
  slug: string
  name: string
  root_path: string
  description: string
}

const emptyForm: ProjectForm = { slug: '', name: '', root_path: '', description: '' }

const statusClass = (status: string) =>
  status === 'ARCHIVED'
    ? 'border-amber-500/30 bg-amber-500/10 text-amber-300'
    : 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300'

export const ProjectsView: React.FC<ProjectsViewProps> = ({
  activeProjectSlug,
  onOpenProject,
  onProjectsChanged,
}) => {
  const { fetchProjects, fetchPlans, createProject, updateProject, deleteProject } = useZenfAPI()
  const [projects, setProjects] = useState<Project[]>([])
  const [plans, setPlans] = useState<Plan[]>([])
  const [includeArchived, setIncludeArchived] = useState(false)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [form, setForm] = useState<ProjectForm>(emptyForm)
  const [editingSlug, setEditingSlug] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<ProjectForm>(emptyForm)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    const [projectResult, planResult] = await Promise.all([
      fetchProjects(includeArchived),
      fetchPlans('', true),
    ])
    if (!projectResult.ok) {
      setError(projectResult.error)
      setLoading(false)
      return
    }
    setProjects(projectResult.data)
    if (planResult.ok) setPlans(planResult.data)
    setLoading(false)
  }, [fetchPlans, fetchProjects, includeArchived])

  useEffect(() => {
    void load()
  }, [load])

  const planCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const plan of plans) {
      counts.set(plan.project_slug, (counts.get(plan.project_slug) || 0) + 1)
    }
    return counts
  }, [plans])

  const handleCreate = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!form.slug.trim() || !form.name.trim()) return
    setBusy('create')
    setError('')
    const result = await createProject({
      slug: form.slug.trim(),
      name: form.name.trim(),
      root_path: form.root_path.trim(),
      description: form.description.trim(),
    })
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    setForm(emptyForm)
    await load()
    onProjectsChanged()
  }

  const startEdit = (project: Project) => {
    setEditingSlug(project.slug)
    setEditForm({
      slug: project.slug,
      name: project.name,
      root_path: project.root_path || '',
      description: project.description || '',
    })
  }

  const handleSave = async (slug: string) => {
    setBusy(`edit:${slug}`)
    setError('')
    const result = await updateProject(slug, {
      name: editForm.name.trim(),
      root_path: editForm.root_path.trim(),
      description: editForm.description.trim(),
    })
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    setEditingSlug(null)
    await load()
    onProjectsChanged()
  }

  const handleArchive = async (project: Project) => {
    setBusy(`archive:${project.slug}`)
    setError('')
    const nextStatus = project.status === 'ARCHIVED' ? 'ACTIVE' : 'ARCHIVED'
    const result = await updateProject(project.slug, { status: nextStatus })
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    await load()
    onProjectsChanged()
  }

  const handleDelete = async (project: Project) => {
    if (!window.confirm(`Delete project "${project.name}" and every plan/task inside it?`)) return
    setBusy(`delete:${project.slug}`)
    setError('')
    const result = await deleteProject(project.slug)
    setBusy('')
    if (!result.ok) {
      setError(result.error)
      return
    }
    await load()
    onProjectsChanged()
  }

  return (
    <div className="flex h-full w-full overflow-y-auto bg-zinc-950 p-6">
      <div className="mx-auto w-full max-w-6xl space-y-6">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-zinc-800 pb-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/30">
              <FolderKanban className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-zinc-100">Projects</h2>
              <p className="text-xs text-zinc-400">
                Top-level working environments. Plans, tasks, and scoped documents live inside a project.
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setIncludeArchived((value) => !value)}
              className={`rounded border px-3 py-1.5 text-xs font-semibold transition ${
                includeArchived
                  ? 'border-amber-500/40 bg-amber-500/10 text-amber-300'
                  : 'border-zinc-800 bg-zinc-900 text-zinc-400 hover:text-zinc-200'
              }`}
            >
              {includeArchived ? 'Showing archived' : 'Show archived'}
            </button>
            <button
              type="button"
              onClick={() => void load()}
              disabled={loading}
              className="flex items-center gap-1.5 rounded border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs font-semibold text-zinc-300 transition hover:bg-zinc-800 disabled:opacity-50"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        </div>

        {error && (
          <div className="rounded-lg border border-rose-500/30 bg-rose-500/10 px-3 py-2 font-mono text-xs text-rose-300">
            {error}
          </div>
        )}

        <form
          onSubmit={handleCreate}
          className="grid gap-3 rounded-xl border border-zinc-800 bg-zinc-900/40 p-4 md:grid-cols-12"
        >
          <div className="md:col-span-3">
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Slug</label>
            <input
              value={form.slug}
              onChange={(e) => setForm({ ...form, slug: e.target.value })}
              placeholder="zenforge"
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-emerald-500"
            />
          </div>
          <div className="md:col-span-3">
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Name</label>
            <input
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="ZenForge"
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1.5 text-xs text-zinc-200 outline-none focus:border-emerald-500"
            />
          </div>
          <div className="md:col-span-4">
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Root path</label>
            <input
              value={form.root_path}
              onChange={(e) => setForm({ ...form, root_path: e.target.value })}
              placeholder="/home/dev/src/owner/zenforge"
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-emerald-500"
            />
          </div>
          <div className="flex items-end md:col-span-2">
            <button
              type="submit"
              disabled={busy === 'create' || !form.slug.trim() || !form.name.trim()}
              className="flex w-full items-center justify-center gap-1.5 rounded bg-emerald-500/15 px-3 py-1.5 text-xs font-bold text-emerald-300 ring-1 ring-emerald-500/30 transition hover:bg-emerald-500/25 disabled:opacity-40"
            >
              <Plus className="h-3.5 w-3.5" />
              Create
            </button>
          </div>
          <div className="md:col-span-12">
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">Description</label>
            <input
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder="Purpose, boundaries, and ownership of this working environment"
              className="w-full rounded border border-zinc-800 bg-zinc-950 px-2.5 py-1.5 text-xs text-zinc-200 outline-none focus:border-emerald-500"
            />
          </div>
        </form>

        {loading ? (
          <div className="py-12 text-center font-mono text-xs text-zinc-500">loading projects…</div>
        ) : projects.length === 0 ? (
          <div className="rounded-xl border border-dashed border-zinc-800 py-12 text-center font-mono text-xs text-zinc-500">
            No projects registered yet. Create one above, or call the <span className="text-emerald-400">projects</span> MCP tool.
          </div>
        ) : (
          <div className="grid gap-3 lg:grid-cols-2">
            {projects.map((project) => {
              const isEditing = editingSlug === project.slug
              return (
                <article
                  key={project.slug}
                  className={`rounded-xl border bg-zinc-900/40 p-4 transition ${
                    activeProjectSlug === project.slug
                      ? 'border-emerald-500/40 shadow-[0_0_20px_rgba(16,185,129,0.06)]'
                      : 'border-zinc-800'
                  }`}
                >
                  {isEditing ? (
                    <div className="space-y-2">
                      <input
                        value={editForm.name}
                        onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                        className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-sm font-semibold text-zinc-100 outline-none focus:border-emerald-500"
                      />
                      <input
                        value={editForm.root_path}
                        onChange={(e) => setEditForm({ ...editForm, root_path: e.target.value })}
                        className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 font-mono text-xs text-zinc-300 outline-none focus:border-emerald-500"
                      />
                      <textarea
                        value={editForm.description}
                        onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                        rows={2}
                        className="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-xs text-zinc-300 outline-none focus:border-emerald-500"
                      />
                      <div className="flex gap-2">
                        <button
                          onClick={() => void handleSave(project.slug)}
                          disabled={busy === `edit:${project.slug}`}
                          className="rounded bg-emerald-500/15 px-3 py-1 text-xs font-semibold text-emerald-300 ring-1 ring-emerald-500/30 hover:bg-emerald-500/25 disabled:opacity-50"
                        >
                          Save
                        </button>
                        <button
                          onClick={() => setEditingSlug(null)}
                          className="rounded border border-zinc-800 px-3 py-1 text-xs text-zinc-400 hover:text-zinc-200"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  ) : (
                    <>
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <h3 className="truncate text-sm font-bold text-zinc-100">{project.name}</h3>
                            <span className={`rounded-full border px-2 py-0.5 text-[10px] font-bold ${statusClass(project.status)}`}>
                              {project.status}
                            </span>
                            {activeProjectSlug === project.slug && (
                              <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-400">
                                active
                              </span>
                            )}
                          </div>
                          <div className="mt-1 font-mono text-[11px] text-zinc-500">{project.slug}</div>
                        </div>
                        <span className="shrink-0 rounded bg-zinc-800 px-2 py-1 text-[10px] font-mono text-zinc-400">
                          {planCounts.get(project.slug) || 0} plans
                        </span>
                      </div>

                      {project.description && (
                        <p className="mt-3 line-clamp-2 text-xs text-zinc-400">{project.description}</p>
                      )}
                      <div className="mt-2 truncate font-mono text-[10.5px] text-zinc-600">
                        {project.root_path || 'no root path recorded'}
                      </div>

                      <div className="mt-4 flex flex-wrap items-center gap-1.5">
                        <button
                          onClick={() => onOpenProject(project.slug)}
                          className="flex items-center gap-1 rounded border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-1 text-[11px] font-semibold text-emerald-300 hover:bg-emerald-500/20"
                        >
                          Open board <ArrowRight className="h-3 w-3" />
                        </button>
                        <button
                          onClick={() => startEdit(project)}
                          className="flex items-center gap-1 rounded border border-zinc-800 px-2.5 py-1 text-[11px] text-zinc-400 hover:text-zinc-200"
                        >
                          <Pencil className="h-3 w-3" /> Edit
                        </button>
                        <button
                          onClick={() => void handleArchive(project)}
                          disabled={busy === `archive:${project.slug}`}
                          className="flex items-center gap-1 rounded border border-zinc-800 px-2.5 py-1 text-[11px] text-zinc-400 hover:text-amber-300 disabled:opacity-50"
                        >
                          {project.status === 'ARCHIVED' ? (
                            <>
                              <ArchiveRestore className="h-3 w-3" /> Restore
                            </>
                          ) : (
                            <>
                              <Archive className="h-3 w-3" /> Archive
                            </>
                          )}
                        </button>
                        <button
                          onClick={() => void handleDelete(project)}
                          disabled={busy === `delete:${project.slug}`}
                          className="flex items-center gap-1 rounded border border-zinc-800 px-2.5 py-1 text-[11px] text-zinc-500 hover:text-rose-300 disabled:opacity-50"
                        >
                          <Trash2 className="h-3 w-3" /> Delete
                        </button>
                      </div>
                    </>
                  )}
                </article>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
