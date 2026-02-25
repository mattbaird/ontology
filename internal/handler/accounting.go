package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/ent/account"
	"github.com/matthewbaird/ontology/ent/bankaccount"
	"github.com/matthewbaird/ontology/ent/journalentry"
	"github.com/matthewbaird/ontology/ent/ledgerentry"
	"github.com/matthewbaird/ontology/ent/reconciliation"
	"github.com/matthewbaird/ontology/ent/schema"
	"github.com/matthewbaird/ontology/internal/types"
)

// AccountingHandler implements HTTP handlers for accounting entities.
type AccountingHandler struct {
	client *ent.Client
}

// NewAccountingHandler creates a new AccountingHandler.
func NewAccountingHandler(client *ent.Client) *AccountingHandler {
	return &AccountingHandler{client: client}
}

// ---------------------------------------------------------------------------
// Account
// ---------------------------------------------------------------------------

type createAccountRequest struct {
	AccountNumber       string                   `json:"account_number"`
	Name                string                   `json:"name"`
	Description         *string                  `json:"description,omitempty"`
	AccountType         string                   `json:"account_type"`
	AccountSubtype      string                   `json:"account_subtype"`
	ParentAccountID     *string                  `json:"parent_account_id,omitempty"`
	Depth               int                      `json:"depth"`
	Dimensions          *types.AccountDimensions `json:"dimensions,omitempty"`
	NormalBalance       string                   `json:"normal_balance"`
	IsHeader            *bool                    `json:"is_header,omitempty"`
	IsSystem            *bool                    `json:"is_system,omitempty"`
	AllowsDirectPosting *bool                    `json:"allows_direct_posting,omitempty"`
	Status              string                   `json:"status"`
	IsTrustAccount      *bool                    `json:"is_trust_account,omitempty"`
	TrustType           *string                  `json:"trust_type,omitempty"`
	BudgetAmountCents   *int64                   `json:"budget_amount_cents,omitempty"`
	BudgetCurrency      *string                  `json:"budget_currency,omitempty"`
	TaxLine             *string                  `json:"tax_line,omitempty"`
}

