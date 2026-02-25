// ontology/accounting.cue
package propeller

import "time"

// ─── Chart of Accounts ──────────────────────────────────────────────────────

#Account: {
	id:             string & !=""
	account_number: string & !=""
	name:           string & !=""
	description?:   string

	account_type: "asset" | "liability" | "equity" | "revenue" | "expense"

	account_subtype: "cash" | "accounts_receivable" | "prepaid" | "fixed_asset" |
		"accumulated_depreciation" | "other_asset" |
		"accounts_payable" | "accrued_liability" | "unearned_revenue" |
		"security_deposits_held" | "other_liability" |
		"owners_equity" | "retained_earnings" | "distributions" |
		"rental_income" | "other_income" | "cam_recovery" |
		"percentage_rent_income" |
		"operating_expense" | "maintenance_expense" | "utility_expense" |
		"management_fee_expense" | "depreciation_expense" | "other_expense"

	// Hierarchy
	parent_account_id?: string
	depth:              int & >=0

	// Multi-dimensional
	dimensions?: #AccountDimensions

	// Behavior
	normal_balance:        "debit" | "credit"
	is_header:             bool | *false
	is_system:             bool | *false
	allows_direct_posting: bool | *true

	// Status
	status: "active" | "inactive" | "archived"

	// Trust accounting flag
	is_trust_account: bool | *false
	trust_type?:      "operating" | "security_deposit" | "escrow"

	// Budgeting
	budget_amount?: #Money

	// Tax
	tax_line?: string

	// CONSTRAINTS:

	// Asset and expense accounts have debit normal balance
	if account_type == "asset" || account_type == "expense" {
		normal_balance: "debit"
	}

	// Liability, equity, and revenue accounts have credit normal balance
	if account_type == "liability" || account_type == "equity" || account_type == "revenue" {
		normal_balance: "credit"
	}

	// Header accounts cannot receive direct postings
	if is_header {
		allows_direct_posting: false
	}

	// Trust accounts must specify trust type
	if is_trust_account {
		trust_type: string & !=""
	}

	audit: #AuditMetadata
}

#AccountDimensions: {
	entity_id?:   string
	property_id?: string
	dimension_1?: string
	dimension_2?: string
	dimension_3?: string
}

// ─── Ledger Entry ────────────────────────────────────────────────────────────

#LedgerEntry: {
	id:         string & !=""
	account_id: string & !=""

	entry_type: "charge" | "payment" | "credit" | "adjustment" |
		"refund" | "deposit" | "nsf" | "write_off" |
		"late_fee" | "management_fee" | "owner_draw"

	amount: #Money

	// Double-entry
	journal_entry_id: string & !=""

	// Temporal
	effective_date: time.Time
	posted_date:    time.Time

	description: string & !=""
	charge_code: string & !=""
	memo?:       string

	// Dimensional references
	property_id: string & !=""
	space_id?:   string
	lease_id?:   string
	person_id?:  string

	// Bank / trust accounting
	bank_account_id?:     string
	bank_transaction_id?: string

	// Reconciliation
	reconciled:         bool | *false
	reconciliation_id?: string
	reconciled_at?:     time.Time

	// For adjustments
	adjusts_entry_id?: string

	// CONSTRAINTS:

	// Payment/refund/nsf entries require a person
	if entry_type == "payment" || entry_type == "refund" || entry_type == "nsf" {
		person_id: string & !=""
	}

	// Charge/late_fee entries require a lease
	if entry_type == "charge" || entry_type == "late_fee" {
		lease_id: string & !=""
	}

	// Adjustments must reference the entry they adjust
	if entry_type == "adjustment" {
		adjusts_entry_id: string & !=""
	}

	// Reconciled entries must have reconciliation details
	if reconciled {
		reconciliation_id: string & !=""
		reconciled_at:     time.Time
	}

	// CONSTRAINT: LedgerEntries are immutable (enforced at Ent hook level)

	audit: #AuditMetadata
}

// ─── Journal Entry ───────────────────────────────────────────────────────────

#JournalEntry: {
	id: string & !=""

	entry_date:  time.Time
	posted_date: time.Time

	description: string & !=""

	source_type: "manual" | "auto_charge" | "payment" | "bank_import" |
		"cam_reconciliation" | "depreciation" | "accrual" |
		"intercompany" | "management_fee" | "system"
	source_id?: string

	// Approval
	status:      "draft" | "pending_approval" | "posted" | "voided"
	approved_by?: string
	approved_at?: time.Time

	// Batch
	batch_id?: string

	// Entity / property scope
	entity_id?:   string
	property_id?: string

	// Reversal tracking
	reverses_journal_id?:    string
	reversed_by_journal_id?: string

	// Line items — minimum 2 lines for double-entry
	lines: [#JournalLine, #JournalLine, ...#JournalLine]

	// CONSTRAINTS:

	// Posted manual entries require approval
	if status == "posted" {
		if source_type == "manual" {
			approved_by: string & !=""
			approved_at: time.Time
		}
	}

	// Voided entries must reference the reversing journal
	if status == "voided" {
		reversed_by_journal_id: string & !=""
	}

	// CONSTRAINT: Sum of debits must equal sum of credits (enforced at Ent hook level)

	audit: #AuditMetadata
}

#JournalLine: {
	account_id:  string & !=""
	debit?:      #NonNegativeMoney
	credit?:     #NonNegativeMoney
	description?: string
	dimensions?: #AccountDimensions
	// CONSTRAINT: Must have exactly one of debit or credit (not both, not neither)
	// (enforced at runtime via Ent hooks)
}

// ─── Bank Account ────────────────────────────────────────────────────────────

#BankAccount: {
	id:   string & !=""
	name: string & !=""

	account_type: "operating" | "trust" | "security_deposit" | "escrow" | "reserve"

	// Linked CoA account
	gl_account_id: string & !=""

	// Bank details
	bank_name:                string & !=""
	routing_number?:          string
	account_number_last_four: =~"^[0-9]{4}$"

	// Scope
	portfolio_id?: string
	property_id?:  string
	entity_id?:    string

	status: "active" | "inactive" | "frozen" | "closed"

	// Current balance
	current_balance?:    #Money
	last_reconciled_at?: time.Time

	// Trust accounting controls
	is_trust:             bool | *false
	trust_state?:         =~"^[A-Z]{2}$"
	commingling_allowed:  bool | *false

	// CONSTRAINTS:
	// Trust accounts must specify state and prohibit commingling
	if is_trust {
		trust_state:        =~"^[A-Z]{2}$"
		commingling_allowed: false
	}

	audit: #AuditMetadata
}

// ─── Reconciliation ──────────────────────────────────────────────────────────

#Reconciliation: {
	id:              string & !=""
	bank_account_id: string & !=""

	period_start: time.Time
	period_end:   time.Time

	statement_balance: #Money
	system_balance:    #Money
	difference:        #Money

	status: "in_progress" | "balanced" | "unbalanced" | "approved"

	matched_transaction_count:   int & >=0
	unmatched_transaction_count: int & >=0

	completed_by?: string
	completed_at?: time.Time
	approved_by?:  string
	approved_at?:  time.Time

	// CONSTRAINTS:
	// Balanced/approved reconciliations must have zero difference
	if status == "balanced" || status == "approved" {
		difference: amount_cents: 0
	}

	audit: #AuditMetadata
}
