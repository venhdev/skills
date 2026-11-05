-- ZenForge v0.2.0 schema. This is a clean-break schema: v0.1.x tracking
-- tables are dropped by the migration runner before this file is applied.

CREATE TABLE projects (
    slug TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX idx_projects_status ON projects (status);

CREATE TABLE plans (
    slug TEXT PRIMARY KEY,
    project_slug TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    archived_at DATETIME,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (project_slug) REFERENCES projects (slug) ON DELETE CASCADE
);
CREATE INDEX idx_plans_project ON plans (project_slug, archived_at);

CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    project_slug TEXT,
    plan_slug TEXT,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    dependencies TEXT NOT NULL DEFAULT '[]',
    priority TEXT NOT NULL DEFAULT 'P2',
    complexity TEXT NOT NULL DEFAULT 'MEDIUM',
    risk_level TEXT NOT NULL DEFAULT 'LOW',
    agent_slug TEXT NOT NULL DEFAULT '',
    required_skills TEXT NOT NULL DEFAULT '[]',
    required_tools TEXT NOT NULL DEFAULT '[]',
    required_docs TEXT NOT NULL DEFAULT '[]',
    target_files TEXT NOT NULL DEFAULT '[]',
    acceptance_criteria TEXT NOT NULL DEFAULT '[]',
    verification_commands TEXT NOT NULL DEFAULT '[]',
    harness TEXT NOT NULL DEFAULT 'orchestrator',
    worktree_path TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    claimed_by TEXT NOT NULL DEFAULT '',
    claimed_at DATETIME,
    orchestrator_id TEXT NOT NULL DEFAULT '',
    worker_role TEXT NOT NULL DEFAULT '',
    claim_session_id TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    review_reason TEXT NOT NULL DEFAULT '',
    review_details TEXT NOT NULL DEFAULT '',
    review_options TEXT NOT NULL DEFAULT '[]',
    review_response TEXT NOT NULL DEFAULT '',
    resolved_by TEXT NOT NULL DEFAULT '',
    conflict_files TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    CHECK (plan_slug IS NULL OR project_slug IS NOT NULL),
    FOREIGN KEY (project_slug) REFERENCES projects (slug) ON DELETE CASCADE,
    FOREIGN KEY (plan_slug) REFERENCES plans (slug) ON DELETE CASCADE
);
CREATE INDEX idx_tasks_project ON tasks (project_slug);
CREATE INDEX idx_tasks_plan ON tasks (plan_slug);
CREATE INDEX idx_tasks_status ON tasks (status);
CREATE INDEX idx_tasks_updated ON tasks (updated_at);
CREATE INDEX idx_tasks_project_status ON tasks (project_slug, status);
CREATE INDEX idx_tasks_plan_status ON tasks (plan_slug, status);
CREATE INDEX idx_tasks_agent ON tasks (agent_slug);

CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    project_slug TEXT,
    key TEXT NOT NULL,
    slug TEXT NOT NULL,
    type TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'PROJECT',
    scope_ref TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    path TEXT NOT NULL,
    content_hash TEXT NOT NULL DEFAULT '',
    byte_size INTEGER NOT NULL DEFAULT 0,
    mime_type TEXT NOT NULL DEFAULT 'text/markdown',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    summary TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (project_slug) REFERENCES projects (slug) ON DELETE CASCADE
);
CREATE UNIQUE INDEX ux_documents_project_key ON documents (IFNULL(project_slug, ''), key);
CREATE UNIQUE INDEX ux_documents_project_slug ON documents (IFNULL(project_slug, ''), slug);
CREATE INDEX idx_documents_project ON documents (project_slug);
CREATE INDEX idx_documents_type ON documents (type);
CREATE INDEX idx_documents_scope ON documents (scope, scope_ref);
CREATE INDEX idx_documents_status ON documents (status);

CREATE TABLE document_tags (
    document_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (document_id, tag),
    FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE
);
CREATE INDEX idx_document_tags_tag ON document_tags (tag);

CREATE TABLE document_links (
    from_id TEXT NOT NULL,
    to_type TEXT NOT NULL,
    to_id TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (from_id, to_type, to_id),
    FOREIGN KEY (from_id) REFERENCES documents (id) ON DELETE CASCADE
);
CREATE INDEX idx_document_links_target ON document_links (to_type, to_id);

