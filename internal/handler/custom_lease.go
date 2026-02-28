// Custom lease handlers — these transitions have business logic that
// cannot be purely generated from the CUE ontology.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/ent/application"
	"github.com/matthewbaird/ontology/ent/journalentry"
	"github.com/matthewbaird/ontology/ent/lease"
	"github.com/matthewbaird/ontology/ent/leasespace"
	"github.com/matthewbaird/ontology/ent/ledgerentry"
	"github.com/matthewbaird/ontology/ent/schema"
	"github.com/matthewbaird/ontology/ent/space"
	"github.com/matthewbaird/ontology/internal/event"
	"github.com/matthewbaird/ontology/internal/types"
)

// Ensure imports are used.
var (
	_ json.RawMessage
	_ = schema.ValidApplicationTransitions
	_ = schema.ValidSpaceTransitions
	_ types.TenantAttributes
)

// ApproveApplication approves a lease application, recording the decision.
func (h *LeaseHandler) ApproveApplication(w http.ResponseWriter, r *http.Request) {
	h.transitionApplication(w, r, "approved")
}

// DenyApplication denies a lease application, recording the decision.
func (h *LeaseHandler) DenyApplication(w http.ResponseWriter, r *http.Request) {
	h.transitionApplication(w, r, "denied")
}

// transitionApplication is a helper for Application transitions that also
// sets decision_by and decision_at fields.
func (h *LeaseHandler) transitionApplication(w http.ResponseWriter, r *http.Request, targetStatus string) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	a, err := h.client.Application.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	if err := ValidateTransition(schema.ValidApplicationTransitions, string(a.Status), targetStatus); err != nil {
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		return
	}
	builder := h.client.Application.UpdateOneID(id).
		SetStatus(application.Status(targetStatus)).
		SetDecisionBy(audit.Actor).
		SetDecisionAt(time.Now()).
		SetUpdatedBy(audit.Actor).
		SetSource(application.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}
	updated, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}

	// Record event after successful save.
	propID, _ := updated.QueryProperty().FirstID(r.Context())
	payload := event.ApplicationDecidedPayload{
		ApplicationID: id.String(),
		PropertyID:    uuidToString(propID),
		PersonID:      updated.ApplicantPersonID.String(),
		DecisionBy:    audit.Actor,
	}
	if targetStatus == "approved" {
		recordEvent(r.Context(), event.NewApplicationApproved(payload))
	} else {
		if updated.DecisionReason != nil {
			payload.Reason = *updated.DecisionReason
		}
		recordEvent(r.Context(), event.NewApplicationDenied(payload))
	}

	writeJSON(w, http.StatusOK, updated)
}

// SendForSignature sends a lease for electronic signature.
func (h *LeaseHandler) SendForSignature(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	current, err := h.client.Lease.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	if err := ValidateTransition(schema.ValidLeaseTransitions, string(current.Status), "pending_signature"); err != nil {
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		return
	}
	updated, err := h.client.Lease.UpdateOneID(id).
		SetStatus(lease.StatusPendingSignature).
		SetUpdatedBy(audit.Actor).
		SetSource(lease.Source(audit.Source)).
		Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// RecordNotice records a tenant's notice date on a lease.
func (h *LeaseHandler) RecordNotice(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	type noticeReq struct {
		NoticeDate *time.Time `json:"notice_date,omitempty"`
	}
	var req noticeReq
	_ = decodeJSON(r, &req)
	builder := h.client.Lease.UpdateOneID(id).
		SetUpdatedBy(audit.Actor).
		SetSource(lease.Source(audit.Source))
	if req.NoticeDate != nil {
		builder.SetNoticeDate(*req.NoticeDate)
	} else {
		builder.SetNoticeDate(time.Now())
	}
	updated, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// SearchLeases performs an advanced search across leases.
func (h *LeaseHandler) SearchLeases(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "SearchLeases not yet implemented")
}

// GetLeaseLedger returns ledger entries for a specific lease.
func (h *LeaseHandler) GetLeaseLedger(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "GetLeaseLedger not yet implemented")
}

// PostCharge posts a charge to a lease.
func (h *LeaseHandler) PostCharge(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "PostCharge not yet implemented")
}

// ApplyCredit applies a credit to a lease.
func (h *LeaseHandler) ApplyCredit(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "ApplyCredit not yet implemented")
}

