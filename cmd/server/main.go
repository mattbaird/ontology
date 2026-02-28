package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"ariga.io/atlas-go-sdk/atlasexec"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/internal/activity"
	"github.com/matthewbaird/ontology/internal/seed"
	"github.com/matthewbaird/ontology/internal/server"
	"github.com/matthewbaird/ontology/internal/signals"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/matthewbaird/ontology/ent/runtime"
	_ "modernc.org/sqlite"
)

func main() {
	demo := flag.Bool("demo", false, "seed activity store with demo data")
	flag.Parse()

	signals.Init()

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

	// Apply pending Atlas versioned migrations.
	// Convert Go SQLite DSN (file:path?params) to Atlas URL (sqlite://path?params).
	atlasURL := "sqlite://" + strings.TrimPrefix(dsn, "file:")
	atlasClient, err := atlasexec.NewClient(".", "atlas")
	if err != nil {
		log.Fatalf("initializing atlas client: %v", err)
	}
	res, err := atlasClient.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
		URL:    atlasURL,
		DirURL: "file://ent/migrations",
	})
	if err != nil {
		log.Fatalf("applying migrations: %v", err)
	}
	log.Printf("database migrated: %d applied\n", len(res.Applied))

	store := activity.NewMemoryStore()
	if *demo {
		if err := activity.SeedDemoData(ctx, store); err != nil {
			log.Fatalf("seeding demo data: %v", err)
		}
		if err := seed.SeedJurisdictions(ctx, client); err != nil {
			log.Fatalf("seeding jurisdictions: %v", err)
		}
	}

	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	if err := server.Run(ctx, server.Config{
		Port:          port,
		DBClient:      client,
		ActivityStore: store,
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
