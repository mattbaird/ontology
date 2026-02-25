package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/ent/application"
	"github.com/matthewbaird/ontology/ent/lease"
	"github.com/matthewbaird/ontology/ent/schema"
	"github.com/matthewbaird/ontology/internal/types"
)

// LeaseHandler implements HTTP handlers for Lease and Application.
type LeaseHandler struct {
	client *ent.Client
}

// NewLeaseHandler creates a new LeaseHandler.
func NewLeaseHandler(client *ent.Client) *LeaseHandler {
	return &LeaseHandler{client: client}
}

// ---------------------------------------------------------------------------
// Lease
// ---------------------------------------------------------------------------

type createLeaseRequest struct {
	PropertyID                 string                    `json:"property_id"`
	TenantRoleIDs              []string                  `json:"tenant_role_ids"`
	GuarantorRoleIDs           []string                  `json:"guarantor_role_ids,omitempty"`
	LeaseType                  string                    `json:"lease_type"`
	Status                     string                    `json:"status"`
	Term                       types.DateRange           `json:"term"`
	BaseRentAmountCents        int64                     `json:"base_rent_amount_cents"`
	BaseRentCurrency           string                    `json:"base_rent_currency,omitempty"`
	SecurityDepositAmountCents int64                     `json:"security_deposit_amount_cents"`
	SecurityDepositCurrency    string                    `json:"security_deposit_currency,omitempty"`
	NoticeRequiredDays         int                       `json:"notice_required_days"`
	RentSchedule               []types.RentScheduleEntry `json:"rent_schedule,omitempty"`
	RecurringCharges           []types.RecurringCharge   `json:"recurring_charges,omitempty"`
	LateFeePolicy              *types.LateFeePolicy      `json:"late_fee_policy,omitempty"`
	CAMTerms                   *types.CAMTerms           `json:"cam_terms,omitempty"`
	TenantImprovement          *types.TenantImprovement  `json:"tenant_improvement,omitempty"`
	RenewalOptions             []types.RenewalOption     `json:"renewal_options,omitempty"`
	Subsidy                    *types.SubsidyTerms       `json:"subsidy,omitempty"`
	MoveInDate                 *time.Time                `json:"move_in_date,omitempty"`
	UnitID                     *string                   `json:"unit_id,omitempty"`
}

