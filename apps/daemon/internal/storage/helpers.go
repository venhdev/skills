package storage

import (
	"database/sql"
	"fmt"
	"strings"
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

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
// The driver does not expose a stable typed error across all builds, so the
// message check is the portable option.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "constraint failed: unique")
}

// nullableString converts an empty string into SQL NULL. Setting project_slug
// and plan_slug to NULL (rather than ”) is what makes the orphan/project task
// checks and the expression indexes work.
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// joinConditions concatenates SQL predicates with AND.
func joinConditions(conditions []string) string {
	return strings.Join(conditions, " AND ")
}
