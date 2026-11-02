export type TaskStatus =
  | 'PENDING'
  | 'READY'
  | 'RUNNING'
  | 'COMPLETED'
  | 'FAILED'
  | 'BLOCKED'
  | 'CANCELLED'

export interface Plan {
  slug: string
  title: string
  repo_path: string
  status: string
  created_at: string
  updated_at: string
}

export interface Task {
  id: string
  plan_id: string
  repo_path: string
  title: string
  dependencies: string[]
  harness: string
  skill_command: string
  worktree_path: string
  status: TaskStatus
  created_at: string
  updated_at: string
}

export interface Run {
  id: string
  task_id: string
  repo_path: string
  pid: number
  pgid: number
  status: TaskStatus
  log_path: string
  exit_code: number
  started_at: string
  finished_at?: string
  last_beat_at: string
}

export type ArtifactType = 'DIFF' | 'IMAGE' | 'LOG' | 'TEST_RESULT' | 'MARKDOWN'

export interface Artifact {
  id: string
  run_id: string
  task_id: string
  type: ArtifactType
  file_path: string
  mime_type: string
  byte_size: number
  sha256: string
  created_at: string
}

export interface EventMessage {
  id?: number
  run_id?: string
  task_id?: string
  type: string
  payload: string
  timestamp: string
}

export interface WindowSize {
  rows: number
  cols: number
}
