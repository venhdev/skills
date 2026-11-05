package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"zenfd/internal/domain"
)

// AgentRepository persists reusable worker definitions. Execution stays with an
// external runner; the daemon stores the definition, goal, and schedule.
type AgentRepository struct {
	db *DB
}

const agentSelectColumns = `
slug, name, persona, mode, goal, start_conditions, stop_conditions, schedule,
skills, model, runner, enabled, created_at, updated_at`

func scanAgent(s scanner) (*domain.Agent, error) {
	var a domain.Agent
	var persona, mode, startJSON, stopJSON, skillsJSON string
	if err := s.Scan(
		&a.Slug, &a.Name, &persona, &mode, &a.Goal, &startJSON, &stopJSON,
		&a.Schedule, &skillsJSON, &a.Model, &a.Runner, &a.Enabled,
		&a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}
	a.Persona = domain.AgentPersona(persona)
	a.Mode = domain.AgentMode(mode)
	a.StartConditions = unmarshalStrings(startJSON)
	a.StopConditions = unmarshalStrings(stopJSON)
	a.Skills = unmarshalStrings(skillsJSON)
	return &a, nil
}

func agentArgs(agent domain.Agent) ([]any, error) {
	startJSON, err := json.Marshal(agent.StartConditions)
	if err != nil {
		return nil, err
	}
	stopJSON, err := json.Marshal(agent.StopConditions)
	if err != nil {
		return nil, err
	}
	skillsJSON, err := json.Marshal(agent.Skills)
	if err != nil {
		return nil, err
	}
	return []any{
		agent.Slug, agent.Name, string(agent.Persona), string(agent.Mode), agent.Goal,
		string(startJSON), string(stopJSON), agent.Schedule, string(skillsJSON),
		agent.Model, agent.Runner, agent.Enabled, agent.CreatedAt, agent.UpdatedAt,
	}, nil
}

func (r *AgentRepository) CreateAgent(ctx context.Context, agent domain.Agent) error {
	now := time.Now().UTC()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now
	if agent.Persona == "" {
		agent.Persona = domain.PersonaCustom
	}
	if agent.Mode == "" {
		agent.Mode = domain.AgentModeHeadless
	}
	if err := agent.Validate(); err != nil {
		return err
	}
	args, err := agentArgs(agent)
	if err != nil {
		return fmt.Errorf("failed to encode agent %s: %w", agent.Slug, err)
	}

	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO agents (
				slug, name, persona, mode, goal, start_conditions, stop_conditions,
				schedule, skills, model, runner, enabled, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, args...)
		if err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("%w: agent %s", ErrAlreadyExists, agent.Slug)
			}
			return fmt.Errorf("failed to create agent %s: %w", agent.Slug, err)
		}
		return nil
	})
}

func (r *AgentRepository) GetAgent(ctx context.Context, slug string) (*domain.Agent, error) {
	row := r.db.QueryRow(ctx, "SELECT "+agentSelectColumns+" FROM agents WHERE slug = ?", slug)
	agent, err := scanAgent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: agent %s", ErrNotFound, slug)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query agent %s: %w", slug, err)
	}
	return agent, nil
}

func (r *AgentRepository) ListAgents(ctx context.Context, includeDisabled bool) ([]domain.Agent, error) {
	query := "SELECT " + agentSelectColumns + " FROM agents"
	if !includeDisabled {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY slug ASC"

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	agents := []domain.Agent{}
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, *agent)
	}
	return agents, rows.Err()
}

func (r *AgentRepository) UpdateAgent(ctx context.Context, slug string, patch domain.AgentPatch) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		agent, err := getAgentTx(ctx, tx, slug)
		if err != nil {
			return err
		}
		if patch.Name != nil {
			agent.Name = *patch.Name
		}
		if patch.Persona != nil {
			agent.Persona = *patch.Persona
		}
		if patch.Mode != nil {
			agent.Mode = *patch.Mode
		}
		if patch.Goal != nil {
			agent.Goal = *patch.Goal
		}
		if patch.StartConditions != nil {
			agent.StartConditions = *patch.StartConditions
		}
		if patch.StopConditions != nil {
			agent.StopConditions = *patch.StopConditions
		}
		if patch.Schedule != nil {
			agent.Schedule = *patch.Schedule
		}
		if patch.Skills != nil {
			agent.Skills = *patch.Skills
		}
		if patch.Model != nil {
			agent.Model = *patch.Model
		}
		if patch.Runner != nil {
			agent.Runner = *patch.Runner
		}
		if patch.Enabled != nil {
			agent.Enabled = *patch.Enabled
		}
		if err := agent.Validate(); err != nil {
			return err
		}
		agent.UpdatedAt = now
		args, err := agentArgs(*agent)
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `
			UPDATE agents SET
				name = ?, persona = ?, mode = ?, goal = ?, start_conditions = ?,
				stop_conditions = ?, schedule = ?, skills = ?, model = ?, runner = ?,
				enabled = ?, created_at = ?, updated_at = ?
			WHERE slug = ?`,
			append(args[1:], slug)...)
		if err != nil {
			return fmt.Errorf("failed to update agent %s: %w", slug, err)
		}
		rows, err := affectedOrErr(res, "update agent", slug)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: agent %s", ErrNotFound, slug)
		}
		return nil
	})
}

