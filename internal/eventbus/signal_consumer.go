package eventbus

import (
	"context"
	"log"

	"github.com/matthewbaird/ontology/internal/event"
	"github.com/matthewbaird/ontology/internal/signals"
)

// SignalConsumer classifies domain events against the signal registry
// and logs when escalation-worthy patterns are detected.
type SignalConsumer struct{}

// NewSignalConsumer creates a new signal classification consumer.
func NewSignalConsumer() *SignalConsumer {
	return &SignalConsumer{}
}

// HandleEvent classifies the domain event against the signal registry.
func (c *SignalConsumer) HandleEvent(_ context.Context, evt event.DomainEvent) error {
	// The signal registry uses PascalCase event types (e.g., "PaymentRecorded")
	// while DomainEvents use snake_case (e.g., "payment_received").
	// Try both the raw event type and common mappings.
	sEvt := signals.DomainEvent{
		EventID:   evt.ID,
		EventType: evt.EventType,
		Payload:   evt.Payload,
	}

	result, ok := signals.ClassifyEvent(sEvt)
	if !ok {
		return nil
	}

	log.Printf("signal: classified %s as [%s] weight=%s polarity=%s — %s",
		evt.EventType, result.Category, result.Weight, result.Polarity, result.Description)

	return nil
}
