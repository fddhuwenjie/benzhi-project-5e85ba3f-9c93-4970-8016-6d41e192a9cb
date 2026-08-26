package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"windtunnel-release/internal/domain"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}
type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, domain.Invalid("body", "JSON 请求无效: "+err.Error()))
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, domain.Invalid("body", "请求只能包含一个 JSON 对象"))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrConflict):
		status, code = http.StatusConflict, "revision_conflict"
	case errors.Is(err, domain.ErrIdempotency):
		status, code = http.StatusConflict, "request_id_conflict"
	case errors.Is(err, domain.ErrInvalidState):
		status, code = http.StatusUnprocessableEntity, "invalid_state"
	case errors.Is(err, domain.ErrArchived):
		status, code = http.StatusLocked, "archived"
	case errors.Is(err, domain.ErrUnauthorizedRole):
		status, code = http.StatusForbidden, "forbidden_role"
	}
	var validation *domain.ValidationError
	detail := errorDetail{Code: code, Message: err.Error()}
	if errors.As(err, &validation) {
		status, detail.Code, detail.Field = http.StatusBadRequest, "validation_failed", validation.Field
	}
	writeJSON(w, status, errorBody{Error: detail})
}

func queryInt(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return fallback
	}
	return result
}