// ── Command Implementations ─────────────────────────────────────────────────

// SubmitApplication creates a new application and optionally records an
// application fee as a LedgerEntry. Overrides the generated CreateApplication.
func (h *LeaseHandler) SubmitApplication(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	type submitReq struct {
		PropertyID             string `json:"property_id"`
		SpaceID                string `json:"space_id,omitempty"`
		ApplicantPersonID      string `json:"applicant_person_id"`
		DesiredMoveIn          string `json:"desired_move_in"`
		DesiredLeaseTermMonths int    `json:"desired_lease_term_months"`
		ApplicationFeeAmountCents int64  `json:"application_fee_amount_cents"`
		ApplicationFeeCurrency    string `json:"application_fee_currency,omitempty"`
	}
	var req submitReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	propID, err := uuid.Parse(req.PropertyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid property_id")
		return
	}
	applicantID, err := uuid.Parse(req.ApplicantPersonID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid applicant_person_id")
		return
	}
	moveIn, err := time.Parse(time.RFC3339, req.DesiredMoveIn)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DATE", "desired_move_in must be RFC3339")
		return
	}
	if req.DesiredLeaseTermMonths <= 0 {
		writeError(w, http.StatusBadRequest, "INVALID_TERM", "desired_lease_term_months must be > 0")
		return
	}

	currency := req.ApplicationFeeCurrency
	if currency == "" {
		currency = "USD"
	}

	tx, err := h.client.Tx(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TX_ERROR", err.Error())
		return
	}

	appBuilder := tx.Application.Create().
		SetStatus(application.StatusSubmitted).
		SetApplicantPersonID(applicantID).
		SetPropertyID(propID).
		SetDesiredMoveIn(moveIn).
		SetDesiredLeaseTermMonths(req.DesiredLeaseTermMonths).
		SetApplicationFeeAmountCents(req.ApplicationFeeAmountCents).
		SetApplicationFeeCurrency(currency).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(application.Source(audit.Source))

	if req.SpaceID != "" {
		spaceID, err := uuid.Parse(req.SpaceID)
		if err != nil {
			tx.Rollback()
			writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid space_id")
			return
		}
		appBuilder.SetSpaceID(spaceID)
	}
	if audit.CorrelationID != nil {
		appBuilder.SetCorrelationID(*audit.CorrelationID)
	}

	app, err := appBuilder.Save(r.Context())
	if err != nil {
		tx.Rollback()
		entErrorToHTTP(w, err)
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "COMMIT_ERROR", err.Error())
		return
	}

	recordEvent(r.Context(), event.NewApplicationSubmitted(event.ApplicationSubmittedPayload{
		ApplicationID: app.ID.String(),
		PropertyID:    req.PropertyID,
		SpaceID:       req.SpaceID,
		PersonID:      req.ApplicantPersonID,
		DesiredMoveIn: moveIn,
	}))

	writeJSON(w, http.StatusCreated, app)
}

