import React from 'react'
import {
  CheckCircle2,
  PlayCircle,
  Clock,
  AlertTriangle,
  Unplug,
  GitMerge,
  AlertCircle,
  Ban,
  CircleDashed,
  XCircle,
} from 'lucide-react'
import type { TaskStatus } from './contracts'

export interface TaskStatusMetadata {
  label: string
  border: string
  bg: string
  text: string
  glow: string
  icon: React.ReactNode
  badge?: string
}

export const TASK_STATUS_CONFIG: Record<TaskStatus, TaskStatusMetadata> = {
  COMPLETED: {
    label: 'Completed',
    border: 'border-emerald-500/40',
    bg: 'bg-emerald-950/20',
    text: 'text-emerald-400',
    glow: 'hover:shadow-[0_0_12px_rgba(16,185,129,0.2)]',
    badge: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-400',
    icon: React.createElement(CheckCircle2, { className: 'h-3.5 w-3.5 text-emerald-400' }),
  },
  RUNNING: {
    label: 'Running',
    border: 'border-violet-500/80',
    bg: 'bg-violet-950/30',
    text: 'text-violet-400',
    glow: 'shadow-[0_0_16px_rgba(139,92,246,0.3)] animate-pulse',
    badge: 'border-violet-500/40 bg-violet-500/10 text-violet-400 animate-pulse',
    icon: React.createElement(PlayCircle, { className: 'h-3.5 w-3.5 text-violet-400 animate-spin' }),
  },
  READY: {
    label: 'Ready',
    border: 'border-amber-500/40',
    bg: 'bg-amber-950/20',
    text: 'text-amber-400',
    glow: 'hover:shadow-[0_0_12px_rgba(245,158,11,0.2)]',
    badge: 'border-amber-500/40 bg-amber-500/10 text-amber-400',
    icon: React.createElement(Clock, { className: 'h-3.5 w-3.5 text-amber-400' }),
  },
  AWAITING_REVIEW: {
    label: 'Awaiting Review',
    border: 'border-orange-500/70',
    bg: 'bg-orange-950/30',
    text: 'text-orange-400',
    glow: 'shadow-[0_0_16px_rgba(249,115,22,0.3)] animate-pulse',
    badge: 'border-orange-500/40 bg-orange-500/15 text-orange-400 animate-pulse',
    icon: React.createElement(AlertTriangle, { className: 'h-3.5 w-3.5 text-orange-400' }),
  },
  SUSPENDED: {
    label: 'Suspended',
    border: 'border-amber-600/70',
    bg: 'bg-amber-950/30',
    text: 'text-amber-400',
    glow: 'shadow-[0_0_14px_rgba(217,119,6,0.3)]',
    badge: 'border-amber-600/50 bg-amber-600/15 text-amber-400 font-bold',
    icon: React.createElement(Unplug, { className: 'h-3.5 w-3.5 text-amber-500' }),
  },
  AWAITING_MERGE: {
    label: 'Awaiting Merge',
    border: 'border-cyan-500/70',
    bg: 'bg-cyan-950/30',
    text: 'text-cyan-400',
    glow: 'shadow-[0_0_16px_rgba(6,182,212,0.3)]',
    badge: 'border-cyan-500/40 bg-cyan-500/15 text-cyan-400',
    icon: React.createElement(GitMerge, { className: 'h-3.5 w-3.5 text-cyan-400' }),
  },
  CONFLICT: {
    label: 'Conflict',
    border: 'border-rose-500/80',
    bg: 'bg-rose-950/40',
    text: 'text-rose-400',
    glow: 'shadow-[0_0_16px_rgba(244,63,94,0.4)]',
    badge: 'border-rose-500/40 bg-rose-500/20 text-rose-400 font-bold',
    icon: React.createElement(AlertCircle, { className: 'h-3.5 w-3.5 text-rose-400' }),
  },
  FAILED: {
    label: 'Failed',
    border: 'border-rose-500/60',
    bg: 'bg-rose-950/30',
    text: 'text-rose-400',
    glow: 'shadow-[0_0_14px_rgba(244,63,94,0.3)]',
    badge: 'border-rose-500/40 bg-rose-500/10 text-rose-400',
    icon: React.createElement(AlertCircle, { className: 'h-3.5 w-3.5 text-rose-400' }),
  },
  BLOCKED: {
    label: 'Blocked',
    border: 'border-zinc-800',
    bg: 'bg-zinc-900/40',
    text: 'text-zinc-500',
    glow: '',
    badge: 'border-zinc-700 bg-zinc-800 text-zinc-400',
    icon: React.createElement(Ban, { className: 'h-3.5 w-3.5 text-zinc-500' }),
  },
  PENDING: {
    label: 'Pending',
    border: 'border-zinc-800',
    bg: 'bg-zinc-900/40',
    text: 'text-zinc-500',
    glow: '',
    badge: 'border-zinc-800 bg-zinc-900 text-zinc-500',
    icon: React.createElement(CircleDashed, { className: 'h-3.5 w-3.5 text-zinc-500' }),
  },
  CANCELLED: {
    label: 'Cancelled',
    border: 'border-zinc-800',
    bg: 'bg-zinc-900/40',
    text: 'text-zinc-600',
    glow: '',
    badge: 'border-zinc-800 bg-zinc-900 text-zinc-500',
    icon: React.createElement(XCircle, { className: 'h-3.5 w-3.5 text-zinc-600' }),
  },
}

export function getTaskStatusStyle(status: TaskStatus): string {
  const config = TASK_STATUS_CONFIG[status]
  if (!config) {
    return 'border-zinc-800 bg-zinc-900 text-zinc-500'
  }
  return config.badge || `${config.border} ${config.bg} ${config.text}`
}
