package domain

import (
	"fmt"
	"strings"
)

// TransitionTaskRequest defines the input to a domain state machine transition.
type TransitionTaskRequest struct {
	Action       string      `json:"action"` // COMPLETE, FAIL, RESET, CANCEL, CHECKPOINT
	Reason       string      `json:"reason,omitempty"`
	Summary      string      `json:"summary,omitempty"`
	MergeOutcome string      `json:"merge_outcome,omitempty"` // MERGED, CONFLICT (for worktree merge gate)
	PhaseID      string      `json:"phase_id,omitempty"`      // for CHECKPOINT
	PhaseStatus  PhaseStatus `json:"phase_status,omitempty"`  // for CHECKPOINT (default COMPLETED)
	Notes        string      `json:"notes,omitempty"`
}

// TransitionTaskResult contains the outcome of an evaluated domain state transition.
type TransitionTaskResult struct {
	TargetStatus  TaskStatus
	AuditNote     string
	CurrentPhase  string
	Phases        []TaskPhase
	PhasesUpdated bool
}

// EvaluateTransition applies the core domain state machine rules to a task.
// This is the Single Source of Truth for lifecycle transitions, ensuring MCP,
// REST API, and CLI all uphold identical invariants (including the worktree merge
// gate and intra-task phase checkpoints).
func EvaluateTransition(task Task, req TransitionTaskRequest) (*TransitionTaskResult, error) {
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if action == "" {
		return nil, fmt.Errorf("transition action cannot be empty")
	}

	note := ""
	if req.Summary != "" {
		note = "Summary: " + req.Summary
	}
	if req.Reason != "" && action != "RESET" && action != "CANCEL" {
		if note != "" {
			note += "\n"
		}
		note += "Reason: " + req.Reason
	}
	if req.Notes != "" {
		if note != "" {
			note += "\n"
		}
		note += "Notes: " + req.Notes
	}

	result := &TransitionTaskResult{
		AuditNote:    note,
		CurrentPhase: task.CurrentPhase,
		Phases:       task.Phases,
	}

	switch action {
	case "COMPLETE":
		switch task.Status {
		case StatusRunning:
			if task.WorktreePath != "" {
				result.TargetStatus = StatusAwaitingMerge
			} else {
				result.TargetStatus = StatusCompleted
			}
			// If all phases were running/pending, mark all completed
			if len(task.Phases) > 0 {
				updatedPhases := make([]TaskPhase, len(task.Phases))
				copy(updatedPhases, task.Phases)
				for i := range updatedPhases {
					if updatedPhases[i].Status != PhaseCompleted {
						updatedPhases[i].Status = PhaseCompleted
					}
				}
				result.Phases = updatedPhases
				result.PhasesUpdated = true
			}
			return result, nil

		case StatusAwaitingMerge:
			switch strings.ToUpper(req.MergeOutcome) {
			case "MERGED":
				result.TargetStatus = StatusCompleted
				return result, nil
			case "CONFLICT":
				result.TargetStatus = StatusConflict
				return result, nil
			default:
				return nil, fmt.Errorf("merge_outcome MERGED or CONFLICT is required when task is AWAITING_MERGE")
			}

		case StatusConflict:
			if strings.ToUpper(req.MergeOutcome) != "MERGED" {
				return nil, fmt.Errorf("merge_outcome MERGED is required when task is CONFLICT")
			}
			result.TargetStatus = StatusCompleted
			return result, nil

		default:
			return nil, fmt.Errorf("cannot COMPLETE task in %s status", task.Status)
		}

	case "CHECKPOINT":
		if task.Status != StatusRunning {
			return nil, fmt.Errorf("cannot CHECKPOINT task in %s status (must be RUNNING)", task.Status)
		}
		result.TargetStatus = StatusRunning

		if len(task.Phases) == 0 {
			// No phases declared; checkpoint acts as an audit note update
			return result, nil
		}

		updatedPhases := make([]TaskPhase, len(task.Phases))
		copy(updatedPhases, task.Phases)

		targetPhaseID := req.PhaseID
		if targetPhaseID == "" {
			targetPhaseID = task.CurrentPhase
		}

		targetStatus := req.PhaseStatus
		if targetStatus == "" {
			targetStatus = PhaseCompleted
		}

		foundIdx := -1
		for i, p := range updatedPhases {
			if targetPhaseID != "" && p.ID == targetPhaseID {
				foundIdx = i
				break
			}
			if targetPhaseID == "" && p.Status == PhaseRunning {
				foundIdx = i
				break
			}
		}

		if foundIdx == -1 {
			// Default to first pending/running phase
			for i, p := range updatedPhases {
				if p.Status != PhaseCompleted {
					foundIdx = i
					break
				}
			}
		}

		if foundIdx >= 0 {
			updatedPhases[foundIdx].Status = targetStatus
			if req.Notes != "" {
				updatedPhases[foundIdx].Notes = req.Notes
			}

			// If completed and there is a next phase, advance CurrentPhase to next
			if targetStatus == PhaseCompleted && foundIdx+1 < len(updatedPhases) {
				result.CurrentPhase = updatedPhases[foundIdx+1].ID
				if updatedPhases[foundIdx+1].Status == PhasePending {
					updatedPhases[foundIdx+1].Status = PhaseRunning
				}
			} else if targetStatus == PhaseCompleted && foundIdx+1 == len(updatedPhases) {
				result.CurrentPhase = updatedPhases[foundIdx].ID
			} else {
				result.CurrentPhase = updatedPhases[foundIdx].ID
			}

			result.Phases = updatedPhases
			result.PhasesUpdated = true
		}

		return result, nil

	case "FAIL":
		if task.Status != StatusRunning {
			return nil, fmt.Errorf("cannot FAIL task in %s status", task.Status)
		}
		result.TargetStatus = StatusFailed
		if len(task.Phases) > 0 && task.CurrentPhase != "" {
			updatedPhases := make([]TaskPhase, len(task.Phases))
			copy(updatedPhases, task.Phases)
			for i := range updatedPhases {
				if updatedPhases[i].ID == task.CurrentPhase {
					updatedPhases[i].Status = PhaseFailed
					break
				}
			}
			result.Phases = updatedPhases
			result.PhasesUpdated = true
		}
		return result, nil

	case "RESET":
		switch task.Status {
		case StatusRunning, StatusFailed, StatusCancelled, StatusConflict:
			result.TargetStatus = StatusReady
			if len(task.Phases) > 0 {
				updatedPhases := make([]TaskPhase, len(task.Phases))
				copy(updatedPhases, task.Phases)
				for i := range updatedPhases {
					updatedPhases[i].Status = PhasePending
				}
				result.Phases = updatedPhases
				result.CurrentPhase = updatedPhases[0].ID
				result.PhasesUpdated = true
			}
			return result, nil
		default:
			return nil, fmt.Errorf("cannot RESET task in %s status", task.Status)
		}

	case "CANCEL":
		switch task.Status {
		case StatusPending, StatusReady, StatusRunning, StatusBlocked, StatusAwaitingReview, StatusAwaitingMerge:
			result.TargetStatus = StatusCancelled
			return result, nil
		default:
			return nil, fmt.Errorf("cannot CANCEL task in %s status", task.Status)
		}

	default:
		return nil, fmt.Errorf("unsupported transition action: %s", req.Action)
	}
}