// MoveInTenant executes the full move-in command:
// validate lease → set move_in_date → transition lease→active →
// transition spaces→occupied → update TenantAttributes → create deposit LedgerEntry.
func (h *LeaseHandler) MoveInTenant(w http.ResponseWriter, r *http.Request) {
	leaseID, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}

	type moveInReq struct {
		ActualMoveInDate    string `json:"actual_move_in_date"`
		InspectionCompleted bool   `json:"inspection_completed"`
		KeyHandoffNotes     string `json:"key_handoff_notes,omitempty"`
	}
	var req moveInReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	moveInDate, err := time.Parse(time.RFC3339, req.ActualMoveInDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DATE", "actual_move_in_date must be RFC3339")
		return
	}
	if !req.InspectionCompleted {
		writeError(w, http.StatusBadRequest, "INSPECTION_REQUIRED", "inspection must be completed before move-in")
		return
	}

	// Load lease with its spaces.
	l, err := h.client.Lease.Get(r.Context(), leaseID)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}

	// Lease must be pending_signature or active to move in.
	validStatuses := map[string]bool{"pending_signature": true, "active": true}
	if !validStatuses[string(l.Status)] {
		writeError(w, http.StatusConflict, "INVALID_STATE",
			fmt.Sprintf("lease status %q does not allow move-in (need pending_signature or active)", l.Status))
		return
	}

	// Find lease spaces.
	leaseSpaces, err := h.client.LeaseSpace.Query().
		Where().
		All(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	// Filter to this lease's spaces.
	var spaceIDs []uuid.UUID
	for _, ls := range leaseSpaces {
		lsLease, err := ls.QueryLease().FirstID(r.Context())
		if err == nil && lsLease == leaseID {
			spID, err := ls.QuerySpace().FirstID(r.Context())
			if err == nil {
				spaceIDs = append(spaceIDs, spID)
			}
		}
	}

	tx, err := h.client.Tx(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TX_ERROR", err.Error())
		return
	}

	// 1. Update lease: set move_in_date, transition to active if pending_signature.
	leaseBuilder := tx.Lease.UpdateOneID(leaseID).
		SetMoveInDate(moveInDate).
		SetUpdatedBy(audit.Actor).
		SetSource(lease.Source(audit.Source))
	if l.Status == lease.StatusPendingSignature {
		leaseBuilder.SetStatus(lease.StatusActive)
	}
	if audit.CorrelationID != nil {
		leaseBuilder.SetCorrelationID(*audit.CorrelationID)
	}
	updated, err := leaseBuilder.Save(r.Context())
	if err != nil {
		tx.Rollback()
		entErrorToHTTP(w, err)
		return
	}

	// 2. Transition spaces to occupied.
	var spaceNumbers []string
	for _, sid := range spaceIDs {
		sp, err := tx.Space.Get(r.Context(), sid)
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}
		if err := ValidateTransition(schema.ValidSpaceTransitions, string(sp.Status), "occupied"); err != nil {
			// Space may already be occupied — skip rather than fail.
			continue
		}
		_, err = tx.Space.UpdateOneID(sid).
			SetStatus(space.StatusOccupied).
			SetActiveLeaseID(leaseID.String()).
			SetUpdatedBy(audit.Actor).
			SetSource(space.Source(audit.Source)).
			Save(r.Context())
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}
		spaceNumbers = append(spaceNumbers, sp.SpaceNumber)
	}

	// 3. Update tenant PersonRoles with move_in_date.
	for _, roleIDStr := range l.TenantRoleIds {
		roleID, err := uuid.Parse(roleIDStr)
		if err != nil {
			continue
		}
		role, err := tx.PersonRole.Get(r.Context(), roleID)
		if err != nil {
			continue
		}
		attrs := role.Attributes
		if attrs == nil {
			attrs = &types.TenantAttributes{Type: "tenant", Standing: "good", ScreeningStatus: "pending", OccupancyStatus: "current", LiabilityStatus: "current"}
		}
		attrs.MoveInDate = &moveInDate
		_, err = tx.PersonRole.UpdateOneID(roleID).
			SetAttributes(attrs).
			SetUpdatedBy(audit.Actor).
			Save(r.Context())
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}
	}

	// 4. Create security deposit LedgerEntry.
	if l.SecurityDepositAmountCents > 0 {
		propID, _ := uuid.Parse(l.PropertyID)

		// Create a journal entry for the deposit.
		je, err := tx.JournalEntry.Create().
			SetEntryDate(moveInDate).
			SetPostedDate(time.Now()).
			SetDescription("Security deposit charge for move-in").
			SetSourceType(journalentry.SourceTypeSystem).
			SetStatus(journalentry.StatusPosted).
			SetPropertyID(l.PropertyID).
			SetLines([]types.JournalLine{{
				AccountID:   "deposit", // placeholder
				Debit:       &types.Money{AmountCents: l.SecurityDepositAmountCents, Currency: l.SecurityDepositCurrency},
				Description: "Security deposit",
			}}).
			SetCreatedBy(audit.Actor).
			SetUpdatedBy(audit.Actor).
			SetSource(journalentry.Source(audit.Source)).
			Save(r.Context())
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}

		leBuilder := tx.LedgerEntry.Create().
			SetEntryType(ledgerentry.EntryTypeDeposit).
			SetAmountAmountCents(l.SecurityDepositAmountCents).
			SetAmountCurrency(l.SecurityDepositCurrency).
			SetEffectiveDate(moveInDate).
			SetPostedDate(time.Now()).
			SetDescription("Security deposit").
			SetChargeCode("security_deposit").
			SetJournalEntryID(je.ID).
			SetPropertyID(propID).
			SetLeaseID(leaseID).
			SetCreatedBy(audit.Actor).
			SetUpdatedBy(audit.Actor).
			SetSource(ledgerentry.Source(audit.Source))
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
	spaceNumber := ""
	if len(spaceNumbers) > 0 {
		spaceNumber = spaceNumbers[0]
	}
	spaceIDStrs := make([]string, len(spaceIDs))
	for i, sid := range spaceIDs {
		spaceIDStrs[i] = sid.String()
	}
	// Find first tenant person ID.
	personID := ""
	if len(l.TenantRoleIds) > 0 {
		roleID, _ := uuid.Parse(l.TenantRoleIds[0])
		if pid, err := h.client.PersonRole.Query().Where().All(r.Context()); err == nil {
			for _, pr := range pid {
				if pr.ID == roleID {
					if prs, err := pr.QueryPerson().FirstID(r.Context()); err == nil {
						personID = prs.String()
					}
					break
				}
			}
		}
	}
	recordEvent(r.Context(), event.NewTenantMovedIn(event.TenantMovedInPayload{
		LeaseID:     leaseID.String(),
		PropertyID:  l.PropertyID,
		SpaceIDs:    spaceIDStrs,
		PersonID:    personID,
		MoveInDate:  moveInDate,
		LeaseType:   string(l.LeaseType),
		BaseRent:    types.Money{AmountCents: l.BaseRentAmountCents, Currency: l.BaseRentCurrency},
		SpaceNumber: spaceNumber,
	}))

	writeJSON(w, http.StatusOK, updated)
}

