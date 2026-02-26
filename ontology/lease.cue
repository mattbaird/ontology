// ontology/lease.cue
package propeller

import "time"

// ─── Lease ───────────────────────────────────────────────────────────────────

#Lease: {
	id:          string & !=""
	property_id: string & !="" // Denormalized for query efficiency

	// Tenant references — via PersonRole, not directly to Person
	tenant_role_ids:     [...string]
	guarantor_role_ids?: [...string]

	lease_type: "fixed_term" | "month_to_month" |
		"commercial_nnn" | "commercial_nn" | "commercial_n" | "commercial_gross" | "commercial_modified_gross" |
		"affordable" | "section_8" | "student" |
		"ground_lease" | "short_term" | "membership"

	status: "draft" | "pending_approval" | "pending_signature" | "active" |
		"expired" | "month_to_month_holdover" | "renewed" |
		"terminated" | "eviction"

	description?:       string @text()
	// Liability
	liability_type: *"joint_and_several" | "joint_and_several" | "individual" | "by_the_bed" | "proportional"

	// Term
	term: #DateRange
	lease_commencement_date?: time.Time
	rent_commencement_date?:  time.Time

	// Financial — base rent
	base_rent:        #NonNegativeMoney
	security_deposit: #NonNegativeMoney @sensitive()

	// Rent schedule
	rent_schedule?: [...#RentScheduleEntry]

	// Recurring charges beyond base rent
	recurring_charges?: [...#RecurringCharge]

	// Late fee policy
	late_fee_policy?: #LateFeePolicy

	// Commercial-specific
	cam_terms?:          #CAMTerms
	tenant_improvement?: #TenantImprovement
	renewal_options?:    [...#RenewalOption]
	usage_charges?:      [...#UsageBasedCharge]
	percentage_rent?:    #PercentageRent
	expansion_rights?:   [...#ExpansionRight]
	contraction_rights?: [...#ContractionRight]

	// Affordable housing-specific
	subsidy?: #SubsidyTerms

	// Move-in / move-out
	move_in_date?:       time.Time
	move_out_date?:      time.Time
	notice_date?:        time.Time
	notice_required_days: *30 | int & >=0

	// Short-term specific
	check_in_time?:       string
	check_out_time?:      string
	cleaning_fee?:        #NonNegativeMoney
	platform_booking_id?: string

	// Membership specific
	membership_tier?: "hot_desk" | "dedicated_desk" | "office" | "suite" | "virtual"

	// Sublease
	parent_lease_id?:  string
	is_sublease:       bool | *false
	sublease_billing:  *"through_master_tenant" | "through_master_tenant" | "direct_to_landlord"

	// Signing
	signing_method?: "electronic" | "wet_ink" | "both"
	signed_at?:      time.Time
	document_id?:    string

	// CONSTRAINTS:

	// Fixed-term and student leases must have an end date
	if lease_type == "fixed_term" || lease_type == "student" {
		term: end: time.Time
	}

	// Commercial lease types require CAM terms
	if lease_type == "commercial_nnn" || lease_type == "commercial_nn" || lease_type == "commercial_n" || lease_type == "commercial_gross" || lease_type == "commercial_modified_gross" {
		cam_terms: #CAMTerms
	}

	// NNN leases require all three pass-throughs
	if lease_type == "commercial_nnn" {
		cam_terms: includes_property_tax: true
		cam_terms: includes_insurance:    true
		cam_terms: includes_utilities:    true
	}

	// NN leases require tax and insurance
	if lease_type == "commercial_nn" {
		cam_terms: includes_property_tax: true
		cam_terms: includes_insurance:    true
	}

	// N leases require tax
	if lease_type == "commercial_n" {
		cam_terms: includes_property_tax: true
	}

	// Section 8 leases require subsidy terms
	if lease_type == "section_8" {
		subsidy: #SubsidyTerms
	}

	// Active leases must have a move-in date
	if status == "active" {
		move_in_date: time.Time
	}

	// Active/expired/renewed leases must be signed
	if status == "active" || status == "expired" || status == "renewed" {
		signed_at: time.Time
	}

	// Subleases must reference a parent lease
	if is_sublease {
		parent_lease_id: string & !=""
	}

	// CONSTRAINT: rent_commencement_date must be on or after lease_commencement_date
	// (enforced at runtime via Ent hooks)

	audit: #AuditMetadata
}

#RentScheduleEntry: {
	effective_period: #DateRange
	fixed_amount?:    #NonNegativeMoney
	adjustment?:      #RentAdjustment
	description:      string & !=""
	charge_code:      string & !=""
}

#RecurringCharge: {
	id:               string & !=""
	charge_code:      string & !=""
	description:      string & !=""
	amount:           #NonNegativeMoney
	frequency:        "monthly" | "quarterly" | "annually" | "one_time"
	effective_period: #DateRange
	taxable:          bool | *false
	space_id?:        string
}

#LateFeePolicy: {
	grace_period_days: *5 | int & >=0
	fee_type:          "flat" | "percent" | "per_day" | "tiered"
	flat_amount?:      #NonNegativeMoney
	percent?:          float & >0 & <=100
	per_day_amount?:   #NonNegativeMoney
	max_fee?:          #NonNegativeMoney
	tiers?: [...{
		days_late_min: int & >=0
		days_late_max: int
		amount:        #NonNegativeMoney
	}]
}

#CAMTerms: {
	reconciliation_type:    "estimated_with_annual_reconciliation" | "fixed" | "actual"
	pro_rata_share_percent: float & >0 & <=100
	estimated_monthly_cam:  #NonNegativeMoney
	annual_cap?:            #NonNegativeMoney
	base_year?:             int
	includes_property_tax:  bool
	includes_insurance:     bool
	includes_utilities:     bool
	excluded_categories?:   [...string]
	base_year_expenses?:    #NonNegativeMoney
	expense_stop?:          #NonNegativeMoney
	category_terms?:        [...#CAMCategoryTerms]

	// CONSTRAINTS:
	// If reconciliation_type is "fixed", annual_cap is not allowed
	// base_year and expense_stop are mutually exclusive
	// (both enforced at runtime via Ent hooks)
}

#TenantImprovement: {
	allowance:                #NonNegativeMoney
	amortized:                bool | *false
	amortization_term_months?: int & >0
	interest_rate_percent?:   float & >=0
	completion_deadline?:     time.Time
}

#RenewalOption: {
	option_number:       int & >=1
	term_months:         int & >0
	rent_adjustment:     "fixed" | "cpi" | "percent_increase" | "market"
	fixed_rent?:         #NonNegativeMoney
	percent_increase?:   float & >=0
	notice_required_days: *90 | int & >=0
	must_exercise_by?:   time.Time
	cpi_floor?:          float & >=0
	cpi_ceiling?:        float & >0
}

