// codegen/apigen.cue
// Service mapping: maps ontology entities to Connect-RPC services.
// Each service defines CRUD ops + named state transition RPCs.
package codegen

#ServiceDef: {
	name:         string
	base_path:    string
	entities:     [...string]
	operations:   [...#OperationDef]
}

#OperationDef: {
	name:        string // RPC method name
	entity:      string
	type:        "create" | "get" | "list" | "update" | "delete" | "transition"
	// For transition operations
	from_status?: [...string]
	to_status?:   string
	extra_fields?: [...string] // Additional request fields
	description:  string
}

services: [...#ServiceDef]
services: [
	{
		name: "PersonService"
		base_path: "/v1/persons"
		entities: ["Person", "Organization", "PersonRole"]
		operations: [
			{name: "CreatePerson", entity: "Person", type: "create", description: "Create a new person"},
			{name: "GetPerson", entity: "Person", type: "get", description: "Get person by ID"},
			{name: "ListPersons", entity: "Person", type: "list", description: "List persons with filtering"},
			{name: "UpdatePerson", entity: "Person", type: "update", description: "Update person fields"},
			{name: "CreateOrganization", entity: "Organization", type: "create", description: "Create a new organization"},
			{name: "GetOrganization", entity: "Organization", type: "get", description: "Get organization by ID"},
			{name: "ListOrganizations", entity: "Organization", type: "list", description: "List organizations"},
			{name: "UpdateOrganization", entity: "Organization", type: "update", description: "Update organization"},
			{name: "CreatePersonRole", entity: "PersonRole", type: "create", description: "Assign a role to a person"},
			{name: "GetPersonRole", entity: "PersonRole", type: "get", description: "Get person role by ID"},
			{name: "ListPersonRoles", entity: "PersonRole", type: "list", description: "List person roles"},
			{name: "ActivateRole", entity: "PersonRole", type: "transition", from_status: ["pending"], to_status: "active",
				description: "Activate a pending person role"},
			{name: "DeactivateRole", entity: "PersonRole", type: "transition", from_status: ["active"], to_status: "inactive",
				description: "Deactivate an active person role"},
			{name: "TerminateRole", entity: "PersonRole", type: "transition", from_status: ["active", "inactive", "pending"], to_status: "terminated",
				description: "Terminate a person role"},
		]
	},
	{
		name: "PropertyService"
		base_path: "/v1/properties"
		entities: ["Portfolio", "Property", "Unit"]
		operations: [
			{name: "CreatePortfolio", entity: "Portfolio", type: "create", description: "Create a new portfolio"},
			{name: "GetPortfolio", entity: "Portfolio", type: "get", description: "Get portfolio by ID"},
			{name: "ListPortfolios", entity: "Portfolio", type: "list", description: "List portfolios"},
			{name: "UpdatePortfolio", entity: "Portfolio", type: "update", description: "Update portfolio"},
			{name: "ActivatePortfolio", entity: "Portfolio", type: "transition", from_status: ["onboarding"], to_status: "active",
				description: "Activate a portfolio after onboarding"},
			{name: "CreateProperty", entity: "Property", type: "create", description: "Create a new property"},
			{name: "GetProperty", entity: "Property", type: "get", description: "Get property by ID"},
			{name: "ListProperties", entity: "Property", type: "list", description: "List properties with filtering"},
			{name: "UpdateProperty", entity: "Property", type: "update", description: "Update property fields"},
			{name: "ActivateProperty", entity: "Property", type: "transition", from_status: ["onboarding"], to_status: "active",
				description: "Activate a property after onboarding"},
			{name: "CreateUnit", entity: "Unit", type: "create", description: "Create a new unit within a property"},
			{name: "GetUnit", entity: "Unit", type: "get", description: "Get unit by ID"},
			{name: "ListUnits", entity: "Unit", type: "list", description: "List units with filtering"},
			{name: "UpdateUnit", entity: "Unit", type: "update", description: "Update unit fields"},
		]
	},
	{
		name: "LeaseService"
		base_path: "/v1/leases"
		entities: ["Lease", "Application"]
		operations: [
			{name: "CreateLease", entity: "Lease", type: "create", description: "Create a new lease draft"},
			{name: "GetLease", entity: "Lease", type: "get", description: "Get lease by ID"},
			{name: "ListLeases", entity: "Lease", type: "list", description: "List leases with filtering"},
			{name: "UpdateLease", entity: "Lease", type: "update", description: "Update lease fields (draft only)"},
			{name: "SubmitForApproval", entity: "Lease", type: "transition", from_status: ["draft"], to_status: "pending_approval",
				description: "Submit lease draft for approval"},
			{name: "ApproveLease", entity: "Lease", type: "transition", from_status: ["pending_approval"], to_status: "pending_signature",
				description: "Approve lease for signing"},
			{name: "ActivateLease", entity: "Lease", type: "transition", from_status: ["pending_signature"], to_status: "active",
				extra_fields: ["move_in_date", "confirmed_rent"],
				description: "Activate a signed lease. Side effects: Unit status -> occupied, security deposit charge created"},
			{name: "TerminateLease", entity: "Lease", type: "transition", from_status: ["active", "month_to_month_holdover"], to_status: "terminated",
				extra_fields: ["reason", "move_out_date"],
				description: "Terminate a lease early. Requires reason for audit trail"},
			{name: "RenewLease", entity: "Lease", type: "transition", from_status: ["expired", "month_to_month_holdover"], to_status: "renewed",
				description: "Renew a lease. Creates a new lease entity for the renewal term"},
			{name: "StartEviction", entity: "Lease", type: "transition", from_status: ["active", "month_to_month_holdover"], to_status: "eviction",
				extra_fields: ["reason"],
				description: "Begin eviction process"},
			{name: "CreateApplication", entity: "Application", type: "create", description: "Submit a new lease application"},
			{name: "GetApplication", entity: "Application", type: "get", description: "Get application by ID"},
			{name: "ListApplications", entity: "Application", type: "list", description: "List applications"},
			{name: "ApproveApplication", entity: "Application", type: "transition",
				from_status: ["under_review"], to_status: "approved",
				extra_fields: ["conditions"],
				description: "Approve a lease application"},
			{name: "DenyApplication", entity: "Application", type: "transition",
				from_status: ["under_review", "conditionally_approved"], to_status: "denied",
				extra_fields: ["reason"],
				description: "Deny a lease application. Reason required for fair housing compliance"},
		]
	},
	{
		name: "AccountingService"
		base_path: "/v1/accounting"
		entities: ["Account", "LedgerEntry", "JournalEntry", "BankAccount", "Reconciliation"]
		operations: [
			{name: "CreateAccount", entity: "Account", type: "create", description: "Create a new GL account"},
			{name: "GetAccount", entity: "Account", type: "get", description: "Get GL account by ID"},
			{name: "ListAccounts", entity: "Account", type: "list", description: "List GL accounts"},
			{name: "UpdateAccount", entity: "Account", type: "update", description: "Update GL account"},
			{name: "GetLedgerEntry", entity: "LedgerEntry", type: "get", description: "Get ledger entry by ID"},
			{name: "ListLedgerEntries", entity: "LedgerEntry", type: "list", description: "List ledger entries with filtering"},
			{name: "CreateJournalEntry", entity: "JournalEntry", type: "create", description: "Create a new journal entry"},
			{name: "GetJournalEntry", entity: "JournalEntry", type: "get", description: "Get journal entry by ID"},
			{name: "ListJournalEntries", entity: "JournalEntry", type: "list", description: "List journal entries"},
			{name: "PostJournalEntry", entity: "JournalEntry", type: "transition",
				from_status: ["draft", "pending_approval"], to_status: "posted",
				extra_fields: ["approval_notes"],
				description: "Post a journal entry. Lines must balance (debits = credits)"},
			{name: "VoidJournalEntry", entity: "JournalEntry", type: "transition",
				from_status: ["posted"], to_status: "voided",
				extra_fields: ["reason"],
				description: "Void a posted journal entry. Creates reversal entry"},
			{name: "CreateBankAccount", entity: "BankAccount", type: "create", description: "Create a bank account"},
			{name: "GetBankAccount", entity: "BankAccount", type: "get", description: "Get bank account by ID"},
			{name: "ListBankAccounts", entity: "BankAccount", type: "list", description: "List bank accounts"},
			{name: "UpdateBankAccount", entity: "BankAccount", type: "update", description: "Update bank account"},
			{name: "CreateReconciliation", entity: "Reconciliation", type: "create", description: "Start a bank reconciliation"},
			{name: "GetReconciliation", entity: "Reconciliation", type: "get", description: "Get reconciliation by ID"},
			{name: "ListReconciliations", entity: "Reconciliation", type: "list", description: "List reconciliations"},
			{name: "ApproveReconciliation", entity: "Reconciliation", type: "transition",
				from_status: ["balanced"], to_status: "approved",
				description: "Approve a balanced reconciliation"},
		]
	},
]
