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

### `create_building`

Create a new building

- **Type:** create
- **Entity:** Building

### `get_building`

Get building by ID

- **Type:** get
- **Entity:** Building

### `list_buildings`

List buildings

- **Type:** list
- **Entity:** Building

### `update_building`

Update building

- **Type:** update
- **Entity:** Building

### `deactivate_building`

Deactivate a building

- **Type:** transition
- **Entity:** Building

### `start_building_renovation`

Start building renovation

- **Type:** transition
- **Entity:** Building

### `activate_building`

Activate a building

- **Type:** transition
- **Entity:** Building

### `create_space`

Create a new space within a property

- **Type:** create
- **Entity:** Space

### `get_space`

Get space by ID

- **Type:** get
- **Entity:** Space

### `list_spaces`

List spaces with filtering

- **Type:** list
- **Entity:** Space

### `update_space`

Update space fields

- **Type:** update
- **Entity:** Space

### `occupy_space`

Mark a space as occupied

- **Type:** transition
- **Entity:** Space

### `record_space_notice`

Record notice to vacate

- **Type:** transition
- **Entity:** Space

### `rescind_space_notice`

Rescind a notice to vacate

- **Type:** transition
- **Entity:** Space

### `start_make_ready`

Start make-ready process

- **Type:** transition
- **Entity:** Space

### `mark_space_vacant`

Mark a space as vacant

- **Type:** transition
- **Entity:** Space

### `mark_space_down`

Mark a space as down (out of service)

- **Type:** transition
- **Entity:** Space

### `mark_space_model`

Mark a space as a model unit

- **Type:** transition
- **Entity:** Space

### `reserve_space`

Reserve a space

- **Type:** transition
- **Entity:** Space

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

### `send_for_signature`

Send lease for electronic signature

- **Type:** transition
- **Entity:** Lease

### `activate_lease`

Activate a signed lease

- **Type:** transition
- **Entity:** Lease

### `terminate_lease`

Terminate a lease early. Requires reason for audit trail

- **Type:** transition
- **Entity:** Lease

### `renew_lease`

Renew a lease

- **Type:** transition
- **Entity:** Lease

### `initiate_eviction`

Begin eviction process

- **Type:** transition
- **Entity:** Lease

### `record_notice`

Record tenant notice date

- **Type:** transition
- **Entity:** Lease

### `search_leases`

Search leases with advanced filters

- **Type:** create
- **Entity:** Lease

### `get_lease_ledger`

Get ledger entries for a lease

- **Type:** get
- **Entity:** Lease

### `record_payment`

Record a payment on a lease

- **Type:** create
- **Entity:** Lease

### `post_charge`

Post a charge to a lease

- **Type:** create
- **Entity:** Lease

### `apply_credit`

Apply a credit to a lease

- **Type:** create
- **Entity:** Lease

### `create_lease_space`

Create a lease-space association

- **Type:** create
- **Entity:** LeaseSpace

### `get_lease_space`

Get lease-space by ID

- **Type:** get
- **Entity:** LeaseSpace

### `list_lease_spaces`

List lease-space associations

- **Type:** list
- **Entity:** LeaseSpace

### `update_lease_space`

Update lease-space association

- **Type:** update
- **Entity:** LeaseSpace

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

Deny a lease application

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

## ActivityService

### `get_entity_activity`

Get chronological activity feed for any entity. Use as FIRST call when assessing an entity.

- **Type:** get
- **Entity:** ActivityEntry

### `get_signal_summary`

Get pre-aggregated signal summary for an entity with category breakdowns, sentiment, and escalations.

- **Type:** get
- **Entity:** ActivityEntry

### `get_portfolio_signals`

Batch screen multiple entities ranked by concern level. Use for portfolio-wide risk assessment.

- **Type:** create
- **Entity:** ActivityEntry

### `search_activity`

Full-text search across all activity streams by keyword.

- **Type:** create
- **Entity:** ActivityEntry