func (h *AccountingHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req createAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.Account.Create().
		SetAccountNumber(req.AccountNumber).
		SetName(req.Name).
		SetAccountType(account.AccountType(req.AccountType)).
		SetAccountSubtype(account.AccountSubtype(req.AccountSubtype)).
		SetDepth(req.Depth).
		SetNormalBalance(account.NormalBalance(req.NormalBalance)).
		SetStatus(account.Status(req.Status)).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(account.Source(audit.Source))

	if req.Description != nil {
		builder.SetNillableDescription(req.Description)
	}
	if req.ParentAccountID != nil {
		builder.SetNillableParentAccountID(req.ParentAccountID)
	}
	if req.Dimensions != nil {
		builder.SetDimensions(req.Dimensions)
	}
	if req.IsHeader != nil {
		builder.SetIsHeader(*req.IsHeader)
	}
	if req.IsSystem != nil {
		builder.SetIsSystem(*req.IsSystem)
	}
	if req.AllowsDirectPosting != nil {
		builder.SetAllowsDirectPosting(*req.AllowsDirectPosting)
	}
	if req.IsTrustAccount != nil {
		builder.SetIsTrustAccount(*req.IsTrustAccount)
	}
	if req.TrustType != nil {
		tt := account.TrustType(*req.TrustType)
		builder.SetNillableTrustType(&tt)
	}
	if req.BudgetAmountCents != nil {
		builder.SetNillableBudgetAmountAmountCents(req.BudgetAmountCents)
	}
	if req.BudgetCurrency != nil {
		builder.SetNillableBudgetAmountCurrency(req.BudgetCurrency)
	}
	if req.TaxLine != nil {
		builder.SetNillableTaxLine(req.TaxLine)
	}
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	a, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *AccountingHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	a, err := h.client.Account.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *AccountingHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.Account.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Asc(account.FieldAccountNumber)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type updateAccountRequest struct {
	Name                *string                  `json:"name,omitempty"`
	Description         *string                  `json:"description,omitempty"`
	Status              *string                  `json:"status,omitempty"`
	AllowsDirectPosting *bool                    `json:"allows_direct_posting,omitempty"`
	Dimensions          *types.AccountDimensions `json:"dimensions,omitempty"`
	BudgetAmountCents   *int64                   `json:"budget_amount_cents,omitempty"`
	BudgetCurrency      *string                  `json:"budget_currency,omitempty"`
	TaxLine             *string                  `json:"tax_line,omitempty"`
}

func (h *AccountingHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req updateAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.Account.UpdateOneID(id).
		SetUpdatedBy(audit.Actor).
		SetSource(account.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	if req.Name != nil {
		builder.SetName(*req.Name)
	}
	if req.Description != nil {
		builder.SetNillableDescription(req.Description)
	}
	if req.Status != nil {
		builder.SetStatus(account.Status(*req.Status))
	}
	if req.AllowsDirectPosting != nil {
		builder.SetAllowsDirectPosting(*req.AllowsDirectPosting)
	}
	if req.Dimensions != nil {
		builder.SetDimensions(req.Dimensions)
	}
	if req.BudgetAmountCents != nil {
		builder.SetNillableBudgetAmountAmountCents(req.BudgetAmountCents)
	}
	if req.BudgetCurrency != nil {
		builder.SetNillableBudgetAmountCurrency(req.BudgetCurrency)
	}
	if req.TaxLine != nil {
		builder.SetNillableTaxLine(req.TaxLine)
	}

	a, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// ---------------------------------------------------------------------------
// LedgerEntry (read-only)
// ---------------------------------------------------------------------------

func (h *AccountingHandler) GetLedgerEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	le, err := h.client.LedgerEntry.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, le)
}

func (h *AccountingHandler) ListLedgerEntries(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.LedgerEntry.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Desc(ledgerentry.FieldEffectiveDate)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ---------------------------------------------------------------------------
// JournalEntry
// ---------------------------------------------------------------------------

type createJournalEntryRequest struct {
	EntryDate   time.Time          `json:"entry_date"`
	PostedDate  time.Time          `json:"posted_date"`
	Description string             `json:"description"`
	SourceType  string             `json:"source_type"`
	SourceID    *string            `json:"source_id,omitempty"`
	Status      string             `json:"status"`
	EntityID    *string            `json:"entity_id,omitempty"`
	PropertyID  *string            `json:"property_id,omitempty"`
	BatchID     *string            `json:"batch_id,omitempty"`
	Lines       []types.JournalLine `json:"lines"`
}

func (h *AccountingHandler) CreateJournalEntry(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req createJournalEntryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.JournalEntry.Create().
		SetEntryDate(req.EntryDate).
		SetPostedDate(req.PostedDate).
		SetDescription(req.Description).
		SetSourceType(journalentry.SourceType(req.SourceType)).
		SetStatus(journalentry.Status(req.Status)).
		SetLines(req.Lines).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(journalentry.Source(audit.Source))

	if req.SourceID != nil {
		builder.SetNillableSourceID(req.SourceID)
	}
	if req.EntityID != nil {
		builder.SetNillableEntityID(req.EntityID)
	}
	if req.PropertyID != nil {
		builder.SetNillablePropertyID(req.PropertyID)
	}
	if req.BatchID != nil {
		builder.SetNillableBatchID(req.BatchID)
	}
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	je, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, je)
}

func (h *AccountingHandler) GetJournalEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	je, err := h.client.JournalEntry.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, je)
}

