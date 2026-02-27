// api/v1/property_api.cue
// Versioned request/response shapes for the property API.
package api_v1

#PropertyListResponse: close({
	properties: [...#PropertySummary]
	pagination: #PaginationResponse
})

#PropertySummary: close({
	id:   string
	name: string
	property_type: "single_family" | "multi_family" | "commercial_office" |
		"commercial_retail" | "mixed_use" | "industrial" |
		"affordable_housing" | "student_housing" | "senior_living" |
		"vacation_rental" | "mobile_home_park" |
		"self_storage" | "coworking" | "data_center" | "medical_office"
	status: "active" | "inactive" | "under_renovation" | "for_sale" | "onboarding"

	// Address summary
	city:  string
	state: string

	// Key metrics
	total_spaces:    int
	occupied_spaces: int
	vacancy_rate:    float

	// Denormalized
	portfolio_name: string
})

#PropertyDetailResponse: close({
	#PropertySummary

	address: close({
		line1:       string
		line2?:      string
		city:        string
		state:       string
		postal_code: string
		county?:     string
	})

	year_built:           int
	total_square_footage: float

	// Buildings
	buildings: [...close({
		id:            string
		name:          string
		building_type: string
		status:        string
		space_count:   int
	})]

	// Jurisdiction info
	jurisdictions: [...close({
		id:   string
		name: string
		type: string
	})]

	// Financial summary
	monthly_revenue?: #MoneyResponse
	monthly_expenses?: #MoneyResponse
})
