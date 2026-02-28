package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/internal/types"
)

// DomainEvent carries the canonical shape of every domain event.
type DomainEvent struct {
	ID               string
	EventType        string
	OccurredAt       time.Time
	AffectedEntities []types.SourceRef
	Summary          string
	Category         string // "lease", "payment", "property", "accounting", "application"
	Weight           string // "critical", "major", "minor", "info"
	Polarity         string // "positive", "negative", "neutral"
	Payload          json.RawMessage
}

func newID() string { return uuid.New().String() }

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// ── Lease events ─────────────────────────────────────────────────────────────

// TenantMovedInPayload carries event-specific data for TenantMovedIn.
type TenantMovedInPayload struct {
	LeaseID     string     `json:"lease_id"`
	PropertyID  string     `json:"property_id"`
	SpaceIDs    []string   `json:"space_ids"`
	PersonID    string     `json:"person_id"`
	MoveInDate  time.Time  `json:"move_in_date"`
	LeaseType   string     `json:"lease_type"`
	BaseRent    types.Money `json:"base_rent"`
	SpaceNumber string     `json:"space_number"`
}

func NewTenantMovedIn(p TenantMovedInPayload) DomainEvent {
	refs := []types.SourceRef{
		{EntityType: "lease", EntityID: p.LeaseID, Role: "subject"},
		{EntityType: "property", EntityID: p.PropertyID, Role: "context"},
		{EntityType: "person", EntityID: p.PersonID, Role: "related"},
	}
	for _, sid := range p.SpaceIDs {
		refs = append(refs, types.SourceRef{EntityType: "space", EntityID: sid, Role: "target"})
	}
	return DomainEvent{
		ID:               newID(),
		EventType:        "tenant_moved_in",
		OccurredAt:       time.Now(),
		AffectedEntities: refs,
		Summary:          fmt.Sprintf("Tenant moved in to lease %s", p.LeaseID[:8]),
		Category:         "lease",
		Weight:           "major",
		Polarity:         "positive",
		Payload:          mustJSON(p),
	}
}

// PaymentReceivedPayload carries event-specific data for PaymentReceived.
type PaymentReceivedPayload struct {
	LeaseID        string      `json:"lease_id"`
	PropertyID     string      `json:"property_id"`
	PersonID       string      `json:"person_id"`
	Amount         types.Money `json:"amount"`
	PaymentMethod  string      `json:"payment_method"`
	ReceivedDate   time.Time   `json:"received_date"`
	ReferenceNumber string    `json:"reference_number,omitempty"`
	NewBalance     types.Money `json:"new_balance"`
	Standing       string      `json:"standing"`
	JournalEntryID string      `json:"journal_entry_id"`
}

func NewPaymentReceived(p PaymentReceivedPayload) DomainEvent {
	return DomainEvent{
		ID:        newID(),
		EventType: "payment_received",
		OccurredAt: time.Now(),
		AffectedEntities: []types.SourceRef{
			{EntityType: "lease", EntityID: p.LeaseID, Role: "subject"},
			{EntityType: "property", EntityID: p.PropertyID, Role: "context"},
			{EntityType: "person", EntityID: p.PersonID, Role: "related"},
		},
		Summary:  fmt.Sprintf("Payment of %d cents received on lease %s", p.Amount.AmountCents, p.LeaseID[:8]),
		Category: "payment",
		Weight:   "minor",
		Polarity: "positive",
		Payload:  mustJSON(p),
	}
}

// LeaseRenewedPayload carries event-specific data for LeaseRenewed.
type LeaseRenewedPayload struct {
	OldLeaseID       string          `json:"old_lease_id"`
	NewLeaseID       string          `json:"new_lease_id"`
	PropertyID       string          `json:"property_id"`
	PreviousRent     types.Money     `json:"previous_rent"`
	NewRent          types.Money     `json:"new_rent"`
	NewTerm          types.DateRange `json:"new_term"`
	RentChangePct    float64         `json:"rent_change_percent"`
	WithinCap        bool            `json:"within_cap"`
}

func NewLeaseRenewed(p LeaseRenewedPayload) DomainEvent {
	return DomainEvent{
		ID:        newID(),
		EventType: "lease_renewed",
		OccurredAt: time.Now(),
		AffectedEntities: []types.SourceRef{
			{EntityType: "lease", EntityID: p.OldLeaseID, Role: "subject"},
			{EntityType: "lease", EntityID: p.NewLeaseID, Role: "target"},
			{EntityType: "property", EntityID: p.PropertyID, Role: "context"},
		},
		Summary:  fmt.Sprintf("Lease %s renewed as %s", p.OldLeaseID[:8], p.NewLeaseID[:8]),
		Category: "lease",
		Weight:   "major",
		Polarity: "positive",
		Payload:  mustJSON(p),
	}
}

