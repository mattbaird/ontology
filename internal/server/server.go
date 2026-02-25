// Package server assembles all HTTP handlers and starts the server.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/internal/handler"
)

// Config holds server configuration.
type Config struct {
	Port     int
	DBClient *ent.Client
}

// Run starts the HTTP server with all routes registered.
func Run(ctx context.Context, cfg Config) error {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// --- PersonService ---
	ph := handler.NewPersonHandler(cfg.DBClient)
	mux.HandleFunc("POST /v1/persons", ph.CreatePerson)
	mux.HandleFunc("GET /v1/persons/{id}", ph.GetPerson)
	mux.HandleFunc("GET /v1/persons", ph.ListPersons)
	mux.HandleFunc("PATCH /v1/persons/{id}", ph.UpdatePerson)
	mux.HandleFunc("POST /v1/organizations", ph.CreateOrganization)
	mux.HandleFunc("GET /v1/organizations/{id}", ph.GetOrganization)
	mux.HandleFunc("GET /v1/organizations", ph.ListOrganizations)
	mux.HandleFunc("PATCH /v1/organizations/{id}", ph.UpdateOrganization)
	mux.HandleFunc("POST /v1/person-roles", ph.CreatePersonRole)
	mux.HandleFunc("GET /v1/person-roles/{id}", ph.GetPersonRole)
	mux.HandleFunc("GET /v1/person-roles", ph.ListPersonRoles)
	mux.HandleFunc("POST /v1/person-roles/{id}/activate", ph.ActivateRole)
	mux.HandleFunc("POST /v1/person-roles/{id}/deactivate", ph.DeactivateRole)
	mux.HandleFunc("POST /v1/person-roles/{id}/terminate", ph.TerminateRole)

	// --- PropertyService ---
	proph := handler.NewPropertyHandler(cfg.DBClient)
	mux.HandleFunc("POST /v1/portfolios", proph.CreatePortfolio)
	mux.HandleFunc("GET /v1/portfolios/{id}", proph.GetPortfolio)
	mux.HandleFunc("GET /v1/portfolios", proph.ListPortfolios)
	mux.HandleFunc("PATCH /v1/portfolios/{id}", proph.UpdatePortfolio)
	mux.HandleFunc("POST /v1/portfolios/{id}/activate", proph.ActivatePortfolio)
	mux.HandleFunc("POST /v1/properties", proph.CreateProperty)
	mux.HandleFunc("GET /v1/properties/{id}", proph.GetProperty)
	mux.HandleFunc("GET /v1/properties", proph.ListProperties)
	mux.HandleFunc("PATCH /v1/properties/{id}", proph.UpdateProperty)
	mux.HandleFunc("POST /v1/properties/{id}/activate", proph.ActivateProperty)
	mux.HandleFunc("POST /v1/units", proph.CreateUnit)
	mux.HandleFunc("GET /v1/units/{id}", proph.GetUnit)
	mux.HandleFunc("GET /v1/units", proph.ListUnits)
	mux.HandleFunc("PATCH /v1/units/{id}", proph.UpdateUnit)

	// --- LeaseService ---
	lh := handler.NewLeaseHandler(cfg.DBClient)
	mux.HandleFunc("POST /v1/leases", lh.CreateLease)
	mux.HandleFunc("GET /v1/leases/{id}", lh.GetLease)
	mux.HandleFunc("GET /v1/leases", lh.ListLeases)
	mux.HandleFunc("PATCH /v1/leases/{id}", lh.UpdateLease)
	mux.HandleFunc("POST /v1/leases/{id}/submit", lh.SubmitForApproval)
	mux.HandleFunc("POST /v1/leases/{id}/approve", lh.ApproveLease)
	mux.HandleFunc("POST /v1/leases/{id}/activate", lh.ActivateLease)
	mux.HandleFunc("POST /v1/leases/{id}/terminate", lh.TerminateLease)
	mux.HandleFunc("POST /v1/leases/{id}/renew", lh.RenewLease)
	mux.HandleFunc("POST /v1/leases/{id}/evict", lh.StartEviction)
	mux.HandleFunc("POST /v1/applications", lh.CreateApplication)
	mux.HandleFunc("GET /v1/applications/{id}", lh.GetApplication)
	mux.HandleFunc("GET /v1/applications", lh.ListApplications)
	mux.HandleFunc("POST /v1/applications/{id}/approve", lh.ApproveApplication)
	mux.HandleFunc("POST /v1/applications/{id}/deny", lh.DenyApplication)

	// --- AccountingService ---
	ah := handler.NewAccountingHandler(cfg.DBClient)
	mux.HandleFunc("POST /v1/accounts", ah.CreateAccount)
	mux.HandleFunc("GET /v1/accounts/{id}", ah.GetAccount)
	mux.HandleFunc("GET /v1/accounts", ah.ListAccounts)
	mux.HandleFunc("PATCH /v1/accounts/{id}", ah.UpdateAccount)
	mux.HandleFunc("GET /v1/ledger-entries/{id}", ah.GetLedgerEntry)
	mux.HandleFunc("GET /v1/ledger-entries", ah.ListLedgerEntries)
	mux.HandleFunc("POST /v1/journal-entries", ah.CreateJournalEntry)
	mux.HandleFunc("GET /v1/journal-entries/{id}", ah.GetJournalEntry)
	mux.HandleFunc("GET /v1/journal-entries", ah.ListJournalEntries)
	mux.HandleFunc("POST /v1/journal-entries/{id}/post", ah.PostJournalEntry)
	mux.HandleFunc("POST /v1/journal-entries/{id}/void", ah.VoidJournalEntry)
	mux.HandleFunc("POST /v1/bank-accounts", ah.CreateBankAccount)
	mux.HandleFunc("GET /v1/bank-accounts/{id}", ah.GetBankAccount)
	mux.HandleFunc("GET /v1/bank-accounts", ah.ListBankAccounts)
	mux.HandleFunc("PATCH /v1/bank-accounts/{id}", ah.UpdateBankAccount)
	mux.HandleFunc("POST /v1/reconciliations", ah.CreateReconciliation)
	mux.HandleFunc("GET /v1/reconciliations/{id}", ah.GetReconciliation)
	mux.HandleFunc("GET /v1/reconciliations", ah.ListReconciliations)
	mux.HandleFunc("POST /v1/reconciliations/{id}/approve", ah.ApproveReconciliation)

	// Wrap with middleware
	wrapped := handler.Recovery(handler.Logging(mux))

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("starting server on %s (%d routes registered)", addr, 50)

	server := &http.Server{
		Addr:    addr,
		Handler: wrapped,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}
