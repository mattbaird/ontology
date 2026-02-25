// ontology/signals.cue
// Signal taxonomy types, annotation schema, inference rules, escalation rule schema,
// and the complete signal registration catalog. This is the CUE source of truth
// for the signal discovery system (Layers 2 and 3).
package propeller

import "time"

// ─── Signal Taxonomy ───────────────────────────────────────────────────────────

#SignalCategory:
	"financial" |
	"maintenance" |
	"communication" |
	"compliance" |
	"behavioral" |
	"market" |
	"relationship" |
	"lifecycle"

#SignalWeight:
	"critical" |
	"strong" |
	"moderate" |
	"weak" |
	"info"

#SignalPolarity:
	"positive" |
	"negative" |
	"neutral" |
	"contextual"

#EntityRole:
	"subject" |
	"target" |
	"related" |
	"context"

// ─── Signal Annotation Schema ──────────────────────────────────────────────────

// SignalAnnotation is the _signal metadata attached to ontology fields.
// Use "ignore" for fields whose changes carry no signal value.
#SignalAnnotation: "ignore" | #SignalAnnotationDef

#SignalAnnotationDef: {
	category:        #SignalCategory
	weight:          #SignalWeight
	polarity:        #SignalPolarity
	description:     string
	on_value?: [string]: {
		weight?:   #SignalWeight
		polarity?: #SignalPolarity
	}
	absent_signal?:    bool
	interpretation?:   string
}

// ─── Escalation Rules ──────────────────────────────────────────────────────────

#EscalationTriggerType:
	"count" |
	"cross_category" |
	"absence" |
	"trend"

#CategoryRequirement: {
	category:   #SignalCategory
	polarity?:  #SignalPolarity
	min_count:  int & >0
}

#EscalationRule: {
	id:          string & !=""
	description: string

	trigger_type: #EscalationTriggerType

	// Count-based triggers
	signal_category?: #SignalCategory
	signal_polarity?: #SignalPolarity
	count?:           int & >0
	within_days?:     int & >0

	// Cross-category triggers
	required_categories?: [...#CategoryRequirement]

	// Absence-based triggers
	expected_signal_category?: #SignalCategory
	absent_for_days?:          int & >0
	applies_to_condition?:     string

	// Trend-based triggers
	trend_direction?:   "increasing" | "decreasing"
	trend_metric?:      string
	trend_window_days?: int & >0

	// Result
	escalated_weight:      #SignalWeight
	escalated_description: string
	recommended_action?:   string
}

// ─── Inference Rules ───────────────────────────────────────────────────────────

#InferenceRules: {
	state_machine_terminal_state: {
		category:  "lifecycle"
		weight:    "critical"
		polarity:  "negative"
		rationale: "Terminal states end an entity's active life."
	}
	state_machine_forward_progression: {
		category:  "lifecycle"
		weight:    "info"
		polarity:  "positive"
		rationale: "Forward transitions indicate normal progress."
	}
	state_machine_regression: {
		category:  "lifecycle"
		weight:    "moderate"
		polarity:  "negative"
		rationale: "Backward transitions indicate rejection or reversal."
	}
}

// ─── Signal Registration ───────────────────────────────────────────────────────

#SignalRegistration: {
	id:          string & !=""
	event_type:  string & !=""
	condition?:  string
	category:    #SignalCategory
	weight:      #SignalWeight
	polarity:    #SignalPolarity
	description: string
	interpretation_guidance?: string
	escalation_rules?: [...#EscalationRule]
}

// ─── Activity Entry ────────────────────────────────────────────────────────────

#SourceRef: {
	entity_type: #EntityType
	entity_id:   string & !=""
	role:        #EntityRole
}

#ActivityEntry: {
	event_id:            string & !=""
	event_type:          string & !=""
	occurred_at:         time.Time
	indexed_entity_type: #EntityType
	indexed_entity_id:   string & !=""
	entity_role:         #EntityRole
	source_refs: [...#SourceRef]
	summary:  string & !=""
	category: #SignalCategory
	weight:   #SignalWeight
	polarity: #SignalPolarity
	payload: _
}

// ─── Signal Registration Catalog ───────────────────────────────────────────────
// ~30 signal registrations derived from the spec Section 6.2.

