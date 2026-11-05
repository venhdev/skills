package domain

import "errors"

// Sentinel errors shared by the storage implementations and the transports that
// map them onto HTTP/MCP responses.
var (
	ErrNotFound       = errors.New("entity not found")
	ErrAlreadyExists  = errors.New("entity already exists")
	ErrInvalidState   = errors.New("invalid state transition")
	ErrConflictStatus = errors.New("task status conflict during transition")
	ErrInvalidGraph   = errors.New("invalid task dependency graph")
)
