// api/v1/person_api.cue
// Versioned request/response shapes for the person API.
// PII fields (SSN, DOB) are excluded from API responses.
package api_v1

#PersonListResponse: close({
	persons:    [...#PersonSummary]
	pagination: #PaginationResponse
})

#PersonSummary: close({
	id:           string
	display_name: string
	email?:       string
	phone?:       string

	// Active roles
	roles: [...close({
		role_type:  string
		scope_type: string
		status:     string
	})]
})

#PersonDetailResponse: close({
	#PersonSummary

	first_name:  string
	middle_name?: string
	last_name:   string

	preferred_contact:    string
	language_preference:  string
	do_not_contact:       bool
	identity_verified:    bool

	contact_methods: [...close({
		type:     string
		value:    string
		primary:  bool
		verified: bool
		label?:   string
	})]

	// Explicitly EXCLUDED: ssn_last_four, date_of_birth (PII)
	// Explicitly EXCLUDED: audit metadata (internal)
})

#OrganizationListResponse: close({
	organizations: [...#OrganizationSummary]
	pagination:    #PaginationResponse
})

#OrganizationSummary: close({
	id:         string
	legal_name: string
	dba_name?:  string
	org_type:   string
	status:     string
})
