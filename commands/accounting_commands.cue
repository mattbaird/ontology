// commands/accounting_commands.cue
// CQRS write-side commands for the accounting domain.
package commands

import (
	"time"

	"github.com/matthewbaird/ontology/ontology:propeller"
)

#PostJournalEntry: close({
	entry_date:  time.Time
	description: string
	source_type: "manual" | "auto_charge" | "payment" | "bank_import" |
		"cam_reconciliation" | "depreciation" | "accrual" |
		"intercompany" | "management_fee" | "system"
	property_id?: string
	entity_id?:   string
	lines: [...close({
		account_id:   string
		debit?:       propeller.#NonNegativeMoney
		credit?:      propeller.#NonNegativeMoney
		description?: string
	})]

	// Execution plan:
	// 1. Validate lines balance (sum debits == sum credits)
	// 2. Create JournalEntry entity (status: draft or posted based on source_type)
	// 3. Create LedgerEntry for each line
	// 4. If source_type == "manual" and auto-post, transition to pending_approval
	// 5. Emit: JournalEntryPosted event (if posted)

	_affects:             ["journal_entry", "ledger_entry"]
	_requires_permission: "journal_entry:create"
})

#Reconcile: close({
	bank_account_id:  string
	period_start:     time.Time
	period_end:       time.Time
	statement_date:   time.Time
	statement_balance: propeller.#Money
	matched_entries: [...close({
		ledger_entry_id: string
		bank_reference?: string
	})]

	// Execution plan:
	// 1. Create or update Reconciliation entity (status: in_progress)
	// 2. Mark matched LedgerEntries as reconciled
	// 3. Calculate GL balance for period
	// 4. If statement_balance == gl_balance, transition to balanced
	// 5. Else transition to unbalanced
	// 6. Emit: ReconciliationCompleted event (when balanced/approved)

	_affects:             ["reconciliation", "ledger_entry"]
	_requires_permission: "reconciliation:start"
})
