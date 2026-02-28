package eventbus

import (
	"context"
	"log"

	"github.com/matthewbaird/ontology/internal/event"
)

// LogConsumer logs all domain events for observability.
type LogConsumer struct{}

func NewLogConsumer() *LogConsumer { return &LogConsumer{} }

func (c *LogConsumer) HandleEvent(_ context.Context, evt event.DomainEvent) error {
	entities := make([]string, len(evt.AffectedEntities))
	for i, ref := range evt.AffectedEntities {
		entities[i] = ref.EntityType + ":" + ref.EntityID[:8]
	}
	log.Printf("event: %s [%s/%s] %s — entities=%v",
		evt.EventType, evt.Category, evt.Weight, evt.Summary, entities)
	return nil
}
