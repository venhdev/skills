package domain

import (
	"fmt"
	"time"
)

// NewRunID builds a unique run identifier for a task. The gateway globs
// run-<task>-*.log files, so every producer must keep this format.
func NewRunID(taskID string) string {
	return fmt.Sprintf("run-%s-%d", taskID, time.Now().UnixNano())
}