CREATE VIRTUAL TABLE documents_fts USING fts5(
    document_id UNINDEXED,
    key,
    title,
    summary,
    content,
    tokenize = 'porter'
);

CREATE TABLE agents (
    slug TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    persona TEXT NOT NULL DEFAULT 'CUSTOM',
    mode TEXT NOT NULL DEFAULT 'HEADLESS',
    goal TEXT NOT NULL DEFAULT '',
    start_conditions TEXT NOT NULL DEFAULT '[]',
    stop_conditions TEXT NOT NULL DEFAULT '[]',
    schedule TEXT NOT NULL DEFAULT '',
    skills TEXT NOT NULL DEFAULT '[]',
    model TEXT NOT NULL DEFAULT '',
    runner TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX idx_agents_enabled ON agents (enabled);

CREATE TABLE agent_sessions (
    id TEXT PRIMARY KEY,
    agent_slug TEXT,
    session_type TEXT NOT NULL,
    parent_id TEXT NOT NULL DEFAULT '',
    external_ref TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    started_at DATETIME NOT NULL,
    last_seen_at DATETIME NOT NULL,
    ended_at DATETIME,
    FOREIGN KEY (agent_slug) REFERENCES agents (slug) ON DELETE SET NULL
);
CREATE INDEX idx_sessions_type_status ON agent_sessions (session_type, status);
CREATE INDEX idx_sessions_agent ON agent_sessions (agent_slug);
CREATE INDEX idx_sessions_seen ON agent_sessions (last_seen_at);

CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'OUTPUT',
    name TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL,
    file_path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    byte_size INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
);
CREATE INDEX idx_artifacts_task ON artifacts (task_id);
CREATE INDEX idx_artifacts_role ON artifacts (role);

CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_slug TEXT,
    plan_slug TEXT,
    task_id TEXT,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    type TEXT NOT NULL,
    actor TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL,
    timestamp DATETIME NOT NULL
);
CREATE INDEX idx_events_entity ON events (entity_type, entity_id);
CREATE INDEX idx_events_project ON events (project_slug);
CREATE INDEX idx_events_plan ON events (plan_slug);
CREATE INDEX idx_events_task ON events (task_id);
CREATE INDEX idx_events_timestamp ON events (timestamp);

-- Document FTS synchronization -------------------------------------------------

CREATE TRIGGER trg_documents_fts_insert AFTER INSERT ON documents BEGIN
    INSERT INTO documents_fts (document_id, key, title, summary, content)
    VALUES (NEW.id, NEW.key, NEW.title, NEW.summary, NEW.content);
END;

CREATE TRIGGER trg_documents_fts_update AFTER UPDATE ON documents BEGIN
    DELETE FROM documents_fts WHERE document_id = OLD.id;
    INSERT INTO documents_fts (document_id, key, title, summary, content)
    VALUES (NEW.id, NEW.key, NEW.title, NEW.summary, NEW.content);
END;

CREATE TRIGGER trg_documents_fts_delete AFTER DELETE ON documents BEGIN
    DELETE FROM documents_fts WHERE document_id = OLD.id;
END;

-- Activity log ----------------------------------------------------------------

