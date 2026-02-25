// ontology/property.cue
package propeller

import "time"

// ─── Portfolio ───────────────────────────────────────────────────────────────
// The top-level organizational grouping. A management company may manage
// multiple portfolios for different ownership entities.

#Portfolio: {
	id:       string & !=""
	name:     string & !=""
	owner_id: string & !="" // Organization ID of the ownership entity

	management_type: "self_managed" | "third_party" | "hybrid"

	// Trust accounting — drives major architectural decisions downstream
	requires_trust_accounting: bool
	trust_bank_account_id?:    string

	status: "active" | "inactive" | "onboarding" | "offboarding"

	// Financial settings at portfolio level
	default_late_fee_policy?: string // Reference to fee schedule
	default_payment_methods?: [...("ach" | "credit_card" | "check" | "cash" | "money_order")]
	fiscal_year_start_month: *1 | int & >=1 & <=12 // January default

	// CONSTRAINT: Trust accounting requires a linked bank account
	if requires_trust_accounting {
		trust_bank_account_id: string & !=""
	}

	audit: #AuditMetadata
}

// ─── Property ────────────────────────────────────────────────────────────────

#Property: {
	id:           string & !=""
	portfolio_id: string & !=""
	name:         string & !=""
	address:      #Address

	property_type: "single_family" | "multi_family" | "commercial_office" |
		"commercial_retail" | "mixed_use" | "industrial" |
		"affordable_housing" | "student_housing" | "senior_living" |
		"vacation_rental" | "mobile_home_park"

	status: "active" | "inactive" | "under_renovation" | "for_sale" | "onboarding"

	// Physical
	year_built:           int & >=1800 & <=2030
	total_square_footage: float & >0
	total_units:          int & >=1
	lot_size_sqft?:       float & >0
	stories?:             int & >=1
	parking_spaces?:      int & >=0

	// Regulatory — these drive business rules across the entire system
	jurisdiction_id?:         string // Links to local ordinance rules
	rent_controlled:          bool | *false
	compliance_programs?: [...("LIHTC" | "Section8" | "HUD" | "HOME" | "RAD" | "VASH" | "PBV")]
	requires_lead_disclosure: bool | *false // Pre-1978 buildings

	// Financial — property-level overrides
	chart_of_accounts_id?: string // If different from portfolio default
	bank_account_id?:      string // If different from portfolio trust account

	// Insurance
	insurance_policy_number?: string
	insurance_expiry?:        time.Time

	// CONSTRAINTS:

	// Single-family = exactly 1 unit
	if property_type == "single_family" {
		total_units: 1
	}

	// Affordable housing MUST specify compliance programs
	if property_type == "affordable_housing" {
		compliance_programs: [_, ...] // At least one
	}

	// Rent control requires jurisdiction
	if rent_controlled {
		jurisdiction_id: string & !=""
	}

	// Pre-1978 buildings require lead disclosure
	if year_built < 1978 {
		requires_lead_disclosure: true
	}

	audit: #AuditMetadata
}

// ─── Unit ────────────────────────────────────────────────────────────────────

#Unit: {
	id:          string & !=""
	property_id: string & !=""
	unit_number: string & !="" // "101", "A", "Suite 200", etc.

	unit_type: "residential" | "commercial_office" | "commercial_retail" |
		"storage" | "parking" | "common_area"

	status: "vacant" | "occupied" | "notice_given" | "make_ready" |
		"down" | "model" | "reserved"

	// Physical
	square_footage: float & >0
	bedrooms?:      int & >=0
	bathrooms?:     float & >=0
	floor?:         int

	// Features
	amenities?:     [...string]
	floor_plan?:    string // Reference to floor plan template
	ada_accessible: bool | *false
	pet_friendly:   bool | *true
	furnished:      bool | *false

	// Financial
	market_rent?: #NonNegativeMoney

	// Active lease — computed from relationship traversal
	active_lease_id?: string

	// For affordable housing — unit-level income restrictions
	ami_restriction?: int & >=0 & <=150 // % of Area Median Income

	// CONSTRAINTS:

	// Occupied units MUST have an active lease
	if status == "occupied" {
		active_lease_id: string & !=""
	}

	// Residential units should have bedroom/bathroom counts
	if unit_type == "residential" {
		bedrooms:  int
		bathrooms: float
	}

	// Parking/storage don't have bedrooms
	if unit_type == "parking" || unit_type == "storage" {
		bedrooms:  0 | *0
		bathrooms: 0 | *0
	}

	audit: #AuditMetadata
}
