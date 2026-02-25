package activity

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/matthewbaird/ontology/internal/signals"
	"github.com/matthewbaird/ontology/internal/types"
)

// MemoryStore implements Store using in-memory slices.
// Intended for demos and testing — no Postgres required.
type MemoryStore struct {
	mu      sync.RWMutex
	entries []types.ActivityEntry
}

// NewMemoryStore creates a new empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) WriteEntries(_ context.Context, entries []types.ActivityEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entries...)
	return nil
}

func (s *MemoryStore) QueryByEntity(_ context.Context, entityType, entityID string, opts QueryOptions) ([]types.ActivityEntry, string, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matched []types.ActivityEntry
	for _, e := range s.entries {
		if e.IndexedEntityType != entityType || e.IndexedEntityID != entityID {
			continue
		}
		if opts.Since != nil && e.OccurredAt.Before(*opts.Since) {
			continue
		}
		if opts.Until != nil && e.OccurredAt.After(*opts.Until) {
			continue
		}
		if len(opts.Categories) > 0 && !contains(opts.Categories, e.Category) {
			continue
		}
		if opts.MinWeight != "" && opts.MinWeight != "info" {
			if !signals.IsAtLeastWeight(e.Weight, opts.MinWeight) {
				continue
			}
		}
		if opts.Cursor != "" {
			cursorTime, err := time.Parse(time.RFC3339Nano, opts.Cursor)
			if err == nil && !e.OccurredAt.Before(cursorTime) {
				continue
			}
		}
		matched = append(matched, e)
	}

	// Sort by occurred_at DESC.
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].OccurredAt.After(matched[j].OccurredAt)
	})

	totalCount := len(matched)
	limit := opts.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var nextCursor string
	if len(matched) > limit {
		matched = matched[:limit]
		nextCursor = matched[len(matched)-1].OccurredAt.Format(time.RFC3339Nano)
	}

	return matched, nextCursor, totalCount, nil
}

func (s *MemoryStore) Search(_ context.Context, query string, opts SearchOptions) ([]types.ActivityEntry, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(query)
	var matched []types.ActivityEntry
	for _, e := range s.entries {
		if !strings.Contains(strings.ToLower(e.Summary), q) {
			continue
		}
		if opts.EntityType != "" && e.IndexedEntityType != opts.EntityType {
			continue
		}
		if opts.Since != nil && e.OccurredAt.Before(*opts.Since) {
			continue
		}
		if len(opts.Categories) > 0 && !contains(opts.Categories, e.Category) {
			continue
		}
		matched = append(matched, e)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].OccurredAt.After(matched[j].OccurredAt)
	})

	totalCount := len(matched)
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	if len(matched) > limit {
		matched = matched[:limit]
	}

	return matched, totalCount, nil
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
