// policies/command_permissions.cue
// Maps commands to required permission keys.
// This is the authoritative source for which permission is needed to execute each command.
package policies

#CommandPermission: close({
	command:     string
	permission:  string
	description: string
})

command_permissions: [...#CommandPermission]
command_permissions: [
	// Lease commands
	{command: "MoveInTenant", permission: "lease:move_in", description: "Execute tenant move-in process"},
	{command: "RecordPayment", permission: "payment:record", description: "Record a rent payment"},
	{command: "RenewLease", permission: "lease:renew", description: "Renew an existing lease"},
	{command: "InitiateEviction", permission: "lease:eviction", description: "Begin eviction proceedings"},

	// Property commands
	{command: "OnboardProperty", permission: "property:create", description: "Onboard a new property"},

	// Accounting commands
	{command: "PostJournalEntry", permission: "journal_entry:create", description: "Create and post a journal entry"},
	{command: "Reconcile", permission: "reconciliation:start", description: "Start bank reconciliation"},

	// Application commands
	{command: "SubmitApplication", permission: "application:process", description: "Submit a rental application"},
	{command: "ApproveApplication", permission: "application:approve", description: "Approve a rental application"},
	{command: "DenyApplication", permission: "application:deny", description: "Deny a rental application"},
]
