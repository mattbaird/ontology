// Package executor converts QueryPlans into Ent query builder calls
// and returns results.
package executor

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/internal/repl/planner"
)

// EntityDispatcher bridges dynamic PQL entity names to typed Ent operations.
// Each entity gets a generated implementation.
type EntityDispatcher interface {
	// Query creates a new query handle for list operations.
	Query(client *ent.Client) QueryHandle
	// Get fetches a single entity by UUID.
	Get(ctx context.Context, client *ent.Client, id uuid.UUID) (any, error)
}

// QueryHandle provides a chainable interface over a typed Ent query builder.
type QueryHandle interface {
	Where(specs ...planner.PredicateSpec) QueryHandle
	WithEdge(name string) QueryHandle
	OrderBy(field string, desc bool) QueryHandle
	Limit(n int) QueryHandle
	Offset(n int) QueryHandle
	All(ctx context.Context) ([]any, error)
	Count(ctx context.Context) (int, error)
}

// MutationDispatcher adds create/update/delete operations.
// Each generated entity dispatcher implements this interface.
type MutationDispatcher interface {
	// Create inserts a new entity with the given field assignments.
	Create(ctx context.Context, client *ent.Client, fields map[string]any) (any, error)
	// Update modifies an existing entity by UUID with the given field assignments.
	Update(ctx context.Context, client *ent.Client, id uuid.UUID, fields map[string]any) (any, error)
	// Delete removes an entity by UUID.
	Delete(ctx context.Context, client *ent.Client, id uuid.UUID) error
}

// DispatchRegistry maps PQL entity names to their dispatchers.
type DispatchRegistry struct {
	dispatchers map[string]EntityDispatcher
}

// NewDispatchRegistry creates an empty dispatch registry.
func NewDispatchRegistry() *DispatchRegistry {
	return &DispatchRegistry{
		dispatchers: make(map[string]EntityDispatcher),
	}
}

// Register adds a dispatcher for an entity.
func (r *DispatchRegistry) Register(entity string, d EntityDispatcher) {
	r.dispatchers[entity] = d
}

// Get returns the dispatcher for an entity, or nil.
func (r *DispatchRegistry) Get(entity string) EntityDispatcher {
	return r.dispatchers[entity]
}

// GetMutation returns the MutationDispatcher for an entity, or an error
// if the dispatcher doesn't support mutations.
func (r *DispatchRegistry) GetMutation(entity string) (MutationDispatcher, error) {
	d := r.dispatchers[entity]
	if d == nil {
		return nil, fmt.Errorf("no dispatcher for entity '%s'", entity)
	}
	md, ok := d.(MutationDispatcher)
	if !ok {
		return nil, fmt.Errorf("entity '%s' does not support mutations", entity)
	}
	return md, nil
}
