// ontology/state_machines.cue
package propeller

// Every status enum in the ontology has a corresponding transition map here.
// These are generated into Ent hooks that reject invalid transitions at the
// persistence layer. No code path can violate these transitions.

// #StateMachine defines a state machine as a map of source state → valid target states.
#StateMachine: {
	[string]: [...string]
}

// #StateMachines is the unified map of all entity state machines.
// Keyed by snake_case entity name for direct lookup from generators.
#StateMachines: {
	lease: #StateMachine & {
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

	space: #StateMachine & {
		vacant:         ["occupied", "make_ready", "down", "model", "reserved"]
		occupied:       ["notice_given"]
		notice_given:   ["make_ready", "occupied"] // Can rescind notice
		make_ready:     ["vacant", "down"]
		down:           ["make_ready", "vacant"]
		model:          ["vacant", "occupied"]
		reserved:       ["vacant", "occupied"]
		owner_occupied: ["vacant"]
	}

	building: #StateMachine & {
		active:           ["inactive", "under_renovation"]
		inactive:         ["active"]
		under_renovation: ["active"]
	}

	application: #StateMachine & {
		submitted:              ["screening", "withdrawn"]
		screening:              ["under_review", "withdrawn"]
		under_review:           ["approved", "conditionally_approved", "denied", "withdrawn"]
		approved:               ["expired"] // If lease not signed in time
		conditionally_approved: ["approved", "denied", "withdrawn", "expired"]
		denied:                 [] // Terminal
		withdrawn:              [] // Terminal
		expired:                [] // Terminal
	}

	journal_entry: #StateMachine & {
		draft:            ["pending_approval", "posted"] // Auto-generated can go straight to posted
		pending_approval: ["posted", "draft"]             // Reject sends back to draft
		posted:           ["voided"]
		voided:           [] // Terminal
	}

	portfolio: #StateMachine & {
		onboarding:  ["active"]
		active:      ["inactive", "offboarding"]
		inactive:    ["active", "offboarding"]
		offboarding: ["inactive"] // After all properties migrated
	}

	property: #StateMachine & {
		onboarding:       ["active"]
		active:           ["inactive", "under_renovation", "for_sale"]
		inactive:         ["active"]
		under_renovation: ["active", "for_sale"]
		for_sale:         ["active", "inactive"]
	}

	person_role: #StateMachine & {
		pending:    ["active", "terminated"]
		active:     ["inactive", "terminated"]
		inactive:   ["active", "terminated"]
		terminated: [] // Terminal
	}

	organization: #StateMachine & {
		active:    ["inactive", "suspended", "dissolved"]
		inactive:  ["active", "dissolved"]
		suspended: ["active", "dissolved"]
		dissolved: [] // Terminal
	}

	bank_account: #StateMachine & {
		active:   ["inactive", "frozen", "closed"]
		inactive: ["active", "closed"]
		frozen:   ["active", "closed"]
		closed:   [] // Terminal
	}

	reconciliation: #StateMachine & {
		in_progress: ["balanced", "unbalanced"]
		balanced:    ["approved", "in_progress"] // Reopen if errors found
		unbalanced:  ["in_progress"]
		approved:    [] // Terminal
	}

	jurisdiction: #StateMachine & {
		pending:   ["active"]
		active:    ["dissolved", "merged"]
		dissolved: [] // Terminal
		merged:    [] // Terminal
	}

	jurisdiction_rule: #StateMachine & {
		draft:      ["active"]
		active:     ["superseded", "expired", "repealed"]
		superseded: [] // Terminal
		expired:    [] // Terminal
		repealed:   [] // Terminal
	}
}