// RecordPayment records a payment on a lease: creates a JournalEntry + LedgerEntry
// and updates tenant balance/standing.
func (h *LeaseHandler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	leaseID, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}

	type paymentReq struct {
		AmountCents     int64  `json:"amount_cents"`
		Currency        string `json:"currency,omitempty"`
		PaymentMethod   string `json:"payment_method"`
		ReferenceNumber string `json:"reference_number,omitempty"`
		ReceivedDate    string `json:"received_date"`
		Memo            string `json:"memo,omitempty"`
	}
	var req paymentReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.AmountCents <= 0 {
		writeError(w, http.StatusBadRequest, "INVALID_AMOUNT", "amount_cents must be positive")
		return
	}
	receivedDate, err := time.Parse(time.RFC3339, req.ReceivedDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DATE", "received_date must be RFC3339")
		return
	}
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	l, err := h.client.Lease.Get(r.Context(), leaseID)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	if l.Status != lease.StatusActive && l.Status != lease.StatusMonthToMonthHoldover {
		writeError(w, http.StatusConflict, "INVALID_STATE",
			fmt.Sprintf("cannot record payment on lease with status %q", l.Status))
		return
	}
	propID, _ := uuid.Parse(l.PropertyID)

	tx, err := h.client.Tx(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TX_ERROR", err.Error())
		return
	}

	// Create JournalEntry for payment.
	desc := fmt.Sprintf("Payment received via %s", req.PaymentMethod)
	if req.ReferenceNumber != "" {
		desc += " ref:" + req.ReferenceNumber
	}
	je, err := tx.JournalEntry.Create().
		SetEntryDate(receivedDate).
		SetPostedDate(time.Now()).
		SetDescription(desc).
		SetSourceType(journalentry.SourceTypePayment).
		SetStatus(journalentry.StatusPosted).
		SetPropertyID(l.PropertyID).
		SetLines([]types.JournalLine{{
			AccountID:   "cash",
			Debit:       &types.Money{AmountCents: req.AmountCents, Currency: currency},
			Description: desc,
		}, {
			AccountID:   "accounts_receivable",
			Credit:      &types.Money{AmountCents: req.AmountCents, Currency: currency},
			Description: desc,
		}}).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(journalentry.Source(audit.Source)).
		Save(r.Context())
	if err != nil {
		tx.Rollback()
		entErrorToHTTP(w, err)
		return
	}

	// Find the first tenant person for the LedgerEntry person edge.
	var personID uuid.UUID
	if len(l.TenantRoleIds) > 0 {
		roleID, err := uuid.Parse(l.TenantRoleIds[0])
		if err == nil {
			if role, err := tx.PersonRole.Get(r.Context(), roleID); err == nil {
				if pid, err := role.QueryPerson().FirstID(r.Context()); err == nil {
					personID = pid
				}
			}
		}
	}

	leBuilder := tx.LedgerEntry.Create().
		SetEntryType(ledgerentry.EntryTypePayment).
		SetAmountAmountCents(req.AmountCents).
		SetAmountCurrency(currency).
		SetEffectiveDate(receivedDate).
		SetPostedDate(time.Now()).
		SetDescription(desc).
		SetChargeCode("payment").
		SetJournalEntryID(je.ID).
		SetPropertyID(propID).
		SetLeaseID(leaseID).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(ledgerentry.Source(audit.Source))
	if personID != uuid.Nil {
		leBuilder.SetPersonID(personID)
	}
	if req.Memo != "" {
		leBuilder.SetMemo(req.Memo)
	}
	if audit.CorrelationID != nil {
		leBuilder.SetCorrelationID(*audit.CorrelationID)
	}
	if _, err := leBuilder.Save(r.Context()); err != nil {
		tx.Rollback()
		entErrorToHTTP(w, err)
		return
	}

	// Update tenant standing/balance.
	var newBalance types.Money
	standing := "good"
	if len(l.TenantRoleIds) > 0 {
		roleID, err := uuid.Parse(l.TenantRoleIds[0])
		if err == nil {
			if role, err := tx.PersonRole.Get(r.Context(), roleID); err == nil {
				attrs := role.Attributes
				if attrs == nil {
					attrs = &types.TenantAttributes{Type: "tenant", Standing: "good", ScreeningStatus: "complete", OccupancyStatus: "current", LiabilityStatus: "current"}
				}
				cb := int64(0)
				if attrs.CurrentBalance != nil {
					cb = attrs.CurrentBalance.AmountCents
				}
				cb -= req.AmountCents
				attrs.CurrentBalance = &types.Money{AmountCents: cb, Currency: currency}
				newBalance = *attrs.CurrentBalance
				if cb <= 0 {
					attrs.Standing = "good"
				}
				standing = attrs.Standing
				tx.PersonRole.UpdateOneID(roleID).
					SetAttributes(attrs).
					SetUpdatedBy(audit.Actor).
					Save(r.Context())
			}
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "COMMIT_ERROR", err.Error())
		return
	}

	recordEvent(r.Context(), event.NewPaymentReceived(event.PaymentReceivedPayload{
		LeaseID:         leaseID.String(),
		PropertyID:      l.PropertyID,
		PersonID:        personID.String(),
		Amount:          types.Money{AmountCents: req.AmountCents, Currency: currency},
		PaymentMethod:   req.PaymentMethod,
		ReceivedDate:    receivedDate,
		ReferenceNumber: req.ReferenceNumber,
		NewBalance:      newBalance,
		Standing:        standing,
		JournalEntryID:  je.ID.String(),
	}))

	writeJSON(w, http.StatusOK, map[string]any{
		"journal_entry_id": je.ID,
		"lease_id":         leaseID,
		"amount_cents":     req.AmountCents,
		"new_balance":      newBalance,
		"standing":         standing,
	})
}

