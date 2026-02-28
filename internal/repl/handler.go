// Package repl provides the WebSocket-based REPL for PQL queries.
package repl

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/internal/repl/autocomplete"
	"github.com/matthewbaird/ontology/internal/repl/executor"
	"github.com/matthewbaird/ontology/internal/repl/meta"
	"github.com/matthewbaird/ontology/internal/repl/planner"
	"github.com/matthewbaird/ontology/internal/repl/schema"
	"github.com/matthewbaird/ontology/internal/repl/session"
	"github.com/matthewbaird/ontology/internal/repl/wire"
)

// RegisterRoutes registers REPL HTTP and WebSocket routes on the given router.
func RegisterRoutes(r chi.Router, client *ent.Client, registry *schema.Registry, dispatchers *executor.DispatchRegistry) {
	// Create session manager (30 min idle, 24 hr max)
	sessions := session.NewManager(24*time.Hour, 30*time.Minute)

	// Create components
	pl := planner.New(registry)
	exec := executor.New(client, dispatchers)
	ac := autocomplete.New(registry)
	metaHandler := meta.New(registry)

	// WebSocket handler
	wsHandler := wire.NewHandler(sessions, pl, exec, ac, metaHandler)

	r.Route("/api/repl", func(r chi.Router) {
		// WebSocket endpoint
		r.Get("/ws", wsHandler.ServeHTTP)

		// Schema endpoint (REST, for inspector/tooling)
		r.Get("/schema", func(w http.ResponseWriter, r *http.Request) {
			entities := registry.AllEntities()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(entities)
		})

		// Session create endpoint (REST alternative to WebSocket)
		r.Post("/session", func(w http.ResponseWriter, r *http.Request) {
			sess := sessions.Create()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sess)
		})
	})
}