// EvictionInitiatedPayload carries event-specific data for EvictionInitiated.
type EvictionInitiatedPayload struct {
	LeaseID              string      `json:"lease_id"`
	PropertyID           string      `json:"property_id"`
	PersonID             string      `json:"person_id"`
	Reason               string      `json:"reason"`
	BalanceOwed          *types.Money `json:"balance_owed,omitempty"`
	JustCauseJurisdiction bool       `json:"just_cause_jurisdiction"`
	CurePeriodDays       int         `json:"cure_period_days"`
	RelocationRequired   bool        `json:"relocation_required"`
	RightToCounsel       bool        `json:"right_to_counsel"`
}

func NewEvictionInitiated(p EvictionInitiatedPayload) DomainEvent {
	return DomainEvent{
		ID:        newID(),
		EventType: "eviction_initiated",
		OccurredAt: time.Now(),
		AffectedEntities: []types.SourceRef{
			{EntityType: "lease", EntityID: p.LeaseID, Role: "subject"},
			{EntityType: "property", EntityID: p.PropertyID, Role: "context"},
			{EntityType: "person", EntityID: p.PersonID, Role: "related"},
		},
		Summary:  fmt.Sprintf("Eviction initiated on lease %s for %s", p.LeaseID[:8], p.Reason),
		Category: "lease",
		Weight:   "critical",
		Polarity: "negative",
		Payload:  mustJSON(p),
	}
}

// ── Application events ───────────────────────────────────────────────────────

// ApplicationSubmittedPayload carries event-specific data for ApplicationSubmitted.
type ApplicationSubmittedPayload struct {
	ApplicationID string    `json:"application_id"`
	PropertyID    string    `json:"property_id"`
	SpaceID       string    `json:"space_id,omitempty"`
	PersonID      string    `json:"person_id"`
	DesiredMoveIn time.Time `json:"desired_move_in"`
}

func NewApplicationSubmitted(p ApplicationSubmittedPayload) DomainEvent {
	refs := []types.SourceRef{
		{EntityType: "application", EntityID: p.ApplicationID, Role: "subject"},
		{EntityType: "property", EntityID: p.PropertyID, Role: "context"},
		{EntityType: "person", EntityID: p.PersonID, Role: "related"},
	}
	if p.SpaceID != "" {
		refs = append(refs, types.SourceRef{EntityType: "space", EntityID: p.SpaceID, Role: "target"})
	}
	return DomainEvent{
		ID:               newID(),
		EventType:        "application_submitted",
		OccurredAt:       time.Now(),
		AffectedEntities: refs,
		Summary:          fmt.Sprintf("Application submitted for property %s", p.PropertyID[:8]),
		Category:         "application",
		Weight:           "minor",
		Polarity:         "neutral",
		Payload:          mustJSON(p),
	}
}

// ApplicationDecidedPayload carries event-specific data for application approval/denial.
type ApplicationDecidedPayload struct {
	ApplicationID string `json:"application_id"`
	PropertyID    string `json:"property_id"`
	PersonID      string `json:"person_id"`
	Decision      string `json:"decision"` // "approved" or "denied"
	DecisionBy    string `json:"decision_by"`
	Reason        string `json:"reason,omitempty"`
}

func NewApplicationApproved(p ApplicationDecidedPayload) DomainEvent {
	p.Decision = "approved"
	return DomainEvent{
		ID:        newID(),
		EventType: "application_approved",
		OccurredAt: time.Now(),
		AffectedEntities: []types.SourceRef{
			{EntityType: "application", EntityID: p.ApplicationID, Role: "subject"},
			{EntityType: "property", EntityID: p.PropertyID, Role: "context"},
			{EntityType: "person", EntityID: p.PersonID, Role: "related"},
		},
		Summary:  fmt.Sprintf("Application %s approved", p.ApplicationID[:8]),
		Category: "application",
		Weight:   "major",
		Polarity: "positive",
		Payload:  mustJSON(p),
	}
}