CREATE TRIGGER trg_audit_project_insert AFTER INSERT ON projects BEGIN
    INSERT INTO events (project_slug, entity_type, entity_id, type, payload, timestamp)
    VALUES (NEW.slug, 'project', NEW.slug, 'project.created',
        json_object('slug', NEW.slug, 'name', NEW.name, 'status', NEW.status), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_project_update AFTER UPDATE ON projects BEGIN
    INSERT INTO events (project_slug, entity_type, entity_id, type, payload, timestamp)
    VALUES (NEW.slug, 'project', NEW.slug, 'project.updated',
        json_object('slug', NEW.slug, 'name', NEW.name, 'status', NEW.status,
            'kind', CASE
                WHEN OLD.status <> NEW.status THEN 'status'
                WHEN OLD.name <> NEW.name THEN 'name'
                WHEN OLD.root_path <> NEW.root_path THEN 'root_path'
                ELSE 'metadata' END), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_project_delete AFTER DELETE ON projects BEGIN
    INSERT INTO events (project_slug, entity_type, entity_id, type, payload, timestamp)
    VALUES (OLD.slug, 'project', OLD.slug, 'project.deleted',
        json_object('slug', OLD.slug, 'name', OLD.name), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_plan_insert AFTER INSERT ON plans BEGIN
    INSERT INTO events (project_slug, plan_slug, entity_type, entity_id, type, payload, timestamp)
    VALUES (NEW.project_slug, NEW.slug, 'plan', NEW.slug, 'plan.created',
        json_object('slug', NEW.slug, 'project_slug', NEW.project_slug, 'title', NEW.title, 'status', NEW.status),
        CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_plan_update AFTER UPDATE ON plans BEGIN
    INSERT INTO events (project_slug, plan_slug, entity_type, entity_id, type, payload, timestamp)
    VALUES (NEW.project_slug, NEW.slug, 'plan', NEW.slug, 'plan.updated',
        json_object('slug', NEW.slug, 'project_slug', NEW.project_slug,
            'kind', CASE
                WHEN OLD.status <> NEW.status THEN 'status'
                WHEN OLD.archived_at IS NULL AND NEW.archived_at IS NOT NULL THEN 'archived'
                WHEN OLD.archived_at IS NOT NULL AND NEW.archived_at IS NULL THEN 'unarchived'
                WHEN OLD.title <> NEW.title THEN 'title'
                ELSE 'metadata' END,
            'from', OLD.status, 'to', NEW.status), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_plan_delete AFTER DELETE ON plans BEGIN
    INSERT INTO events (project_slug, plan_slug, entity_type, entity_id, type, payload, timestamp)
    VALUES (OLD.project_slug, OLD.slug, 'plan', OLD.slug, 'plan.deleted',
        json_object('slug', OLD.slug, 'project_slug', OLD.project_slug, 'title', OLD.title), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_task_insert AFTER INSERT ON tasks BEGIN
    INSERT INTO events (project_slug, plan_slug, task_id, entity_type, entity_id, type, payload, timestamp)
    VALUES (NEW.project_slug, NEW.plan_slug, NEW.id, 'task', NEW.id, 'task.created',
        json_object('task_id', NEW.id, 'project_slug', NEW.project_slug, 'plan_slug', NEW.plan_slug,
            'title', NEW.title, 'status', NEW.status), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_task_update AFTER UPDATE ON tasks BEGIN
    INSERT INTO events (project_slug, plan_slug, task_id, entity_type, entity_id, type, payload, timestamp)
    VALUES (NEW.project_slug, NEW.plan_slug, NEW.id, 'task', NEW.id, 'task.updated',
        json_object('task_id', NEW.id, 'project_slug', NEW.project_slug, 'plan_slug', NEW.plan_slug,
            'kind', CASE
                WHEN OLD.status <> NEW.status THEN 'status'
                WHEN OLD.notes <> NEW.notes THEN 'notes'
                WHEN OLD.dependencies <> NEW.dependencies THEN 'dependencies'
                WHEN OLD.claimed_by <> NEW.claimed_by THEN 'claim'
                ELSE 'metadata' END,
            'from', OLD.status, 'to', NEW.status, 'claimed_by', NEW.claimed_by), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_task_delete AFTER DELETE ON tasks BEGIN
    INSERT INTO events (project_slug, plan_slug, task_id, entity_type, entity_id, type, payload, timestamp)
    VALUES (OLD.project_slug, OLD.plan_slug, OLD.id, 'task', OLD.id, 'task.deleted',
        json_object('task_id', OLD.id, 'project_slug', OLD.project_slug, 'plan_slug', OLD.plan_slug, 'title', OLD.title),
        CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_document_insert AFTER INSERT ON documents BEGIN
    INSERT INTO events (project_slug, entity_type, entity_id, type, payload, timestamp)
    VALUES (NEW.project_slug, 'document', NEW.id, 'document.created',
        json_object('id', NEW.id, 'project_slug', NEW.project_slug, 'key', NEW.key, 'slug', NEW.slug,
            'type', NEW.type, 'scope', NEW.scope, 'title', NEW.title, 'path', NEW.path), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_document_update AFTER UPDATE ON documents BEGIN
    INSERT INTO events (project_slug, entity_type, entity_id, type, payload, timestamp)
    VALUES (NEW.project_slug, 'document', NEW.id, 'document.updated',
        json_object('id', NEW.id, 'project_slug', NEW.project_slug, 'key', NEW.key, 'slug', NEW.slug,
            'type', NEW.type, 'scope', NEW.scope, 'title', NEW.title,
            'kind', CASE
                WHEN OLD.status <> NEW.status THEN 'status'
                WHEN OLD.path <> NEW.path THEN 'path'
                WHEN OLD.content_hash <> NEW.content_hash THEN 'content'
                WHEN OLD.title <> NEW.title THEN 'title'
                ELSE 'metadata' END,
            'version', NEW.version), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_document_delete AFTER DELETE ON documents BEGIN
    INSERT INTO events (project_slug, entity_type, entity_id, type, payload, timestamp)
    VALUES (OLD.project_slug, 'document', OLD.id, 'document.deleted',
        json_object('id', OLD.id, 'project_slug', OLD.project_slug, 'key', OLD.key, 'slug', OLD.slug, 'title', OLD.title),
        CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_document_tag_insert AFTER INSERT ON document_tags BEGIN
    INSERT INTO events (entity_type, entity_id, type, payload, timestamp)
    VALUES ('document', NEW.document_id, 'document.tagged',
        json_object('document_id', NEW.document_id, 'tag', NEW.tag), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_document_tag_delete AFTER DELETE ON document_tags BEGIN
    INSERT INTO events (entity_type, entity_id, type, payload, timestamp)
    VALUES ('document', OLD.document_id, 'document.untagged',
        json_object('document_id', OLD.document_id, 'tag', OLD.tag), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_agent_insert AFTER INSERT ON agents BEGIN
    INSERT INTO events (entity_type, entity_id, type, payload, timestamp)
    VALUES ('agent', NEW.slug, 'agent.created',
        json_object('slug', NEW.slug, 'name', NEW.name, 'persona', NEW.persona, 'mode', NEW.mode, 'enabled', NEW.enabled),
        CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_agent_update AFTER UPDATE ON agents BEGIN
    INSERT INTO events (entity_type, entity_id, type, payload, timestamp)
    VALUES ('agent', NEW.slug, 'agent.updated',
        json_object('slug', NEW.slug, 'name', NEW.name, 'persona', NEW.persona, 'mode', NEW.mode, 'enabled', NEW.enabled),
        CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_agent_delete AFTER DELETE ON agents BEGIN
    INSERT INTO events (entity_type, entity_id, type, payload, timestamp)
    VALUES ('agent', OLD.slug, 'agent.deleted',
        json_object('slug', OLD.slug, 'name', OLD.name), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_session_insert AFTER INSERT ON agent_sessions BEGIN
    INSERT INTO events (entity_type, entity_id, type, payload, timestamp)
    VALUES ('session', NEW.id, 'session.registered',
        json_object('id', NEW.id, 'agent_slug', NEW.agent_slug, 'session_type', NEW.session_type,
            'parent_id', NEW.parent_id, 'status', NEW.status), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_session_update AFTER UPDATE ON agent_sessions BEGIN
    INSERT INTO events (entity_type, entity_id, type, payload, timestamp)
    VALUES ('session', NEW.id, 'session.updated',
        json_object('id', NEW.id, 'agent_slug', NEW.agent_slug, 'session_type', NEW.session_type,
            'status', NEW.status), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_session_delete AFTER DELETE ON agent_sessions BEGIN
    INSERT INTO events (entity_type, entity_id, type, payload, timestamp)
    VALUES ('session', OLD.id, 'session.deleted',
        json_object('id', OLD.id, 'agent_slug', OLD.agent_slug, 'session_type', OLD.session_type), CURRENT_TIMESTAMP);
END;

CREATE TRIGGER trg_audit_artifact_insert AFTER INSERT ON artifacts BEGIN
    INSERT INTO events (task_id, entity_type, entity_id, type, payload, timestamp)
    VALUES (NEW.task_id, 'artifact', NEW.id, 'artifact.added',
        json_object('artifact_id', NEW.id, 'task_id', NEW.task_id, 'role', NEW.role,
            'name', NEW.name, 'type', NEW.type, 'mime_type', NEW.mime_type), CURRENT_TIMESTAMP);
END;
