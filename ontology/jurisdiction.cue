// ontology/jurisdiction.cue
package propeller

import (
	"strings"
	"time"
)

// ─── Named Enum Types ───────────────────────────────────────────────────────

#JurisdictionType: "federal" | "state" | "county" | "city" |
	"special_district" | "unincorporated_area"

#JurisdictionStatus: "active" | "dissolved" | "merged" | "pending"

#JurisdictionRuleType: "security_deposit_limit" | "notice_period" | "rent_increase_cap" |
	"required_disclosure" | "eviction_procedure" | "late_fee_cap" |
	"rent_control" | "habitability_standard" | "tenant_screening_restriction" |
	"lease_term_restriction" | "fee_restriction" | "relocation_assistance" |
	"right_to_counsel" | "just_cause_eviction" | "source_of_income_protection" |
	"lead_paint_disclosure" | "mold_disclosure" | "bed_bug_disclosure" |
	"flood_zone_disclosure" | "utility_billing_restriction" |
	"short_term_rental_restriction"

#JurisdictionRuleStatus: "draft" | "active" | "superseded" | "expired" | "repealed"

// ─── Jurisdiction ───────────────────────────────────────────────────────────
// A geographic or administrative region whose laws and regulations apply to
// properties within its boundaries. Supports hierarchical composition
// (federal → state → county → city → special district).

#Jurisdiction: close({
	#StatefulEntity

	name: string & strings.MinRunes(1) @display()
	jurisdiction_type: "federal" | "state" | "county" | "city" |
		"special_district" | "unincorporated_area"

	parent_jurisdiction_id?: string // self-referential hierarchy

	fips_code?:   =~"^[0-9]{5,10}$"
	state_code?:  =~"^[A-Z]{2}$"
	country_code: *"US" | =~"^[A-Z]{2}$"

	status: "active" | "dissolved" | "merged" | "pending"

	successor_jurisdiction_id?: string
	effective_date?:            time.Time
	dissolution_date?:          time.Time

	governing_body?: string
	regulatory_url?: string

	// CONSTRAINTS:
	// Dissolved/merged must have successor and dissolution date
	if status == "dissolved" || status == "merged" {
		successor_jurisdiction_id: string & !=""
		dissolution_date:          time.Time
	}

	// Hidden: generator metadata
	_display_template: "{name}"
})

// ─── PropertyJurisdiction ───────────────────────────────────────────────────
// M2M join entity linking properties to the jurisdictions they fall under.
// A property may be subject to multiple jurisdictions (federal + state +
// county + city), each with different regulatory rules.

#PropertyJurisdiction: close({
	#BaseEntity

	property_id:     string & !="" @immutable()
	jurisdiction_id: string & !="" @immutable()

	effective_date: time.Time
	end_date?:      time.Time // null = currently active

	lookup_source: "address_geocode" | "manual" | "api_lookup" | "imported"
	verified:      bool | *false
	verified_at?:  time.Time
	verified_by?:  string
})

// ─── JurisdictionRule ───────────────────────────────────────────────────────
// A specific regulatory rule within a jurisdiction. Rules are typed, and the
// rule_definition field contains structured data specific to each rule type.

