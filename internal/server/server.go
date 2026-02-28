// Package server assembles all HTTP handlers and starts the server.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/internal/activity"
	"github.com/matthewbaird/ontology/internal/event"
	"github.com/matthewbaird/ontology/internal/eventbus"
	"github.com/matthewbaird/ontology/internal/handler"
	"github.com/matthewbaird/ontology/internal/jurisdiction"
	"github.com/matthewbaird/ontology/internal/repl"
	"github.com/matthewbaird/ontology/internal/repl/executor"
	"github.com/matthewbaird/ontology/internal/repl/schema"
)

// Config holds server configuration.
type Config struct {
	Port          int
	DBClient      *ent.Client
	ActivityStore activity.Store // optional; if set, activity routes are registered
}

// Run starts the HTTP server with all routes registered.
func Run(ctx context.Context, cfg Config) error {
	r := chi.NewRouter()
	r.Use(handler.CORS, handler.Logging, handler.Recovery)

	// Wire event recorder and event bus if activity store is configured.
	if cfg.ActivityStore != nil {
		bus := eventbus.New(256)
		bus.Subscribe("log", eventbus.NewLogConsumer())
		bus.Subscribe("signals", eventbus.NewSignalConsumer())

		recorder := event.NewActivityRecorder(cfg.ActivityStore)
		recorder.SetPublisher(bus)
		handler.SetRecorder(recorder)

		bus.Start(ctx)
		log.Printf("event bus started with 2 consumers")
	}

	// Register jurisdiction enforcement hook on lease mutations.
	cfg.DBClient.Lease.Use(jurisdiction.LeaseHook(cfg.DBClient))

	// Generated routes for standard CRUD and transitions.
	RegisterRoutes(r, cfg.DBClient)
	// Custom routes with non-standard path patterns.
	// Registered after generated routes so custom handlers override generated ones.
	RegisterCustomRoutes(r, cfg.DBClient)

	// Activity/signal routes (optional — no Ent dependency).
	if cfg.ActivityStore != nil {
		RegisterActivityRoutes(r, cfg.ActivityStore)
	}

	// REPL routes (always registered in dev mode).
	replRegistry := schema.InitRegistry()
	replDispatchers := executor.InitDispatchers()
	repl.RegisterRoutes(r, cfg.DBClient, replRegistry, replDispatchers)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("starting server on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}
