package storage

import (
	"zenfd/internal/contextpacket"
	"zenfd/internal/domain"
)

// The storage package re-exports the domain sentinels so existing call sites and
// tests keep using storage.Err* while transports can classify by domain.Err*.
var (
	ErrNotFound       = domain.ErrNotFound
	ErrAlreadyExists  = domain.ErrAlreadyExists
	ErrInvalidState   = domain.ErrInvalidState
	ErrConflictStatus = domain.ErrConflictStatus
	ErrInvalidGraph   = domain.ErrInvalidGraph
)

// Store is the storage composition root. Each embedded repository owns one
// aggregate; Store only wires them and promotes their methods for existing
// callers.
type Store struct {
	*PlanRepository
	*TaskRepository
	*TaskLifecycle
	*ArtifactRepository
	*EventRepository
	*StateFingerprinter
}

// NewStore creates a Store backed by the SQLite WAL DB.
func NewStore(db *DB) *Store {
	plans := &PlanRepository{db: db}
	tasks := &TaskRepository{db: db}
	artifacts := &ArtifactRepository{db: db}
	builder := contextpacket.New(artifacts, nil)

	return &Store{
		PlanRepository:     plans,
		TaskRepository:     tasks,
		TaskLifecycle:      &TaskLifecycle{db: db, tasks: tasks, packets: builder},
		ArtifactRepository: artifacts,
		EventRepository:    &EventRepository{db: db},
		StateFingerprinter: &StateFingerprinter{db: db},
	}
}

var (
	_ domain.PlanRepository     = (*PlanRepository)(nil)
	_ domain.TaskRepository     = (*TaskRepository)(nil)
	_ domain.TaskLifecycle      = (*TaskLifecycle)(nil)
	_ domain.ArtifactRepository = (*ArtifactRepository)(nil)
	_ domain.ArtifactReader     = (*ArtifactRepository)(nil)
	_ domain.EventRepository    = (*EventRepository)(nil)
	_ domain.StateFingerprinter = (*StateFingerprinter)(nil)
)
