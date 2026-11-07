package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"zenf/internal/domain"
)

// APIError represents the canonical error payload following RFC-7807 semantics.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse wraps APIError in an outer error envelope.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

func writeJSONError(w http.ResponseWriter, code string, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: msg,
		},
	})
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusUnprocessableEntity:
		return "UNPROCESSABLE_ENTITY"
	default:
		return "INTERNAL_ERROR"
	}
}

func writeHttpError(w http.ResponseWriter, msg string, status int) {
	writeJSONError(w, codeForStatus(status), status, msg)
}

// writeStorageError maps domain sentinels onto HTTP status codes so clients can
// distinguish a conflict from a server failure.
func writeStorageError(w http.ResponseWriter, op string, err error) {
	msg := fmt.Sprintf("%s: %v", op, err)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSONError(w, "RESOURCE_NOT_FOUND", http.StatusNotFound, msg)
	case errors.Is(err, domain.ErrAlreadyExists),
		errors.Is(err, domain.ErrConflictStatus),
		errors.Is(err, domain.ErrInvalidState):
		writeJSONError(w, "STATE_CONFLICT", http.StatusConflict, msg)
	case errors.Is(err, domain.ErrInvalidGraph):
		writeJSONError(w, "INVALID_GRAPH", http.StatusUnprocessableEntity, msg)
	default:
		writeJSONError(w, "STORAGE_ERROR", http.StatusInternalServerError, msg)
	}
}