signal_registrations: [...#SignalRegistration]
signal_registrations: [
	// === Financial (7) ===
	{
		id:          "payment_on_time"
		event_type:  "PaymentRecorded"
		condition:   "days_past_due == 0"
		category:    "financial"
		weight:      "info"
		polarity:    "positive"
		description: "Payment received on time"
		interpretation_guidance: "On-time payment is the baseline. Consistent on-time payment over months is a strong positive signal for retention."
	},
	{
		id:          "payment_late"
		event_type:  "PaymentRecorded"
		condition:   "days_past_due > 0"
		category:    "financial"
		weight:      "moderate"
		polarity:    "negative"
		description: "Payment received late"
		interpretation_guidance: "Single late payment is common and not actionable alone. Check for pattern: 3+ in 6 months indicates financial stress."
		escalation_rules: [
			{
				id:                    "fin_late_pattern"
				description:           "Repeated late payments indicate financial stress or disengagement"
				trigger_type:          "count"
				signal_category:       "financial"
				signal_polarity:       "negative"
				count:                 3
				within_days:           180
				escalated_weight:      "strong"
				escalated_description: "3+ late payments in 6 months. Pattern, not one-off."
				recommended_action:    "Proactive outreach to understand situation. Offer payment plan if appropriate."
			},
			{
				id:                    "fin_late_acute"
				description:           "Rapid late payment acceleration"
				trigger_type:          "count"
				signal_category:       "financial"
				signal_polarity:       "negative"
				count:                 3
				within_days:           90
				escalated_weight:      "critical"
				escalated_description: "3 late payments in 90 days. Likely financial distress."
				recommended_action:    "Immediate outreach with assistance resources. Consider hardship agreement."
			},
		]
	},
	{
		id:          "payment_nsf"
		event_type:  "PaymentReturned"
		condition:   "return_reason == nsf"
		category:    "financial"
		weight:      "strong"
		polarity:    "negative"
		description: "Payment returned NSF (insufficient funds)"
		interpretation_guidance: "NSF is a stronger signal than simple lateness — it indicates the tenant attempted to pay but lacks funds."
		escalation_rules: [
			{
				id:                    "fin_nsf_repeat"
				description:           "Multiple returned payments"
				trigger_type:          "count"
				signal_category:       "financial"
				signal_polarity:       "negative"
				count:                 2
				within_days:           90
				escalated_weight:      "critical"
				escalated_description: "2+ NSF events in 90 days. Serious financial difficulty."
				recommended_action:    "In-person conversation. Payment method change required."
			},
		]
	},
	{
		id:          "late_fee_assessed"
		event_type:  "LateFeeAssessed"
		category:    "financial"
		weight:      "moderate"
		polarity:    "negative"
		description: "Late fee assessed on account"
		interpretation_guidance: "Late fee is a lagging indicator — the lateness already happened. Track whether the fee itself is paid promptly."
	},
	{
		id:          "balance_increasing"
		event_type:  "BalanceChanged"
		condition:   "direction == increasing"
		category:    "financial"
		weight:      "moderate"
		polarity:    "negative"
		description: "Account balance increasing (growing debt)"
		interpretation_guidance: "Rising balance suggests charges outpacing payments. Compare to rent amount to assess severity."
	},
	{
		id:          "payment_partial"
		event_type:  "PaymentRecorded"
		condition:   "amount < amount_due"
		category:    "financial"
		weight:      "moderate"
		polarity:    "negative"
		description: "Partial payment received"
		interpretation_guidance: "Partial payments may indicate effort (positive) or decline (negative). Check trend direction and communication context."
	},
	{
		id:          "write_off_posted"
		event_type:  "WriteOffPosted"
		category:    "financial"
		weight:      "strong"
		polarity:    "negative"
		description: "Balance written off as uncollectible"
		interpretation_guidance: "Write-off is typically a terminal financial signal. Check if tenant is still active — may indicate pending move-out."
	},

	// === Maintenance (5) ===
	{
		id:          "complaint_filed"
		event_type:  "ComplaintCreated"
		category:    "maintenance"
		weight:      "moderate"
		polarity:    "negative"
		description: "Complaint filed by tenant"
		interpretation_guidance: "Distinguish controllable (broken equipment, pests) from uncontrollable (neighbor noise, street traffic). Resolution speed strongly affects retention."
		escalation_rules: [
			{
				id:                    "maint_complaint_pattern"
				description:           "Repeated complaints indicate persistent dissatisfaction"
				trigger_type:          "count"
				signal_category:       "maintenance"
				signal_polarity:       "negative"
				count:                 3
				within_days:           180
				escalated_weight:      "strong"
				escalated_description: "3+ complaints in 6 months. High dissatisfaction."
				recommended_action:    "Personal outreach from property manager. Address root cause, not just symptoms."
			},
		]
	},
	{
		id:          "maintenance_request"
		event_type:  "WorkOrderCreated"
		condition:   "type == maintenance_request"
		category:    "maintenance"
		weight:      "info"
		polarity:    "contextual"
		description: "Maintenance request submitted"
		interpretation_guidance: "Maintenance requests are generally positive — tenant is engaged and communicating. Track resolution time."
		escalation_rules: [
			{
				id:                    "maint_recurring_same_issue"
				description:           "Same issue recurring indicates inadequate repair"
				trigger_type:          "count"
				signal_category:       "maintenance"
				count:                 2
				within_days:           90
				escalated_weight:      "strong"
				escalated_description: "Recurring maintenance issue in same space."
				recommended_action:    "Escalate to different vendor or replace equipment. Apologize to tenant."
			},
		]
	},
	{
		id:          "emergency_maintenance"
		event_type:  "WorkOrderCreated"
		condition:   "priority == emergency"
		category:    "maintenance"
		weight:      "strong"
		polarity:    "negative"
		description: "Emergency maintenance request"
		interpretation_guidance: "Single emergency is normal wear. Check resolution time. Unresolved emergency > 48 hours is critical for retention AND legal liability."
		escalation_rules: [
			{
				id:                    "maint_unresolved_critical"
				description:           "Emergency maintenance unresolved beyond safety threshold"
				trigger_type:          "count"
				signal_category:       "maintenance"
				count:                 1
				within_days:           2
				escalated_weight:      "critical"
				escalated_description: "Emergency work order open > 48 hours."
				recommended_action:    "Immediate escalation. Legal liability exposure. Assign backup vendor."
			},
		]
	},
	{
		id:          "work_order_unresolved"
		event_type:  "WorkOrderOverdue"
		category:    "maintenance"
		weight:      "moderate"
		polarity:    "negative"
		description: "Work order past expected resolution date"
		interpretation_guidance: "Unresolved work orders are exponentially worse than resolved ones for tenant satisfaction."
	},
	{
		id:          "recurring_issue"
		event_type:  "WorkOrderCreated"
		condition:   "is_recurring == true"
		category:    "maintenance"
		weight:      "strong"
		polarity:    "negative"
		description: "Recurring maintenance issue detected"
		interpretation_guidance: "Recurring issues indicate systemic problems. Replace equipment or change vendor rather than patching."
	},

	// === Communication (4) ===
	{
		id:          "outreach_no_response"
		event_type:  "OutreachAttempted"
		condition:   "response == none"
		category:    "communication"
		weight:      "moderate"
		polarity:    "negative"
		description: "Outreach attempt with no response"
		interpretation_guidance: "Silence is the most dangerous communication signal. Try alternate contact methods."
		escalation_rules: [
			{
				id:                    "comm_unresponsive"
				description:           "Tenant unresponsive to multiple contact attempts"
				trigger_type:          "count"
				signal_category:       "communication"
				signal_polarity:       "negative"
				count:                 2
				within_days:           30
				escalated_weight:      "strong"
				escalated_description: "Unresponsive to 2+ contact attempts in 30 days."
				recommended_action:    "Try alternate contact method. If all fail, consider in-person visit."
			},
			{
				id:                    "comm_unresponsive_critical"
				description:           "Tenant completely unreachable"
				trigger_type:          "count"
				signal_category:       "communication"
				signal_polarity:       "negative"
				count:                 3
				within_days:           30
				escalated_weight:      "critical"
				escalated_description: "3+ unanswered contact attempts in 30 days."
				recommended_action:    "In-person visit or formal notice via certified mail."
			},
		]
	},
	{
		id:          "tenant_initiated_contact"
		event_type:  "TenantContactReceived"
		category:    "communication"
		weight:      "info"
		polarity:    "positive"
		description: "Tenant initiated contact"
		interpretation_guidance: "Tenant-initiated contact is almost always positive regardless of content — it shows engagement."
	},
	{
		id:          "portal_activity_drop"
		event_type:  "PortalActivityChanged"
		condition:   "direction == decreasing"
		category:    "communication"
		weight:      "weak"
		polarity:    "negative"
		description: "Portal login frequency decreased"
		interpretation_guidance: "Portal activity is a leading indicator — drops precede other changes. Weak signal alone, strengthens other negative signals."
	},
	{
		id:          "communication_preference_changed"
		event_type:  "ContactPreferenceUpdated"
		category:    "communication"
		weight:      "weak"
		polarity:    "neutral"
		description: "Communication preference changed"
		interpretation_guidance: "May indicate changed phone number or lifestyle. Ensure future outreach uses updated preference."
	},

	// === Compliance (3) ===
	{
		id:          "lease_violation"
		event_type:  "LeaseViolationRecorded"
		category:    "compliance"
		weight:      "strong"
		polarity:    "negative"
		description: "Lease violation recorded"
		interpretation_guidance: "Document thoroughly. Check for pattern — 2+ violations in 12 months is grounds for non-renewal."
		escalation_rules: [
			{
				id:                    "compliance_repeat_violation"
				description:           "Multiple lease violations indicate non-compliance pattern"
				trigger_type:          "count"
				signal_category:       "compliance"
				signal_polarity:       "negative"
				count:                 2
				within_days:           365
				escalated_weight:      "critical"
				escalated_description: "2+ lease violations in 12 months."
				recommended_action:    "Formal notice. Document for potential non-renewal or termination."
			},
		]
	},
	{
		id:          "violation_cured"
		event_type:  "LeaseViolationCured"
		category:    "compliance"
		weight:      "moderate"
		polarity:    "positive"
		description: "Lease violation cured by tenant"
		interpretation_guidance: "Cured violation is positive — tenant is responsive to notices. Speed of cure matters."
	},
	{
		id:          "inspection_failed"
		event_type:  "InspectionCompleted"
		condition:   "result == failed"
		category:    "compliance"
		weight:      "strong"
		polarity:    "negative"
		description: "Space failed inspection"
		interpretation_guidance: "Failed inspection may have regulatory implications. Check jurisdiction requirements for follow-up timeline."
	},

	// === Behavioral (3) ===
	{
		id:          "parking_violation"
		event_type:  "ParkingViolationRecorded"
		category:    "behavioral"
		weight:      "weak"
		polarity:    "negative"
		description: "Parking violation recorded"
		interpretation_guidance: "Often first visible sign of norm disengagement. Individual instances are weak; look for pattern."
		escalation_rules: [
			{
				id:                    "behavioral_pattern"
				description:           "Multiple behavioral changes suggest disengagement"
				trigger_type:          "count"
				signal_category:       "behavioral"
				signal_polarity:       "negative"
				count:                 3
				within_days:           90
				escalated_weight:      "moderate"
				escalated_description: "Multiple behavioral changes in 90 days."
				recommended_action:    "Casual check-in. May precede non-renewal."
			},
		]
	},
	{
		id:          "amenity_usage_change"
		event_type:  "AmenityUsageChanged"
		category:    "behavioral"
		weight:      "weak"
		polarity:    "contextual"
		description: "Amenity usage pattern changed"
		interpretation_guidance: "Individual behavioral signals are weak. Only meaningful in combination with other signals."
	},
	{
		id:          "occupancy_pattern_change"
		event_type:  "OccupancyPatternChanged"
		category:    "behavioral"
		weight:      "weak"
		polarity:    "contextual"
		description: "Occupancy pattern change detected"
		interpretation_guidance: "Extended absences or unusual patterns may indicate subletting or abandonment. Verify with communication."
	},

	// === Relationship (5) ===
	{
		id:          "occupant_added"
		event_type:  "OccupantAdded"
		category:    "relationship"
		weight:      "moderate"
		polarity:    "contextual"
		description: "Occupant added to lease"
		interpretation_guidance: "Growing household is generally positive. Check if it creates compliance concerns (occupancy limits)."
	},
	{
		id:          "occupant_removed"
		event_type:  "OccupantRemoved"
		category:    "relationship"
		weight:      "moderate"
		polarity:    "negative"
		description: "Occupant removed from lease"
		interpretation_guidance: "Check if remaining occupant income supports rent. May indicate relationship change or financial restructuring."
	},
	{
		id:          "guarantor_change"
		event_type:  "GuarantorChanged"
		category:    "relationship"
		weight:      "moderate"
		polarity:    "contextual"
		description: "Guarantor added, removed, or changed"
		interpretation_guidance: "Guarantor removal may signal changed family dynamics. Addition may signal financial concern requiring guarantee."
	},
	{
		id:          "emergency_contact_updated"
		event_type:  "EmergencyContactUpdated"
		category:    "relationship"
		weight:      "weak"
		polarity:    "neutral"
		description: "Emergency contact information updated"
		interpretation_guidance: "Routine update. Significant only if combined with other relationship changes."
	},
	{
		id:          "roommate_departed"
		event_type:  "RoommateDeparted"
		category:    "relationship"
		weight:      "strong"
		polarity:    "negative"
		description: "Roommate departed the lease"
		interpretation_guidance: "Check remaining tenant income against rent. Roommate departure is a high-signal event for financial risk."
		escalation_rules: [
			{
				id:          "cross_relationship_financial"
				description: "Household change with financial impact"
				trigger_type: "cross_category"
				required_categories: [
					{category: "relationship", polarity: "negative", min_count: 1},
					{category: "financial", polarity: "negative", min_count: 1},
				]
				within_days:           90
				escalated_weight:      "strong"
				escalated_description: "Roommate/occupant departure coinciding with payment issues."
				recommended_action:    "Check if remaining occupant income supports rent. Offer restructure options."
			},
		]
	},

	// === Lifecycle (7) ===
	{
		id:          "lease_expiring_90"
		event_type:  "LeaseExpirationApproaching"
		condition:   "days_remaining <= 90"
		category:    "lifecycle"
		weight:      "moderate"
		polarity:    "neutral"
		description: "Lease expiring within 90 days"
		interpretation_guidance: "90-day window is when most renewal decisions are made. Check signal summary before sending renewal offer."
		escalation_rules: [
			{
				id:          "cross_maintenance_lifecycle"
				description: "Active complaints near lease expiration — retention at risk"
				trigger_type: "cross_category"
				required_categories: [
					{category: "maintenance", polarity: "negative", min_count: 2},
					{category: "lifecycle", min_count: 1},
				]
				within_days:           90
				escalated_weight:      "strong"
				escalated_description: "Unresolved maintenance issues with lease expiring."
				recommended_action:    "Resolve maintenance first, then present renewal. Do not send renewal offer while complaints are open."
			},
		]
	},
	{
		id:          "lease_expiring_30"
		event_type:  "LeaseExpirationApproaching"
		condition:   "days_remaining <= 30"
		category:    "lifecycle"
		weight:      "strong"
		polarity:    "negative"
		description: "Lease expiring within 30 days"
		interpretation_guidance: "No renewal response by 30 days out means tenant is likely leaving. Escalate immediately."
	},
	{
		id:          "notice_given"
		event_type:  "NoticeRecorded"
		category:    "lifecycle"
		weight:      "critical"
		polarity:    "negative"
		description: "Tenant gave notice to vacate"
		interpretation_guidance: "Terminal signal for this lease. Begin make-ready planning. Check if retention conversation is still possible."
	},
	{
		id:          "renewal_offered"
		event_type:  "RenewalOfferSent"
		category:    "lifecycle"
		weight:      "info"
		polarity:    "neutral"
		description: "Renewal offer sent to tenant"
		interpretation_guidance: "Track response time. No response within 14 days should be escalated."
	},
	{
		id:          "renewal_signed"
		event_type:  "RenewalSigned"
		category:    "lifecycle"
		weight:      "strong"
		polarity:    "positive"
		description: "Lease renewal signed"
		interpretation_guidance: "Strong positive outcome. Reset signal assessments for the new term."
	},
	{
		id:          "move_in_anniversary"
		event_type:  "MoveInAnniversary"
		category:    "lifecycle"
		weight:      "info"
		polarity:    "positive"
		description: "Move-in anniversary milestone"
		interpretation_guidance: "Long tenancy is positive. 2+ year tenants are high-value — handle complaints with extra care."
	},
	{
		id:          "option_exercise_deadline"
		event_type:  "OptionDeadlineApproaching"
		category:    "lifecycle"
		weight:      "strong"
		polarity:    "neutral"
		description: "Renewal/expansion option exercise deadline approaching"
		interpretation_guidance: "Legally binding deadline. Missing it forfeits the option. Ensure tenant and manager are both aware."
	},
]

// Cross-category escalation rules that span multiple signal registrations.
cross_category_escalation_rules: [...#EscalationRule]
cross_category_escalation_rules: [
	{
		id:          "cross_financial_communication"
		description: "Financial decline combined with communication decline — highest non-renewal predictor"
		trigger_type: "cross_category"
		required_categories: [
			{category: "financial", polarity: "negative", min_count: 2},
			{category: "communication", polarity: "negative", min_count: 1},
		]
		within_days:           90
		escalated_weight:      "critical"
		escalated_description: "Financial problems AND communication avoidance within 90 days."
		recommended_action:    "Highest priority intervention. In-person if possible. Have assistance resources ready."
	},
	{
		id:          "absence_zero_maintenance"
		description: "Long-term tenant with no maintenance activity may indicate disengagement"
		trigger_type: "absence"
		expected_signal_category: "maintenance"
		absent_for_days:          365
		applies_to_condition:     "tenant with active lease > 24 months"
		escalated_weight:         "weak"
		escalated_description:    "No maintenance requests in 12+ months from long-term tenant."
		recommended_action:       "Not actionable alone. Note for context when evaluating other signals."
	},
	{
		id:          "absence_portal_activity"
		description: "Portal login cessation"
		trigger_type: "absence"
		expected_signal_category: "communication"
		absent_for_days:          60
		applies_to_condition:     "tenant who previously logged in at least monthly"
		escalated_weight:         "weak"
		escalated_description:    "Previously active portal user has stopped logging in."
		recommended_action:       "Weak signal alone. Strengthens interpretation of other negative signals."
	},
	{
		id:          "trend_payment_degrading"
		description: "Payment timing getting progressively later"
		trigger_type: "trend"
		trend_direction:   "increasing"
		trend_metric:      "days_past_due_at_payment"
		trend_window_days: 180
		escalated_weight:      "moderate"
		escalated_description: "Payment timing trending later over 6 months."
		recommended_action:    "Early intervention before pattern becomes critical. Friendly check-in."
	},
]

// Category descriptions for agent context generation.
signal_category_descriptions: [string]: string
signal_category_descriptions: {
	financial:     "Payment patterns, balance changes, fee assessments, NSF events, collection actions, credit changes."
	maintenance:   "Work orders, complaints, inspection results, emergency repairs, recurring issues. Both tenant-initiated and property-initiated."
	communication: "Outbound contact attempts, response rates, response times, channel preferences, portal activity. Silence is a signal."
	compliance:    "Lease violations, policy infractions, regulatory notices, inspection failures, permit issues."
	behavioral:    "Occupancy pattern changes, amenity usage changes, parking behavior, portal login frequency, maintenance request pattern changes. Proxy indicators of engagement or disengagement."
	market:        "Comparable rents, vacancy rate changes, local market trends, seasonal patterns."
	relationship:  "Household composition changes, guarantor changes, emergency contact updates, roommate dynamics, co-signer events."
	lifecycle:     "Lease milestone events: approaching expiration, renewal window open, notice period started, move-in anniversary, option exercise deadlines."
}

signal_weight_descriptions: [string]: string
signal_weight_descriptions: {
	critical: "Requires immediate action."
	strong:   "Likely to affect outcome if unaddressed."
	moderate: "Worth noting, contributes to pattern."
	weak:     "Background signal, meaningful only in aggregate."
	info:     "No signal value alone but provides context."
}

signal_polarity_descriptions: [string]: string
signal_polarity_descriptions: {
	positive:   "Favorable indicator."
	negative:   "Unfavorable indicator."
	neutral:    "Neither favorable nor unfavorable on its own."
	contextual: "Polarity depends on context. Agent must reason."
}
