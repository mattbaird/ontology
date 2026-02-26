// ontology/person.cue
package propeller

import "time"

// ─── Person ──────────────────────────────────────────────────────────────────
// A Person is any individual who interacts with the property management system.
// Roles (tenant, owner, manager, vendor contact) are relationships, not types.

#Person: {
	id:           string & !=""
	first_name:   string & !=""
	last_name:    string & !=""
	display_name: string & !="" @display()

	date_of_birth?: time.Time @pii()
	ssn_last_four?: =~"^[0-9]{4}$" @pii()

	contact_methods: [#ContactMethod, ...#ContactMethod] // At least one required
	preferred_contact: *"email" | "email" | "sms" | "phone" | "mail" | "portal"

	// Communication preferences
	language_preference: *"en" | =~"^[a-z]{2}$"
	timezone?:           string
	do_not_contact:      bool | *false

	// Identity verification state
	identity_verified:    bool | *false
	verification_method?: "manual" | "id_check" | "credit_check" | "ssn_verify"
	verified_at?:         time.Time

	// Tags for flexible categorization
	tags?: [...string]

	audit: #AuditMetadata
}

// ─── Organization ────────────────────────────────────────────────────────────

#Organization: {
	id:         string & !=""
	legal_name: string & !="" @display()
	dba_name?:  string @display()

	org_type: "management_company" | "ownership_entity" | "vendor" |
		"corporate_tenant" | "government_agency" | "hoa" |
		"investment_fund" | "other"

	tax_id?:      string @pii()
	tax_id_type?: "ein" | "ssn" | "itin" | "foreign"

	status: "active" | "inactive" | "suspended" | "dissolved"

	address?:         #Address
	contact_methods?: [...#ContactMethod]

	// Regulatory
	state_of_incorporation?: #USState
	formation_date?:         time.Time

	// For management companies
	management_license?: string
	license_state?:      #USState
	license_expiry?:     time.Time

	audit: #AuditMetadata
}

// ─── PersonRole ──────────────────────────────────────────────────────────────

#PersonRole: {
	id:        string & !=""
	person_id: string & !=""
	role_type: "tenant" | "owner" | "property_manager" | "maintenance_tech" |
		"leasing_agent" | "accountant" | "vendor_contact" |
		"guarantor" | "emergency_contact" | "authorized_occupant" | "co_signer"

	scope_type: "organization" | "portfolio" | "property" | "building" | "space" | "lease"
	scope_id:   string & !=""

	status: "active" | "inactive" | "pending" | "terminated"

	effective: #DateRange

	// Role-specific attributes stored as structured JSON
	attributes?: #TenantAttributes | #OwnerAttributes | #ManagerAttributes | #GuarantorAttributes

	audit: #AuditMetadata
}

// Role-specific attribute sets
#TenantAttributes: {
	_type:            "tenant"
	standing:         *"good" | "good" | "late" | "collections" | "eviction"
	screening_status: "not_started" | "in_progress" | "approved" | "denied" | "conditional"
	screening_date?:  time.Time
	current_balance?: #Money
	move_in_date?:    time.Time
	move_out_date?:   time.Time
	pet_count?:       int & >=0
	vehicle_count?:   int & >=0
	occupancy_status: *"occupying" | "occupying" | "vacated" | "never_occupied"
	liability_status: *"active" | "active" | "released" | "guarantor_only"
}

#OwnerAttributes: {
	_type:                "owner"
	ownership_percent:    float & >0 & <=100
	distribution_method:  *"ach" | "ach" | "check" | "hold"
	management_fee_percent?: float & >=0 & <=100
	tax_reporting:        *"1099" | "1099" | "k1" | "none"
	reserve_amount?:      #NonNegativeMoney
}

#ManagerAttributes: {
	_type:               "manager"
	license_number?:     string
	license_state?:      =~"^[A-Z]{2}$"
	approval_limit?:     #NonNegativeMoney
	can_sign_leases:     bool | *false
	can_approve_expenses: bool | *true
}

#GuarantorAttributes: {
	_type:            "guarantor"
	guarantee_type:   "full" | "partial" | "conditional"
	guarantee_amount?: #PositiveMoney
	guarantee_term?:  #DateRange
	credit_score?:    int & >=300 & <=850
}
