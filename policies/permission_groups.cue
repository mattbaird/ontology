// policies/permission_groups.cue
// Business-defined permission groups. These are NOT generated from entity schemas.
// The ontology tells you what entities exist. Business policy decides who can do what.
package policies

#PermissionGroup: close({
	name:        string
	description: string
	inherits?:   [...string] // inherit from other groups
	commands:    [...string] // which commands this group can execute
	queries:     [...string] // which query endpoints this group can access
})

permission_groups: {
	organization_admin: #PermissionGroup & {
		name:        "Organization Admin"
		description: "Full access to all operations within the organization"
		commands:    ["*"]
		queries:     ["*"]
	}

	portfolio_admin: #PermissionGroup & {
		name:        "Portfolio Admin"
		description: "Full access within assigned portfolios"
		inherits:    ["property_manager"]
		commands: [
			"property:create", "property:transfer",
			"account:create", "account:modify",
			"bank_account:create",
			"journal_entry:approve",
			"reconciliation:approve",
		]
		queries: ["portfolio:*", "property:*", "accounting:*"]
	}

	property_manager: #PermissionGroup & {
		name:        "Property Manager"
		description: "Day-to-day property operations"
		inherits:    ["leasing_agent", "maintenance_coordinator"]
		commands: [
			"lease:move_in", "lease:move_out", "lease:renew",
			"lease:eviction",
			"payment:record", "payment:reverse",
			"charge:create", "charge:waive",
			"journal_entry:create",
		]
		queries: ["property:detail", "lease:*", "person:*", "accounting:property_level"]
	}

	leasing_agent: #PermissionGroup & {
		name:        "Leasing Agent"
		description: "Leasing operations only"
		commands: [
			"application:process", "application:approve", "application:deny",
			"lease:create", "lease:submit_for_approval",
			"lease:send_for_signature",
		]
		queries: ["property:list", "space:list", "application:*", "lease:read"]
	}

	maintenance_coordinator: #PermissionGroup & {
		name:        "Maintenance Coordinator"
		description: "Work order and maintenance operations"
		commands: [
			"work_order:create", "work_order:assign", "work_order:complete",
			"inspection:schedule", "inspection:record",
		]
		queries: ["property:detail", "space:detail", "work_order:*"]
	}

	accountant: #PermissionGroup & {
		name:        "Accountant"
		description: "Financial operations"
		commands: [
			"payment:record", "charge:create",
			"journal_entry:create", "journal_entry:post",
			"reconciliation:start", "reconciliation:complete",
			"owner_distribution:calculate", "owner_distribution:process",
		]
		queries: ["accounting:*", "lease:financial", "property:financial"]
	}

	viewer: #PermissionGroup & {
		name:        "Viewer"
		description: "Read-only access"
		commands:    []
		queries:     ["property:list", "lease:list", "person:list"]
	}
}
