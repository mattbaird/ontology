package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent"
)

// AuditInfo holds audit metadata extracted from request headers.
type AuditInfo struct {
	Actor         string
	Source        string
	CorrelationID *string
}

// writeJSON marshals v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

// writeError writes a structured JSON error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
		"code":  code,
	})
}

// decodeJSON decodes the request body into v.
func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// parseUUID extracts and validates a UUID path parameter.
func parseUUID(w http.ResponseWriter, r *http.Request, paramName string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, paramName)
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid UUID: "+raw)
		return uuid.Nil, false
	}
	return id, true
}

// Pagination holds parsed pagination parameters.
type Pagination struct {
	Limit  int
	Offset int
}

// parsePagination extracts page_size and offset from query params.
func parsePagination(r *http.Request) Pagination {
	p := Pagination{Limit: 20, Offset: 0}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.Limit = n
		}
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.Offset = n
		}
	}
	return p
}

// entErrorToHTTP maps Ent errors to appropriate HTTP responses.
func entErrorToHTTP(w http.ResponseWriter, err error) {
	if ent.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	if ent.IsValidationError(err) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if ent.IsConstraintError(err) {
		writeError(w, http.StatusConflict, "CONSTRAINT_ERROR", err.Error())
		return
	}
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

// parseAuditContext extracts audit metadata from request headers.
func parseAuditContext(w http.ResponseWriter, r *http.Request) (AuditInfo, bool) {
	actor := r.Header.Get("X-Actor")
	if actor == "" {
		writeError(w, http.StatusBadRequest, "MISSING_ACTOR", "X-Actor header is required")
		return AuditInfo{}, false
	}
	source := r.Header.Get("X-Source")
	if source == "" {
		source = "user"
	}
	info := AuditInfo{
		Actor:  actor,
		Source: source,
	}
	if cid := r.Header.Get("X-Correlation-ID"); cid != "" {
		info.CorrelationID = &cid
	}
	return info, true
}