func (h *AccountingHandler) ListJournalEntries(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.JournalEntry.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Desc(journalentry.FieldEntryDate)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// PostJournalEntry transitions a journal entry to "posted" and creates
// LedgerEntries for each line. Lines must balance (total debits == total credits).
func (h *AccountingHandler) PostJournalEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}

	je, err := h.client.JournalEntry.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	if err := ValidateTransition(schema.ValidJournalEntryTransitions, string(je.Status), "posted"); err != nil {
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		return
	}

	// Validate lines balance
	var totalDebit, totalCredit int64
	for _, line := range je.Lines {
		if line.Debit != nil {
			totalDebit += line.Debit.AmountCents
		}
		if line.Credit != nil {
			totalCredit += line.Credit.AmountCents
		}
	}
	if totalDebit != totalCredit {
		writeError(w, http.StatusBadRequest, "UNBALANCED",
			fmt.Sprintf("journal entry lines do not balance: debits=%d, credits=%d", totalDebit, totalCredit))
		return
	}

	// Use a transaction to post the journal entry and create ledger entries
	tx, err := h.client.Tx(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TX_ERROR", err.Error())
		return
	}

	now := time.Now()

	// Update journal entry status
	updated, err := tx.JournalEntry.UpdateOneID(id).
		SetStatus(journalentry.StatusPosted).
		SetPostedDate(now).
		SetUpdatedBy(audit.Actor).
		SetSource(journalentry.Source(audit.Source)).
		Save(r.Context())
	if err != nil {
		tx.Rollback()
		entErrorToHTTP(w, err)
		return
	}

	// Create ledger entries for each line
	for _, line := range je.Lines {
		acctID, err := uuid.Parse(line.AccountID)
		if err != nil {
			tx.Rollback()
			writeError(w, http.StatusBadRequest, "INVALID_ACCOUNT_ID", "invalid account_id in line: "+line.AccountID)
			return
		}

		var amountCents int64
		var entryType ledgerentry.EntryType
		if line.Debit != nil && line.Debit.AmountCents > 0 {
			amountCents = line.Debit.AmountCents
			entryType = ledgerentry.EntryTypeCharge
		} else if line.Credit != nil && line.Credit.AmountCents > 0 {
			amountCents = line.Credit.AmountCents
			entryType = ledgerentry.EntryTypePayment
		} else {
			continue
		}

		desc := line.Description
		if desc == "" {
			desc = je.Description
		}

		leBuilder := tx.LedgerEntry.Create().
			SetEntryType(entryType).
			SetAmountAmountCents(amountCents).
			SetEffectiveDate(je.EntryDate).
			SetPostedDate(now).
			SetDescription(desc).
			SetChargeCode("journal").
			SetJournalEntryID(id).
			SetAccountID(acctID).
			SetCreatedBy(audit.Actor).
			SetUpdatedBy(audit.Actor).
			SetSource(ledgerentry.Source(audit.Source))

		// Property edge is required — use journal entry's property_id if available
		if je.PropertyID != nil {
			propID, err := uuid.Parse(*je.PropertyID)
			if err == nil {
				leBuilder.SetPropertyID(propID)
			}
		}

		if audit.CorrelationID != nil {
			leBuilder.SetCorrelationID(*audit.CorrelationID)
		}

		if _, err := leBuilder.Save(r.Context()); err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "COMMIT_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *AccountingHandler) VoidJournalEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	je, err := h.client.JournalEntry.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	if err := ValidateTransition(schema.ValidJournalEntryTransitions, string(je.Status), "voided"); err != nil {
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		return
	}
	builder := h.client.JournalEntry.UpdateOneID(id).
		SetStatus(journalentry.StatusVoided).
		SetUpdatedBy(audit.Actor).
		SetSource(journalentry.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}
	updated, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ---------------------------------------------------------------------------
// BankAccount
// ---------------------------------------------------------------------------

type createBankAccountRequest struct {
	Name                  string  `json:"name"`
	AccountType           string  `json:"account_type"`
	BankName              string  `json:"bank_name"`
	RoutingNumber         *string `json:"routing_number,omitempty"`
	AccountNumberLastFour string  `json:"account_number_last_four"`
	PropertyID            *string `json:"property_id,omitempty"`
	EntityID              *string `json:"entity_id,omitempty"`
	Status                string  `json:"status"`
	IsTrust               *bool   `json:"is_trust,omitempty"`
	TrustState            *string `json:"trust_state,omitempty"`
	ComminglingAllowed    *bool   `json:"commingling_allowed,omitempty"`
	GLAccountID           string  `json:"gl_account_id"`
}

func (h *AccountingHandler) CreateBankAccount(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req createBankAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	glAcctID, err := uuid.Parse(req.GLAccountID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid gl_account_id")
		return
	}

	builder := h.client.BankAccount.Create().
		SetName(req.Name).
		SetAccountType(bankaccount.AccountType(req.AccountType)).
		SetBankName(req.BankName).
		SetAccountNumberLastFour(req.AccountNumberLastFour).
		SetStatus(bankaccount.Status(req.Status)).
		SetGlAccountID(glAcctID).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(bankaccount.Source(audit.Source))

	if req.RoutingNumber != nil {
		builder.SetNillableRoutingNumber(req.RoutingNumber)
	}
	if req.PropertyID != nil {
		builder.SetNillablePropertyID(req.PropertyID)
	}
	if req.EntityID != nil {
		builder.SetNillableEntityID(req.EntityID)
	}
	if req.IsTrust != nil {
		builder.SetIsTrust(*req.IsTrust)
	}
	if req.TrustState != nil {
		builder.SetNillableTrustState(req.TrustState)
	}
	if req.ComminglingAllowed != nil {
		builder.SetComminglingAllowed(*req.ComminglingAllowed)
	}
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	ba, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ba)
}

