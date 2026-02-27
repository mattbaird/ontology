// events/lease_events.cue
// Domain events for the leasing domain.
// Events carry what changed and why it matters — they are projections, not entity dumps.
package events

import (
	"time"

	"github.com/matthewbaird/ontology/ontology:propeller"
)

#TenantMovedIn: close({
	// Identifiers
	lease_id:    string
	property_id: string
	space_ids:   [...string]
	person_id:   string

	// Facts about what happened
	move_in_date: time.Time

	// Contextual data consumers commonly need
	lease_type:   "fixed_term" | "month_to_month" |
		"commercial_nnn" | "commercial_nn" | "commercial_n" | "commercial_gross" | "commercial_modified_gross" |
		"affordable" | "section_8" | "student" |
		"ground_lease" | "short_term" | "membership"
	base_rent:    propeller.#NonNegativeMoney
	space_number: string // denormalized for convenience
})

#PaymentReceived: close({
	lease_id:          string
	property_id:       string
	person_id:         string

	amount:            propeller.#PositiveMoney
	payment_method:    string
	received_date:     time.Time
	reference_number?: string

	// Post-payment state
	new_balance: propeller.#Money
	standing:    "good" | "late" | "collections" | "eviction"

	journal_entry_id: string // for audit trail
})

#LeaseRenewed: close({
	old_lease_id: string
	new_lease_id: string
	property_id:  string

	previous_rent:       propeller.#NonNegativeMoney
	new_rent:            propeller.#NonNegativeMoney
	new_term:            propeller.#DateRange
	rent_change_percent: float // computed, not stored on entity

	// Jurisdiction context for compliance audit
	jurisdiction_rule_ids?: [...string]
	within_cap:             bool
})

#EvictionInitiated: close({
	lease_id:    string
	property_id: string
	person_id:   string

	reason:        string
	balance_owed?: propeller.#NonNegativeMoney

	// Jurisdiction context
	just_cause_jurisdiction: bool
	cure_period_days:        int
	relocation_required:     bool
	right_to_counsel:        bool
})
