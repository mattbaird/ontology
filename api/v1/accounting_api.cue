// api/v1/accounting_api.cue
// Versioned request/response shapes for the accounting API.
package api_v1

#AccountListResponse: close({
	accounts:   [...#AccountSummary]
	pagination: #PaginationResponse
})

#AccountSummary: close({
	id:              string
	account_number:  string
	name:            string
	account_type:    "asset" | "liability" | "equity" | "revenue" | "expense"
	account_subtype: string
	normal_balance:  "debit" | "credit"
	status:          string
	is_header:       bool
	depth:           int
	balance?:        #MoneyResponse
})

#JournalEntryListResponse: close({
	journal_entries: [...#JournalEntrySummary]
	pagination:      #PaginationResponse
})

#JournalEntrySummary: close({
	id:          string
	entry_date:  string // ISO date
	posted_date: string // ISO date
	description: string
	source_type: string
	status:      "draft" | "pending_approval" | "posted" | "voided"
	line_count:  int
	total_debits: #MoneyResponse
})

#ReconciliationListResponse: close({
	reconciliations: [...#ReconciliationSummary]
	pagination:      #PaginationResponse
})

#ReconciliationSummary: close({
	id:               string
	bank_account_name: string
	period_start:      string // ISO date
	period_end:        string // ISO date
	status:            "in_progress" | "balanced" | "unbalanced" | "approved"
	statement_balance: #MoneyResponse
	gl_balance:        #MoneyResponse
	difference:        #MoneyResponse
	unreconciled_items: int
})
