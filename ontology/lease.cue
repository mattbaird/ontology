// ontology/lease.cue
package propeller

import "time"

// ─── Lease ───────────────────────────────────────────────────────────────────

#Lease: {
	id:          string & !=""
	unit_id:     string & !=""
	property_id: string & !="" // Denormalized for query efficiency

	// Tenant references — via PersonRole, not directly to Person
	tenant_role_ids:     [...string]
	guarantor_role_ids?: [...string]

	lease_type: "fixed_term" | "month_to_month" |
		"commercial_nnn" | "commercial_gross" | "commercial_modified_gross" |
		"affordable" | "section_8" | "student"

	status: "draft" | "pending_approval" | "pending_signature" | "active" |
		"expired" | "month_to_month_holdover" | "renewed" |
		"terminated" | "eviction"

	// Term
	term: #DateRange

	// Financial — base rent
	base_rent:        #NonNegativeMoney
	security_deposit: #NonNegativeMoney

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

	// Affordable housing-specific
	subsidy?: #SubsidyTerms

	// Move-in / move-out
	move_in_date?:       time.Time
	move_out_date?:      time.Time
	notice_date?:        time.Time
	notice_required_days: *30 | int & >=0

	// Signing
	signing_method?: "electronic" | "wet_ink" | "both"
	signed_at?:      time.Time
	document_id?:    string

	audit: #AuditMetadata
}

#RentScheduleEntry: {
	effective_period: #DateRange
	amount:           #NonNegativeMoney
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

// ─── Application ─────────────────────────────────────────────────────────────

#Application: {
	id:                  string & !=""
	property_id:         string & !=""
	unit_id?:            string
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
	decision_reason?: string
	conditions?:      [...string]

	// Financial
	application_fee: #NonNegativeMoney
	fee_paid:        bool | *false

	audit: #AuditMetadata
}
