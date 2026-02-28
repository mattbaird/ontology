// Package eventbus provides an in-process pub/sub event bus for domain events.
// Handlers publish events after commit; subscribers process them asynchronously.
// This replaces NATS for the POC — subscribers run in-process via goroutines.
package eventbus

import (
	"context"
	"log"
	"sync"

	"github.com/matthewbaird/ontology/internal/event"
)

// Handler processes a domain event. Implementations must be safe for
// concurrent calls from different goroutines.
type Handler interface {
	HandleEvent(ctx context.Context, evt event.DomainEvent) error
}

// HandlerFunc adapts a plain function to the Handler interface.
type HandlerFunc func(ctx context.Context, evt event.DomainEvent) error

func (f HandlerFunc) HandleEvent(ctx context.Context, evt event.DomainEvent) error {
	return f(ctx, evt)
}

// Bus is a simple in-process event bus. Events are published to a buffered
// channel and dispatched to all subscribers in a single consumer goroutine.
// This serialises event processing which is fine for the POC and avoids
// concurrent-write issues with SQLite.
type Bus struct {
	mu          sync.RWMutex
	subscribers []namedHandler
	events      chan event.DomainEvent
	done        chan struct{}
}

type namedHandler struct {
	name    string
	handler Handler
}

// New creates a new Bus with the given channel buffer size.
func New(bufSize int) *Bus {
	if bufSize < 1 {
		bufSize = 256
	}
	return &Bus{
		events: make(chan event.DomainEvent, bufSize),
		done:   make(chan struct{}),
	}
}

// Subscribe registers a named handler. Must be called before Start.
func (b *Bus) Subscribe(name string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = append(b.subscribers, namedHandler{name: name, handler: h})
}

// Publish sends an event to the bus. Non-blocking — if the buffer is full
// the event is dropped and a warning is logged.
func (b *Bus) Publish(ctx context.Context, evt event.DomainEvent) {
	select {
	case b.events <- evt:
	default:
		log.Printf("eventbus: buffer full, dropping event %s (%s)", evt.EventType, evt.ID)
	}
}

// Start begins the consumer goroutine. It processes events until the
// context is cancelled or Stop is called.
func (b *Bus) Start(ctx context.Context) {
	go func() {
		defer close(b.done)
		for {
			select {
			case evt := <-b.events:
				b.dispatch(ctx, evt)
			case <-ctx.Done():
				// Drain remaining events before exiting.
				for {
					select {
					case evt := <-b.events:
						b.dispatch(ctx, evt)
					default:
						return
					}
				}
			}
		}
	}()
}

// Stop waits for the consumer goroutine to finish.
func (b *Bus) Stop() {
	close(b.events)
	<-b.done
}

func (b *Bus) dispatch(ctx context.Context, evt event.DomainEvent) {
	b.mu.RLock()
	subs := b.subscribers
	b.mu.RUnlock()

	for _, s := range subs {
		if err := s.handler.HandleEvent(ctx, evt); err != nil {
			log.Printf("eventbus: %s handler error for %s: %v", s.name, evt.EventType, err)
		}
	}
}