#SubsidyTerms: {
	program:                  "section_8" | "pbv" | "vash" | "home" | "lihtc"
	housing_authority:        string & !=""
	hap_contract_id?:         string
	contract_rent:            #NonNegativeMoney
	tenant_portion:           #NonNegativeMoney
	subsidy_portion:          #NonNegativeMoney
	utility_allowance:        #NonNegativeMoney
	annual_recert_date?:      time.Time
	income_limit_ami_percent: int & >0 & <=150
}

// ─── New Value Types ─────────────────────────────────────────────────────────

#UsageBasedCharge: {
	id:                string & !=""
	charge_code:       string & !=""
	description:       string & !=""
	unit_of_measure:   "kwh" | "gallon" | "cubic_foot" | "therm" | "hour" | "gb"
	rate_per_unit:     #PositiveMoney
	meter_id?:         string
	billing_frequency: "monthly" | "quarterly"
	cap?:              #NonNegativeMoney
	space_id?:         string
}

#PercentageRent: {
	rate:                   float & >0 & <=100
	breakpoint_type:        "natural" | "artificial"
	natural_breakpoint?:    #NonNegativeMoney
	artificial_breakpoint?: #NonNegativeMoney
	reporting_frequency:    "monthly" | "quarterly" | "annually"
	audit_rights:           bool | *true
}

#RentAdjustment: {
	method:      "cpi" | "fixed_percent" | "fixed_amount_increase" | "market_review"
	base_amount: #NonNegativeMoney

	// For CPI
	cpi_index?:   "CPI-U" | "CPI-W" | "regional"
	cpi_floor?:   float & >=0
	cpi_ceiling?: float & >0

	// For fixed percent
	percent_increase?: float & >0

	// For fixed amount
	amount_increase?: #PositiveMoney

	// For market review
	market_review_mechanism?: string
}

#ExpansionRight: {
	type:                 "first_right_of_refusal" | "first_right_to_negotiate" | "must_take" | "option"
	target_space_ids:     [...string]
	exercise_deadline?:   time.Time
	terms?:               string
	notice_required_days: int & >=0
}

#ContractionRight: {
	minimum_retained_sqft:  float & >0
	earliest_exercise_date: time.Time
	penalty?:               #NonNegativeMoney
	notice_required_days:   int & >=0
}

#CAMCategoryTerms: {
	category: "property_tax" | "insurance" | "utilities" | "janitorial" |
		"landscaping" | "security" | "management_fee" | "repairs" |
		"snow_removal" | "elevator" | "hvac_maintenance" | "other"
	tenant_pays:  bool
	landlord_cap?: #NonNegativeMoney
	tenant_cap?:   #NonNegativeMoney
	escalation?:   float
}

// ─── LeaseSpace ──────────────────────────────────────────────────────────────
// First-class M2M join between Lease and Space.

#LeaseSpace: {
	id:       string & !=""
	lease_id: string & !=""
	space_id: string & !=""

	is_primary: bool | *true

	relationship: "primary" | "expansion" | "sublease" | "shared_access" |
		"parking" | "storage" | "loading_dock" | "rooftop" |
		"patio" | "signage" | "included" | "membership"

	effective: #DateRange

	square_footage_leased?: float & >0

	audit: #AuditMetadata
}

// ─── Application ─────────────────────────────────────────────────────────────

#Application: {
	id:                  string & !=""
	property_id:         string & !=""
	space_id?:           string
	applicant_person_id: string & !=""

	status: "submitted" | "screening" | "under_review" | "approved" |
		"conditionally_approved" | "denied" | "withdrawn" | "expired"

	desired_move_in:           time.Time
	desired_lease_term_months: int & >0

	// Screening
	screening_request_id?: string
	screening_completed?:  time.Time
	credit_score?:         int & >=300 & <=850
	background_clear:      bool | *false
	income_verified:       bool | *false
	income_to_rent_ratio?: float & >=0

	// Decision
	decision_by?:     string
	decision_at?:     time.Time
	decision_reason?: string @text()
	conditions?:      [...string]

	// Financial
	application_fee: #NonNegativeMoney
	fee_paid:        bool | *false

	// CONSTRAINTS:

	// Decisions require decision metadata
	if status == "approved" || status == "conditionally_approved" || status == "denied" {
		decision_by: string & !=""
		decision_at: time.Time
	}

	// Denials require a reason (fair housing compliance)
	if status == "denied" {
		decision_reason: string & !=""
	}

	audit: #AuditMetadata
}
