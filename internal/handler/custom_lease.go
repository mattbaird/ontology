// Custom lease handlers — these transitions have business logic that
// cannot be purely generated from the CUE ontology.
package handler

import (
	"net/http"
	"time"

	"github.com/matthewbaird/ontology/ent/application"
	"github.com/matthewbaird/ontology/ent/lease"
	"github.com/matthewbaird/ontology/ent/schema"
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
	// TODO: implement advanced lease search
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "SearchLeases not yet implemented")
}

// GetLeaseLedger returns ledger entries for a specific lease.
func (h *LeaseHandler) GetLeaseLedger(w http.ResponseWriter, r *http.Request) {
	// TODO: implement lease ledger query
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "GetLeaseLedger not yet implemented")
}

// RecordPayment records a payment on a lease.
func (h *LeaseHandler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	// TODO: implement payment recording
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "RecordPayment not yet implemented")
}

// PostCharge posts a charge to a lease.
func (h *LeaseHandler) PostCharge(w http.ResponseWriter, r *http.Request) {
	// TODO: implement charge posting
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "PostCharge not yet implemented")
}

// ApplyCredit applies a credit to a lease.
func (h *LeaseHandler) ApplyCredit(w http.ResponseWriter, r *http.Request) {
	// TODO: implement credit application
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "ApplyCredit not yet implemented")
}
