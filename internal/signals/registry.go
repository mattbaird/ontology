// Package signals provides the signal registry, classifier, and aggregator
// for the signal discovery system.
package signals

import (
	"github.com/matthewbaird/ontology/internal/types"
)

// WeightOrder maps signal weights to numeric severity (lower = more severe).
var WeightOrder = map[string]int{
	"critical": 1,
	"strong":   2,
	"moderate": 3,
	"weak":     4,
	"info":     5,
}

// registryByEventType is the lookup map built at Init time.
var registryByEventType map[string][]types.SignalRegistration

// crossCategoryEscalations holds escalation rules that span multiple categories.
var crossCategoryEscalations []types.EscalationRule

// SignalRegistry contains all signal registrations from the CUE ontology.
var SignalRegistry = []types.SignalRegistration{
	// === Financial (7) ===
	{
		ID:          "payment_on_time",
		EventType:   "PaymentRecorded",
		Condition:   "days_past_due == 0",
		Category:    "financial",
		Weight:      "info",
		Polarity:    "positive",
		Description: "Payment received on time",
		InterpretationGuidance: "On-time payment is the baseline. Consistent on-time payment over months is a strong positive signal for retention.",
	},
	{
		ID:          "payment_late",
		EventType:   "PaymentRecorded",
		Condition:   "days_past_due > 0",
		Category:    "financial",
		Weight:      "moderate",
		Polarity:    "negative",
		Description: "Payment received late",
		InterpretationGuidance: "Single late payment is common and not actionable alone. Check for pattern: 3+ in 6 months indicates financial stress.",
		EscalationRules: []types.EscalationRule{
			{
				ID:                   "fin_late_pattern",
				Description:          "Repeated late payments indicate financial stress or disengagement",
				TriggerType:          "count",
				SignalCategory:       "financial",
				SignalPolarity:       "negative",
				Count:                3,
				WithinDays:           180,
				EscalatedWeight:      "strong",
				EscalatedDescription: "3+ late payments in 6 months. Pattern, not one-off.",
				RecommendedAction:    "Proactive outreach to understand situation. Offer payment plan if appropriate.",
			},
			{
				ID:                   "fin_late_acute",
				Description:          "Rapid late payment acceleration",
				TriggerType:          "count",
				SignalCategory:       "financial",
				SignalPolarity:       "negative",
				Count:                3,
				WithinDays:           90,
				EscalatedWeight:      "critical",
				EscalatedDescription: "3 late payments in 90 days. Likely financial distress.",
				RecommendedAction:    "Immediate outreach with assistance resources. Consider hardship agreement.",
			},
		},
	},
	{
		ID:          "payment_nsf",
		EventType:   "PaymentReturned",
		Condition:   "return_reason == nsf",
		Category:    "financial",
		Weight:      "strong",
		Polarity:    "negative",
		Description: "Payment returned NSF (insufficient funds)",
		InterpretationGuidance: "NSF is a stronger signal than simple lateness — it indicates the tenant attempted to pay but lacks funds.",
		EscalationRules: []types.EscalationRule{
			{
				ID:                   "fin_nsf_repeat",
				Description:          "Multiple returned payments",
				TriggerType:          "count",
				SignalCategory:       "financial",
				SignalPolarity:       "negative",
				Count:                2,
				WithinDays:           90,
				EscalatedWeight:      "critical",
				EscalatedDescription: "2+ NSF events in 90 days. Serious financial difficulty.",
				RecommendedAction:    "In-person conversation. Payment method change required.",
			},
		},
	},
	{
		ID:          "late_fee_assessed",
		EventType:   "LateFeeAssessed",
		Category:    "financial",
		Weight:      "moderate",
		Polarity:    "negative",
		Description: "Late fee assessed on account",
		InterpretationGuidance: "Late fee is a lagging indicator — the lateness already happened. Track whether the fee itself is paid promptly.",
	},
	{
		ID:          "balance_increasing",
		EventType:   "BalanceChanged",
		Condition:   "direction == increasing",
		Category:    "financial",
		Weight:      "moderate",
		Polarity:    "negative",
		Description: "Account balance increasing (growing debt)",
		InterpretationGuidance: "Rising balance suggests charges outpacing payments. Compare to rent amount to assess severity.",
	},
	{
		ID:          "payment_partial",
		EventType:   "PaymentRecorded",
		Condition:   "amount < amount_due",
		Category:    "financial",
		Weight:      "moderate",
		Polarity:    "negative",
		Description: "Partial payment received",
		InterpretationGuidance: "Partial payments may indicate effort (positive) or decline (negative). Check trend direction and communication context.",
	},
	{
		ID:          "write_off_posted",
		EventType:   "WriteOffPosted",
		Category:    "financial",
		Weight:      "strong",
		Polarity:    "negative",
		Description: "Balance written off as uncollectible",
		InterpretationGuidance: "Write-off is typically a terminal financial signal. Check if tenant is still active — may indicate pending move-out.",
	},

	// === Maintenance (5) ===
	{
		ID:          "complaint_filed",
		EventType:   "ComplaintCreated",
		Category:    "maintenance",
		Weight:      "moderate",
		Polarity:    "negative",
		Description: "Complaint filed by tenant",
		InterpretationGuidance: "Distinguish controllable (broken equipment, pests) from uncontrollable (neighbor noise, street traffic). Resolution speed strongly affects retention.",
		EscalationRules: []types.EscalationRule{
			{
				ID:                   "maint_complaint_pattern",
				Description:          "Repeated complaints indicate persistent dissatisfaction",
				TriggerType:          "count",
				SignalCategory:       "maintenance",
				SignalPolarity:       "negative",
				Count:                3,
				WithinDays:           180,
				EscalatedWeight:      "strong",
				EscalatedDescription: "3+ complaints in 6 months. High dissatisfaction.",
				RecommendedAction:    "Personal outreach from property manager. Address root cause, not just symptoms.",
			},
		},
	},
	{
		ID:          "maintenance_request",
		EventType:   "WorkOrderCreated",
		Condition:   "type == maintenance_request",
		Category:    "maintenance",
		Weight:      "info",
		Polarity:    "contextual",
		Description: "Maintenance request submitted",
		InterpretationGuidance: "Maintenance requests are generally positive — tenant is engaged and communicating. Track resolution time.",
		EscalationRules: []types.EscalationRule{
			{
				ID:                   "maint_recurring_same_issue",
				Description:          "Same issue recurring indicates inadequate repair",
				TriggerType:          "count",
				SignalCategory:       "maintenance",
				Count:                2,
				WithinDays:           90,
				EscalatedWeight:      "strong",
				EscalatedDescription: "Recurring maintenance issue in same space.",
				RecommendedAction:    "Escalate to different vendor or replace equipment. Apologize to tenant.",
			},
		},
	},
	{
		ID:          "emergency_maintenance",
		EventType:   "WorkOrderCreated",
		Condition:   "priority == emergency",
		Category:    "maintenance",
		Weight:      "strong",
		Polarity:    "negative",
		Description: "Emergency maintenance request",
		InterpretationGuidance: "Single emergency is normal wear. Check resolution time. Unresolved emergency > 48 hours is critical for retention AND legal liability.",
		EscalationRules: []types.EscalationRule{
			{
				ID:                   "maint_unresolved_critical",
				Description:          "Emergency maintenance unresolved beyond safety threshold",
				TriggerType:          "count",
				SignalCategory:       "maintenance",
				Count:                1,
				WithinDays:           2,
				EscalatedWeight:      "critical",
				EscalatedDescription: "Emergency work order open > 48 hours.",
				RecommendedAction:    "Immediate escalation. Legal liability exposure. Assign backup vendor.",
			},
		},
	},
	{
		ID:          "work_order_unresolved",
		EventType:   "WorkOrderOverdue",
		Category:    "maintenance",
		Weight:      "moderate",
		Polarity:    "negative",
		Description: "Work order past expected resolution date",
		InterpretationGuidance: "Unresolved work orders are exponentially worse than resolved ones for tenant satisfaction.",
	},
	{
		ID:          "recurring_issue",
		EventType:   "WorkOrderCreated",
		Condition:   "is_recurring == true",
		Category:    "maintenance",
		Weight:      "strong",
		Polarity:    "negative",
		Description: "Recurring maintenance issue detected",
		InterpretationGuidance: "Recurring issues indicate systemic problems. Replace equipment or change vendor rather than patching.",
	},

	// === Communication (4) ===
	{
		ID:          "outreach_no_response",
		EventType:   "OutreachAttempted",
		Condition:   "response == none",
		Category:    "communication",
		Weight:      "moderate",
		Polarity:    "negative",
		Description: "Outreach attempt with no response",
		InterpretationGuidance: "Silence is the most dangerous communication signal. Try alternate contact methods.",
		EscalationRules: []types.EscalationRule{
			{
				ID:                   "comm_unresponsive",
				Description:          "Tenant unresponsive to multiple contact attempts",
				TriggerType:          "count",
				SignalCategory:       "communication",
				SignalPolarity:       "negative",
				Count:                2,
				WithinDays:           30,
				EscalatedWeight:      "strong",
				EscalatedDescription: "Unresponsive to 2+ contact attempts in 30 days.",
				RecommendedAction:    "Try alternate contact method. If all fail, consider in-person visit.",
			},
			{
				ID:                   "comm_unresponsive_critical",
				Description:          "Tenant completely unreachable",
				TriggerType:          "count",
				SignalCategory:       "communication",
				SignalPolarity:       "negative",
				Count:                3,
				WithinDays:           30,
				EscalatedWeight:      "critical",
				EscalatedDescription: "3+ unanswered contact attempts in 30 days.",
				RecommendedAction:    "In-person visit or formal notice via certified mail.",
			},
		},
	},
	{
		ID:          "tenant_initiated_contact",
		EventType:   "TenantContactReceived",
		Category:    "communication",
		Weight:      "info",
		Polarity:    "positive",
		Description: "Tenant initiated contact",
		InterpretationGuidance: "Tenant-initiated contact is almost always positive regardless of content — it shows engagement.",
	},
	{
		ID:          "portal_activity_drop",
		EventType:   "PortalActivityChanged",
		Condition:   "direction == decreasing",
		Category:    "communication",
		Weight:      "weak",
		Polarity:    "negative",
		Description: "Portal login frequency decreased",
		InterpretationGuidance: "Portal activity is a leading indicator — drops precede other changes. Weak signal alone, strengthens other negative signals.",
	},
	{
		ID:          "communication_preference_changed",
		EventType:   "ContactPreferenceUpdated",
		Category:    "communication",
		Weight:      "weak",
		Polarity:    "neutral",
		Description: "Communication preference changed",
		InterpretationGuidance: "May indicate changed phone number or lifestyle. Ensure future outreach uses updated preference.",
	},

	// === Compliance (3) ===
	{
		ID:          "lease_violation",
		EventType:   "LeaseViolationRecorded",
		Category:    "compliance",
		Weight:      "strong",
		Polarity:    "negative",
		Description: "Lease violation recorded",
		InterpretationGuidance: "Document thoroughly. Check for pattern — 2+ violations in 12 months is grounds for non-renewal.",
		EscalationRules: []types.EscalationRule{
			{
				ID:                   "compliance_repeat_violation",
				Description:          "Multiple lease violations indicate non-compliance pattern",
				TriggerType:          "count",
				SignalCategory:       "compliance",
				SignalPolarity:       "negative",
				Count:                2,
				WithinDays:           365,
				EscalatedWeight:      "critical",
				EscalatedDescription: "2+ lease violations in 12 months.",
				RecommendedAction:    "Formal notice. Document for potential non-renewal or termination.",
			},
		},
	},
	{
		ID:          "violation_cured",
		EventType:   "LeaseViolationCured",
		Category:    "compliance",
		Weight:      "moderate",
		Polarity:    "positive",
		Description: "Lease violation cured by tenant",
		InterpretationGuidance: "Cured violation is positive — tenant is responsive to notices. Speed of cure matters.",
	},
	{
		ID:          "inspection_failed",
		EventType:   "InspectionCompleted",
		Condition:   "result == failed",
		Category:    "compliance",
		Weight:      "strong",
		Polarity:    "negative",
		Description: "Space failed inspection",
		InterpretationGuidance: "Failed inspection may have regulatory implications. Check jurisdiction requirements for follow-up timeline.",
	},

	// === Behavioral (3) ===
	{
		ID:          "parking_violation",
		EventType:   "ParkingViolationRecorded",
		Category:    "behavioral",
		Weight:      "weak",
		Polarity:    "negative",
		Description: "Parking violation recorded",
		InterpretationGuidance: "Often first visible sign of norm disengagement. Individual instances are weak; look for pattern.",
		EscalationRules: []types.EscalationRule{
			{
				ID:                   "behavioral_pattern",
				Description:          "Multiple behavioral changes suggest disengagement",
				TriggerType:          "count",
				SignalCategory:       "behavioral",
				SignalPolarity:       "negative",
				Count:                3,
				WithinDays:           90,
				EscalatedWeight:      "moderate",
				EscalatedDescription: "Multiple behavioral changes in 90 days.",
				RecommendedAction:    "Casual check-in. May precede non-renewal.",
			},
		},
	},
	{
		ID:          "amenity_usage_change",
		EventType:   "AmenityUsageChanged",
		Category:    "behavioral",
		Weight:      "weak",
		Polarity:    "contextual",
		Description: "Amenity usage pattern changed",
		InterpretationGuidance: "Individual behavioral signals are weak. Only meaningful in combination with other signals.",
	},
	{
		ID:          "occupancy_pattern_change",
		EventType:   "OccupancyPatternChanged",
		Category:    "behavioral",
		Weight:      "weak",
		Polarity:    "contextual",
		Description: "Occupancy pattern change detected",
		InterpretationGuidance: "Extended absences or unusual patterns may indicate subletting or abandonment. Verify with communication.",
	},

	// === Relationship (5) ===
	{
		ID:          "occupant_added",
		EventType:   "OccupantAdded",
		Category:    "relationship",
		Weight:      "moderate",
		Polarity:    "contextual",
		Description: "Occupant added to lease",
		InterpretationGuidance: "Growing household is generally positive. Check if it creates compliance concerns (occupancy limits).",
	},
	{
		ID:          "occupant_removed",
		EventType:   "OccupantRemoved",
		Category:    "relationship",
		Weight:      "moderate",
		Polarity:    "negative",
		Description: "Occupant removed from lease",
		InterpretationGuidance: "Check if remaining occupant income supports rent. May indicate relationship change or financial restructuring.",
	},
	{
		ID:          "guarantor_change",
		EventType:   "GuarantorChanged",
		Category:    "relationship",
		Weight:      "moderate",
		Polarity:    "contextual",
		Description: "Guarantor added, removed, or changed",
		InterpretationGuidance: "Guarantor removal may signal changed family dynamics. Addition may signal financial concern requiring guarantee.",
	},
	{
		ID:          "emergency_contact_updated",
		EventType:   "EmergencyContactUpdated",
		Category:    "relationship",
		Weight:      "weak",
		Polarity:    "neutral",
		Description: "Emergency contact information updated",
		InterpretationGuidance: "Routine update. Significant only if combined with other relationship changes.",
	},
	{
		ID:          "roommate_departed",
		EventType:   "RoommateDeparted",
		Category:    "relationship",
		Weight:      "strong",
		Polarity:    "negative",
		Description: "Roommate departed the lease",
		InterpretationGuidance: "Check remaining tenant income against rent. Roommate departure is a high-signal event for financial risk.",
		EscalationRules: []types.EscalationRule{
			{
				ID:          "cross_relationship_financial",
				Description: "Household change with financial impact",
				TriggerType: "cross_category",
				RequiredCategories: []types.CategoryRequirement{
					{Category: "relationship", Polarity: "negative", MinCount: 1},
					{Category: "financial", Polarity: "negative", MinCount: 1},
				},
				WithinDays:           90,
				EscalatedWeight:      "strong",
				EscalatedDescription: "Roommate/occupant departure coinciding with payment issues.",
				RecommendedAction:    "Check if remaining occupant income supports rent. Offer restructure options.",
			},
		},
	},

	// === Lifecycle (7) ===
	{
		ID:          "lease_expiring_90",
		EventType:   "LeaseExpirationApproaching",
		Condition:   "days_remaining <= 90",
		Category:    "lifecycle",
		Weight:      "moderate",
		Polarity:    "neutral",
		Description: "Lease expiring within 90 days",
		InterpretationGuidance: "90-day window is when most renewal decisions are made. Check signal summary before sending renewal offer.",
		EscalationRules: []types.EscalationRule{
			{
				ID:          "cross_maintenance_lifecycle",
				Description: "Active complaints near lease expiration — retention at risk",
				TriggerType: "cross_category",
				RequiredCategories: []types.CategoryRequirement{
					{Category: "maintenance", Polarity: "negative", MinCount: 2},
					{Category: "lifecycle", MinCount: 1},
				},
				WithinDays:           90,
				EscalatedWeight:      "strong",
				EscalatedDescription: "Unresolved maintenance issues with lease expiring.",
				RecommendedAction:    "Resolve maintenance first, then present renewal. Do not send renewal offer while complaints are open.",
			},
		},
	},
	{
		ID:          "lease_expiring_30",
		EventType:   "LeaseExpirationApproaching",
		Condition:   "days_remaining <= 30",
		Category:    "lifecycle",
		Weight:      "strong",
		Polarity:    "negative",
		Description: "Lease expiring within 30 days",
		InterpretationGuidance: "No renewal response by 30 days out means tenant is likely leaving. Escalate immediately.",
	},
	{
		ID:          "notice_given",
		EventType:   "NoticeRecorded",
		Category:    "lifecycle",
		Weight:      "critical",
		Polarity:    "negative",
		Description: "Tenant gave notice to vacate",
		InterpretationGuidance: "Terminal signal for this lease. Begin make-ready planning. Check if retention conversation is still possible.",
	},
	{
		ID:          "renewal_offered",
		EventType:   "RenewalOfferSent",
		Category:    "lifecycle",
		Weight:      "info",
		Polarity:    "neutral",
		Description: "Renewal offer sent to tenant",
		InterpretationGuidance: "Track response time. No response within 14 days should be escalated.",
	},
	{
		ID:          "renewal_signed",
		EventType:   "RenewalSigned",
		Category:    "lifecycle",
		Weight:      "strong",
		Polarity:    "positive",
		Description: "Lease renewal signed",
		InterpretationGuidance: "Strong positive outcome. Reset signal assessments for the new term.",
	},
	{
		ID:          "move_in_anniversary",
		EventType:   "MoveInAnniversary",
		Category:    "lifecycle",
		Weight:      "info",
		Polarity:    "positive",
		Description: "Move-in anniversary milestone",
		InterpretationGuidance: "Long tenancy is positive. 2+ year tenants are high-value — handle complaints with extra care.",
	},
	{
		ID:          "option_exercise_deadline",
		EventType:   "OptionDeadlineApproaching",
		Category:    "lifecycle",
		Weight:      "strong",
		Polarity:    "neutral",
		Description: "Renewal/expansion option exercise deadline approaching",
		InterpretationGuidance: "Legally binding deadline. Missing it forfeits the option. Ensure tenant and manager are both aware.",
	},
}

