// api/v1/common_api.cue
// Shared API types for the external API contract.
// The external API is a versioned contract with its own shapes.
// It imports ontology enums for validation but defines its own response structures.
package api_v1

#PaginationRequest: close({
	page?:      int & >=1 | *1
	page_size?: int & >=1 & <=100 | *25
	sort_by?:   string
	sort_dir?:  "asc" | "desc" | *"desc"
})

#PaginationResponse: close({
	page:      int
	page_size: int
	total:     int
	has_more:  bool
})

#ErrorResponse: close({
	code:    string
	message: string
	details?: [...close({
		field?:  string
		reason:  string
	})]
})

// MoneyResponse uses dollars (not cents) for API consumers.
#MoneyResponse: close({
	amount:   number // dollars, not cents
	currency: string
})
