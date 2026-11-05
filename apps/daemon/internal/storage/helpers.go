package storage

import (
	"database/sql"
	"fmt"
)

// affectedOrErr returns the affected row count and preserves driver errors
// instead of collapsing them into a zero count.
func affectedOrErr(res sql.Result, op string, id string) (int64, error) {
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s for %s: %w", op, id, err)
	}
	return rows, nil
}
