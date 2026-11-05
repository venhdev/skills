package storage

import (
	"context"
	"database/sql"
	"fmt"

	"zenfd/internal/domain"
)

// insertRunTx writes a run row inside an existing transaction. It is the only
// run write path: the claim transition registers its attempt through this
// helper so task status and run registration commit atomically.
func insertRunTx(ctx context.Context, tx *sql.Tx, run domain.Run) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO runs (id, task_id, pid, pgid, status, log_path, exit_code, started_at, finished_at, last_beat_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.TaskID, run.PID, run.PGID, string(run.Status), run.LogPath,
		run.ExitCode, run.StartedAt, run.FinishedAt, run.LastBeatAt); err != nil {
		return fmt.Errorf("failed to register run %s for task %s: %w", run.ID, run.TaskID, err)
	}
	return nil
}
