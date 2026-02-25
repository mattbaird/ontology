// Package worker contains event consumer workers that maintain derived data stores.
package worker

import (
	"context"
	"encoding/json"
	"log"
)

// DomainEvent is the canonical event structure emitted by Ent hooks.
type DomainEvent struct {
	Type          string                 `json:"type"`
	EntityType    string                 `json:"entity_type"`
	EntityID      string                 `json:"entity_id"`
	Timestamp     string                 `json:"timestamp"`
	Actor         string                 `json:"actor"`
	Source        string                 `json:"source"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	PreviousState string                 `json:"previous_state,omitempty"`
	NewState      string                 `json:"new_state,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
}

// GraphSyncWorker consumes domain events from NATS and maintains
// a Neo4j projection of the ontology's relationship graph.
type GraphSyncWorker struct {
	// neo4jDriver would be injected here
}

// NewGraphSyncWorker creates a new graph sync worker.
func NewGraphSyncWorker() *GraphSyncWorker {
	return &GraphSyncWorker{}
}

// HandleEvent processes a domain event and updates the graph projection.
func (w *GraphSyncWorker) HandleEvent(ctx context.Context, data []byte) error {
	var event DomainEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	log.Printf("graph_sync: processing %s for %s/%s", event.Type, event.EntityType, event.EntityID)

	// Route to entity-specific handler
	switch event.EntityType {
	case "Person", "Organization", "PersonRole":
		return w.syncPersonGraph(ctx, event)
	case "Portfolio", "Property", "Building", "Space", "LeaseSpace":
		return w.syncPropertyGraph(ctx, event)
	case "Lease", "Application":
		return w.syncLeaseGraph(ctx, event)
	case "Account", "LedgerEntry", "JournalEntry", "BankAccount", "Reconciliation":
		return w.syncAccountingGraph(ctx, event)
	default:
		log.Printf("graph_sync: unknown entity type %s", event.EntityType)
	}

	return nil
}

func (w *GraphSyncWorker) syncPersonGraph(ctx context.Context, event DomainEvent) error {
	// TODO: Create/update person node in Neo4j
	// - Upsert node with entity_type, id, status, name
	// - Create/update edges for PersonRole relationships
	return nil
}

func (w *GraphSyncWorker) syncPropertyGraph(ctx context.Context, event DomainEvent) error {
	// TODO: Create/update property hierarchy in Neo4j
	// - Portfolio -> Property -> Unit relationship chain
	return nil
}

func (w *GraphSyncWorker) syncLeaseGraph(ctx context.Context, event DomainEvent) error {
	// TODO: Create/update lease relationships in Neo4j
	// - Lease connects Unit to PersonRoles (tenant/guarantor)
	return nil
}

func (w *GraphSyncWorker) syncAccountingGraph(ctx context.Context, event DomainEvent) error {
	// TODO: Create/update accounting relationships in Neo4j
	// - Account hierarchy (parent/child)
	// - LedgerEntry -> Account, JournalEntry
	return nil
}
