// ontology/state_machines.cue
package propeller

// Every status enum in the ontology has a corresponding transition map here.
// These are generated into Ent hooks that reject invalid transitions at the
// persistence layer. No code path can violate these transitions.

#LeaseTransitions: {
	draft:                   ["pending_approval", "pending_signature", "terminated"]
	pending_approval:        ["draft", "pending_signature", "terminated"]
	pending_signature:       ["active", "draft", "terminated"]
	active:                  ["expired", "month_to_month_holdover", "terminated", "eviction"]
	expired:                 ["active", "month_to_month_holdover", "renewed", "terminated"]
	month_to_month_holdover: ["active", "renewed", "terminated", "eviction"]
	renewed:                 [] // Terminal — a new lease is created
	terminated:              [] // Terminal
	eviction:                ["terminated"]
}

#SpaceTransitions: {
	vacant:         ["occupied", "make_ready", "down", "model", "reserved"]
	occupied:       ["notice_given"]
	notice_given:   ["make_ready", "occupied"] // Can rescind notice
	make_ready:     ["vacant", "down"]
	down:           ["make_ready", "vacant"]
	model:          ["vacant", "occupied"]
	reserved:       ["vacant", "occupied"]
	owner_occupied: ["vacant"]
}

#BuildingTransitions: {
	active:           ["inactive", "under_renovation"]
	inactive:         ["active"]
	under_renovation: ["active"]
}

#ApplicationTransitions: {
	submitted:              ["screening", "withdrawn"]
	screening:              ["under_review", "withdrawn"]
	under_review:           ["approved", "conditionally_approved", "denied", "withdrawn"]
	approved:               ["expired"] // If lease not signed in time
	conditionally_approved: ["approved", "denied", "withdrawn", "expired"]
	denied:                 [] // Terminal
	withdrawn:              [] // Terminal
	expired:                [] // Terminal
}

#JournalEntryTransitions: {
	draft:            ["pending_approval", "posted"] // Auto-generated can go straight to posted
	pending_approval: ["posted", "draft"]             // Reject sends back to draft
	posted:           ["voided"]
	voided:           [] // Terminal
}

#PortfolioTransitions: {
	onboarding:  ["active"]
	active:      ["inactive", "offboarding"]
	inactive:    ["active", "offboarding"]
	offboarding: ["inactive"] // After all properties migrated
}

#PropertyTransitions: {
	onboarding:       ["active"]
	active:           ["inactive", "under_renovation", "for_sale"]
	inactive:         ["active"]
	under_renovation: ["active", "for_sale"]
	for_sale:         ["active", "inactive"]
}

#PersonRoleTransitions: {
	pending:    ["active", "terminated"]
	active:     ["inactive", "terminated"]
	inactive:   ["active", "terminated"]
	terminated: [] // Terminal
}

#OrganizationTransitions: {
	active:    ["inactive", "suspended", "dissolved"]
	inactive:  ["active", "dissolved"]
	suspended: ["active", "dissolved"]
	dissolved: [] // Terminal
}

#BankAccountTransitions: {
	active:   ["inactive", "frozen", "closed"]
	inactive: ["active", "closed"]
	frozen:   ["active", "closed"]
	closed:   [] // Terminal
}

#ReconciliationTransitions: {
	in_progress: ["balanced", "unbalanced"]
	balanced:    ["approved", "in_progress"] // Reopen if errors found
	unbalanced:  ["in_progress"]
	approved:    [] // Terminal
}
