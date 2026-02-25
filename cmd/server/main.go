package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/internal/server"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/matthewbaird/ontology/ent/runtime"
	_ "modernc.org/sqlite"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "file:ontology.db?_pragma=foreign_keys(1)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	db.SetMaxOpenConns(1)

	// Enable foreign keys explicitly — required for SQLite.
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		log.Fatalf("enabling foreign keys: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("running schema migration: %v", err)
	}
	log.Println("database migrated successfully")

	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	if err := server.Run(ctx, server.Config{
		Port:     port,
		DBClient: client,
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
