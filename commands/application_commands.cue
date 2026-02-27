// commands/application_commands.cue
// CQRS write-side commands for the application domain.
package commands

import (
	"time"

	"github.com/matthewbaird/ontology/ontology:propeller"
)

#SubmitApplication: close({
	property_id:               string
	space_id?:                 string
	applicant_person_id:       string
	desired_move_in:           time.Time
	desired_lease_term_months: int & >0
	application_fee:           propeller.#NonNegativeMoney

	// Execution plan:
	// 1. Create Application entity (status: submitted)
	// 2. If application_fee > 0, create LedgerEntry for fee
	// 3. Initiate screening if auto-screen enabled
	// 4. Emit: ApplicationSubmitted event

	_affects:             ["application", "ledger_entry"]
	_requires_permission: "application:process"
})

#ApproveApplication: close({
	application_id:    string
	decision_by:       string
	conditions?:       [...string]
	decision_reason?:  string

	// Execution plan:
	// 1. Validate application is in status "under_review" or "conditionally_approved"
	// 2. Set decision metadata (decision_by, decision_at, conditions)
	// 3. Transition status -> approved (or conditionally_approved if conditions)
	// 4. Emit: ApplicationApproved event

	_affects:             ["application"]
	_requires_permission: "application:approve"
})

#DenyApplication: close({
	application_id:  string
	decision_by:     string
	decision_reason: string // Required for fair housing compliance

	// Execution plan:
	// 1. Validate application is in status "under_review" or "conditionally_approved"
	// 2. Set decision metadata
	// 3. Transition status -> denied
	// 4. Emit: ApplicationDenied event

	_affects:             ["application"]
	_requires_permission: "application:deny"
})
