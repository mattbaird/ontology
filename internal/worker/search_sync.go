package worker

import (
	"context"
	"encoding/json"
	"log"
)

// SearchSyncWorker consumes domain events from NATS and maintains
// Meilisearch indexes for full-text search across all entity types.
type SearchSyncWorker struct {
	// meiliClient would be injected here
}

// NewSearchSyncWorker creates a new search sync worker.
func NewSearchSyncWorker() *SearchSyncWorker {
	return &SearchSyncWorker{}
}

// HandleEvent processes a domain event and updates search indexes.
func (w *SearchSyncWorker) HandleEvent(ctx context.Context, data []byte) error {
	var event DomainEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	log.Printf("search_sync: indexing %s/%s", event.EntityType, event.EntityID)

	// Each entity type has its own Meilisearch index.
	// The index name matches the snake_case entity type.
	// Only searchable/filterable fields are indexed.

	// TODO: Fetch full entity from Ent and index to Meilisearch
	// - Properties: name, address fields, property_type, status
	// - Units: unit_number, unit_type, status, amenities
	// - Persons: first_name, last_name, display_name, contact info
	// - Leases: lease_type, status, tenant info
	// - Accounts: account_number, name, account_type

	return nil
}
