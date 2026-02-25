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
	writeJSON(w, http.StatusOK, updated)
}
