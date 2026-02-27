// ontology/property.cue
package propeller

import (
	"strings"
	"time"
)

// ─── Named Enum Types ───────────────────────────────────────────────────────

#PortfolioStatus: "active" | "inactive" | "onboarding" | "offboarding"

#PropertyType: "single_family" | "multi_family" | "commercial_office" |
	"commercial_retail" | "mixed_use" | "industrial" |
	"affordable_housing" | "student_housing" | "senior_living" |
	"vacation_rental" | "mobile_home_park" |
	"self_storage" | "coworking" | "data_center" | "medical_office"

#PropertyStatus: "active" | "inactive" | "under_renovation" | "for_sale" | "onboarding"

#BuildingType: "residential" | "commercial" | "mixed_use" | "parking_structure" |
	"industrial" | "storage" | "auxiliary"

#BuildingStatus: "active" | "inactive" | "under_renovation"

#SpaceType: "residential_unit" | "commercial_office" | "commercial_retail" |
	"storage" | "parking" | "common_area" |
	"industrial" | "lot_pad" | "bed_space" | "desk_space" |
	"parking_garage" | "private_office" | "warehouse" | "amenity" |
	"rack" | "cage" | "server_room" | "other"

#SpaceStatus: "vacant" | "occupied" | "notice_given" | "make_ready" |
	"down" | "model" | "reserved" | "owner_occupied"

// ─── Portfolio ───────────────────────────────────────────────────────────────
// The top-level organizational grouping. A management company may manage
// multiple portfolios for different ownership entities.

#Portfolio: close({
	#StatefulEntity
	name:     string & strings.MinRunes(1) @display()
	owner_id: string & !="" // Organization ID of the ownership entity

	management_type: "self_managed" | "third_party" | "hybrid"

	description?: string @text()

	status: "active" | "inactive" | "onboarding" | "offboarding"

	// Default financial references
	default_chart_of_accounts_id?: string
	default_bank_account_id?:      string

	// Hidden: generator metadata
	_display_template: "{name}"
})

// ─── Property ────────────────────────────────────────────────────────────────

#Property: close({
	#StatefulEntity
	portfolio_id: string & !=""
	name:         string & strings.MinRunes(1) @display()
	address:      #Address

	property_type: "single_family" | "multi_family" | "commercial_office" |
		"commercial_retail" | "mixed_use" | "industrial" |
		"affordable_housing" | "student_housing" | "senior_living" |
		"vacation_rental" | "mobile_home_park" |
		"self_storage" | "coworking" | "data_center" | "medical_office"

	status: "active" | "inactive" | "under_renovation" | "for_sale" | "onboarding"

	// Physical
	year_built:           int & >=1800 & <=2030
	total_square_footage: float & >0
	total_spaces:         int & >=1
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

	// Single-family = exactly 1 space
	if property_type == "single_family" {
		total_spaces: 1
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

	// Hidden: generator metadata
	_display_template: "{name}"
})

// ─── Building ────────────────────────────────────────────────────────────────
// Optional grouping between Property and Space. Not all properties have
// distinct buildings (e.g., single-family), but campuses and multi-building
// complexes need this level.

#Building: close({
	#StatefulEntity
	property_id: string & !=""
	name:        string & strings.MinRunes(1) @display()

	building_type: "residential" | "commercial" | "mixed_use" | "parking_structure" |
		"industrial" | "storage" | "auxiliary"

	address?: #Address
	description?: string @text()

	status: "active" | "inactive" | "under_renovation"

	floors?:                       int & >=1
	year_built?:                   int & >=1800 & <=2030
	total_square_footage?:         float & >0
	total_rentable_square_footage?: float & >0

	// Hidden: generator metadata
	_display_template: "{name}"
})

// ─── Space ──────────────────────────────────────────────────────────────────
// A leasable (or non-leasable) area within a property or building.
// Replaces the former "Unit" entity with expanded capabilities.

#Space: close({
	#StatefulEntity
	property_id: string & !=""
	space_number: string & strings.MinRunes(1) @display() // "101", "A", "Suite 200", etc.

	space_type: "residential_unit" | "commercial_office" | "commercial_retail" |
		"storage" | "parking" | "common_area" |
		"industrial" | "lot_pad" | "bed_space" | "desk_space" |
		"parking_garage" | "private_office" | "warehouse" | "amenity" |
		"rack" | "cage" | "server_room" | "other"

	status: "vacant" | "occupied" | "notice_given" | "make_ready" |
		"down" | "model" | "reserved" | "owner_occupied"

	// Hierarchy
	building_id?:      string
	parent_space_id?:  string

	// Leasability
	leasable:           bool | *true
	shared_with_parent: bool | *false

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

	// Specialized infrastructure for commercial/industrial spaces
	specialized_infrastructure?: [...("medical_plumbing" | "clean_room" | "high_voltage" | "loading_dock" | "commercial_kitchen" | "server_room" | "cold_storage" | "hazmat_ventilation" | "grease_trap" | "exhaust_hood")]

	// Financial
	market_rent?: #NonNegativeMoney

	// For affordable housing — space-level income restrictions
	ami_restriction?: int & >=0 & <=150 // % of Area Median Income

	// Active lease (computed from LeaseSpace relationship traversal)
	active_lease_id?: string

	// CONSTRAINTS:

	// Residential spaces should have bedroom/bathroom counts
	if space_type == "residential_unit" {
		bedrooms:  int
		bathrooms: float
	}

	// Parking/storage/lot_pad don't have bedrooms
	if space_type == "parking" || space_type == "storage" || space_type == "lot_pad" {
		bedrooms:  0 | *0
		bathrooms: 0 | *0
	}

	// Common areas are not directly leasable
	if space_type == "common_area" {
		leasable: false
	}

	// Occupied spaces must have an active lease
	if status == "occupied" {
		active_lease_id: string & !=""
	}

	// CONSTRAINT: If parent_space_id is set, building_id should match parent's building_id
	// (enforced at runtime via Ent hooks — cross-entity validation)

	// Hidden: generator metadata
	_display_template: "Space {space_number}"
})
