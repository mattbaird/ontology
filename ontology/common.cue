// ontology/common.cue
package propeller

import (
	"strings"
	"time"
)

// ─── Monetary ────────────────────────────────────────────────────────────────

// Money represents a monetary amount. All calculations use integer cents
// to eliminate floating-point errors in financial operations.
#Money: close({
	amount_cents: int
	currency:     *"USD" | =~"^[A-Z]{3}$" // ISO 4217, defaults to USD
})

#NonNegativeMoney: #Money & {
	amount_cents: >=0
}

#PositiveMoney: #Money & {
	amount_cents: >0
}

// ─── Temporal ────────────────────────────────────────────────────────────────

#DateRange: close({
	start: time.Time
	end?:  time.Time // Open-ended if unset
	// CONSTRAINT: end must be after start
	if end != _|_ {
		end: time.Time // Runtime validation ensures end > start
	}
})

// ─── Geographic ──────────────────────────────────────────────────────────────

#USState:
	"AL" | "AK" | "AZ" | "AR" | "CA" | "CO" | "CT" | "DE" | "FL" | "GA" |
	"HI" | "ID" | "IL" | "IN" | "IA" | "KS" | "KY" | "LA" | "ME" | "MD" |
	"MA" | "MI" | "MN" | "MS" | "MO" | "MT" | "NE" | "NV" | "NH" | "NJ" |
	"NM" | "NY" | "NC" | "ND" | "OH" | "OK" | "OR" | "PA" | "RI" | "SC" |
	"SD" | "TN" | "TX" | "UT" | "VT" | "VA" | "WA" | "WV" | "WI" | "WY" |
	"DC" | "PR" | "VI" | "GU" | "AS" | "MP"

#Address: close({
	line1:       string & strings.MinRunes(1)
	line2?:      string
	city:        string & strings.MinRunes(1)
	state:       #USState
	postal_code: =~"^[0-9]{5}(-[0-9]{4})?$"
	country:     *"US" | =~"^[A-Z]{2}$"
	latitude?:   float & >=-90 & <=90
	longitude?:  float & >=-180 & <=180
	county?:     string // Important for tax jurisdictions
})

// ─── Identity ────────────────────────────────────────────────────────────────

// EntityRef is the universal relationship primitive. Every cross-entity
// reference in the ontology uses this type, ensuring that relationships
// are always typed and semantically meaningful.
#EntityRef: close({
	entity_type:  #EntityType
	entity_id:    string & !=""
	relationship: #RelationshipType
})

#EntityType:
	"person" | "organization" | "portfolio" | "property" | "building" | "space" | "lease_space" |
	"lease" | "work_order" | "vendor" | "ledger_entry" | "journal_entry" |
	"account" | "bank_account" | "application" | "inspection" | "document" |
	"jurisdiction" | "jurisdiction_rule" | "property_jurisdiction"

#RelationshipType:
	"belongs_to" | "contains" | "managed_by" | "owned_by" |
	"leased_to" | "occupied_by" | "reported_by" | "assigned_to" |
	"billed_to" | "paid_by" | "performed_by" | "approved_by" |
	"guarantor_for" | "emergency_contact_for" | "employed_by" |
	"related_to" | "parent_of" | "child_of" | "sublease_of"

// ─── Audit ───────────────────────────────────────────────────────────────────

#AuditSource: "user" | "agent" | "import" | "system" | "migration"

// AuditMetadata is attached to every domain entity. It provides full
// traceability for every change, which is critical for:
// - Compliance (trust accounting, fair housing)
// - Agent accountability (which agent made this change, under what authority)
// - Debugging (correlation IDs trace chains of related changes)
#AuditMetadata: close({
	created_by:      string & !="" @computed() // User ID, agent ID, or "system"
	updated_by:      string & !="" @computed()
	created_at:      time.Time @computed()
	updated_at:      time.Time @computed()
	source:          ("user" | "agent" | "import" | "system" | "migration") @computed()
	correlation_id?: string @computed() // Links related changes across entities
	agent_goal_id?:  string @computed() // If source == "agent", which goal triggered this
})

// ─── Contact ─────────────────────────────────────────────────────────────────

#ContactType: "email" | "phone" | "sms" | "mail" | "portal"

#ContactMethod: close({
	type:     "email" | "phone" | "sms" | "mail" | "portal"
	value:    string & strings.MinRunes(1)
	primary:  bool | *false
	verified: bool | *false
	opt_out:  bool | *false // Communication preference
	label?:   string        // "work", "home", "mobile", etc.
})
