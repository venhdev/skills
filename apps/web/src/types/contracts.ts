export type TaskStatus =
  | 'PENDING'
  | 'READY'
  | 'RUNNING'
  | 'AWAITING_REVIEW'
  | 'SUSPENDED'
  | 'AWAITING_MERGE'
  | 'CONFLICT'
  | 'COMPLETED'
  | 'FAILED'
  | 'BLOCKED'
  | 'CANCELLED'

export type TaskPriority = 'P0' | 'P1' | 'P2' | 'P3'
export type TaskComplexity = 'LOW' | 'MEDIUM' | 'HIGH' | 'SPIKE'
export type TaskRisk = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'

export interface Plan {
  slug: string
  title: string
  status: string
  created_at: string
  updated_at: string
}

export interface Task {
  id: string
  plan_id: string
  title: string
  description?: string
  dependencies: string[]
  priority?: TaskPriority
  complexity?: TaskComplexity
  risk_level?: TaskRisk
  required_skills?: string[]
  required_tools?: string[]
  target_files?: string[]
  acceptance_criteria?: string[]
  verification_commands?: string[]
  harness: string
  skill_command: string
  worktree_path?: string
  status: TaskStatus
  claimed_by?: string
  orchestrator_id?: string
  worker_role?: string
  pid?: number
  orchestrator_pid?: number
  claimed_at?: string
  notes?: string
  review_reason?: string
  review_details?: string
  review_options?: string[]
  review_response?: string
  resolved_by?: string
  conflict_files?: string[]
  created_at: string
  updated_at: string
}

export interface ContextPacket {
  task: Task
  target_files: string[]
  skill_paths: string[]
  input_attachments: Artifact[]
  verification_commands: string[]
  prompt_markdown: string
}

export interface Run {
  id: string
  task_id: string
  pid: number
  pgid: number
  status: TaskStatus
  log_path: string
  exit_code: number
  started_at: string
  finished_at?: string
  last_beat_at: string
}

export type ArtifactType = 'IMAGE' | 'LOG' | 'TEST_RESULT' | 'MARKDOWN' | 'DOC' | 'FILE'

export interface Artifact {
  id: string
  run_id?: string
  task_id: string
  role: 'INPUT' | 'OUTPUT'
  name: string
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
