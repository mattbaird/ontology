// Custom activity handlers for the signal discovery system.
// These handlers don't use Ent — they operate on the separate activity store.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matthewbaird/ontology/internal/activity"
	"github.com/matthewbaird/ontology/internal/signals"
	"github.com/matthewbaird/ontology/internal/types"
)

// ActivityHandler implements HTTP handlers for ActivityService.
type ActivityHandler struct {
	store activity.Store
}

// NewActivityHandler creates a new ActivityHandler.
func NewActivityHandler(store activity.Store) *ActivityHandler {
	return &ActivityHandler{store: store}
}

// HandleGetEntityActivity returns a chronological activity feed for any entity.
// GET /v1/activity/entity/{entity_type}/{entity_id}
func (h *ActivityHandler) HandleGetEntityActivity(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entity_type")
	entityID := chi.URLParam(r, "entity_id")
	if entityType == "" || entityID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_PARAMS", "entity_type and entity_id are required")
		return
	}

	opts := activity.DefaultQueryOptions()
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			opts.Since = &t
		}
	}
	if u := r.URL.Query().Get("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			opts.Until = &t
		}
	}
	if cats := r.URL.Query().Get("categories"); cats != "" {
		opts.Categories = strings.Split(cats, ",")
	}
	if mw := r.URL.Query().Get("min_weight"); mw != "" {
		opts.MinWeight = mw
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			if n > 500 {
				n = 500
			}
			opts.Limit = n
		}
	}
	if c := r.URL.Query().Get("cursor"); c != "" {
		opts.Cursor = c
	}

	entries, nextCursor, totalCount, err := h.store.QueryByEntity(r.Context(), entityType, entityID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}

	resp := struct {
		Activities []types.ActivityEntry `json:"activities"`
		NextCursor string                `json:"next_cursor,omitempty"`
		TotalCount int                   `json:"total_count"`
		Period     struct {
			Since time.Time `json:"since"`
			Until time.Time `json:"until"`
		} `json:"period"`
	}{
		Activities: entries,
		NextCursor: nextCursor,
		TotalCount: totalCount,
	}
	if opts.Since != nil {
		resp.Period.Since = *opts.Since
	}
	if opts.Until != nil {
		resp.Period.Until = *opts.Until
	}
	if resp.Activities == nil {
		resp.Activities = []types.ActivityEntry{}
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleGetSignalSummary returns a pre-aggregated signal summary for an entity.
// GET /v1/activity/summary/{entity_type}/{entity_id}
func (h *ActivityHandler) HandleGetSignalSummary(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entity_type")
	entityID := chi.URLParam(r, "entity_id")
	if entityType == "" || entityID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_PARAMS", "entity_type and entity_id are required")
		return
	}

	// Default: 12 months lookback.
	since := time.Now().AddDate(-1, 0, 0)
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}
	until := time.Now()

	opts := activity.QueryOptions{
		Since:     &since,
		Until:     &until,
		MinWeight: "info",
		Limit:     500, // fetch all for aggregation
	}

	entries, _, _, err := h.store.QueryByEntity(r.Context(), entityType, entityID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}

	summary := signals.Aggregate(entries, entityType, entityID, since, until)
	writeJSON(w, http.StatusOK, summary)
}

// HandleGetPortfolioSignals screens multiple entities ranked by concern level.
// POST /v1/activity/portfolio
func (h *ActivityHandler) HandleGetPortfolioSignals(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScopeType   string `json:"scope_type"`
		ScopeID     string `json:"scope_id"`
		EntityType  string `json:"entity_type"`
		MinSeverity string `json:"min_severity"`
		SortBy      string `json:"sort_by"`
		Limit       int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.ScopeType == "" || req.ScopeID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_PARAMS", "scope_type and scope_id are required")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.MinSeverity == "" {
		req.MinSeverity = "moderate"
	}

	// For now, return an empty result. Full implementation requires querying
	// entities within the scope (via Ent) and then batch-querying activity.
	resp := struct {
		Entities   []interface{} `json:"entities"`
		TotalCount int           `json:"total_count"`
	}{
		Entities:   []interface{}{},
		TotalCount: 0,
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleSearchActivity performs full-text search across activity streams.
// POST /v1/activity/search
func (h *ActivityHandler) HandleSearchActivity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query      string   `json:"query"`
		ScopeType  string   `json:"scope_type,omitempty"`
		ScopeID    string   `json:"scope_id,omitempty"`
		EntityType string   `json:"entity_type,omitempty"`
		Since      string   `json:"since,omitempty"`
		Categories []string `json:"categories,omitempty"`
		Limit      int      `json:"limit,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "MISSING_PARAMS", "query is required")
		return
	}

	opts := activity.DefaultSearchOptions()
	opts.ScopeType = req.ScopeType
	opts.ScopeID = req.ScopeID
	opts.EntityType = req.EntityType
	opts.Categories = req.Categories
	if req.Limit > 0 {
		opts.Limit = req.Limit
	}
	if req.Since != "" {
		if t, err := time.Parse(time.RFC3339, req.Since); err == nil {
			opts.Since = &t
		}
	}

	entries, totalCount, err := h.store.Search(r.Context(), req.Query, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SEARCH_FAILED", err.Error())
		return
	}

	resp := struct {
		Results    []types.ActivityEntry `json:"results"`
		TotalCount int                   `json:"total_count"`
	}{
		Results:    entries,
		TotalCount: totalCount,
	}
	if resp.Results == nil {
		resp.Results = []types.ActivityEntry{}
	}

	writeJSON(w, http.StatusOK, resp)
}