#JurisdictionRule: close({
	#StatefulEntity

	jurisdiction_id: string & !=""
	rule_type: "security_deposit_limit" | "notice_period" | "rent_increase_cap" |
		"required_disclosure" | "eviction_procedure" | "late_fee_cap" |
		"rent_control" | "habitability_standard" | "tenant_screening_restriction" |
		"lease_term_restriction" | "fee_restriction" | "relocation_assistance" |
		"right_to_counsel" | "just_cause_eviction" | "source_of_income_protection" |
		"lead_paint_disclosure" | "mold_disclosure" | "bed_bug_disclosure" |
		"flood_zone_disclosure" | "utility_billing_restriction" |
		"short_term_rental_restriction"

	status: "draft" | "active" | "superseded" | "expired" | "repealed"

	// Applicability filters
	applies_to_lease_types?:    [...string]
	applies_to_property_types?: [...string]
	applies_to_space_types?:    [...string]

	// Exemptions
	exemptions?: #RuleExemptions

	// Typed rule definition — schema varies by rule_type
	// Stored as JSON; validated at application layer per rule_type
	rule_definition: _ // top type → json.RawMessage

	// Legal reference
	statute_reference?: string
	ordinance_number?:  string
	statute_url?:       string

	// Temporal validity
	effective_date:    time.Time
	expiration_date?:  time.Time
	superseded_by_id?: string

	// Verification
	last_verified?:       time.Time
	verified_by?:         string
	verification_source?: string

	// CONSTRAINTS:
	// Superseded rules must reference their successor
	if status == "superseded" {
		superseded_by_id: string & !=""
	}

	// Hidden: generator metadata
	_display_template: "{rule_type}"
})

// ─── Rule Exemptions ────────────────────────────────────────────────────────

#RuleExemptions: close({
	owner_occupied?:           bool
	owner_occupied_max_units?: int & >=1
	units_built_after?:        time.Time
	units_built_within_years?: int & >0
	single_family_exempt?:     bool
	small_property_max_units?: int & >=1
	subsidized_exempt?:        bool
	corporate_owner_only?:     bool
	custom_exemptions?:        [...string]
})

// ─── Rule Definition Schemas ────────────────────────────────────────────────
// These define the expected structure of rule_definition for each rule_type.
// Validation is performed at the application layer, not in the schema.

#SecurityDepositLimitRule: close({
	max_months:                      float & >0
	furnished_max_months?:           float & >0
	additional_pet_deposit_allowed?: bool
	max_pet_deposit?:                #NonNegativeMoney
	refund_deadline_days:            int & >0
	itemization_required:            bool
	interest_required?:              bool
	interest_rate?:                  float
	notes?:                          string
})

#NoticePeriodRule: close({
	tenancy_under_1_year_days:       int & >=0
	tenancy_over_1_year_days:        int & >=0
	increase_over_threshold_days?:   int & >=0
	increase_threshold_percent?:     float
	month_to_month_termination_days: int & >=0
	fixed_term_non_renewal_days?:    int & >=0
	notes?:                          string
})

#RentIncreaseCapRule: close({
	cap_type:       "fixed_percent" | "cpi_plus_fixed" | "cpi_only" | "board_determined" | "none"
	fixed_percent?: float
	max_percent?:   float
	cpi_index?:     string
	frequency:      "annual" | "biannual" | "per_tenancy"
	applies_to:     "existing_tenants_only" | "all_tenants" | "rent_controlled_units"
	vacancy_decontrol?: bool
	notes?:         string
})

#LateFeeCapRule: close({
	max_flat?:             #NonNegativeMoney
	max_percent?:          float
	grace_period_min_days: int & >=0
	compound_prohibited?:  bool
	notes?:                string
})

#EvictionProcedureRule: close({
	just_cause_required:             bool
	just_causes?:                    [...string]
	cure_period_days:                int & >=0
	notice_type:                     string
	mandatory_mediation?:            bool
	relocation_assistance_required?: bool
	relocation_amount?:              #NonNegativeMoney
	right_to_counsel?:               bool
	winter_eviction_moratorium?:     bool
	moratorium_months?:              [...int & >=1 & <=12]
	notes?:                          string
})

#RequiredDisclosureRule: close({
	disclosure_type:  string
	timing:          "before_signing" | "at_signing" | "within_days_of_signing" | "annually"
	timing_days?:    int
	form_required?:  bool
	form_reference?: string
	notes?:          string
})

#ShortTermRentalRestrictionRule: close({
	permitted:                       bool
	license_required?:               bool
	max_days_per_year?:              int
	hosted_only?:                    bool
	primary_residence_only?:         bool
	transient_occupancy_tax_rate?:   float
	platform_registration_required?: bool
	notes?:                          string
})
