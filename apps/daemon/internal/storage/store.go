package storage

import (
	"errors"

	"zenfd/internal/contextpacket"
	"zenfd/internal/domain"
	"zenfd/internal/domain/prompt"
)

var (
	ErrNotFound       = errors.New("entity not found")
	ErrInvalidState   = errors.New("invalid state transition")
	ErrConflictStatus = errors.New("task status conflict during transition")
)

// Store is the storage composition root. Each embedded repository owns one
// aggregate; Store only wires them and promotes their methods for existing
// callers.
type Store struct {
	*PlanRepository
	*TaskRepository
	*TaskLifecycle
	*ArtifactRepository
	*EventLog
	*StateFingerprinter
}

// Option configures NewStore.
type Option func(*storeConfig)

type storeConfig struct {
	compiler prompt.Compiler
}

// WithPromptCompiler overrides the prompt compiler used for context packets.
func WithPromptCompiler(c prompt.Compiler) Option {
	return func(cfg *storeConfig) { cfg.compiler = c }
}

// NewStore creates a Store backed by the SQLite WAL DB.
func NewStore(db *DB, opts ...Option) *Store {
	cfg := storeConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	plans := &PlanRepository{db: db}
	tasks := &TaskRepository{db: db}
	artifacts := &ArtifactRepository{db: db}
	builder := contextpacket.New(artifacts, cfg.compiler)

	return &Store{
		PlanRepository:     plans,
		TaskRepository:     tasks,
		TaskLifecycle:      &TaskLifecycle{db: db, tasks: tasks, packets: builder},
		ArtifactRepository: artifacts,
		EventLog:           &EventLog{db: db},
		StateFingerprinter: &StateFingerprinter{db: db},
	}
}

var (
	_ domain.PlanRepository     = (*PlanRepository)(nil)
	_ domain.TaskRepository     = (*TaskRepository)(nil)
	_ domain.TaskLifecycle      = (*TaskLifecycle)(nil)
	_ domain.ArtifactRepository = (*ArtifactRepository)(nil)
	_ domain.ArtifactReader     = (*ArtifactRepository)(nil)
	_ domain.EventLog           = (*EventLog)(nil)
	_ domain.StateFingerprinter = (*StateFingerprinter)(nil)
)