// CrossCategoryEscalationRules are escalation rules that span multiple signal categories.
var CrossCategoryEscalationRules = []types.EscalationRule{
	{
		ID:          "cross_financial_communication",
		Description: "Financial decline combined with communication decline — highest non-renewal predictor",
		TriggerType: "cross_category",
		RequiredCategories: []types.CategoryRequirement{
			{Category: "financial", Polarity: "negative", MinCount: 2},
			{Category: "communication", Polarity: "negative", MinCount: 1},
		},
		WithinDays:           90,
		EscalatedWeight:      "critical",
		EscalatedDescription: "Financial problems AND communication avoidance within 90 days.",
		RecommendedAction:    "Highest priority intervention. In-person if possible. Have assistance resources ready.",
	},
	{
		ID:                     "absence_zero_maintenance",
		Description:            "Long-term tenant with no maintenance activity may indicate disengagement",
		TriggerType:            "absence",
		ExpectedSignalCategory: "maintenance",
		AbsentForDays:          365,
		AppliesToCondition:     "tenant with active lease > 24 months",
		EscalatedWeight:        "weak",
		EscalatedDescription:   "No maintenance requests in 12+ months from long-term tenant.",
		RecommendedAction:      "Not actionable alone. Note for context when evaluating other signals.",
	},
	{
		ID:                     "absence_portal_activity",
		Description:            "Portal login cessation",
		TriggerType:            "absence",
		ExpectedSignalCategory: "communication",
		AbsentForDays:          60,
		AppliesToCondition:     "tenant who previously logged in at least monthly",
		EscalatedWeight:        "weak",
		EscalatedDescription:   "Previously active portal user has stopped logging in.",
		RecommendedAction:      "Weak signal alone. Strengthens interpretation of other negative signals.",
	},
	{
		ID:              "trend_payment_degrading",
		Description:     "Payment timing getting progressively later",
		TriggerType:     "trend",
		TrendDirection:  "increasing",
		TrendMetric:     "days_past_due_at_payment",
		TrendWindowDays: 180,
		EscalatedWeight:      "moderate",
		EscalatedDescription: "Payment timing trending later over 6 months.",
		RecommendedAction:    "Early intervention before pattern becomes critical. Friendly check-in.",
	},
}

// Init builds lookup maps. Call once at startup.
func Init() {
	registryByEventType = make(map[string][]types.SignalRegistration, len(SignalRegistry))
	for _, reg := range SignalRegistry {
		registryByEventType[reg.EventType] = append(registryByEventType[reg.EventType], reg)
	}
	crossCategoryEscalations = CrossCategoryEscalationRules
}

// LookupSignals returns all signal registrations matching the given event type.
func LookupSignals(eventType string) []types.SignalRegistration {
	return registryByEventType[eventType]
}

// GetCrossCategoryEscalations returns all cross-category escalation rules.
func GetCrossCategoryEscalations() []types.EscalationRule {
	return crossCategoryEscalations
}

// WeightSeverity returns the numeric severity for a weight (lower = more severe).
// Returns 6 for unknown weights.
func WeightSeverity(weight string) int {
	if s, ok := WeightOrder[weight]; ok {
		return s
	}
	return 6
}

// IsAtLeastWeight returns true if actual is at least as severe as minimum.
func IsAtLeastWeight(actual, minimum string) bool {
	return WeightSeverity(actual) <= WeightSeverity(minimum)
}
