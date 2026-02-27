// commands/lease_commands.cue
// CQRS write-side commands for the leasing domain.
// Commands define what you can DO — they import ontology types for field validation
// but define their own shapes, execution plans, and permission requirements.
package commands

import (
	"time"

	"github.com/matthewbaird/ontology/ontology:propeller"
)

#MoveInTenant: close({
	lease_id:             string
	actual_move_in_date:  time.Time
	key_handoff_notes?:   string
	inspection_completed: bool
	initial_meter_readings?: [...close({
		meter_id:  string
		reading:   float & >=0
		photo_id?: string
	})]

	// Execution plan:
	// 1. Validate lease is in status "pending_signature" or "active"
	// 2. Lease: set move_in_date = actual_move_in_date
	// 3. Lease: if status is "pending_signature", transition -> active
	// 4. Space(s): transition primary space(s) -> occupied
	// 5. PersonRole(tenant): set move_in_date on TenantAttributes
	// 6. Create LedgerEntries: security deposit charge, prorated first month rent
	// 7. If initial_meter_readings provided, record baseline readings
	// 8. Emit: TenantMovedIn event

	_affects:             ["lease", "space", "person_role", "ledger_entry"]
	_requires_permission: "lease:move_in"
})

#RecordPayment: close({
	lease_id:          string
	amount:            propeller.#PositiveMoney
	payment_method:    "ach" | "check" | "cash" | "money_order" | "credit_card"
	reference_number?: string
	received_date:     time.Time
	memo?:             string
	bank_account_id?:  string
	allocations?: [...close({
		charge_id: string
		amount:    propeller.#PositiveMoney
	})]

	// Execution plan:
	// 1. Create JournalEntry with lines: debit cash, credit receivable
	// 2. Create LedgerEntry(payment) linked to lease and person
	// 3. If allocations provided, apply to specific charges; else auto-allocate oldest first
	// 4. Update TenantAttributes.current_balance
	// 5. If balance reaches $0, update TenantAttributes.standing -> "good"
	// 6. Emit: PaymentReceived event

	_affects:             ["ledger_entry", "journal_entry", "person_role"]
	_requires_permission: "payment:record"
})

#RenewLease: close({
	lease_id:                   string
	new_term:                   propeller.#DateRange
	new_base_rent:              propeller.#NonNegativeMoney
	rent_change_reason?:        string
	retain_existing_charges:    *true | bool
	updated_charges?: [...close({
		charge_id?:  string // existing charge to modify, or omit for new
		charge_code: string
		description: string
		amount:      propeller.#NonNegativeMoney
		frequency:   "monthly" | "quarterly" | "annually" | "one_time"
	})]
	updated_cam_terms?:          propeller.#CAMTerms
	renewal_option_exercised?:   int // which renewal option number, if applicable

	// Execution plan:
	// 1. Validate jurisdiction constraints (rent increase cap, notice period)
	// 2. Create new Lease entity (renewal = new lease, not mutation of old)
	// 3. Copy LeaseSpace records to new lease
	// 4. Transition old lease status -> renewed
	// 5. Transition new lease status -> active (or pending_signature)
	// 6. If renewal_option_exercised, mark option as used
	// 7. Emit: LeaseRenewed event

	_affects:             ["lease", "lease_space"]
	_requires_permission: "lease:renew"
	_jurisdiction_checks: ["rent_increase_cap", "notice_period", "required_disclosure"]
})

#InitiateEviction: close({
	lease_id: string
	reason:   "nonpayment" | "lease_violation" | "nuisance" |
		"illegal_activity" | "owner_move_in" | "renovation" | "no_cause"
	violation_details?: string
	balance_owed?:      propeller.#NonNegativeMoney
	cure_offered:       bool
	cure_deadline?:     time.Time

	// Execution plan:
	// 1. Validate jurisdiction: is this a valid eviction reason? (just cause check)
	// 2. Validate jurisdiction: cure period requirements met?
	// 3. Validate jurisdiction: is there a winter moratorium?
	// 4. Transition lease status -> eviction
	// 5. Update TenantAttributes.standing -> "eviction"
	// 6. If relocation_assistance_required by jurisdiction, calculate amount
	// 7. If right_to_counsel jurisdiction, note in communications
	// 8. Emit: EvictionInitiated event

	_affects:             ["lease", "person_role"]
	_requires_permission: "lease:eviction"
	_jurisdiction_checks: ["just_cause_eviction", "eviction_procedure", "relocation_assistance",
		"right_to_counsel"]
})
