// codegen/apigen.cue
// Service mapping: maps ontology entities to Connect-RPC services.
// Each service defines CRUD ops + named state transition RPCs.
package codegen

#ServiceDef: {
	name:       string
	base_path:  string
	entities:   [...string]
	operations: [...#OperationDef]
}

#OperationDef: {
	name:        string // RPC method name
	entity:      string
	type:        "create" | "get" | "list" | "update" | "delete" | "transition"
	// REST route path segment for the entity (e.g., "persons", "person-roles")
	entity_path?: string
	// For transition operations
	action?:      string // URL path segment (e.g., "approve", "activate")
	from_status?: [...string]
	to_status?:   string
	extra_fields?: [...string] // Additional request fields
	// Mark transition as having custom handler logic (not generated)
	custom?:     bool | *false
	description: string
}

services: [...#ServiceDef]
services: [
	{
		name:      "PersonService"
		base_path: "/v1"
		entities: ["Person", "Organization", "PersonRole"]
		operations: [
			{name: "CreatePerson", entity: "Person", entity_path: "persons", type: "create", description: "Create a new person"},
			{name: "GetPerson", entity: "Person", entity_path: "persons", type: "get", description: "Get person by ID"},
			{name: "ListPersons", entity: "Person", entity_path: "persons", type: "list", description: "List persons with filtering"},
			{name: "UpdatePerson", entity: "Person", entity_path: "persons", type: "update", description: "Update person fields"},
			{name: "CreateOrganization", entity: "Organization", entity_path: "organizations", type: "create", description: "Create a new organization"},
			{name: "GetOrganization", entity: "Organization", entity_path: "organizations", type: "get", description: "Get organization by ID"},
			{name: "ListOrganizations", entity: "Organization", entity_path: "organizations", type: "list", description: "List organizations"},
			{name: "UpdateOrganization", entity: "Organization", entity_path: "organizations", type: "update", description: "Update organization"},
			{name: "CreatePersonRole", entity: "PersonRole", entity_path: "person-roles", type: "create", description: "Assign a role to a person"},
			{name: "GetPersonRole", entity: "PersonRole", entity_path: "person-roles", type: "get", description: "Get person role by ID"},
			{name: "ListPersonRoles", entity: "PersonRole", entity_path: "person-roles", type: "list", description: "List person roles"},
			{name: "ActivateRole", entity: "PersonRole", entity_path: "person-roles", type: "transition", action: "activate",
				from_status: ["pending"], to_status: "active",
				description: "Activate a pending person role"},
			{name: "DeactivateRole", entity: "PersonRole", entity_path: "person-roles", type: "transition", action: "deactivate",
				from_status: ["active"], to_status: "inactive",
				description: "Deactivate an active person role"},
			{name: "TerminateRole", entity: "PersonRole", entity_path: "person-roles", type: "transition", action: "terminate",
				from_status: ["active", "inactive", "pending"], to_status: "terminated",
				description: "Terminate a person role"},
		]
	},
	{
		name:      "PropertyService"
		base_path: "/v1"
		entities: ["Portfolio", "Property", "Building", "Space"]
		operations: [
			// Portfolio CRUD + transitions
			{name: "CreatePortfolio", entity: "Portfolio", entity_path: "portfolios", type: "create", description: "Create a new portfolio"},
			{name: "GetPortfolio", entity: "Portfolio", entity_path: "portfolios", type: "get", description: "Get portfolio by ID"},
			{name: "ListPortfolios", entity: "Portfolio", entity_path: "portfolios", type: "list", description: "List portfolios"},
			{name: "UpdatePortfolio", entity: "Portfolio", entity_path: "portfolios", type: "update", description: "Update portfolio"},
			{name: "ActivatePortfolio", entity: "Portfolio", entity_path: "portfolios", type: "transition", action: "activate",
				from_status: ["onboarding"], to_status: "active",
				description: "Activate a portfolio after onboarding"},

			// Property CRUD + transitions
			{name: "CreateProperty", entity: "Property", entity_path: "properties", type: "create", description: "Create a new property"},
			{name: "GetProperty", entity: "Property", entity_path: "properties", type: "get", description: "Get property by ID"},
			{name: "ListProperties", entity: "Property", entity_path: "properties", type: "list", description: "List properties with filtering"},
			{name: "UpdateProperty", entity: "Property", entity_path: "properties", type: "update", description: "Update property fields"},
			{name: "ActivateProperty", entity: "Property", entity_path: "properties", type: "transition", action: "activate",
				from_status: ["onboarding"], to_status: "active",
				description: "Activate a property after onboarding"},

			// Building CRUD + transitions
			{name: "CreateBuilding", entity: "Building", entity_path: "buildings", type: "create", description: "Create a new building"},
			{name: "GetBuilding", entity: "Building", entity_path: "buildings", type: "get", description: "Get building by ID"},
			{name: "ListBuildings", entity: "Building", entity_path: "buildings", type: "list", description: "List buildings"},
			{name: "UpdateBuilding", entity: "Building", entity_path: "buildings", type: "update", description: "Update building"},
			{name: "DeactivateBuilding", entity: "Building", entity_path: "buildings", type: "transition", action: "deactivate",
				from_status: ["active"], to_status: "inactive",
				description: "Deactivate a building"},
			{name: "StartBuildingRenovation", entity: "Building", entity_path: "buildings", type: "transition", action: "renovate",
				from_status: ["active"], to_status: "under_renovation",
				description: "Start building renovation"},
			{name: "ActivateBuilding", entity: "Building", entity_path: "buildings", type: "transition", action: "activate",
				from_status: ["inactive", "under_renovation"], to_status: "active",
				description: "Activate a building"},

			// Space CRUD + transitions
			{name: "CreateSpace", entity: "Space", entity_path: "spaces", type: "create", description: "Create a new space within a property"},
			{name: "GetSpace", entity: "Space", entity_path: "spaces", type: "get", description: "Get space by ID"},
			{name: "ListSpaces", entity: "Space", entity_path: "spaces", type: "list", description: "List spaces with filtering"},
			{name: "UpdateSpace", entity: "Space", entity_path: "spaces", type: "update", description: "Update space fields"},
			{name: "OccupySpace", entity: "Space", entity_path: "spaces", type: "transition", action: "occupy",
				from_status: ["vacant", "model", "reserved"], to_status: "occupied",
				description: "Mark a space as occupied"},
			{name: "RecordSpaceNotice", entity: "Space", entity_path: "spaces", type: "transition", action: "notice",
				from_status: ["occupied"], to_status: "notice_given",
				description: "Record notice to vacate"},
			{name: "RescindSpaceNotice", entity: "Space", entity_path: "spaces", type: "transition", action: "rescind-notice",
				from_status: ["notice_given"], to_status: "occupied",
				description: "Rescind a notice to vacate"},
			{name: "StartMakeReady", entity: "Space", entity_path: "spaces", type: "transition", action: "make-ready",
				from_status: ["notice_given", "down"], to_status: "make_ready",
				description: "Start make-ready process"},
			{name: "MarkSpaceVacant", entity: "Space", entity_path: "spaces", type: "transition", action: "vacate",
				from_status: ["make_ready", "down", "model", "reserved", "owner_occupied"], to_status: "vacant",
				description: "Mark a space as vacant"},
			{name: "MarkSpaceDown", entity: "Space", entity_path: "spaces", type: "transition", action: "mark-down",
				from_status: ["vacant", "make_ready"], to_status: "down",
				description: "Mark a space as down (out of service)"},
			{name: "MarkSpaceModel", entity: "Space", entity_path: "spaces", type: "transition", action: "mark-model",
				from_status: ["vacant"], to_status: "model",
				description: "Mark a space as a model unit"},
			{name: "ReserveSpace", entity: "Space", entity_path: "spaces", type: "transition", action: "reserve",
				from_status: ["vacant"], to_status: "reserved",
				description: "Reserve a space"},
		]
	},
	{
		name:      "LeaseService"
		base_path: "/v1"
		entities: ["Lease", "LeaseSpace", "Application"]
		operations: [
			// Lease CRUD + transitions
			{name: "CreateLease", entity: "Lease", entity_path: "leases", type: "create", description: "Create a new lease draft"},
			{name: "GetLease", entity: "Lease", entity_path: "leases", type: "get", description: "Get lease by ID"},
			{name: "ListLeases", entity: "Lease", entity_path: "leases", type: "list", description: "List leases with filtering"},
			{name: "UpdateLease", entity: "Lease", entity_path: "leases", type: "update", description: "Update lease fields (draft only)"},
			{name: "SubmitForApproval", entity: "Lease", entity_path: "leases", type: "transition", action: "submit",
				from_status: ["draft"], to_status: "pending_approval",
				description: "Submit lease draft for approval"},
			{name: "ApproveLease", entity: "Lease", entity_path: "leases", type: "transition", action: "approve",
				from_status: ["pending_approval"], to_status: "pending_signature",
				description: "Approve lease for signing"},
			{name: "SendForSignature", entity: "Lease", entity_path: "leases", type: "transition", action: "sign",
				from_status: ["pending_approval"], to_status: "pending_signature",
				custom: true,
				description: "Send lease for electronic signature"},
			{name: "ActivateLease", entity: "Lease", entity_path: "leases", type: "transition", action: "activate",
				from_status: ["pending_signature"], to_status: "active",
				extra_fields: ["move_in_date"],
				description: "Activate a signed lease"},
			{name: "TerminateLease", entity: "Lease", entity_path: "leases", type: "transition", action: "terminate",
				from_status: ["active", "month_to_month_holdover"], to_status: "terminated",
				extra_fields: ["reason", "move_out_date"],
				description: "Terminate a lease early. Requires reason for audit trail"},
			{name: "RenewLease", entity: "Lease", entity_path: "leases", type: "transition", action: "renew",
				from_status: ["expired", "month_to_month_holdover"], to_status: "renewed",
				description: "Renew a lease"},
			{name: "InitiateEviction", entity: "Lease", entity_path: "leases", type: "transition", action: "evict",
				from_status: ["active", "month_to_month_holdover"], to_status: "eviction",
				description: "Begin eviction process"},
			{name: "RecordNotice", entity: "Lease", entity_path: "leases", type: "transition", action: "notice",
				custom: true,
				description: "Record tenant notice date"},
			{name: "SearchLeases", entity: "Lease", entity_path: "leases", type: "create",
				custom: true,
				description: "Search leases with advanced filters"},
			{name: "GetLeaseLedger", entity: "Lease", entity_path: "leases", type: "get",
				custom: true,
				description: "Get ledger entries for a lease"},
			{name: "RecordPayment", entity: "Lease", entity_path: "leases", type: "create",
				custom: true,
				description: "Record a payment on a lease"},
			{name: "PostCharge", entity: "Lease", entity_path: "leases", type: "create",
				custom: true,
				description: "Post a charge to a lease"},
			{name: "ApplyCredit", entity: "Lease", entity_path: "leases", type: "create",
				custom: true,
				description: "Apply a credit to a lease"},

			// LeaseSpace CRUD
			{name: "CreateLeaseSpace", entity: "LeaseSpace", entity_path: "lease-spaces", type: "create", description: "Create a lease-space association"},
			{name: "GetLeaseSpace", entity: "LeaseSpace", entity_path: "lease-spaces", type: "get", description: "Get lease-space by ID"},
			{name: "ListLeaseSpaces", entity: "LeaseSpace", entity_path: "lease-spaces", type: "list", description: "List lease-space associations"},
			{name: "UpdateLeaseSpace", entity: "LeaseSpace", entity_path: "lease-spaces", type: "update", description: "Update lease-space association"},

			// Application CRUD + transitions
			{name: "CreateApplication", entity: "Application", entity_path: "applications", type: "create", description: "Submit a new lease application"},
			{name: "GetApplication", entity: "Application", entity_path: "applications", type: "get", description: "Get application by ID"},
			{name: "ListApplications", entity: "Application", entity_path: "applications", type: "list", description: "List applications"},
			{name: "ApproveApplication", entity: "Application", entity_path: "applications", type: "transition", action: "approve",
				from_status: ["under_review"], to_status: "approved",
				custom: true,
				description: "Approve a lease application"},
			{name: "DenyApplication", entity: "Application", entity_path: "applications", type: "transition", action: "deny",
				from_status: ["under_review", "conditionally_approved"], to_status: "denied",
				custom: true,
				description: "Deny a lease application"},
		]
	},
	{
		name:      "AccountingService"
		base_path: "/v1"
		entities: ["Account", "LedgerEntry", "JournalEntry", "BankAccount", "Reconciliation"]
		operations: [
			{name: "CreateAccount", entity: "Account", entity_path: "accounts", type: "create", description: "Create a new GL account"},
			{name: "GetAccount", entity: "Account", entity_path: "accounts", type: "get", description: "Get GL account by ID"},
			{name: "ListAccounts", entity: "Account", entity_path: "accounts", type: "list", description: "List GL accounts"},
			{name: "UpdateAccount", entity: "Account", entity_path: "accounts", type: "update", description: "Update GL account"},
			{name: "GetLedgerEntry", entity: "LedgerEntry", entity_path: "ledger-entries", type: "get", description: "Get ledger entry by ID"},
			{name: "ListLedgerEntries", entity: "LedgerEntry", entity_path: "ledger-entries", type: "list", description: "List ledger entries with filtering"},
			{name: "CreateJournalEntry", entity: "JournalEntry", entity_path: "journal-entries", type: "create", description: "Create a new journal entry"},
			{name: "GetJournalEntry", entity: "JournalEntry", entity_path: "journal-entries", type: "get", description: "Get journal entry by ID"},
			{name: "ListJournalEntries", entity: "JournalEntry", entity_path: "journal-entries", type: "list", description: "List journal entries"},
			{name: "PostJournalEntry", entity: "JournalEntry", entity_path: "journal-entries", type: "transition", action: "post",
				from_status: ["draft", "pending_approval"], to_status: "posted",
				custom: true,
				description: "Post a journal entry. Lines must balance (debits = credits)"},
			{name: "VoidJournalEntry", entity: "JournalEntry", entity_path: "journal-entries", type: "transition", action: "void",
				from_status: ["posted"], to_status: "voided",
				custom: true,
				description: "Void a posted journal entry. Creates reversal entry"},
			{name: "CreateBankAccount", entity: "BankAccount", entity_path: "bank-accounts", type: "create", description: "Create a bank account"},
			{name: "GetBankAccount", entity: "BankAccount", entity_path: "bank-accounts", type: "get", description: "Get bank account by ID"},
			{name: "ListBankAccounts", entity: "BankAccount", entity_path: "bank-accounts", type: "list", description: "List bank accounts"},
			{name: "UpdateBankAccount", entity: "BankAccount", entity_path: "bank-accounts", type: "update", description: "Update bank account"},
			{name: "CreateReconciliation", entity: "Reconciliation", entity_path: "reconciliations", type: "create", description: "Start a bank reconciliation"},
			{name: "GetReconciliation", entity: "Reconciliation", entity_path: "reconciliations", type: "get", description: "Get reconciliation by ID"},
			{name: "ListReconciliations", entity: "Reconciliation", entity_path: "reconciliations", type: "list", description: "List reconciliations"},
			{name: "ApproveReconciliation", entity: "Reconciliation", entity_path: "reconciliations", type: "transition", action: "approve",
				from_status: ["balanced"], to_status: "approved",
				custom: true,
				description: "Approve a balanced reconciliation"},
		]
	},
	{
		name:      "ActivityService"
		base_path: "/v1"
		entities: []
		operations: [
			{name: "GetEntityActivity", entity: "ActivityEntry", entity_path: "activity/entity", type: "get",
				custom: true,
				description: "Get chronological activity feed for any entity. Use as FIRST call when assessing an entity."},
			{name: "GetSignalSummary", entity: "ActivityEntry", entity_path: "activity/summary", type: "get",
				custom: true,
				description: "Get pre-aggregated signal summary for an entity with category breakdowns, sentiment, and escalations."},
			{name: "GetPortfolioSignals", entity: "ActivityEntry", entity_path: "activity/portfolio", type: "create",
				custom: true,
				description: "Batch screen multiple entities ranked by concern level. Use for portfolio-wide risk assessment."},
			{name: "SearchActivity", entity: "ActivityEntry", entity_path: "activity/search", type: "create",
				custom: true,
				description: "Full-text search across all activity streams by keyword."},
		]
	},
	{
		name:      "JurisdictionService"
		base_path: "/v1"
		entities: ["Jurisdiction", "PropertyJurisdiction", "JurisdictionRule"]
		operations: [
			// Jurisdiction CRUD + transitions
			{name: "CreateJurisdiction", entity: "Jurisdiction", entity_path: "jurisdictions", type: "create", description: "Create a new jurisdiction"},
			{name: "GetJurisdiction", entity: "Jurisdiction", entity_path: "jurisdictions", type: "get", description: "Get jurisdiction by ID"},
			{name: "ListJurisdictions", entity: "Jurisdiction", entity_path: "jurisdictions", type: "list", description: "List jurisdictions with filtering"},
			{name: "UpdateJurisdiction", entity: "Jurisdiction", entity_path: "jurisdictions", type: "update", description: "Update jurisdiction fields"},
			{name: "ActivateJurisdiction", entity: "Jurisdiction", entity_path: "jurisdictions", type: "transition", action: "activate",
				from_status: ["pending"], to_status: "active",
				description: "Activate a pending jurisdiction"},
			{name: "DissolveJurisdiction", entity: "Jurisdiction", entity_path: "jurisdictions", type: "transition", action: "dissolve",
				from_status: ["active"], to_status: "dissolved",
				extra_fields: ["successor_jurisdiction_id", "dissolution_date"],
				description: "Mark a jurisdiction as dissolved"},
			{name: "MergeJurisdiction", entity: "Jurisdiction", entity_path: "jurisdictions", type: "transition", action: "merge",
				from_status: ["active"], to_status: "merged",
				extra_fields: ["successor_jurisdiction_id", "dissolution_date"],
				description: "Merge a jurisdiction into another"},

			// PropertyJurisdiction CRUD
			{name: "CreatePropertyJurisdiction", entity: "PropertyJurisdiction", entity_path: "property-jurisdictions", type: "create", description: "Link a property to a jurisdiction"},
			{name: "GetPropertyJurisdiction", entity: "PropertyJurisdiction", entity_path: "property-jurisdictions", type: "get", description: "Get property-jurisdiction link by ID"},
			{name: "ListPropertyJurisdictions", entity: "PropertyJurisdiction", entity_path: "property-jurisdictions", type: "list", description: "List property-jurisdiction links"},
			{name: "UpdatePropertyJurisdiction", entity: "PropertyJurisdiction", entity_path: "property-jurisdictions", type: "update", description: "Update property-jurisdiction link"},

			// JurisdictionRule CRUD + transitions
			{name: "CreateJurisdictionRule", entity: "JurisdictionRule", entity_path: "jurisdiction-rules", type: "create", description: "Create a new jurisdiction rule"},
			{name: "GetJurisdictionRule", entity: "JurisdictionRule", entity_path: "jurisdiction-rules", type: "get", description: "Get jurisdiction rule by ID"},
			{name: "ListJurisdictionRules", entity: "JurisdictionRule", entity_path: "jurisdiction-rules", type: "list", description: "List jurisdiction rules"},
			{name: "UpdateJurisdictionRule", entity: "JurisdictionRule", entity_path: "jurisdiction-rules", type: "update", description: "Update jurisdiction rule"},
			{name: "ActivateRule", entity: "JurisdictionRule", entity_path: "jurisdiction-rules", type: "transition", action: "activate",
				from_status: ["draft"], to_status: "active",
				description: "Activate a draft rule"},
			{name: "SupersedeRule", entity: "JurisdictionRule", entity_path: "jurisdiction-rules", type: "transition", action: "supersede",
				from_status: ["active"], to_status: "superseded",
				extra_fields: ["superseded_by_id"],
				description: "Mark a rule as superseded by a newer rule"},
			{name: "ExpireRule", entity: "JurisdictionRule", entity_path: "jurisdiction-rules", type: "transition", action: "expire",
				from_status: ["active"], to_status: "expired",
				description: "Mark a rule as expired"},
			{name: "RepealRule", entity: "JurisdictionRule", entity_path: "jurisdiction-rules", type: "transition", action: "repeal",
				from_status: ["active"], to_status: "repealed",
				description: "Mark a rule as repealed"},
		]
	},
]
