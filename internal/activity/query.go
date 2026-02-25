// Package activity provides the activity store interface and implementations
// for the entity activity stream (Layer 1 of the signal discovery system).
package activity

import "time"

// QueryOptions controls filtering and pagination for entity activity queries.
type QueryOptions struct {
	Since      *time.Time // default: 6 months ago
	Until      *time.Time // default: now
	Categories []string   // filter to specific signal categories
	MinWeight  string     // minimum weight threshold (default: "info")
	Limit      int        // max results (default: 100, max: 500)
	Cursor     string     // cursor for pagination
}

// SearchOptions controls filtering for full-text activity search.
type SearchOptions struct {
	ScopeType  string     // "portfolio", "property", "building"
	ScopeID    string     // ID of scope entity
	EntityType string     // filter to specific entity type
	Since      *time.Time // filter by time
	Categories []string   // filter to specific signal categories
	Limit      int        // max results (default: 20)
}

// DefaultQueryOptions returns QueryOptions with sensible defaults.
func DefaultQueryOptions() QueryOptions {
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	now := time.Now()
	return QueryOptions{
		Since:     &sixMonthsAgo,
		Until:     &now,
		MinWeight: "info",
		Limit:     100,
	}
}

// DefaultSearchOptions returns SearchOptions with sensible defaults.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Limit: 20,
	}
}