func (h *LeaseHandler) CreateLease(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req createLeaseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.Lease.Create().
		SetPropertyID(req.PropertyID).
		SetTenantRoleIds(req.TenantRoleIDs).
		SetLeaseType(lease.LeaseType(req.LeaseType)).
		SetStatus(lease.Status(req.Status)).
		SetTerm(&req.Term).
		SetBaseRentAmountCents(req.BaseRentAmountCents).
		SetSecurityDepositAmountCents(req.SecurityDepositAmountCents).
		SetNoticeRequiredDays(req.NoticeRequiredDays).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(lease.Source(audit.Source))

	if req.BaseRentCurrency != "" {
		builder.SetBaseRentCurrency(req.BaseRentCurrency)
	}
	if req.SecurityDepositCurrency != "" {
		builder.SetSecurityDepositCurrency(req.SecurityDepositCurrency)
	}
	if len(req.GuarantorRoleIDs) > 0 {
		builder.SetGuarantorRoleIds(req.GuarantorRoleIDs)
	}
	if len(req.RentSchedule) > 0 {
		builder.SetRentSchedule(req.RentSchedule)
	}
	if len(req.RecurringCharges) > 0 {
		builder.SetRecurringCharges(req.RecurringCharges)
	}
	if req.LateFeePolicy != nil {
		builder.SetLateFeePolicy(req.LateFeePolicy)
	}
	if req.CAMTerms != nil {
		builder.SetCamTerms(req.CAMTerms)
	}
	if req.TenantImprovement != nil {
		builder.SetTenantImprovement(req.TenantImprovement)
	}
	if len(req.RenewalOptions) > 0 {
		builder.SetRenewalOptions(req.RenewalOptions)
	}
	if req.Subsidy != nil {
		builder.SetSubsidy(req.Subsidy)
	}
	if req.MoveInDate != nil {
		builder.SetNillableMoveInDate(req.MoveInDate)
	}
	if req.UnitID != nil {
		uid, err := uuid.Parse(*req.UnitID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid unit_id: "+*req.UnitID)
			return
		}
		builder.SetUnitID(uid)
	}
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	l, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (h *LeaseHandler) GetLease(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	l, err := h.client.Lease.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (h *LeaseHandler) ListLeases(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.Lease.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Desc(lease.FieldCreatedAt)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type updateLeaseRequest struct {
	PropertyID                 *string                   `json:"property_id,omitempty"`
	TenantRoleIDs              []string                  `json:"tenant_role_ids,omitempty"`
	GuarantorRoleIDs           []string                  `json:"guarantor_role_ids,omitempty"`
	LeaseType                  *string                   `json:"lease_type,omitempty"`
	Status                     *string                   `json:"status,omitempty"`
	Term                       *types.DateRange          `json:"term,omitempty"`
	BaseRentAmountCents        *int64                    `json:"base_rent_amount_cents,omitempty"`
	BaseRentCurrency           *string                   `json:"base_rent_currency,omitempty"`
	SecurityDepositAmountCents *int64                    `json:"security_deposit_amount_cents,omitempty"`
	SecurityDepositCurrency    *string                   `json:"security_deposit_currency,omitempty"`
	NoticeRequiredDays         *int                      `json:"notice_required_days,omitempty"`
	RentSchedule               []types.RentScheduleEntry `json:"rent_schedule,omitempty"`
	RecurringCharges           []types.RecurringCharge   `json:"recurring_charges,omitempty"`
	LateFeePolicy              *types.LateFeePolicy      `json:"late_fee_policy,omitempty"`
	RenewalOptions             []types.RenewalOption     `json:"renewal_options,omitempty"`
}

func (h *LeaseHandler) UpdateLease(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req updateLeaseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.Lease.UpdateOneID(id).
		SetUpdatedBy(audit.Actor).
		SetSource(lease.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	if req.PropertyID != nil {
		builder.SetPropertyID(*req.PropertyID)
	}
	if req.TenantRoleIDs != nil {
		builder.SetTenantRoleIds(req.TenantRoleIDs)
	}
	if req.GuarantorRoleIDs != nil {
		builder.SetGuarantorRoleIds(req.GuarantorRoleIDs)
	}
	if req.LeaseType != nil {
		builder.SetLeaseType(lease.LeaseType(*req.LeaseType))
	}
	if req.Status != nil {
		builder.SetStatus(lease.Status(*req.Status))
	}
	if req.Term != nil {
		builder.SetTerm(req.Term)
	}
	if req.BaseRentAmountCents != nil {
		builder.SetBaseRentAmountCents(*req.BaseRentAmountCents)
	}
	if req.BaseRentCurrency != nil {
		builder.SetBaseRentCurrency(*req.BaseRentCurrency)
	}
	if req.SecurityDepositAmountCents != nil {
		builder.SetSecurityDepositAmountCents(*req.SecurityDepositAmountCents)
	}
	if req.SecurityDepositCurrency != nil {
		builder.SetSecurityDepositCurrency(*req.SecurityDepositCurrency)
	}
	if req.NoticeRequiredDays != nil {
		builder.SetNoticeRequiredDays(*req.NoticeRequiredDays)
	}
	if req.RentSchedule != nil {
		builder.SetRentSchedule(req.RentSchedule)
	}
	if req.RecurringCharges != nil {
		builder.SetRecurringCharges(req.RecurringCharges)
	}
	if req.LateFeePolicy != nil {
		builder.SetLateFeePolicy(req.LateFeePolicy)
	}
	if req.RenewalOptions != nil {
		builder.SetRenewalOptions(req.RenewalOptions)
	}

	l, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// Lease transition helpers

func (h *LeaseHandler) transitionLease(w http.ResponseWriter, r *http.Request, targetStatus string, applyExtra func(*ent.LeaseUpdateOne)) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	l, err := h.client.Lease.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	if err := ValidateTransition(schema.ValidLeaseTransitions, string(l.Status), targetStatus); err != nil {
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		return
	}
	builder := h.client.Lease.UpdateOneID(id).
		SetStatus(lease.Status(targetStatus)).
		SetUpdatedBy(audit.Actor).
		SetSource(lease.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}
	if applyExtra != nil {
		applyExtra(builder)
	}
	updated, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *LeaseHandler) SubmitForApproval(w http.ResponseWriter, r *http.Request) {
	h.transitionLease(w, r, "pending_approval", nil)
}

func (h *LeaseHandler) ApproveLease(w http.ResponseWriter, r *http.Request) {
	h.transitionLease(w, r, "pending_signature", nil)
}

type activateLeaseRequest struct {
	MoveInDate *time.Time `json:"move_in_date,omitempty"`
}

func (h *LeaseHandler) ActivateLease(w http.ResponseWriter, r *http.Request) {
	var req activateLeaseRequest
	// Body is optional
	_ = decodeJSON(r, &req)
	h.transitionLease(w, r, "active", func(b *ent.LeaseUpdateOne) {
		if req.MoveInDate != nil {
			b.SetNillableMoveInDate(req.MoveInDate)
		}
	})
}

type terminateLeaseRequest struct {
	Reason      string     `json:"reason,omitempty"`
	MoveOutDate *time.Time `json:"move_out_date,omitempty"`
}

func (h *LeaseHandler) TerminateLease(w http.ResponseWriter, r *http.Request) {
	var req terminateLeaseRequest
	_ = decodeJSON(r, &req)
	h.transitionLease(w, r, "terminated", func(b *ent.LeaseUpdateOne) {
		if req.MoveOutDate != nil {
			b.SetNillableMoveOutDate(req.MoveOutDate)
		}
	})
}

func (h *LeaseHandler) RenewLease(w http.ResponseWriter, r *http.Request) {
	h.transitionLease(w, r, "renewed", nil)
}

func (h *LeaseHandler) StartEviction(w http.ResponseWriter, r *http.Request) {
	h.transitionLease(w, r, "eviction", nil)
}

// ---------------------------------------------------------------------------
// Application
// ---------------------------------------------------------------------------

type createApplicationRequest struct {
	ApplicantPersonID       string    `json:"applicant_person_id"`
	Status                  string    `json:"status"`
	DesiredMoveIn           time.Time `json:"desired_move_in"`
	DesiredLeaseTermMonths  int       `json:"desired_lease_term_months"`
	ApplicationFeeAmountCents int64   `json:"application_fee_amount_cents"`
	ApplicationFeeCurrency  string    `json:"application_fee_currency,omitempty"`
	PropertyID              string    `json:"property_id"`
	UnitID                  *string   `json:"unit_id,omitempty"`
}

func (h *LeaseHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req createApplicationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
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

	builder := h.client.Application.Create().
		SetApplicantPersonID(req.ApplicantPersonID).
		SetApplicantID(applicantID).
		SetStatus(application.Status(req.Status)).
		SetDesiredMoveIn(req.DesiredMoveIn).
		SetDesiredLeaseTermMonths(req.DesiredLeaseTermMonths).
		SetApplicationFeeAmountCents(req.ApplicationFeeAmountCents).
		SetPropertyID(propID).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(application.Source(audit.Source))

	if req.ApplicationFeeCurrency != "" {
		builder.SetApplicationFeeCurrency(req.ApplicationFeeCurrency)
	}
	if req.UnitID != nil {
		uid, err := uuid.Parse(*req.UnitID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid unit_id")
			return
		}
		builder.SetUnitID(uid)
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

func (h *LeaseHandler) GetApplication(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	a, err := h.client.Application.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *LeaseHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.Application.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Desc(application.FieldCreatedAt)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

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

func (h *LeaseHandler) ApproveApplication(w http.ResponseWriter, r *http.Request) {
	h.transitionApplication(w, r, "approved")
}

func (h *LeaseHandler) DenyApplication(w http.ResponseWriter, r *http.Request) {
	h.transitionApplication(w, r, "denied")
}
