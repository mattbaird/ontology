// Package event provides domain event recording for command handlers.
// Events are fanned out as ActivityEntry records via the activity.Store interface.
package event

import (
	"context"

	"github.com/matthewbaird/ontology/internal/activity"
	"github.com/matthewbaird/ontology/internal/types"
)

// Recorder writes domain events to the activity store.
type Recorder interface {
	Record(ctx context.Context, evt DomainEvent) error
}

// ActivityRecorder implements Recorder by fanning out a DomainEvent into
// one ActivityEntry per affected entity, then writing via activity.Store.
type ActivityRecorder struct {
	store activity.Store
}

// NewActivityRecorder creates a new ActivityRecorder backed by the given store.
func NewActivityRecorder(store activity.Store) *ActivityRecorder {
	return &ActivityRecorder{store: store}
}

// Record fans out a DomainEvent into ActivityEntry records and writes them.
func (r *ActivityRecorder) Record(ctx context.Context, evt DomainEvent) error {
	entries := make([]types.ActivityEntry, 0, len(evt.AffectedEntities))
	for _, ref := range evt.AffectedEntities {
		entries = append(entries, types.ActivityEntry{
			EventID:           evt.ID,
			EventType:         evt.EventType,
			OccurredAt:        evt.OccurredAt,
			IndexedEntityType: ref.EntityType,
			IndexedEntityID:   ref.EntityID,
			EntityRole:        ref.Role,
			SourceRefs:        evt.AffectedEntities,
			Summary:           evt.Summary,
			Category:          evt.Category,
			Weight:            evt.Weight,
			Polarity:          evt.Polarity,
			Payload:           evt.Payload,
		})
	}
	return r.store.WriteEntries(ctx, entries)
}
