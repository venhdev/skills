import React, { useState, useEffect, useMemo } from 'react'
import {
  FileText,
  GitCompare,
  History,
  FilePlus2,
  FileEdit,
  FileX2,
  FileCheck2,
  Calendar,
  User,
  FolderArchive,
} from 'lucide-react'
import type { PlanRevision, SnapshotFile, FileDiff } from '../../types/contracts'
import { useZenfdAPI } from '../../hooks/useZenfdAPI'

interface DossierExplorerProps {
  planSlug: string
}

export const DossierExplorer: React.FC<DossierExplorerProps> = ({ planSlug }) => {
  const { fetchPlanRevisions, fetchPlanSnapshot, fetchPlanDiff } = useZenfdAPI()

  const [revisions, setRevisions] = useState<PlanRevision[]>([])
  const [selectedRev, setSelectedRev] = useState<string>('')
  const [compareRev, setCompareRev] = useState<string>('')
  const [snapshotFiles, setSnapshotFiles] = useState<SnapshotFile[]>([])
  const [diffs, setDiffs] = useState<FileDiff[]>([])
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [mode, setMode] = useState<'FILES' | 'DIFF'>('FILES')

  // Load Revisions
  useEffect(() => {
    if (!planSlug) return
    const load = async () => {
      try {
        const revs = await fetchPlanRevisions(planSlug)
        setRevisions(revs)
        if (revs.length > 0) {
          const latest = revs[revs.length - 1].revision
          setSelectedRev(latest)
          if (revs.length > 1) {
            setCompareRev(revs[revs.length - 2].revision)
          } else {
            setCompareRev(latest)
          }
        }
      } catch {
        // Ignore load error
      }
    }
    load()
  }, [planSlug])

  // Load Snapshot Files when selectedRev changes
  useEffect(() => {
    if (!planSlug || !selectedRev) return
    const loadFiles = async () => {
      const files = await fetchPlanSnapshot(planSlug, selectedRev)
      setSnapshotFiles(files)
      if (files.length > 0 && !selectedFile) {
        setSelectedFile(files[0].file_path)
      }
    }
    loadFiles()
  }, [planSlug, selectedRev])

  // Load Diffs when compareRev & selectedRev change
  useEffect(() => {
    if (!planSlug || !selectedRev || !compareRev || selectedRev === compareRev) {
      setDiffs([])
      return
    }
    const loadDiff = async () => {
      const res = await fetchPlanDiff(planSlug, compareRev, selectedRev)
      setDiffs(res)
      if (res.length > 0 && !selectedFile) {
        setSelectedFile(res[0].file_path)
      }
    }
    loadDiff()
  }, [planSlug, selectedRev, compareRev])

  const activeSnapshotContent = useMemo(() => {
    return snapshotFiles.find((f) => f.file_path === selectedFile)?.content || ''
  }, [snapshotFiles, selectedFile])

  const activeDiff = useMemo(() => {
    return diffs.find((d) => d.file_path === selectedFile) || null
  }, [diffs, selectedFile])

  return (
    <div className="flex h-full flex-col bg-zinc-950 text-zinc-100">
      {/* Top Header */}
      <div className="flex h-14 shrink-0 items-center justify-between border-b border-zinc-800/80 px-6 bg-zinc-900/40">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-600/20 border border-emerald-500/30 text-emerald-400">
            <FolderArchive className="h-4 w-4" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold tracking-wide text-zinc-100">Dossier Vault & Revisions</h2>
              <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-mono text-emerald-400 border border-emerald-500/20">
                Plan: {planSlug}
              </span>
            </div>
            <p className="text-[11px] text-zinc-500">
              Immutable Folder Snapshots, Semantic Revisioning & Visual Diff Engine
            </p>
          </div>
        </div>

        {/* View Mode Toggle */}
        <div className="flex items-center gap-1 rounded-lg border border-zinc-800 bg-zinc-900 p-0.5 text-xs">
          <button
            onClick={() => setMode('FILES')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1 font-medium transition-all ${
              mode === 'FILES' ? 'bg-zinc-800 text-white' : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <FileText className="h-3.5 w-3.5" />
            <span>Files ({snapshotFiles.length})</span>
          </button>
          <button
            onClick={() => setMode('DIFF')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1 font-medium transition-all ${
              mode === 'DIFF' ? 'bg-zinc-800 text-white' : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <GitCompare className="h-3.5 w-3.5" />
            <span>Compare Diff</span>
          </button>
        </div>
      </div>

      {/* Revision Selection Bar */}
      <div className="flex items-center justify-between border-b border-zinc-800/60 bg-zinc-900/20 px-6 py-2.5">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 text-xs font-mono">
            <History className="h-3.5 w-3.5 text-zinc-400" />
            <span className="text-zinc-400">Active Revision:</span>
            <select
              value={selectedRev}
              onChange={(e) => setSelectedRev(e.target.value)}
              className="rounded-md border border-zinc-800 bg-zinc-900 px-2.5 py-1 text-xs font-mono text-zinc-200 focus:border-emerald-500 focus:outline-none"
            >
              {revisions.map((r) => (
                <option key={r.revision} value={r.revision}>
                  {r.revision} — {r.summary || 'Snapshot'}
                </option>
              ))}
            </select>
          </div>

          {mode === 'DIFF' && (
            <div className="flex items-center gap-2 text-xs font-mono">
              <span className="text-zinc-500">comparing against:</span>
              <select
                value={compareRev}
                onChange={(e) => setCompareRev(e.target.value)}
                className="rounded-md border border-zinc-800 bg-zinc-900 px-2.5 py-1 text-xs font-mono text-zinc-200 focus:border-emerald-500 focus:outline-none"
              >
                {revisions.map((r) => (
                  <option key={r.revision} value={r.revision}>
                    {r.revision}
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>

        {revisions.find((r) => r.revision === selectedRev) && (
          <div className="flex items-center gap-4 text-[11px] font-mono text-zinc-500">
            <div className="flex items-center gap-1">
              <User className="h-3 w-3" />
              <span>{revisions.find((r) => r.revision === selectedRev)?.author || 'system'}</span>
            </div>
            <div className="flex items-center gap-1">
              <Calendar className="h-3 w-3" />
              <span>
                {new Date(
                  revisions.find((r) => r.revision === selectedRev)?.created_at || ''
                ).toLocaleDateString()}
              </span>
            </div>
          </div>
        )}
      </div>

      {/* Main Content Area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left File/Diff Tree */}
        <div className="w-80 shrink-0 border-r border-zinc-800/80 bg-zinc-950/40 p-4 overflow-y-auto">
          {mode === 'FILES' ? (
            <div>
              <div className="text-[11px] font-medium text-zinc-500 uppercase tracking-wider mb-2">
                Preserved Snapshot Files ({snapshotFiles.length})
              </div>
              <div className="space-y-1">
                {snapshotFiles.map((file) => {
                  const isSelected = selectedFile === file.file_path
                  return (
                    <div
                      key={file.file_path}
                      onClick={() => setSelectedFile(file.file_path)}
                      className={`flex items-center justify-between p-2 rounded-lg text-xs font-mono cursor-pointer transition-all ${
                        isSelected
                          ? 'bg-zinc-800 text-white border border-zinc-700'
                          : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                      }`}
                    >
                      <div className="flex items-center gap-2 truncate">
                        <FileText className="h-3.5 w-3.5 shrink-0 text-zinc-500" />
                        <span className="truncate">{file.file_path}</span>
                      </div>
                      <span className="text-[10px] text-zinc-600 font-mono">
                        {(file.byte_size / 1024).toFixed(1)}k
                      </span>
                    </div>
                  )
                })}
              </div>
            </div>
          ) : (
            <div>
              <div className="text-[11px] font-medium text-zinc-500 uppercase tracking-wider mb-2">
                File Differences ({diffs.length})
              </div>
              <div className="space-y-1">
                {diffs.map((diff) => {
                  const isSelected = selectedFile === diff.file_path
                  const isAdded = diff.change_type === 'ADDED'
                  const isMod = diff.change_type === 'MODIFIED'
                  const isDel = diff.change_type === 'DELETED'
                  return (
                    <div
                      key={diff.file_path}
                      onClick={() => setSelectedFile(diff.file_path)}
                      className={`flex items-center justify-between p-2 rounded-lg text-xs font-mono cursor-pointer transition-all ${
                        isSelected
                          ? 'bg-zinc-800 text-white border border-zinc-700'
                          : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                      }`}
                    >
                      <div className="flex items-center gap-2 truncate">
                        {isAdded && <FilePlus2 className="h-3.5 w-3.5 text-emerald-400 shrink-0" />}
                        {isMod && <FileEdit className="h-3.5 w-3.5 text-amber-400 shrink-0" />}
                        {isDel && <FileX2 className="h-3.5 w-3.5 text-rose-400 shrink-0" />}
                        {!isAdded && !isMod && !isDel && (
                          <FileCheck2 className="h-3.5 w-3.5 text-zinc-500 shrink-0" />
                        )}
                        <span className="truncate">{diff.file_path}</span>
                      </div>
                      <span
                        className={`text-[9px] px-1.5 py-0.2 rounded font-mono font-bold ${
                          isAdded
                            ? 'bg-emerald-950 text-emerald-400 border border-emerald-800/60'
                            : isMod
                            ? 'bg-amber-950 text-amber-400 border border-amber-800/60'
                            : isDel
                            ? 'bg-rose-950 text-rose-400 border border-rose-800/60'
                            : 'bg-zinc-900 text-zinc-500'
                        }`}
                      >
                        {diff.change_type}
                      </span>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>

        {/* Right Code/Diff Preview */}
        <div className="flex-1 overflow-y-auto p-6 bg-zinc-950/20">
          {!selectedFile ? (
            <div className="flex h-full items-center justify-center text-center text-xs text-zinc-500 font-mono">
              Select a file on the left to inspect its snapshot content or revision diff.
            </div>
          ) : mode === 'FILES' ? (
            <div className="space-y-4 max-w-5xl mx-auto">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div className="flex items-center gap-2 font-mono text-sm font-bold text-zinc-200">
                  <FileText className="h-4 w-4 text-emerald-400" />
                  <span>{selectedFile}</span>
                </div>
                <span className="text-xs font-mono text-zinc-500">
                  Snapshot: {selectedRev}
                </span>
              </div>
              <pre className="overflow-x-auto rounded-xl border border-zinc-800 bg-zinc-950 p-4 font-mono text-xs text-zinc-300 leading-relaxed">
                {activeSnapshotContent}
              </pre>
            </div>
          ) : activeDiff ? (
            <div className="space-y-4 max-w-5xl mx-auto">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div className="flex items-center gap-2 font-mono text-sm font-bold text-zinc-200">
                  <GitCompare className="h-4 w-4 text-violet-400" />
                  <span>{selectedFile}</span>
                  <span
                    className={`ml-2 text-xs px-2 py-0.5 rounded font-bold uppercase ${
                      activeDiff.change_type === 'ADDED'
                        ? 'bg-emerald-950 text-emerald-400'
                        : activeDiff.change_type === 'MODIFIED'
                        ? 'bg-amber-950 text-amber-400'
                        : activeDiff.change_type === 'DELETED'
                        ? 'bg-rose-950 text-rose-400'
                        : 'bg-zinc-900 text-zinc-500'
                    }`}
                  >
                    {activeDiff.change_type}
                  </span>
                </div>
                <div className="text-xs font-mono text-zinc-500">
                  {compareRev} ➔ {selectedRev}
                </div>
              </div>

              {activeDiff.change_type === 'MODIFIED' ? (
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <div className="text-xs font-mono text-zinc-500 mb-1">
                      Old ({compareRev})
                    </div>
                    <pre className="overflow-x-auto rounded-xl border border-rose-900/30 bg-rose-950/10 p-3 font-mono text-xs text-rose-300 max-h-[600px]">
                      {activeDiff.old_content}
                    </pre>
                  </div>
                  <div>
                    <div className="text-xs font-mono text-zinc-500 mb-1">
                      New ({selectedRev})
                    </div>
                    <pre className="overflow-x-auto rounded-xl border border-emerald-900/30 bg-emerald-950/10 p-3 font-mono text-xs text-emerald-300 max-h-[600px]">
                      {activeDiff.new_content}
                    </pre>
                  </div>
                </div>
              ) : activeDiff.change_type === 'ADDED' ? (
                <div>
                  <div className="text-xs font-mono text-emerald-400 mb-1">
                    New File Added in {selectedRev}
                  </div>
                  <pre className="overflow-x-auto rounded-xl border border-emerald-900/40 bg-emerald-950/10 p-4 font-mono text-xs text-emerald-300">
                    {activeDiff.new_content}
                  </pre>
                </div>
              ) : activeDiff.change_type === 'DELETED' ? (
                <div>
                  <div className="text-xs font-mono text-rose-400 mb-1">
                    File Deleted in {selectedRev}
                  </div>
                  <pre className="overflow-x-auto rounded-xl border border-rose-900/40 bg-rose-950/10 p-4 font-mono text-xs text-rose-300 opacity-60">
                    {activeDiff.old_content}
                  </pre>
                </div>
              ) : (
                <div className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/40 text-center font-mono text-xs text-zinc-500">
                  Identical content across both revisions.
                </div>
              )}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}