func NewApplicationDenied(p ApplicationDecidedPayload) DomainEvent {
	p.Decision = "denied"
	return DomainEvent{
		ID:        newID(),
		EventType: "application_denied",
		OccurredAt: time.Now(),
		AffectedEntities: []types.SourceRef{
			{EntityType: "application", EntityID: p.ApplicationID, Role: "subject"},
			{EntityType: "property", EntityID: p.PropertyID, Role: "context"},
			{EntityType: "person", EntityID: p.PersonID, Role: "related"},
		},
		Summary:  fmt.Sprintf("Application %s denied", p.ApplicationID[:8]),
		Category: "application",
		Weight:   "major",
		Polarity: "negative",
		Payload:  mustJSON(p),
	}
}

// ── Property events ──────────────────────────────────────────────────────────

// PropertyOnboardedPayload carries event-specific data for PropertyOnboarded.
type PropertyOnboardedPayload struct {
	PropertyID    string        `json:"property_id"`
	PortfolioID   string        `json:"portfolio_id"`
	PropertyType  string        `json:"property_type"`
	Address       types.Address `json:"address"`
	SpaceCount    int           `json:"space_count"`
}

func NewPropertyOnboarded(p PropertyOnboardedPayload) DomainEvent {
	return DomainEvent{
		ID:        newID(),
		EventType: "property_onboarded",
		OccurredAt: time.Now(),
		AffectedEntities: []types.SourceRef{
			{EntityType: "property", EntityID: p.PropertyID, Role: "subject"},
			{EntityType: "portfolio", EntityID: p.PortfolioID, Role: "context"},
		},
		Summary:  fmt.Sprintf("Property %s onboarded with %d spaces", p.PropertyID[:8], p.SpaceCount),
		Category: "property",
		Weight:   "major",
		Polarity: "positive",
		Payload:  mustJSON(p),
	}
}

// ── Accounting events ────────────────────────────────────────────────────────

// JournalEntryPostedPayload carries event-specific data for JournalEntryPosted.
type JournalEntryPostedPayload struct {
	JournalEntryID string      `json:"journal_entry_id"`
	PropertyID     string      `json:"property_id,omitempty"`
	EntryDate      time.Time   `json:"entry_date"`
	PostedDate     time.Time   `json:"posted_date"`
	SourceType     string      `json:"source_type"`
	LineCount      int         `json:"line_count"`
	TotalDebits    types.Money `json:"total_debits"`
}

func NewJournalEntryPosted(p JournalEntryPostedPayload) DomainEvent {
	refs := []types.SourceRef{
		{EntityType: "journal_entry", EntityID: p.JournalEntryID, Role: "subject"},
	}
	if p.PropertyID != "" {
		refs = append(refs, types.SourceRef{EntityType: "property", EntityID: p.PropertyID, Role: "context"})
	}
	return DomainEvent{
		ID:               newID(),
		EventType:        "journal_entry_posted",
		OccurredAt:       time.Now(),
		AffectedEntities: refs,
		Summary:          fmt.Sprintf("Journal entry %s posted with %d lines", p.JournalEntryID[:8], p.LineCount),
		Category:         "accounting",
		Weight:           "minor",
		Polarity:         "neutral",
		Payload:          mustJSON(p),
	}
}

// ReconciliationCompletedPayload carries event-specific data for ReconciliationCompleted.
type ReconciliationCompletedPayload struct {
	ReconciliationID string      `json:"reconciliation_id"`
	BankAccountID    string      `json:"bank_account_id"`
	PeriodStart      time.Time   `json:"period_start"`
	PeriodEnd        time.Time   `json:"period_end"`
	StatementBalance types.Money `json:"statement_balance"`
	GLBalance        types.Money `json:"gl_balance"`
	Difference       types.Money `json:"difference"`
	Status           string      `json:"status"`
	UnreconciledItems int        `json:"unreconciled_items"`
}

func NewReconciliationCompleted(p ReconciliationCompletedPayload) DomainEvent {
	polarity := "positive"
	if p.Status == "unbalanced" {
		polarity = "negative"
	}
	return DomainEvent{
		ID:        newID(),
		EventType: "reconciliation_completed",
		OccurredAt: time.Now(),
		AffectedEntities: []types.SourceRef{
			{EntityType: "reconciliation", EntityID: p.ReconciliationID, Role: "subject"},
			{EntityType: "bank_account", EntityID: p.BankAccountID, Role: "context"},
		},
		Summary:  fmt.Sprintf("Reconciliation %s completed: %s", p.ReconciliationID[:8], p.Status),
		Category: "accounting",
		Weight:   "minor",
		Polarity: polarity,
		Payload:  mustJSON(p),
	}
}
