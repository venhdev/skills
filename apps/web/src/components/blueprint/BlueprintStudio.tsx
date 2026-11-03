import React, { useState, useEffect, useMemo, useRef } from 'react'
import {
  Layers,
  Search,
  Network,
  Activity,
  Database,
  FileCode,
  Scale,
  FolderGit2,
  ZoomIn,
  ZoomOut,
  RotateCcw,
  Sparkles,
  Check,
} from 'lucide-react'
import mermaid from 'mermaid'
import type { TechSpec, SpecTier, Task } from '../../types/contracts'
import { useZenfdAPI } from '../../hooks/useZenfdAPI'

// Initialize Mermaid once
try {
  mermaid.initialize({
    startOnLoad: false,
    theme: 'dark',
    securityLevel: 'loose',
    fontFamily: 'JetBrains Mono, monospace',
    themeVariables: {
      darkMode: true,
      background: '#09090b',
      primaryColor: '#7c3aed',
      primaryBorderColor: '#a855f7',
      primaryTextColor: '#f4f4f5',
      lineColor: '#52525b',
      secondaryColor: '#18181b',
      tertiaryColor: '#27272a',
    },
  })
} catch {
  // Ignore re-initialization
}

interface BlueprintStudioProps {
  tasks: Task[]
  onSelectTask?: (task: Task) => void
}

const TIER_CONFIG: Record<
  SpecTier,
  { label: string; description: string; icon: React.ReactNode; color: string }
> = {
  1: {
    label: 'Topology',
    description: 'Tier 1: Boundary Containers & Subsystem Seams',
    icon: <Network className="h-4 w-4 text-violet-400" />,
    color: 'border-violet-500/50 text-violet-400 bg-violet-950/20',
  },
  2: {
    label: 'Dynamics',
    description: 'Tier 2: Workflows, State Transitions & Sequence Handshakes',
    icon: <Activity className="h-4 w-4 text-cyan-400" />,
    color: 'border-cyan-500/50 text-cyan-400 bg-cyan-950/20',
  },
  3: {
    label: 'Persistence',
    description: 'Tier 3: Database Tables, WAL Primitives & ERD Schemas',
    icon: <Database className="h-4 w-4 text-emerald-400" />,
    color: 'border-emerald-500/50 text-emerald-400 bg-emerald-950/20',
  },
  4: {
    label: 'Contracts',
    description: 'Tier 4: Boundary Interfaces, JSON Schemas & Handshakes',
    icon: <FileCode className="h-4 w-4 text-amber-400" />,
    color: 'border-amber-500/50 text-amber-400 bg-amber-950/20',
  },
  5: {
    label: 'ADR',
    description: 'Tier 5: Architecture Decision Records & Trade-offs',
    icon: <Scale className="h-4 w-4 text-rose-400" />,
    color: 'border-rose-500/50 text-rose-400 bg-rose-950/20',
  },
}

// Mermaid Renderer Component with Zoom/Pan
const MermaidRenderer: React.FC<{ code: string }> = ({ code }) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const [svg, setSvg] = useState<string>('')
  const [error, setError] = useState<string | null>(null)
  const [scale, setScale] = useState<number>(1)

  useEffect(() => {
    let isMounted = true
    if (!code || !code.trim()) {
      setSvg('')
      setError(null)
      return
    }

    const id = `mermaid-spec-${Math.random().toString(36).substring(2, 9)}`
    mermaid
      .render(id, code)
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
  }, [code])

  if (error) {
    return (
      <div className="rounded-lg border border-rose-800/50 bg-rose-950/20 p-4 font-mono text-xs text-rose-400">
        <p className="font-bold">Mermaid Render Error</p>
        <p className="mt-1 opacity-80">{error}</p>
        <pre className="mt-2 overflow-x-auto rounded bg-zinc-950/80 p-2 text-zinc-400">
          {code}
        </pre>
      </div>
    )
  }

  if (!svg) {
    return (
      <div className="flex h-64 items-center justify-center text-xs text-zinc-500 font-mono">
        Rendering diagram...
      </div>
    )
  }

  return (
    <div className="relative group rounded-xl border border-zinc-800/80 bg-zinc-950/60 p-4">
      {/* Zoom Controls */}
      <div className="absolute right-3 top-3 z-10 flex items-center gap-1 rounded-lg border border-zinc-800 bg-zinc-900/90 p-1 opacity-60 backdrop-blur transition-opacity group-hover:opacity-100">
        <button
          onClick={() => setScale((s) => Math.min(s + 0.2, 2.0))}
          className="rounded p-1 hover:bg-zinc-800 text-zinc-400 hover:text-white"
          title="Zoom In"
        >
          <ZoomIn className="h-3.5 w-3.5" />
        </button>
        <button
          onClick={() => setScale((s) => Math.max(s - 0.2, 0.4))}
          className="rounded p-1 hover:bg-zinc-800 text-zinc-400 hover:text-white"
          title="Zoom Out"
        >
          <ZoomOut className="h-3.5 w-3.5" />
        </button>
        <button
          onClick={() => setScale(1)}
          className="rounded p-1 hover:bg-zinc-800 text-zinc-400 hover:text-white text-[10px] font-mono px-1.5"
          title="Reset"
        >
          <RotateCcw className="h-3 w-3" />
        </button>
      </div>

      <div
        ref={containerRef}
        className="overflow-auto max-h-[520px] transition-transform duration-150 origin-top-left flex justify-center py-4"
        style={{ transform: `scale(${scale})` }}
        dangerouslySetInnerHTML={{ __html: svg }}
      />
    </div>
  )
}

