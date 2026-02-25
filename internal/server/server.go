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
	"github.com/matthewbaird/ontology/internal/handler"
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
	r.Use(handler.Logging, handler.Recovery)

	// Generated routes for standard CRUD and transitions.
	RegisterRoutes(r, cfg.DBClient)
	// Custom routes with non-standard path patterns.
	RegisterCustomRoutes(r, cfg.DBClient)

	// Activity/signal routes (optional — no Ent dependency).
	if cfg.ActivityStore != nil {
		RegisterActivityRoutes(r, cfg.ActivityStore)
	}

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
