// events/jurisdiction_events.cue
// Domain events for the jurisdiction/compliance domain.
package events

import "time"

#JurisdictionRuleActivated: close({
	jurisdiction_rule_id: string
	jurisdiction_id:      string
	rule_type:            "security_deposit_limit" | "notice_period" | "rent_increase_cap" |
		"required_disclosure" | "eviction_procedure" | "late_fee_cap" |
		"rent_control" | "habitability_standard" | "tenant_screening_restriction" |
		"lease_term_restriction" | "fee_restriction" | "relocation_assistance" |
		"right_to_counsel" | "just_cause_eviction" | "source_of_income_protection" |
		"lead_paint_disclosure" | "mold_disclosure" | "bed_bug_disclosure" |
		"flood_zone_disclosure" | "utility_billing_restriction" |
		"short_term_rental_restriction"
	effective_date: time.Time

	// Which properties are affected?
	affected_property_ids: [...string]
	affected_lease_count:  int

	statute_reference?: string
})

#ComplianceAlertTriggered: close({
	property_id:    string
	jurisdiction_id: string

	alert_type: "deposit_limit_exceeded" | "rent_increase_violation" |
		"notice_period_violation" | "disclosure_missing" |
		"eviction_procedure_violation" | "late_fee_exceeded"

	rule_type: "security_deposit_limit" | "notice_period" | "rent_increase_cap" |
		"required_disclosure" | "eviction_procedure" | "late_fee_cap" |
		"rent_control" | "habitability_standard" | "tenant_screening_restriction" |
		"lease_term_restriction" | "fee_restriction" | "relocation_assistance" |
		"right_to_counsel" | "just_cause_eviction" | "source_of_income_protection" |
		"lead_paint_disclosure" | "mold_disclosure" | "bed_bug_disclosure" |
		"flood_zone_disclosure" | "utility_billing_restriction" |
		"short_term_rental_restriction"

	severity: "critical" | "warning" | "info"
	message:  string

	// Affected entities
	lease_id?:  string
	space_id?:  string
	person_id?: string

	// What was expected vs. actual
	expected_value?: _
	actual_value?:   _
	rule_reference?: string

	triggered_at: time.Time
})
