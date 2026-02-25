package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/matthewbaird/ontology/internal/signals"
	"github.com/matthewbaird/ontology/internal/types"
)

// Indexer consumes domain events, resolves entity references, classifies them,
// and writes activity entries to the store. In production this subscribes to
// NATS JetStream; here we expose ProcessEvent for direct invocation.
type Indexer struct {
	store Store

	// Traversal cache: in-memory maps updated by the event stream.
	mu              sync.RWMutex
	spaceToLease    map[string]string            // space_id → active lease_id
	leaseToTenants  map[string][]string           // lease_id → tenant person_ids
	spaceToBuilding map[string]string             // space_id → building_id
	spaceToProperty map[string]string             // space_id → property_id
}

// NewIndexer creates a new activity indexer.
func NewIndexer(store Store) *Indexer {
	return &Indexer{
		store:           store,
		spaceToLease:    make(map[string]string),
		leaseToTenants:  make(map[string][]string),
		spaceToBuilding: make(map[string]string),
		spaceToProperty: make(map[string]string),
	}
}

// UpdateTraversalCache updates the in-memory relationship cache.
// Called when relationship-bearing events arrive.
func (idx *Indexer) UpdateTraversalCache(spaceID, leaseID, buildingID, propertyID string, tenantPersonIDs []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if spaceID != "" && leaseID != "" {
		idx.spaceToLease[spaceID] = leaseID
	}
	if leaseID != "" && len(tenantPersonIDs) > 0 {
		idx.leaseToTenants[leaseID] = tenantPersonIDs
	}
	if spaceID != "" && buildingID != "" {
		idx.spaceToBuilding[spaceID] = buildingID
	}
	if spaceID != "" && propertyID != "" {
		idx.spaceToProperty[spaceID] = propertyID
	}
}

// ProcessEvent is the main indexing pipeline for a single domain event.
// Steps: extract refs → resolve traversals → classify → generate summary → write entries.
func (idx *Indexer) ProcessEvent(ctx context.Context, event signals.DomainEvent) error {
	// 1. Extract direct entity references from payload.
	directRefs := idx.extractDirectRefs(event)

	// 2. Resolve one-hop traversal references.
	allRefs := idx.resolveTraversals(directRefs)

	// 3. Classify event using signal registry.
	classification, classified := signals.ClassifyEvent(event)
	category := "lifecycle" // default for unclassified events
	weight := "info"
	polarity := "neutral"
	description := event.EventType
	if classified {
		category = classification.Category
		weight = classification.Weight
		polarity = classification.Polarity
		description = classification.Description
	}

	// 4. Generate human-readable summary.
	summary := generateSummary(event, description, directRefs)

	// 5. Build source refs from direct references.
	var sourceRefs []types.SourceRef
	for _, ref := range directRefs {
		sourceRefs = append(sourceRefs, types.SourceRef{
			EntityType: ref.entityType,
			EntityID:   ref.entityID,
			Role:       ref.role,
		})
	}

	// 6. Create one ActivityEntry per referenced entity.
	now := time.Now()
	var entries []types.ActivityEntry
	seen := make(map[string]bool) // deduplicate by entity_type:entity_id
	for _, ref := range allRefs {
		key := ref.entityType + ":" + ref.entityID
		if seen[key] {
			continue
		}
		seen[key] = true

		entries = append(entries, types.ActivityEntry{
			EventID:           event.EventID,
			EventType:         event.EventType,
			OccurredAt:        now,
			IndexedEntityType: ref.entityType,
			IndexedEntityID:   ref.entityID,
			EntityRole:        ref.role,
			SourceRefs:        sourceRefs,
			Summary:           summary,
			Category:          category,
			Weight:            weight,
			Polarity:          polarity,
			Payload:           event.Payload,
		})
	}

	if len(entries) == 0 {
		return nil
	}

	// 7. Write to store.
	return idx.store.WriteEntries(ctx, entries)
}

// entityRef is an internal struct for tracking entity references during indexing.
type entityRef struct {
	entityType string
	entityID   string
	role       string
}

// extractDirectRefs pulls entity IDs from the event payload.
func (idx *Indexer) extractDirectRefs(event signals.DomainEvent) []entityRef {
	var payload map[string]interface{}
	if len(event.Payload) > 0 {
		_ = json.Unmarshal(event.Payload, &payload)
	}

	var refs []entityRef

	// Standard ID fields → entity types.
	fieldMap := map[string]string{
		"person_id":   "person",
		"lease_id":    "lease",
		"space_id":    "space",
		"property_id": "property",
		"building_id": "building",
		"account_id":  "account",
	}

	for field, entityType := range fieldMap {
		if id, ok := payload[field].(string); ok && id != "" {
			role := "target"
			if entityType == "property" {
				role = "context"
			}
			if entityType == "person" {
				role = "subject"
			}
			refs = append(refs, entityRef{
				entityType: entityType,
				entityID:   id,
				role:       role,
			})
		}
	}

	return refs
}

// resolveTraversals adds one-hop related entities from the traversal cache.
func (idx *Indexer) resolveTraversals(directRefs []entityRef) []entityRef {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	allRefs := make([]entityRef, len(directRefs))
	copy(allRefs, directRefs)

	for _, ref := range directRefs {
		switch ref.entityType {
		case "lease":
			// lease → space_ids (via reverse lookup)
			for spaceID, leaseID := range idx.spaceToLease {
				if leaseID == ref.entityID {
					allRefs = append(allRefs, entityRef{
						entityType: "space",
						entityID:   spaceID,
						role:       "related",
					})
				}
			}
			// lease → tenant person_ids
			if tenants, ok := idx.leaseToTenants[ref.entityID]; ok {
				for _, personID := range tenants {
					allRefs = append(allRefs, entityRef{
						entityType: "person",
						entityID:   personID,
						role:       "related",
					})
				}
			}

		case "space":
			// space → active lease_id
			if leaseID, ok := idx.spaceToLease[ref.entityID]; ok {
				allRefs = append(allRefs, entityRef{
					entityType: "lease",
					entityID:   leaseID,
					role:       "related",
				})
			}
			// space → building_id
			if buildingID, ok := idx.spaceToBuilding[ref.entityID]; ok {
				allRefs = append(allRefs, entityRef{
					entityType: "building",
					entityID:   buildingID,
					role:       "context",
				})
			}
			// space → property_id
			if propertyID, ok := idx.spaceToProperty[ref.entityID]; ok {
				allRefs = append(allRefs, entityRef{
					entityType: "property",
					entityID:   propertyID,
					role:       "context",
				})
			}
		}
	}

	return allRefs
}

// generateSummary creates a human-readable summary for the activity entry.
func generateSummary(event signals.DomainEvent, description string, refs []entityRef) string {
	// Find the primary entity for context.
	var entityContext string
	for _, ref := range refs {
		if ref.role == "target" || ref.role == "subject" {
			entityContext = fmt.Sprintf(" for %s %s", ref.entityType, ref.entityID)
			break
		}
	}
	return description + entityContext
}