// ── Route-override commands ─────────────────────────────────────────────────
// RenewLease and InitiateEviction are defined as methods on LeaseHandler in
// gen_lease.go (simple transitions). The full command implementations are
// exposed as package-level functions that return http.HandlerFunc, registered
// in custom_routes.go to override the generated routes.

// RenewLeaseCommand returns an http.HandlerFunc that implements the full
// lease renewal command: create new lease, copy spaces, transition old → renewed.
func RenewLeaseCommand(client *ent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leaseID, ok := parseUUID(w, r, "id")
		if !ok {
			return
		}
		audit, ok := parseAuditContext(w, r)
		if !ok {
			return
		}

		type renewReq struct {
			NewTermStart          string `json:"new_term_start"`
			NewTermEnd            string `json:"new_term_end"`
			NewBaseRentAmountCents int64  `json:"new_base_rent_amount_cents"`
			NewBaseRentCurrency    string `json:"new_base_rent_currency,omitempty"`
			RentChangeReason       string `json:"rent_change_reason,omitempty"`
			RetainExistingCharges  *bool  `json:"retain_existing_charges,omitempty"`
		}
		var req renewReq
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
			return
		}
		termStart, err := time.Parse(time.RFC3339, req.NewTermStart)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_DATE", "new_term_start must be RFC3339")
			return
		}
		termEnd, err := time.Parse(time.RFC3339, req.NewTermEnd)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_DATE", "new_term_end must be RFC3339")
			return
		}
		currency := req.NewBaseRentCurrency
		if currency == "" {
			currency = "USD"
		}

		old, err := client.Lease.Get(r.Context(), leaseID)
		if err != nil {
			entErrorToHTTP(w, err)
			return
		}
		if err := ValidateTransition(schema.ValidLeaseTransitions, string(old.Status), "renewed"); err != nil {
			writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
			return
		}

		tx, err := client.Tx(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "TX_ERROR", err.Error())
			return
		}

		// Create new lease copying key fields from the old.
		newTerm := types.DateRange{Start: termStart, End: &termEnd}
		newLeaseBuilder := tx.Lease.Create().
			SetPropertyID(old.PropertyID).
			SetTenantRoleIds(old.TenantRoleIds).
			SetLeaseType(old.LeaseType).
			SetStatus(lease.StatusActive).
			SetLiabilityType(old.LiabilityType).
			SetTerm(&newTerm).
			SetBaseRentAmountCents(req.NewBaseRentAmountCents).
			SetBaseRentCurrency(currency).
			SetSecurityDepositAmountCents(old.SecurityDepositAmountCents).
			SetSecurityDepositCurrency(old.SecurityDepositCurrency).
			SetNoticeRequiredDays(old.NoticeRequiredDays).
			SetCreatedBy(audit.Actor).
			SetUpdatedBy(audit.Actor).
			SetSource(lease.Source(audit.Source))
		if old.GuarantorRoleIds != nil {
			newLeaseBuilder.SetGuarantorRoleIds(old.GuarantorRoleIds)
		}
		retainCharges := true
		if req.RetainExistingCharges != nil {
			retainCharges = *req.RetainExistingCharges
		}
		if retainCharges && old.RecurringCharges != nil {
			newLeaseBuilder.SetRecurringCharges(old.RecurringCharges)
		}
		if old.LateFeePolicy != nil {
			newLeaseBuilder.SetLateFeePolicy(old.LateFeePolicy)
		}
		if old.RentSchedule != nil {
			newLeaseBuilder.SetRentSchedule(old.RentSchedule)
		}
		if audit.CorrelationID != nil {
			newLeaseBuilder.SetCorrelationID(*audit.CorrelationID)
		}
		newLease, err := newLeaseBuilder.Save(r.Context())
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}

		// Copy LeaseSpace records to the new lease.
		oldSpaces, err := old.QueryLeaseSpaces().All(r.Context())
		if err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "QUERY_ERROR", "failed to query lease spaces")
			return
		}
		for _, ls := range oldSpaces {
			spID, err := ls.QuerySpace().FirstID(r.Context())
			if err != nil {
				continue
			}
			_, err = tx.LeaseSpace.Create().
				SetIsPrimary(ls.IsPrimary).
				SetRelationship(ls.Relationship).
				SetEffective(&newTerm).
				SetLeaseID(newLease.ID).
				SetSpaceID(spID).
				SetCreatedBy(audit.Actor).
				SetUpdatedBy(audit.Actor).
				SetSource(leasespace.Source(audit.Source)).
				Save(r.Context())
			if err != nil {
				tx.Rollback()
				entErrorToHTTP(w, err)
				return
			}
		}

		// Transition old lease to "renewed".
		_, err = tx.Lease.UpdateOneID(leaseID).
			SetStatus(lease.StatusRenewed).
			SetUpdatedBy(audit.Actor).
			SetSource(lease.Source(audit.Source)).
			Save(r.Context())
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}

		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "COMMIT_ERROR", err.Error())
			return
		}

		// Calculate rent change percentage.
		var rentChangePct float64
		if old.BaseRentAmountCents > 0 {
			rentChangePct = float64(req.NewBaseRentAmountCents-old.BaseRentAmountCents) / float64(old.BaseRentAmountCents) * 100
		}

		recordEvent(r.Context(), event.NewLeaseRenewed(event.LeaseRenewedPayload{
			OldLeaseID:   leaseID.String(),
			NewLeaseID:   newLease.ID.String(),
			PropertyID:   old.PropertyID,
			PreviousRent: types.Money{AmountCents: old.BaseRentAmountCents, Currency: old.BaseRentCurrency},
			NewRent:      types.Money{AmountCents: req.NewBaseRentAmountCents, Currency: currency},
			NewTerm:      newTerm,
			RentChangePct: rentChangePct,
			WithinCap:    true, // TODO: check jurisdiction rent cap
		}))

		writeJSON(w, http.StatusOK, map[string]any{
			"old_lease_id":  leaseID,
			"new_lease":     newLease,
			"rent_change_%": rentChangePct,
		})
	}
}