func (h *AccountingHandler) GetBankAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	ba, err := h.client.BankAccount.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ba)
}

func (h *AccountingHandler) ListBankAccounts(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.BankAccount.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Asc(bankaccount.FieldName)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type updateBankAccountRequest struct {
	Name               *string `json:"name,omitempty"`
	Status             *string `json:"status,omitempty"`
	RoutingNumber      *string `json:"routing_number,omitempty"`
	IsTrust            *bool   `json:"is_trust,omitempty"`
	TrustState         *string `json:"trust_state,omitempty"`
	ComminglingAllowed *bool   `json:"commingling_allowed,omitempty"`
}

func (h *AccountingHandler) UpdateBankAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req updateBankAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.BankAccount.UpdateOneID(id).
		SetUpdatedBy(audit.Actor).
		SetSource(bankaccount.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	if req.Name != nil {
		builder.SetName(*req.Name)
	}
	if req.Status != nil {
		builder.SetStatus(bankaccount.Status(*req.Status))
	}
	if req.RoutingNumber != nil {
		builder.SetNillableRoutingNumber(req.RoutingNumber)
	}
	if req.IsTrust != nil {
		builder.SetIsTrust(*req.IsTrust)
	}
	if req.TrustState != nil {
		builder.SetNillableTrustState(req.TrustState)
	}
	if req.ComminglingAllowed != nil {
		builder.SetComminglingAllowed(*req.ComminglingAllowed)
	}

	ba, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ba)
}

// ---------------------------------------------------------------------------
// Reconciliation
// ---------------------------------------------------------------------------

type createReconciliationRequest struct {
	PeriodStart                 time.Time `json:"period_start"`
	PeriodEnd                   time.Time `json:"period_end"`
	StatementBalanceAmountCents int64     `json:"statement_balance_amount_cents"`
	StatementBalanceCurrency    string    `json:"statement_balance_currency,omitempty"`
	SystemBalanceAmountCents    int64     `json:"system_balance_amount_cents"`
	SystemBalanceCurrency       string    `json:"system_balance_currency,omitempty"`
	DifferenceAmountCents       int64     `json:"difference_amount_cents"`
	DifferenceCurrency          string    `json:"difference_currency,omitempty"`
	Status                      string    `json:"status"`
	MatchedTransactionCount     int       `json:"matched_transaction_count"`
	UnmatchedTransactionCount   int       `json:"unmatched_transaction_count"`
	BankAccountID               string    `json:"bank_account_id"`
}

func (h *AccountingHandler) CreateReconciliation(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req createReconciliationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	baID, err := uuid.Parse(req.BankAccountID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid bank_account_id")
		return
	}

	builder := h.client.Reconciliation.Create().
		SetPeriodStart(req.PeriodStart).
		SetPeriodEnd(req.PeriodEnd).
		SetStatementBalanceAmountCents(req.StatementBalanceAmountCents).
		SetSystemBalanceAmountCents(req.SystemBalanceAmountCents).
		SetDifferenceAmountCents(req.DifferenceAmountCents).
		SetStatus(reconciliation.Status(req.Status)).
		SetMatchedTransactionCount(req.MatchedTransactionCount).
		SetUnmatchedTransactionCount(req.UnmatchedTransactionCount).
		SetBankAccountID(baID).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(reconciliation.Source(audit.Source))

	if req.StatementBalanceCurrency != "" {
		builder.SetStatementBalanceCurrency(req.StatementBalanceCurrency)
	}
	if req.SystemBalanceCurrency != "" {
		builder.SetSystemBalanceCurrency(req.SystemBalanceCurrency)
	}
	if req.DifferenceCurrency != "" {
		builder.SetDifferenceCurrency(req.DifferenceCurrency)
	}
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	rec, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (h *AccountingHandler) GetReconciliation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	rec, err := h.client.Reconciliation.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (h *AccountingHandler) ListReconciliations(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.Reconciliation.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Desc(reconciliation.FieldPeriodEnd)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *AccountingHandler) ApproveReconciliation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	rec, err := h.client.Reconciliation.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	if err := ValidateTransition(schema.ValidReconciliationTransitions, string(rec.Status), "approved"); err != nil {
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		return
	}
	now := time.Now()
	builder := h.client.Reconciliation.UpdateOneID(id).
		SetStatus(reconciliation.StatusApproved).
		SetApprovedBy(audit.Actor).
		SetApprovedAt(now).
		SetUpdatedBy(audit.Actor).
		SetSource(reconciliation.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}
	updated, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
