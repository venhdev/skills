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

export interface AgentSession {
  session_id: string
  parent_session_id?: string
  role: 'ORCHESTRATOR' | 'SUBAGENT' | string
  name: string
  pid: number
  status: 'ACTIVE' | 'DISCONNECTED' | 'TERMINATED' | string
  claimed_tasks: string[]
  created_at: string
  last_beat_at: string
}

export interface ProjectDoc {
  slug: string
  repo_path: string
  git_branch: string
  title: string
  category: 'ARCHITECTURE' | 'SCHEMA' | 'ADR' | 'GUIDE' | 'MEMORY' | string
  content_markdown: string
  mermaid_diagrams?: string[]
  version: number
  updated_at: string
}

export interface ContextPacket {
  task: Task
  target_files: string[]
  skill_paths: string[]
  upstream_diffs: string[]
  input_attachments: Artifact[]
  verification_commands: string[]
  prompt_markdown: string
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

export type ArtifactType = 'DIFF' | 'IMAGE' | 'LOG' | 'TEST_RESULT' | 'MARKDOWN' | 'DOC' | 'FILE'

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

export type SpecTier = 1 | 2 | 3 | 4 | 5

export interface TechSpec {
  slug: string
  repo_path: string
  tier: SpecTier
  kind: 'CONTEXT' | 'FLOWCHART' | 'SEQUENCE' | 'STATE' | 'ERD' | 'CONTRACT' | 'ADR' | string
  title: string
  diagram_type: 'mermaid:flowchart' | 'mermaid:sequenceDiagram' | 'mermaid:erDiagram' | 'json:schema' | string
  diagram_code: string
  structured_payload: Record<string, any>
  notes: string
  bound_paths: string[]
  version: number
  updated_at: string
}

export interface CompactSpecPayload {
  slug: string
  kind: string
  title: string
  diagram_code: string
  payload: Record<string, any>
}

export interface PlanRevision {
  id: number
  plan_slug: string
  revision: string
  author: string
  summary: string
  created_at: string
}

export interface SnapshotFile {
  plan_slug: string
  revision: string
  file_path: string
  content: string
  byte_size: number
  sha256: string
  created_at: string
}

export interface FileDiff {
  file_path: string
  change_type: 'ADDED' | 'MODIFIED' | 'DELETED' | 'IDENTICAL'
  old_content?: string
  new_content?: string
}
