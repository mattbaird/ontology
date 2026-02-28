// Custom accounting handlers — these transitions have business logic that
// cannot be purely generated from the CUE ontology.
package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent/journalentry"
	"github.com/matthewbaird/ontology/ent/ledgerentry"
	"github.com/matthewbaird/ontology/ent/reconciliation"
	"github.com/matthewbaird/ontology/ent/schema"
	"github.com/matthewbaird/ontology/internal/event"
	"github.com/matthewbaird/ontology/internal/types"
)

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

	// Record event.
	propID := ""
	if je.PropertyID != nil {
		propID = *je.PropertyID
	}
	recordEvent(r.Context(), event.NewJournalEntryPosted(event.JournalEntryPostedPayload{
		JournalEntryID: id.String(),
		PropertyID:     propID,
		EntryDate:      je.EntryDate,
		PostedDate:     now,
		SourceType:     string(je.SourceType),
		LineCount:      len(je.Lines),
		TotalDebits:    types.Money{AmountCents: totalDebit, Currency: "USD"},
	}))

	writeJSON(w, http.StatusOK, updated)
}

// VoidJournalEntry transitions a posted journal entry to "voided".
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

// ApproveReconciliation transitions a balanced reconciliation to "approved".
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

	// Record event.
	bankAcctID, _ := rec.QueryBankAccount().FirstID(r.Context())
	recordEvent(r.Context(), event.NewReconciliationCompleted(event.ReconciliationCompletedPayload{
		ReconciliationID:  id.String(),
		BankAccountID:     uuidToString(bankAcctID),
		PeriodStart:       rec.PeriodStart,
		PeriodEnd:         rec.PeriodEnd,
		StatementBalance:  types.Money{AmountCents: rec.StatementBalanceAmountCents, Currency: rec.StatementBalanceCurrency},
		GLBalance:         types.Money{AmountCents: rec.GlBalanceAmountCents, Currency: rec.GlBalanceCurrency},
		Difference:        types.Money{},
		Status:            "approved",
		UnreconciledItems: 0,
	}))

	writeJSON(w, http.StatusOK, updated)
}

// StartReconciliation creates a reconciliation, marks matched ledger entries,
// and determines if the reconciliation is balanced or unbalanced.
func (h *AccountingHandler) StartReconciliation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}

	type reconcileReq struct {
		PeriodStart           string `json:"period_start"`
		PeriodEnd             string `json:"period_end"`
		StatementDate         string `json:"statement_date"`
		StatementAmountCents  int64  `json:"statement_balance_amount_cents"`
		StatementCurrency     string `json:"statement_balance_currency,omitempty"`
		MatchedEntries        []struct {
			LedgerEntryID string `json:"ledger_entry_id"`
			BankReference string `json:"bank_reference,omitempty"`
		} `json:"matched_entries"`
	}
	var req reconcileReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	periodStart, err := time.Parse(time.RFC3339, req.PeriodStart)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DATE", "period_start must be RFC3339")
		return
	}
	periodEnd, err := time.Parse(time.RFC3339, req.PeriodEnd)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DATE", "period_end must be RFC3339")
		return
	}
	stmtDate, err := time.Parse(time.RFC3339, req.StatementDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DATE", "statement_date must be RFC3339")
		return
	}
	currency := req.StatementCurrency
	if currency == "" {
		currency = "USD"
	}

	// Verify the bank account exists.
	_, err = h.client.BankAccount.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}

	tx, err := h.client.Tx(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TX_ERROR", err.Error())
		return
	}

	now := time.Now()

	// Calculate GL balance from matched entries.
	var glBalance int64
	for _, me := range req.MatchedEntries {
		leID, err := uuid.Parse(me.LedgerEntryID)
		if err != nil {
			tx.Rollback()
			writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid ledger_entry_id: "+me.LedgerEntryID)
			return
		}
		le, err := tx.LedgerEntry.Get(r.Context(), leID)
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}
		// Payment entries reduce the balance, charges increase it.
		switch le.EntryType {
		case ledgerentry.EntryTypePayment, ledgerentry.EntryTypeCredit, ledgerentry.EntryTypeRefund:
			glBalance += le.AmountAmountCents
		default:
			glBalance -= le.AmountAmountCents
		}
	}

	difference := req.StatementAmountCents - glBalance
	status := reconciliation.StatusBalanced
	if difference != 0 {
		status = reconciliation.StatusUnbalanced
	}

	// Create reconciliation.
	rec, err := tx.Reconciliation.Create().
		SetPeriodStart(periodStart).
		SetPeriodEnd(periodEnd).
		SetStatementDate(stmtDate).
		SetStatementBalanceAmountCents(req.StatementAmountCents).
		SetStatementBalanceCurrency(currency).
		SetGlBalanceAmountCents(glBalance).
		SetGlBalanceCurrency(currency).
		SetDifferenceAmountCents(difference).
		SetDifferenceCurrency(currency).
		SetStatus(status).
		SetUnreconciledItems(0).
		SetReconciledBy(audit.Actor).
		SetReconciledAt(now).
		SetBankAccountID(id).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(reconciliation.Source(audit.Source)).
		Save(r.Context())
	if err != nil {
		tx.Rollback()
		entErrorToHTTP(w, err)
		return
	}

	// Mark matched ledger entries as reconciled.
	for _, me := range req.MatchedEntries {
		leID, _ := uuid.Parse(me.LedgerEntryID)
		_, err = tx.LedgerEntry.UpdateOneID(leID).
			SetReconciled(true).
			SetReconciliationID(rec.ID.String()).
			SetReconciledAt(now).
			Save(r.Context())
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "COMMIT_ERROR", err.Error())
		return
	}

	recordEvent(r.Context(), event.NewReconciliationCompleted(event.ReconciliationCompletedPayload{
		ReconciliationID:  rec.ID.String(),
		BankAccountID:     id.String(),
		PeriodStart:       periodStart,
		PeriodEnd:         periodEnd,
		StatementBalance:  types.Money{AmountCents: req.StatementAmountCents, Currency: currency},
		GLBalance:         types.Money{AmountCents: glBalance, Currency: currency},
		Difference:        types.Money{AmountCents: difference, Currency: currency},
		Status:            string(status),
		UnreconciledItems: 0,
	}))

	writeJSON(w, http.StatusCreated, rec)
}
