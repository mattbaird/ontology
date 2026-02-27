// commands/property_commands.cue
// CQRS write-side commands for the property domain.
package commands

import "github.com/matthewbaird/ontology/ontology:propeller"

#OnboardProperty: close({
	name:          string
	portfolio_id:  string
	address:       propeller.#Address
	property_type: "single_family" | "multi_family" | "commercial_office" |
		"commercial_retail" | "mixed_use" | "industrial" |
		"affordable_housing" | "student_housing" | "senior_living" |
		"vacation_rental" | "mobile_home_park" |
		"self_storage" | "coworking" | "data_center" | "medical_office"
	year_built:   int & >=1800 & <=2030
	total_spaces: int & >=1

	// Optional: bulk space creation during onboarding
	spaces?: [...close({
		space_number:    string
		space_type:      "residential_unit" | "commercial_office" | "commercial_retail" |
			"storage" | "parking" | "common_area" |
			"industrial" | "lot_pad" | "bed_space" | "desk_space" |
			"parking_garage" | "private_office" | "warehouse" | "amenity" |
			"rack" | "cage" | "server_room" | "other"
		floor?:          int
		square_footage?: float & >0
		bedrooms?:       int & >=0
		bathrooms?:      float & >=0
		market_rent?:    propeller.#NonNegativeMoney
	})]

	// Execution plan:
	// 1. Create Property entity (status: onboarding)
	// 2. Geocode address -> derive jurisdiction stack
	// 3. Create PropertyJurisdiction records for each jurisdiction
	// 4. Resolve jurisdiction rules for this property
	// 5. If spaces provided, create Space entities
	// 6. Create default chart of accounts if portfolio has one
	// 7. Emit: PropertyOnboarded event

	_affects:             ["property", "property_jurisdiction", "space"]
	_requires_permission: "property:create"
})