// InitiateEvictionCommand returns an http.HandlerFunc that implements the full
// eviction command: transition lease→eviction, update tenant standing.
func InitiateEvictionCommand(client *ent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leaseID, ok := parseUUID(w, r, "id")
		if !ok {
			return
		}
		audit, ok := parseAuditContext(w, r)
		if !ok {
			return
		}

		type evictReq struct {
			Reason           string `json:"reason"`
			ViolationDetails string `json:"violation_details,omitempty"`
			BalanceOwedCents *int64 `json:"balance_owed_cents,omitempty"`
			CureOffered      bool   `json:"cure_offered"`
			CureDeadline     string `json:"cure_deadline,omitempty"`
		}
		var req evictReq
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
			return
		}
		validReasons := map[string]bool{
			"nonpayment": true, "lease_violation": true, "nuisance": true,
			"illegal_activity": true, "owner_move_in": true, "renovation": true, "no_cause": true,
		}
		if !validReasons[req.Reason] {
			writeError(w, http.StatusBadRequest, "INVALID_REASON",
				fmt.Sprintf("reason must be one of: nonpayment, lease_violation, nuisance, illegal_activity, owner_move_in, renovation, no_cause"))
			return
		}

		l, err := client.Lease.Get(r.Context(), leaseID)
		if err != nil {
			entErrorToHTTP(w, err)
			return
		}
		if err := ValidateTransition(schema.ValidLeaseTransitions, string(l.Status), "eviction"); err != nil {
			writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
			return
		}

		tx, err := client.Tx(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "TX_ERROR", err.Error())
			return
		}

		// Transition lease to eviction.
		updated, err := tx.Lease.UpdateOneID(leaseID).
			SetStatus(lease.StatusEviction).
			SetUpdatedBy(audit.Actor).
			SetSource(lease.Source(audit.Source)).
			Save(r.Context())
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}

		// Update tenant standing to "eviction".
		personID := ""
		for _, roleIDStr := range l.TenantRoleIds {
			roleID, err := uuid.Parse(roleIDStr)
			if err != nil {
				continue
			}
			role, err := tx.PersonRole.Get(r.Context(), roleID)
			if err != nil {
				continue
			}
			attrs := role.Attributes
			if attrs == nil {
				attrs = &types.TenantAttributes{Type: "tenant", Standing: "eviction", ScreeningStatus: "complete", OccupancyStatus: "current", LiabilityStatus: "delinquent"}
			} else {
				attrs.Standing = "eviction"
			}
			tx.PersonRole.UpdateOneID(roleID).
				SetAttributes(attrs).
				SetUpdatedBy(audit.Actor).
				Save(r.Context())

			// Get person ID for event.
			if personID == "" {
				if pid, err := role.QueryPerson().FirstID(r.Context()); err == nil {
					personID = pid.String()
				}
			}
		}

		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "COMMIT_ERROR", err.Error())
			return
		}

		var balanceOwed *types.Money
		if req.BalanceOwedCents != nil {
			balanceOwed = &types.Money{AmountCents: *req.BalanceOwedCents, Currency: "USD"}
		}
		recordEvent(r.Context(), event.NewEvictionInitiated(event.EvictionInitiatedPayload{
			LeaseID:               leaseID.String(),
			PropertyID:            l.PropertyID,
			PersonID:              personID,
			Reason:                req.Reason,
			BalanceOwed:           balanceOwed,
			JustCauseJurisdiction: false, // TODO: check jurisdiction
			CurePeriodDays:        0,     // TODO: check jurisdiction
			RelocationRequired:    false, // TODO: check jurisdiction
			RightToCounsel:        false, // TODO: check jurisdiction
		}))

		writeJSON(w, http.StatusOK, updated)
	}
}

// uuidToString safely converts a uuid.UUID to string, returning "" for nil/zero.
func uuidToString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}
