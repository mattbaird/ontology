// ontology/accounting.cue
package propeller

import "time"

// ─── Named Enum Types ───────────────────────────────────────────────────────

#AccountType:    "asset" | "liability" | "equity" | "revenue" | "expense"
#AccountSubtype: "cash" | "accounts_receivable" | "prepaid" | "fixed_asset" |
	"accumulated_depreciation" | "other_asset" |
	"accounts_payable" | "accrued_liability" | "unearned_revenue" |
	"security_deposits_held" | "other_liability" |
	"owners_equity" | "retained_earnings" | "distributions" |
	"rental_income" | "other_income" | "cam_recovery" |
	"percentage_rent_income" |
	"operating_expense" | "maintenance_expense" | "utility_expense" |
	"management_fee_expense" | "depreciation_expense" | "other_expense"
#NormalBalance: "debit" | "credit"

#EntryType: "charge" | "payment" | "credit" | "adjustment" |
	"refund" | "deposit" | "nsf" | "write_off" |
	"late_fee" | "management_fee" | "owner_draw"

#JournalEntrySource: "manual" | "auto_charge" | "payment" | "bank_import" |
	"cam_reconciliation" | "depreciation" | "accrual" |
	"intercompany" | "management_fee" | "system"
#JournalEntryStatus: "draft" | "pending_approval" | "posted" | "voided"

#BankAccountType:   "operating" | "trust" | "security_deposit" | "escrow" | "reserve"
#BankAccountStatus: "active" | "inactive" | "frozen" | "closed"

#ReconciliationStatus: "in_progress" | "balanced" | "unbalanced" | "approved"

// ─── Chart of Accounts ──────────────────────────────────────────────────────

#Account: close({
	#BaseEntity
	account_number: string & !=""
	name:           string & !="" @display()
	description?:   string @text()

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

	// Hidden: generator metadata
	_display_template: "{account_number} — {name}"
})

#AccountDimensions: close({
	entity_id?:   string
	property_id?: string
	dimension_1?: string
	dimension_2?: string
	dimension_3?: string
})

// ─── Ledger Entry ────────────────────────────────────────────────────────────

#LedgerEntry: close({
	#ImmutableEntity
	account_id: string & !="" @immutable()

	entry_type: ("charge" | "payment" | "credit" | "adjustment" |
		"refund" | "deposit" | "nsf" | "write_off" |
		"late_fee" | "management_fee" | "owner_draw") @immutable()

	amount: #Money @immutable()

	// Double-entry
	journal_entry_id: string & !="" @immutable()

	// Temporal
	effective_date: time.Time @immutable()
	posted_date:    time.Time @immutable()

	description: string & !="" @immutable() @text()
	charge_code: string & !="" @immutable()
	memo?:       string @immutable() @text()

	// Dimensional references
	property_id: string & !="" @immutable()
	space_id?:   string @immutable()
	lease_id?:   string @immutable()
	person_id?:  string @immutable()

	// Bank / trust accounting
	bank_account_id?:     string @immutable()
	bank_transaction_id?: string @immutable()

	// Reconciliation — set during reconciliation process, not immutable
	reconciled:         bool | *false
	reconciliation_id?: string
	reconciled_at?:     time.Time

	// For adjustments
	adjusts_entry_id?: string @immutable()

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

	// Hidden: invariant documentation
	_invariant: "LedgerEntries are append-only. Errors corrected via adjustment entries."
})

// ─── Journal Entry ───────────────────────────────────────────────────────────

// NOTE: #JournalEntry is NOT wrapped in close() because close() + conditional
// blocks (if status == "posted") causes CUE to hide all fields from the Go API,
// breaking generators that need to introspect fields.
#JournalEntry: {
	#StatefulEntity

	entry_date:  time.Time @immutable()
	posted_date: time.Time

	description: string & !="" @immutable() @text()

	source_type: ("manual" | "auto_charge" | "payment" | "bank_import" |
		"cam_reconciliation" | "depreciation" | "accrual" |
		"intercompany" | "management_fee" | "system") @immutable()
	source_id?: string @immutable()

	// Approval — set during state transitions, not immutable
	status:      "draft" | "pending_approval" | "posted" | "voided"
	approved_by?: string
	approved_at?: time.Time

	// Batch
	batch_id?: string @immutable()

	// Entity / property scope
	entity_id?:   string @immutable()
	property_id?: string @immutable()

	// Reversal tracking
	reverses_journal_id?:    string @immutable()
	reversed_by_journal_id?: string

	// Line items — minimum 2 lines for double-entry
	lines: [#JournalLine, #JournalLine, ...#JournalLine] @immutable()

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

	// Hidden: invariant documentation
	_invariant: "Sum of debits must equal sum of credits across all lines."
}

#JournalLine: close({
	account_id:  string & !=""
	debit?:      #NonNegativeMoney
	credit?:     #NonNegativeMoney
	description?: string @text()
	dimensions?: #AccountDimensions
	// CONSTRAINT: Must have exactly one of debit or credit (not both, not neither)
	// (enforced at runtime via Ent hooks)
})

// ─── Bank Account ────────────────────────────────────────────────────────────

#BankAccount: close({
	#StatefulEntity
	name: string & !="" @display()

	account_type: "operating" | "trust" | "security_deposit" | "escrow" | "reserve"

	// Linked CoA account
	gl_account_id: string & !=""

	// Bank details
	institution_name:          string & !=""
	routing_number:            =~"^[0-9]{9}$" @sensitive()
	account_mask:              =~"^\\*{4}[0-9]{4}$" @sensitive()
	account_number_encrypted?: string @sensitive()

	// Plaid integration
	plaid_account_id?:    string
	plaid_access_token?:  string @sensitive()

	// Scope
	portfolio_id?: string
	property_id?:  string
	entity_id?:    string

	status: "active" | "inactive" | "frozen" | "closed"

	// Capabilities
	is_default:       bool | *false
	accepts_deposits: bool | *true
	accepts_payments: bool | *true

	// Current balance
	current_balance?:    #Money @computed()
	last_statement_date?: time.Time

	// Hidden: generator metadata
	_display_template: "{name}"
})

// ─── Reconciliation ──────────────────────────────────────────────────────────

#Reconciliation: close({
	#StatefulEntity
	bank_account_id: string & !=""

	period_start:    time.Time
	period_end:      time.Time
	statement_date:  time.Time

	statement_balance: #Money
	gl_balance:        #Money
	difference?:       #Money @computed()

	status: "in_progress" | "balanced" | "unbalanced" | "approved"

	unreconciled_items?: int & >=0 @computed()

	reconciled_by?: string
	reconciled_at?: time.Time
	approved_by?:   string
	approved_at?:   time.Time

	// CONSTRAINTS:
	// Balanced/approved reconciliations must have zero difference
	if status == "balanced" || status == "approved" {
		difference: amount_cents: 0
	}
})
