// Package event provides domain event recording for command handlers.
// Events are fanned out as ActivityEntry records via the activity.Store interface,
// then published to the in-process event bus for downstream consumers.
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

// Publisher sends domain events to downstream consumers.
type Publisher interface {
	Publish(ctx context.Context, evt DomainEvent)
}

// ActivityRecorder implements Recorder by fanning out a DomainEvent into
// one ActivityEntry per affected entity, then writing via activity.Store.
// If a Publisher is set, the event is also published to the event bus
// after the store write succeeds.
type ActivityRecorder struct {
	store activity.Store
	bus   Publisher
}

// NewActivityRecorder creates a new ActivityRecorder backed by the given store.
func NewActivityRecorder(store activity.Store) *ActivityRecorder {
	return &ActivityRecorder{store: store}
}

// SetPublisher attaches an event bus. Events are published after store writes.
func (r *ActivityRecorder) SetPublisher(p Publisher) {
	r.bus = p
}

// Record fans out a DomainEvent into ActivityEntry records, writes them,
// and publishes to the event bus.
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
	if err := r.store.WriteEntries(ctx, entries); err != nil {
		return err
	}

	// Publish to event bus after successful store write.
	if r.bus != nil {
		r.bus.Publish(ctx, evt)
	}
	return nil
}
