// events/property_events.cue
// Domain events for the property domain.
package events

import (
	"time"

	"github.com/matthewbaird/ontology/ontology:propeller"
)

#PropertyOnboarded: close({
	property_id:  string
	portfolio_id: string
	property_type: "single_family" | "multi_family" | "commercial_office" |
		"commercial_retail" | "mixed_use" | "industrial" |
		"affordable_housing" | "student_housing" | "senior_living" |
		"vacation_rental" | "mobile_home_park" |
		"self_storage" | "coworking" | "data_center" | "medical_office"
	address: propeller.#Address

	jurisdiction_ids: [...string] // resolved from address
	space_count:      int
})

#SpaceStatusChanged: close({
	space_id:     string
	property_id:  string
	space_number: string

	previous_status: "vacant" | "occupied" | "notice_given" | "make_ready" |
		"down" | "model" | "reserved" | "owner_occupied"
	new_status: "vacant" | "occupied" | "notice_given" | "make_ready" |
		"down" | "model" | "reserved" | "owner_occupied"

	// Context
	lease_id?:      string // if status change is lease-related
	changed_at:     time.Time
	changed_reason: string
})
