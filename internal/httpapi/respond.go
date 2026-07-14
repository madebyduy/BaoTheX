// Package httpapi implements the REST API server.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"repwire/internal/domain"
)

// Meta is the pagination block in a list response.
type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// envelope is the standard success response shape: {"data": ..., "meta": ...}.
type envelope struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

// errBody is the standard error response shape.
type errBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data any, meta *Meta) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data, Meta: meta})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	var body errBody
	body.Error.Code = code
	body.Error.Message = message
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeDomainError maps a sentinel domain error to an HTTP response.
func writeDomainError(w http.ResponseWriter, log *slog.Logger, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Resource not found")
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "Resource already exists")
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Not allowed")
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation", err.Error())
	default:
		log.Error("internal error", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Internal server error")
	}
}

// decodeJSON decodes a request body into v, returning false (and writing an
// error) on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return false
	}
	return true
}

// pagination reads ?page= & ?per_page= with sane defaults and caps.
func pagination(r *http.Request) (page, perPage, offset int) {
	page = atoiDefault(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}
	perPage = atoiDefault(r.URL.Query().Get("per_page"), 20)
	if perPage < 1 || perPage > 50 {
		perPage = 20
	}
	return page, perPage, (page - 1) * perPage
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

// pathInt parses an int path value (e.g. r.PathValue("id")).
func pathInt(r *http.Request, name string) (int64, bool) {
	n, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// nonNilItems ensures a nil slice serialises as [] rather than null.
func nonNilItems(items []domain.ContentItem) []domain.ContentItem {
	if items == nil {
		return []domain.ContentItem{}
	}
	return items
}

func nonNilTopics[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func nonNilEntities[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func boolPtr(s string) *bool {
	switch s {
	case "true", "1":
		v := true
		return &v
	case "false", "0":
		v := false
		return &v
	}
	return nil
}
