package harness

import (
	"zenfd/internal/domain"
)

// OrchestratorAdapter represents tasks delegated to external AI Orchestrators or human review.
// It instructs the daemon to avoid spawning OS processes and instead broadcast state to external agents.
type OrchestratorAdapter struct{}

// NewOrchestratorAdapter constructs an OrchestratorAdapter.
func NewOrchestratorAdapter() *OrchestratorAdapter {
	return &OrchestratorAdapter{}
}

func (o *OrchestratorAdapter) Name() string {
	return "orchestrator"
}

// IsAvailable returns false for local OS process spawning because this task is executed externally.
func (o *OrchestratorAdapter) IsAvailable() bool {
	return false
}

func (o *OrchestratorAdapter) BuildExecution(task domain.Task) (ExecutionSpec, error) {
	return ExecutionSpec{
		Binary:     "",
		Args:       nil,
		IsExternal: true,
	}, nil
}
