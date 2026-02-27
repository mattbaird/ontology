// events/accounting_events.cue
// Domain events for the accounting domain.
package events

import (
	"time"

	"github.com/matthewbaird/ontology/ontology:propeller"
)

#JournalEntryPosted: close({
	journal_entry_id: string
	property_id?:     string
	entity_id?:       string

	entry_date:  time.Time
	posted_date: time.Time
	source_type: "manual" | "auto_charge" | "payment" | "bank_import" |
		"cam_reconciliation" | "depreciation" | "accrual" |
		"intercompany" | "management_fee" | "system"

	line_count:   int
	total_debits: propeller.#NonNegativeMoney

	approved_by?: string
})

#ReconciliationCompleted: close({
	reconciliation_id: string
	bank_account_id:   string

	period_start:     time.Time
	period_end:       time.Time
	statement_balance: propeller.#Money
	gl_balance:        propeller.#Money
	difference:        propeller.#Money

	status:             "balanced" | "unbalanced" | "approved"
	unreconciled_items: int
	reconciled_by?:     string
})