export const BlueprintStudio: React.FC<BlueprintStudioProps> = ({ tasks, onSelectTask }) => {
  const { fetchSpecs, searchSpecs } = useZenfdAPI()
  const [specs, setSpecs] = useState<TechSpec[]>([])
  const [selectedTier, setSelectedTier] = useState<SpecTier | 0>(0)
  const [selectedSpec, setSelectedSpec] = useState<TechSpec | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [copiedSlug, setCopiedSlug] = useState<string | null>(null)

  // Load Specs
  const loadSpecs = async () => {
    setLoading(true)
    try {
      const data = await fetchSpecs()
      setSpecs(data)
      if (data.length > 0 && !selectedSpec) {
        setSelectedSpec(data[0])
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadSpecs()
  }, [])

  // FTS5 or client-side filter
  useEffect(() => {
    if (!searchQuery.trim()) {
      loadSpecs()
      return
    }
    const timer = setTimeout(async () => {
      const results = await searchSpecs(searchQuery)
      setSpecs(results)
      if (results.length > 0) {
        setSelectedSpec(results[0])
      }
    }, 250)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // Filtered specs by tier
  const filteredSpecs = useMemo(() => {
    if (selectedTier === 0) return specs
    return specs.filter((s) => s.tier === selectedTier)
  }, [specs, selectedTier])

  // Find tasks that match bound paths of selected spec
  const boundTasks = useMemo(() => {
    if (!selectedSpec || !selectedSpec.bound_paths || selectedSpec.bound_paths.length === 0) {
      return []
    }
    return tasks.filter((t) => {
      if (!t.target_files || t.target_files.length === 0) return false
      return t.target_files.some((tf) =>
        selectedSpec.bound_paths.some((bp) => bp && (tf.startsWith(bp) || tf.includes(bp)))
      )
    })
  }, [selectedSpec, tasks])

  const copySlug = (slug: string) => {
    navigator.clipboard.writeText(slug)
    setCopiedSlug(slug)
    setTimeout(() => setCopiedSlug(null), 1500)
  }

  return (
    <div className="flex h-full flex-col bg-zinc-950 text-zinc-100">
      {/* Top Header Bar */}
      <div className="flex h-14 shrink-0 items-center justify-between border-b border-zinc-800/80 px-6 bg-zinc-900/40">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-violet-600/20 border border-violet-500/30 text-violet-400">
            <Layers className="h-4 w-4" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold tracking-wide text-zinc-100">Blueprint Studio</h2>
              <span className="rounded-full bg-violet-500/10 px-2 py-0.5 text-[10px] font-mono text-violet-400 border border-violet-500/20">
                SSOT Architecture Vault
              </span>
            </div>
            <p className="text-[11px] text-zinc-500">
              5-Tier Technical Specifications & Zero-Round-Trip Injection
            </p>
          </div>
        </div>

        {/* Search Input */}
        <div className="relative w-80">
          <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-zinc-500" />
          <input
            type="text"
            placeholder="Search specs (FTS5 full-text)..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-lg border border-zinc-800 bg-zinc-900/60 py-1.5 pl-9 pr-3 text-xs text-zinc-200 placeholder-zinc-500 focus:border-violet-500 focus:outline-none"
          />
        </div>
      </div>

      {/* Tier Switcher Tabs */}
      <div className="flex items-center gap-1.5 border-b border-zinc-800/60 bg-zinc-900/20 px-6 py-2 overflow-x-auto">
        <button
          onClick={() => setSelectedTier(0)}
          className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-all ${
            selectedTier === 0
              ? 'bg-zinc-800 text-zinc-100 shadow-sm'
              : 'text-zinc-400 hover:bg-zinc-800/50 hover:text-zinc-200'
          }`}
        >
          <Sparkles className="h-3.5 w-3.5 text-zinc-400" />
          <span>All Tiers</span>
          <span className="rounded bg-zinc-800/80 px-1.5 py-0.2 text-[10px] font-mono text-zinc-400">
            {specs.length}
          </span>
        </button>

        {([1, 2, 3, 4, 5] as SpecTier[]).map((tier) => {
          const cfg = TIER_CONFIG[tier]
          const count = specs.filter((s) => s.tier === tier).length
          const isSelected = selectedTier === tier
          return (
            <button
              key={tier}
              onClick={() => setSelectedTier(tier)}
              className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-all ${
                isSelected
                  ? 'bg-zinc-800 text-zinc-100 shadow-sm border border-zinc-700/60'
                  : 'text-zinc-400 hover:bg-zinc-800/50 hover:text-zinc-200'
              }`}
            >
              {cfg.icon}
              <span>{cfg.label}</span>
              <span className="rounded bg-zinc-800/80 px-1.5 py-0.2 text-[10px] font-mono text-zinc-400">
                {count}
              </span>
            </button>
          )
        })}
      </div>

      {/* Main Studio Body: Spec List on Left, Viewer on Right */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left Sidebar: Spec List */}
        <div className="w-80 shrink-0 border-r border-zinc-800/80 bg-zinc-950/40 p-4 overflow-y-auto">
          <div className="flex items-center justify-between text-[11px] font-medium text-zinc-500 uppercase tracking-wider mb-3">
            <span>Specifications ({filteredSpecs.length})</span>
          </div>

          {filteredSpecs.length === 0 ? (
            <div className="rounded-lg border border-dashed border-zinc-800 p-6 text-center text-xs text-zinc-500 font-mono">
              {loading ? 'Loading specifications...' : 'No specifications in this tier.'}
            </div>
          ) : (
            <div className="space-y-2">
              {filteredSpecs.map((spec) => {
                const isSelected = selectedSpec?.slug === spec.slug
                const tierCfg = TIER_CONFIG[spec.tier as SpecTier] || TIER_CONFIG[1]
                return (
                  <div
                    key={spec.slug}
                    onClick={() => setSelectedSpec(spec)}
                    className={`group cursor-pointer rounded-lg border p-3 transition-all ${
                      isSelected
                        ? 'border-violet-500/80 bg-violet-950/20 shadow-[0_0_12px_rgba(139,92,246,0.15)]'
                        : 'border-zinc-800/80 bg-zinc-900/30 hover:border-zinc-700 hover:bg-zinc-900/60'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span
                        className={`rounded px-1.5 py-0.5 text-[9px] font-mono font-semibold uppercase tracking-wider border ${tierCfg.color}`}
                      >
                        T{spec.tier} · {spec.kind}
                      </span>
                      <span className="text-[10px] font-mono text-zinc-500">
                        v{spec.version}
                      </span>
                    </div>

                    <div className="mt-2 text-xs font-medium text-zinc-200 group-hover:text-white line-clamp-1">
                      {spec.title}
                    </div>

                    <div className="mt-1 flex items-center justify-between text-[10px] text-zinc-500 font-mono">
                      <span className="truncate max-w-[170px]">{spec.slug}</span>
                      {spec.bound_paths && spec.bound_paths.length > 0 && (
                        <span className="text-zinc-400">
                          {spec.bound_paths.length} bound
                        </span>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* Right Main Pane: Active Spec Inspection */}
        <div className="flex-1 overflow-y-auto p-6 bg-zinc-950/20">
          {!selectedSpec ? (
            <div className="flex h-full items-center justify-center text-center text-zinc-500 font-mono text-xs">
              Select a specification from the left to inspect architectural diagrams and contracts.
            </div>
          ) : (
            <div className="space-y-6 max-w-5xl mx-auto">
              {/* Spec Header Card */}
              <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-5">
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <span
                        className={`rounded px-2 py-0.5 text-[10px] font-mono font-bold uppercase tracking-wider border ${
                          TIER_CONFIG[selectedSpec.tier as SpecTier]?.color || TIER_CONFIG[1].color
                        }`}
                      >
                        Tier {selectedSpec.tier}: {TIER_CONFIG[selectedSpec.tier as SpecTier]?.label} · {selectedSpec.kind}
                      </span>
                      <span className="text-[11px] font-mono text-zinc-500">
                        Version {selectedSpec.version}
                      </span>
                    </div>

                    <h1 className="mt-2 text-lg font-bold text-zinc-100">{selectedSpec.title}</h1>

                    <div className="mt-1 flex items-center gap-2 text-xs font-mono text-zinc-400">
                      <span>slug:</span>
                      <span className="text-zinc-200">{selectedSpec.slug}</span>
                      <button
                        onClick={() => copySlug(selectedSpec.slug)}
                        className="rounded p-1 hover:bg-zinc-800 text-zinc-500 hover:text-zinc-300"
                        title="Copy Slug"
                      >
                        {copiedSlug === selectedSpec.slug ? (
                          <Check className="h-3 w-3 text-emerald-400" />
                        ) : (
                          <span className="text-[10px] text-zinc-500">copy</span>
                        )}
                      </button>
                    </div>
                  </div>

                  <div className="text-right text-[11px] font-mono text-zinc-500">
                    <div>Updated: {new Date(selectedSpec.updated_at).toLocaleString()}</div>
                    {selectedSpec.repo_path && (
                      <div className="truncate max-w-[200px] text-zinc-600">
                        {selectedSpec.repo_path}
                      </div>
                    )}
                  </div>
                </div>

                {/* Bound Paths Seams */}
                {selectedSpec.bound_paths && selectedSpec.bound_paths.length > 0 && (
                  <div className="mt-4 border-t border-zinc-800/80 pt-3">
                    <div className="text-[11px] font-mono text-zinc-500 mb-1.5 flex items-center gap-1.5">
                      <FolderGit2 className="h-3.5 w-3.5 text-zinc-400" />
                      <span>Bound File Paths (Path-Bound Auto-Linking):</span>
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {selectedSpec.bound_paths.map((p, idx) => (
                        <span
                          key={idx}
                          className="rounded bg-zinc-800/80 px-2 py-0.5 text-[11px] font-mono text-zinc-300 border border-zinc-700/60"
                        >
                          {p}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                {/* Architectural Notes */}
                {selectedSpec.notes && (
                  <div className="mt-4 rounded-lg bg-zinc-950/60 border border-zinc-800/80 p-3 text-xs text-zinc-300 leading-relaxed font-sans">
                    <p className="font-semibold text-zinc-400 text-[11px] uppercase tracking-wider mb-1 font-mono">
                      Architectural Invariants & Notes
                    </p>
                    {selectedSpec.notes}
                  </div>
                )}
              </div>

              {/* Diagram Rendering Section */}
              {selectedSpec.diagram_code && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-xs font-mono text-zinc-400">
                    <span className="font-semibold text-zinc-300 uppercase tracking-wider">
                      Interactive Visual Model ({selectedSpec.diagram_type})
                    </span>
                  </div>
                  <MermaidRenderer code={selectedSpec.diagram_code} />
                </div>
              )}

              {/* Structured Payload Contract (if present) */}
              {selectedSpec.structured_payload &&
                Object.keys(selectedSpec.structured_payload).length > 0 && (
                  <div className="rounded-xl border border-zinc-800 bg-zinc-900/30 p-4">
                    <div className="text-xs font-mono font-semibold text-zinc-300 uppercase tracking-wider mb-2">
                      Structured Schema Payload (Zero-Round-Trip Injection)
                    </div>
                    <pre className="overflow-x-auto rounded-lg bg-zinc-950 p-3 text-xs font-mono text-emerald-400 border border-zinc-800/80">
                      {JSON.stringify(selectedSpec.structured_payload, null, 2)}
                    </pre>
                  </div>
                )}

              {/* Linked Tasks in Current Plan */}
              <div className="rounded-xl border border-zinc-800 bg-zinc-900/30 p-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="text-xs font-mono font-semibold text-zinc-300 uppercase tracking-wider">
                    Governed Tasks ({boundTasks.length})
                  </div>
                  <span className="text-[11px] text-zinc-500 font-mono">
                    Auto-linked via target_files ∩ bound_paths
                  </span>
                </div>

                {boundTasks.length === 0 ? (
                  <div className="text-xs text-zinc-500 font-mono p-3 rounded-lg border border-dashed border-zinc-800/80 text-center">
                    No tasks in the active plan currently target files under this spec's bound paths.
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                    {boundTasks.map((t) => (
                      <div
                        key={t.id}
                        onClick={() => onSelectTask && onSelectTask(t)}
                        className="flex items-center justify-between p-2.5 rounded-lg border border-zinc-800 bg-zinc-950/60 hover:border-violet-500/60 cursor-pointer transition-all"
                      >
                        <div className="flex items-center gap-2 truncate">
                          <span className="font-mono text-xs font-bold text-zinc-200">
                            {t.id}
                          </span>
                          <span className="text-xs text-zinc-400 truncate max-w-[200px]">
                            {t.title}
                          </span>
                        </div>
                        <span className="text-[10px] font-mono px-1.5 py-0.5 rounded uppercase font-semibold text-zinc-400 bg-zinc-800">
                          {t.status}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
