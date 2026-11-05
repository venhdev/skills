export type ProjectStatus = 'ACTIVE' | 'ARCHIVED'
export type PlanStatus = 'ACTIVE' | 'COMPLETED' | 'DRIFTED' | 'ARCHIVED'

export type TaskStatus =
  | 'PENDING'
  | 'READY'
  | 'RUNNING'
  | 'AWAITING_REVIEW'
  | 'AWAITING_MERGE'
  | 'CONFLICT'
  | 'COMPLETED'
  | 'FAILED'
  | 'BLOCKED'
  | 'CANCELLED'

export type TaskPriority = 'P0' | 'P1' | 'P2' | 'P3'
export type TaskComplexity = 'LOW' | 'MEDIUM' | 'HIGH' | 'SPIKE'
export type TaskRisk = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'

export interface Project {
  slug: string
  name: string
  root_path?: string
  description?: string
  status: ProjectStatus
  created_at: string
  updated_at: string
}

export interface ProjectInput {
  slug: string
  name: string
  root_path?: string
  description?: string
}

export interface Plan {
  slug: string
  project_slug: string
  title: string
  status: PlanStatus
  archived_at?: string
  created_at: string
  updated_at: string
}

export interface Task {
  id: string
  project_slug?: string
  plan_slug?: string
  title: string
  description?: string
  dependencies: string[]
  priority?: TaskPriority
  complexity?: TaskComplexity
  risk_level?: TaskRisk
  agent_slug?: string
  required_skills?: string[]
  required_tools?: string[]
  required_docs?: string[]
  target_files?: string[]
  acceptance_criteria?: string[]
  verification_commands?: string[]
  harness: string
  worktree_path?: string
  status: TaskStatus
  claimed_by?: string
  claimed_at?: string
  orchestrator_id?: string
  worker_role?: string
  claim_session_id?: string
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

export type ArtifactType = 'IMAGE' | 'MARKDOWN' | 'DOC' | 'FILE'

export interface Artifact {
  id: string
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

export type DocumentType =
  | 'ARCHITECTURE'
  | 'ADR'
  | 'SPEC'
  | 'RULE'
  | 'GUIDE'
  | 'REFERENCE'
  | 'MEMORY'
  | 'OTHER'

export type DocumentScope = 'GLOBAL' | 'PROJECT' | 'PLAN' | 'TASK'
export type DocumentStatus = 'DRAFT' | 'ACTIVE' | 'SUPERSEDED' | 'ARCHIVED'

export interface Document {
  id: string
  project_slug?: string
  key: string
  slug: string
  type: DocumentType
  scope: DocumentScope
  scope_ref?: string
  title: string
  path: string
  content_hash: string
  byte_size: number
  mime_type: string
  status: DocumentStatus
  summary?: string
  content?: string
  tags?: string[]
  version: number
  created_at: string
  updated_at: string
}

export interface DocumentInput {
  project_slug?: string
  key: string
  slug?: string
  type: DocumentType
  scope: DocumentScope
  scope_ref?: string
  title: string
  path: string
  summary?: string
  content?: string
  status?: DocumentStatus
  tags?: string[]
}

export interface DocumentFilter {
  projectSlug?: string
  includeGlobal?: boolean
  type?: DocumentType | ''
  scope?: DocumentScope | ''
  status?: DocumentStatus | ''
  tag?: string
  limit?: number
}

export interface DocumentSearchResult {
  document: Document
  snippet: string
}

export type AgentPersona = 'EXPLORER' | 'CODER' | 'REVIEWER' | 'CUSTOM'
export type AgentMode = 'HEADLESS' | 'INTERACTIVE'

export interface Agent {
  slug: string
  name: string
  persona: AgentPersona
  mode: AgentMode
  goal?: string
  start_conditions?: string[]
  stop_conditions?: string[]
  schedule?: string
  skills?: string[]
  model?: string
  runner?: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface AgentInput {
  slug: string
  name: string
  persona: AgentPersona
  mode: AgentMode
  goal?: string
  start_conditions?: string[]
  stop_conditions?: string[]
  schedule?: string
  skills?: string[]
  model?: string
  runner?: string
  enabled: boolean
}

export type SessionType = 'MAIN' | 'SUBAGENT' | 'WORKER'
export type SessionStatus = 'ACTIVE' | 'IDLE' | 'DISCONNECTED' | 'TERMINATED'

export interface AgentSession {
  id: string
  agent_slug?: string
  session_type: SessionType
  parent_id?: string
  external_ref?: string
  status: SessionStatus
  started_at: string
  last_seen_at: string
  ended_at?: string
}

export interface SearchResult {
  entity_type: string
  entity_id: string
  title: string
  snippet?: string
  project_slug?: string
}

export interface EventMessage {
  id: number
  project_slug?: string
  plan_slug?: string
  task_id?: string
  entity_type: string
  entity_id: string
  type: string
  actor?: string
  payload: string
  timestamp: string
}
