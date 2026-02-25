// ontology/relationships.cue
package propeller

// This file defines the EDGES between domain models.
// These relationships drive:
//   - Ent edge generation (foreign keys + graph traversal)
//   - Permission model (access paths through the relationship graph)
//   - Agent reasoning (understanding how entities connect)
//   - Event routing (which subscribers care about this entity?)

#OntologyRelationship: {
	from:         string // Source entity type
	to:           string // Target entity type
	edge_name:    string // Name of the edge (lowercase)
	cardinality:  "O2O" | "O2M" | "M2O" | "M2M"
	required:     bool | *false
	semantic:     string // Human-readable relationship meaning
	inverse_name: string // Edge name on the target side
}

relationships: [...#OntologyRelationship]
relationships: [
	// Portfolio relationships
	{from: "Portfolio", to: "Property", edge_name: "properties", cardinality: "O2M",
		semantic: "Portfolio contains Properties", inverse_name: "portfolio"},
	{from: "Portfolio", to: "Organization", edge_name: "owner", cardinality: "M2O", required: true,
		semantic: "Portfolio is owned by Organization", inverse_name: "owned_portfolios"},
	{from: "Portfolio", to: "BankAccount", edge_name: "trust_account", cardinality: "O2O",
		semantic: "Portfolio uses BankAccount for trust funds", inverse_name: "trust_portfolio"},

	// Property relationships
	{from: "Property", to: "Unit", edge_name: "units", cardinality: "O2M",
		semantic: "Property contains Units", inverse_name: "property"},
	{from: "Property", to: "BankAccount", edge_name: "bank_account", cardinality: "M2O",
		semantic: "Property uses BankAccount", inverse_name: "properties"},

	// Unit relationships
	{from: "Unit", to: "Lease", edge_name: "leases", cardinality: "O2M",
		semantic: "Unit has Leases over time", inverse_name: "unit"},
	{from: "Unit", to: "Lease", edge_name: "active_lease", cardinality: "O2O",
		semantic: "Unit has at most one active Lease", inverse_name: "occupied_unit"},

	// Lease relationships
	{from: "Lease", to: "PersonRole", edge_name: "tenant_roles", cardinality: "M2M",
		semantic: "Lease is held by tenant PersonRoles", inverse_name: "leases"},
	{from: "Lease", to: "PersonRole", edge_name: "guarantor_roles", cardinality: "M2M",
		semantic: "Lease is guaranteed by guarantor PersonRoles", inverse_name: "guaranteed_leases"},
	{from: "Lease", to: "LedgerEntry", edge_name: "ledger_entries", cardinality: "O2M",
		semantic: "Lease generates LedgerEntries", inverse_name: "lease"},
	{from: "Lease", to: "Application", edge_name: "application", cardinality: "O2O",
		semantic: "Lease originated from Application", inverse_name: "resulting_lease"},

	// Person relationships
	{from: "Person", to: "PersonRole", edge_name: "roles", cardinality: "O2M",
		semantic: "Person has Roles in various contexts", inverse_name: "person"},
	{from: "Person", to: "Organization", edge_name: "organizations", cardinality: "M2M",
		semantic: "Person is affiliated with Organizations", inverse_name: "people"},

	// Organization relationships
	{from: "Organization", to: "Organization", edge_name: "subsidiaries", cardinality: "O2M",
		semantic: "Organization has subsidiary Organizations", inverse_name: "parent_org"},

	// Accounting relationships
	{from: "Account", to: "Account", edge_name: "children", cardinality: "O2M",
		semantic: "Account has sub-Accounts", inverse_name: "parent"},
	{from: "LedgerEntry", to: "JournalEntry", edge_name: "journal_entry", cardinality: "M2O", required: true,
		semantic: "LedgerEntry belongs to JournalEntry", inverse_name: "ledger_entries"},
	{from: "LedgerEntry", to: "Account", edge_name: "account", cardinality: "M2O", required: true,
		semantic: "LedgerEntry posts to Account", inverse_name: "entries"},
	{from: "LedgerEntry", to: "Property", edge_name: "property", cardinality: "M2O", required: true,
		semantic: "LedgerEntry relates to Property", inverse_name: "property_ledger_entries"},
	{from: "LedgerEntry", to: "Person", edge_name: "person", cardinality: "M2O",
		semantic: "LedgerEntry relates to Person", inverse_name: "person_ledger_entries"},
	{from: "BankAccount", to: "Account", edge_name: "gl_account", cardinality: "M2O", required: true,
		semantic: "BankAccount is tracked via GL Account", inverse_name: "bank_accounts"},
	{from: "Reconciliation", to: "BankAccount", edge_name: "bank_account", cardinality: "M2O", required: true,
		semantic: "Reconciliation is for BankAccount", inverse_name: "reconciliations"},

	// Application relationships
	{from: "Application", to: "Person", edge_name: "applicant", cardinality: "M2O", required: true,
		semantic: "Application was submitted by Person", inverse_name: "applications"},
	{from: "Application", to: "Property", edge_name: "property", cardinality: "M2O", required: true,
		semantic: "Application is for Property", inverse_name: "applications"},
	{from: "Application", to: "Unit", edge_name: "unit", cardinality: "M2O",
		semantic: "Application is for specific Unit", inverse_name: "applications"},
]
