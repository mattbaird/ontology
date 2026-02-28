package handler

import (
	"context"
	"log"

	"github.com/matthewbaird/ontology/internal/event"
)

// defaultRecorder is the package-level event recorder.
// Set during server startup via SetRecorder.
// Handler structs are defined in generated code and cannot be modified,
// so we use a package-level variable instead of a struct field.
var defaultRecorder event.Recorder

// SetRecorder sets the package-level event recorder.
// Call this during server startup before handling requests.
func SetRecorder(r event.Recorder) {
	defaultRecorder = r
}

// recordEvent records a domain event if a recorder is configured.
// Errors are logged but do not fail the request — event recording
// is best-effort and should not block command execution.
func recordEvent(ctx context.Context, evt event.DomainEvent) {
	if defaultRecorder == nil {
		return
	}
	if err := defaultRecorder.Record(ctx, evt); err != nil {
		log.Printf("event recording failed: %v", err)
	}
}
