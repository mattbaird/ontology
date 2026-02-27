// api/v1/lease_api.cue
// Versioned request/response shapes for the lease API.
// Responses are projections — denormalized, dollars not cents, ISO date strings.
package api_v1

// === List / Detail responses ===

#LeaseListResponse: close({
	leases:     [...#LeaseSummary]
	pagination: #PaginationResponse
})

#LeaseSummary: close({
	id: string
	lease_type: "fixed_term" | "month_to_month" |
		"commercial_nnn" | "commercial_nn" | "commercial_n" | "commercial_gross" | "commercial_modified_gross" |
		"affordable" | "section_8" | "student" |
		"ground_lease" | "short_term" | "membership"
	status: "draft" | "pending_approval" | "pending_signature" | "active" |
		"expired" | "month_to_month_holdover" | "renewed" |
		"terminated" | "eviction"
	base_rent:  #MoneyResponse
	term_start: string // ISO date string
	term_end?:  string

	// Denormalized (not on Lease entity internally)
	property_name: string
	primary_space: string
	tenant_name:   string

	// Computed
	days_remaining?: int
})

#LeaseDetailResponse: close({
	#LeaseSummary

	security_deposit: #MoneyResponse
	liability_type:   "joint_and_several" | "individual" | "by_the_bed" | "proportional"
	move_in_date?:    string

	// Flattened from relationships
	spaces: [...close({
		space_id:     string
		space_number: string
		space_type:   "residential_unit" | "commercial_office" | "commercial_retail" |
			"storage" | "parking" | "common_area" |
			"industrial" | "lot_pad" | "bed_space" | "desk_space" |
			"parking_garage" | "private_office" | "warehouse" | "amenity" |
			"rack" | "cage" | "server_room" | "other"
		relationship: "primary" | "expansion" | "sublease" | "shared_access" |
			"parking" | "storage" | "loading_dock" | "rooftop" |
			"patio" | "signage" | "included" | "membership"
		square_footage?: float
	})]

	tenants: [...close({
		person_id: string
		name:      string
		standing:  string
		balance:   #MoneyResponse
	})]

	// Commercial structures (conditionally present)
	cam_terms?:       _
	percentage_rent?: _

	// Jurisdiction constraints currently in effect
	jurisdiction_constraints?: close({
		deposit_limit?:       #MoneyResponse
		rent_increase_cap?:   string // "5% + CPI, max 10%"
		notice_period_days?:  int
		just_cause_required?: bool
		rent_controlled?:     bool
	})
})

// === Command-mapped endpoints ===

// POST /v1/leases/{id}/move-in
#MoveInRequest: close({
	actual_move_in_date:  string // ISO date
	key_handoff_notes?:   string
	inspection_completed: bool
	initial_meter_readings?: [...close({
		meter_id: string
		reading:  number
	})]
})

// POST /v1/leases/{id}/renew
#RenewLeaseRequest: close({
	new_term_start:      string // ISO date
	new_term_end:        string // ISO date
	new_base_rent:       #MoneyResponse
	rent_change_reason?: string
})
