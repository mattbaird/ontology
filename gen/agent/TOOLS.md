# Propeller API Tools

Available operations for the Propeller property management system.
Each tool corresponds to a Connect-RPC API method.

## PersonService

### `create_person`

Create a new person

- **Type:** create
- **Entity:** Person

### `get_person`

Get person by ID

- **Type:** get
- **Entity:** Person

### `list_persons`

List persons with filtering

- **Type:** list
- **Entity:** Person

### `update_person`

Update person fields

- **Type:** update
- **Entity:** Person

### `create_organization`

Create a new organization

- **Type:** create
- **Entity:** Organization

### `get_organization`

Get organization by ID

- **Type:** get
- **Entity:** Organization

### `list_organizations`

List organizations

- **Type:** list
- **Entity:** Organization

### `update_organization`

Update organization

- **Type:** update
- **Entity:** Organization

### `create_person_role`

Assign a role to a person

- **Type:** create
- **Entity:** PersonRole

### `get_person_role`

Get person role by ID

- **Type:** get
- **Entity:** PersonRole

### `list_person_roles`

List person roles

- **Type:** list
- **Entity:** PersonRole

### `activate_role`

Activate a pending person role

- **Type:** transition
- **Entity:** PersonRole

### `deactivate_role`

Deactivate an active person role

- **Type:** transition
- **Entity:** PersonRole

### `terminate_role`

Terminate a person role

- **Type:** transition
- **Entity:** PersonRole

## PropertyService

### `create_portfolio`

Create a new portfolio

- **Type:** create
- **Entity:** Portfolio

### `get_portfolio`

Get portfolio by ID

- **Type:** get
- **Entity:** Portfolio

### `list_portfolios`

List portfolios

- **Type:** list
- **Entity:** Portfolio

### `update_portfolio`

Update portfolio

- **Type:** update
- **Entity:** Portfolio

### `activate_portfolio`

Activate a portfolio after onboarding

- **Type:** transition
- **Entity:** Portfolio

### `create_property`

Create a new property

- **Type:** create
- **Entity:** Property

### `get_property`

Get property by ID

- **Type:** get
- **Entity:** Property

### `list_properties`

List properties with filtering

- **Type:** list
- **Entity:** Property

### `update_property`

Update property fields

- **Type:** update
- **Entity:** Property

### `activate_property`

Activate a property after onboarding

- **Type:** transition
- **Entity:** Property

### `create_unit`

Create a new unit within a property

- **Type:** create
- **Entity:** Unit

### `get_unit`

Get unit by ID

- **Type:** get
- **Entity:** Unit

### `list_units`

List units with filtering

- **Type:** list
- **Entity:** Unit

### `update_unit`

Update unit fields

- **Type:** update
- **Entity:** Unit

## LeaseService

### `create_lease`

Create a new lease draft

- **Type:** create
- **Entity:** Lease

### `get_lease`

Get lease by ID

- **Type:** get
- **Entity:** Lease

### `list_leases`

List leases with filtering

- **Type:** list
- **Entity:** Lease

### `update_lease`

Update lease fields (draft only)

- **Type:** update
- **Entity:** Lease

### `submit_for_approval`

Submit lease draft for approval

- **Type:** transition
- **Entity:** Lease

### `approve_lease`

Approve lease for signing

- **Type:** transition
- **Entity:** Lease

### `activate_lease`

Activate a signed lease. Side effects: Unit status -> occupied, security deposit charge created

- **Type:** transition
- **Entity:** Lease

### `terminate_lease`

Terminate a lease early. Requires reason for audit trail

- **Type:** transition
- **Entity:** Lease

### `renew_lease`

Renew a lease. Creates a new lease entity for the renewal term

- **Type:** transition
- **Entity:** Lease

### `start_eviction`

Begin eviction process

- **Type:** transition
- **Entity:** Lease

### `create_application`

Submit a new lease application

- **Type:** create
- **Entity:** Application

### `get_application`

Get application by ID

- **Type:** get
- **Entity:** Application

### `list_applications`

List applications

- **Type:** list
- **Entity:** Application

### `approve_application`

Approve a lease application

- **Type:** transition
- **Entity:** Application

### `deny_application`

Deny a lease application. Reason required for fair housing compliance

- **Type:** transition
- **Entity:** Application

## AccountingService

### `create_account`

Create a new GL account

- **Type:** create
- **Entity:** Account

### `get_account`

Get GL account by ID

- **Type:** get
- **Entity:** Account

### `list_accounts`

List GL accounts

- **Type:** list
- **Entity:** Account

### `update_account`

Update GL account

- **Type:** update
- **Entity:** Account

### `get_ledger_entry`

Get ledger entry by ID

- **Type:** get
- **Entity:** LedgerEntry

### `list_ledger_entries`

List ledger entries with filtering

- **Type:** list
- **Entity:** LedgerEntry

### `create_journal_entry`

Create a new journal entry

- **Type:** create
- **Entity:** JournalEntry

### `get_journal_entry`

Get journal entry by ID

- **Type:** get
- **Entity:** JournalEntry

### `list_journal_entries`

List journal entries

- **Type:** list
- **Entity:** JournalEntry

### `post_journal_entry`

Post a journal entry. Lines must balance (debits = credits)

- **Type:** transition
- **Entity:** JournalEntry

### `void_journal_entry`

Void a posted journal entry. Creates reversal entry

- **Type:** transition
- **Entity:** JournalEntry

### `create_bank_account`

Create a bank account

- **Type:** create
- **Entity:** BankAccount

### `get_bank_account`

Get bank account by ID

- **Type:** get
- **Entity:** BankAccount

### `list_bank_accounts`

List bank accounts

- **Type:** list
- **Entity:** BankAccount

### `update_bank_account`

Update bank account

- **Type:** update
- **Entity:** BankAccount

### `create_reconciliation`

Start a bank reconciliation

- **Type:** create
- **Entity:** Reconciliation

### `get_reconciliation`

Get reconciliation by ID

- **Type:** get
- **Entity:** Reconciliation

### `list_reconciliations`

List reconciliations

- **Type:** list
- **Entity:** Reconciliation

### `approve_reconciliation`

Approve a balanced reconciliation

- **Type:** transition
- **Entity:** Reconciliation