func getAgentTx(ctx context.Context, tx *sql.Tx, slug string) (*domain.Agent, error) {
	row := tx.QueryRowContext(ctx, "SELECT "+agentSelectColumns+" FROM agents WHERE slug = ?", slug)
	agent, err := scanAgent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: agent %s", ErrNotFound, slug)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query agent %s: %w", slug, err)
	}
	return agent, nil
}

func (r *AgentRepository) DeleteAgent(ctx context.Context, slug string) error {
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM agents WHERE slug = ?", slug)
		if err != nil {
			return fmt.Errorf("failed to delete agent %s: %w", slug, err)
		}
		rows, err := affectedOrErr(res, "delete agent", slug)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: agent %s", ErrNotFound, slug)
		}
		return nil
	})
}

// SessionRepository persists observed MAIN/SUBAGENT sessions and WORKER
// sessions registered by external runners.
type SessionRepository struct {
	db *DB
}

const sessionSelectColumns = `
id, COALESCE(agent_slug, ''), session_type, parent_id, external_ref, status,
started_at, last_seen_at, ended_at`

func scanSession(s scanner) (*domain.AgentSession, error) {
	var session domain.AgentSession
	var sessionType, status string
	var endedAt sql.NullTime
	if err := s.Scan(
		&session.ID, &session.AgentSlug, &sessionType, &session.ParentID,
		&session.ExternalRef, &status, &session.StartedAt, &session.LastSeenAt, &endedAt,
	); err != nil {
		return nil, err
	}
	session.SessionType = domain.SessionType(sessionType)
	session.Status = domain.SessionStatus(status)
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	return &session, nil
}

func (r *SessionRepository) CreateSession(ctx context.Context, session domain.AgentSession) error {
	now := time.Now().UTC()
	if session.StartedAt.IsZero() {
		session.StartedAt = now
	}
	if session.LastSeenAt.IsZero() {
		session.LastSeenAt = now
	}
	if session.Status == "" {
		session.Status = domain.SessionActive
	}
	if err := session.Validate(); err != nil {
		return err
	}

	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO agent_sessions (
				id, agent_slug, session_type, parent_id, external_ref, status,
				started_at, last_seen_at, ended_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			session.ID, nullableString(session.AgentSlug), string(session.SessionType),
			session.ParentID, session.ExternalRef, string(session.Status),
			session.StartedAt, session.LastSeenAt, session.EndedAt)
		if err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("%w: session %s", ErrAlreadyExists, session.ID)
			}
			return fmt.Errorf("failed to create session %s: %w", session.ID, err)
		}
		return nil
	})
}

func (r *SessionRepository) GetSession(ctx context.Context, id string) (*domain.AgentSession, error) {
	row := r.db.QueryRow(ctx, "SELECT "+sessionSelectColumns+" FROM agent_sessions WHERE id = ?", id)
	session, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: session %s", ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query session %s: %w", id, err)
	}
	return session, nil
}

func (r *SessionRepository) ListSessions(ctx context.Context, filter domain.SessionFilter) ([]domain.AgentSession, error) {
	query := "SELECT " + sessionSelectColumns + " FROM agent_sessions"
	var conditions []string
	var args []any

	if filter.Type != "" {
		conditions = append(conditions, "session_type = ?")
		args = append(args, string(filter.Type))
	}
	if filter.AgentSlug != "" {
		conditions = append(conditions, "agent_slug = ?")
		args = append(args, filter.AgentSlug)
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, string(filter.Status))
	}
	if filter.ActiveOnly {
		conditions = append(conditions, "status IN ('ACTIVE', 'IDLE')")
	}
	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions)
	}
	query += " ORDER BY last_seen_at DESC, id ASC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	sessions := []domain.AgentSession{}
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, *session)
	}
	return sessions, rows.Err()
}

func (r *SessionRepository) HeartbeatSession(ctx context.Context, id string, status domain.SessionStatus) error {
	if status != "" && !status.IsValid() {
		return fmt.Errorf("invalid session status: %s", status)
	}
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		var query string
		var args []any
		if status == "" {
			query = "UPDATE agent_sessions SET last_seen_at = ? WHERE id = ?"
			args = []any{now, id}
		} else {
			query = "UPDATE agent_sessions SET last_seen_at = ?, status = ? WHERE id = ?"
			args = []any{now, string(status), id}
		}
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to heartbeat session %s: %w", id, err)
		}
		rows, err := affectedOrErr(res, "heartbeat session", id)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: session %s", ErrNotFound, id)
		}
		return nil
	})
}

func (r *SessionRepository) CloseSession(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE agent_sessions
			SET status = 'TERMINATED', ended_at = ?, last_seen_at = ?
			WHERE id = ?`, now, now, id)
		if err != nil {
			return fmt.Errorf("failed to close session %s: %w", id, err)
		}
		rows, err := affectedOrErr(res, "close session", id)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: session %s", ErrNotFound, id)
		}
		return nil
	})
}
