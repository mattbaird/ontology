# Propeller Domain Ontology

This document describes the complete domain model for the Propeller property management system.
It is auto-generated from the CUE ontology and serves as the agent's world model.

## Entity Types

### Account

Fields:
- `id`
- `account_number`
- `name`
- `description (optional)`
- `account_type`
- `account_subtype`
- `parent_account_id (optional)`
- `depth`
- `dimensions (optional)`
- `normal_balance`
- `is_header`
- `is_system`
- `allows_direct_posting`
- `status`
- `is_trust_account`
- `trust_type (optional)`
- `budget_amount (optional)`
- `tax_line (optional)`

### Application

Fields:
- `id`
- `property_id`
- `unit_id (optional)`
- `applicant_person_id`
- `status`
- `desired_move_in`
- `desired_lease_term_months`
- `screening_request_id (optional)`
- `screening_completed (optional)`
- `credit_score (optional)`
- `background_clear`
- `income_verified`
- `income_to_rent_ratio (optional)`
- `decision_by (optional)`
- `decision_at (optional)`
- `decision_reason (optional)`
- `conditions (optional)`
- `application_fee`
- `fee_paid`

### BankAccount

Fields:
- `id`
- `name`
- `account_type`
- `gl_account_id`
- `bank_name`
- `routing_number (optional)`
- `account_number_last_four`
- `portfolio_id (optional)`
- `property_id (optional)`
- `entity_id (optional)`
- `status`
- `current_balance (optional)`
- `last_reconciled_at (optional)`
- `is_trust`
- `trust_state (optional)`
- `commingling_allowed`

### JournalEntry

Fields:
- `id`
- `entry_date`
- `posted_date`
- `description`
- `source_type`
- `source_id (optional)`
- `status`
- `approved_by (optional)`
- `approved_at (optional)`
- `batch_id (optional)`
- `entity_id (optional)`
- `property_id (optional)`
- `reverses_journal_id (optional)`
- `reversed_by_journal_id (optional)`
- `lines`

### Lease

Fields:
- `id`
- `unit_id`
- `property_id`
- `tenant_role_ids`
- `guarantor_role_ids (optional)`
- `lease_type`
- `status`
- `term`
- `base_rent`
- `security_deposit`
- `rent_schedule (optional)`
- `recurring_charges (optional)`
- `late_fee_policy (optional)`
- `cam_terms (optional)`
- `tenant_improvement (optional)`
- `renewal_options (optional)`
- `subsidy (optional)`
- `move_in_date (optional)`
- `move_out_date (optional)`
- `notice_date (optional)`
- `notice_required_days`
- `signing_method (optional)`
- `signed_at (optional)`
- `document_id (optional)`

### LedgerEntry

Fields:
- `id`
- `account_id`
- `entry_type`
- `amount`
- `journal_entry_id`
- `effective_date`
- `posted_date`
- `description`
- `charge_code`
- `memo (optional)`
- `property_id`
- `unit_id (optional)`
- `lease_id (optional)`
- `person_id (optional)`
- `bank_account_id (optional)`
- `bank_transaction_id (optional)`
- `reconciled`
- `reconciliation_id (optional)`
- `reconciled_at (optional)`
- `adjusts_entry_id (optional)`

### Organization

Fields:
- `id`
- `legal_name`
- `dba_name (optional)`
- `org_type`
- `tax_id (optional)`
- `tax_id_type (optional)`
- `status`
- `address (optional)`
- `contact_methods (optional)`
- `state_of_incorporation (optional)`
- `formation_date (optional)`
- `management_license (optional)`
- `license_state (optional)`
- `license_expiry (optional)`

### Person

Fields:
- `id`
- `first_name`
- `last_name`
- `display_name`
- `date_of_birth (optional)`
- `ssn_last_four (optional)`
- `contact_methods`
- `preferred_contact`
- `language_preference`
- `timezone (optional)`
- `do_not_contact`
- `identity_verified`
- `verification_method (optional)`
- `verified_at (optional)`
- `tags (optional)`

### PersonRole

Fields:
- `id`
- `person_id`
- `role_type`
- `scope_type`
- `scope_id`
- `status`
- `effective`
- `attributes (optional)`

### Portfolio

Fields:
- `id`
- `name`
- `owner_id`
- `management_type`
- `requires_trust_accounting`
- `trust_bank_account_id (optional)`
- `status`
- `default_late_fee_policy (optional)`
- `default_payment_methods (optional)`
- `fiscal_year_start_month`

### Property

Fields:
- `id`
- `portfolio_id`
- `name`
- `address`
- `property_type`
- `status`
- `year_built`
- `total_square_footage`
- `total_units`
- `lot_size_sqft (optional)`
- `stories (optional)`
- `parking_spaces (optional)`
- `jurisdiction_id (optional)`
- `rent_controlled`
- `compliance_programs (optional)`
- `requires_lead_disclosure`
- `chart_of_accounts_id (optional)`
- `bank_account_id (optional)`
- `insurance_policy_number (optional)`
- `insurance_expiry (optional)`

### Reconciliation

Fields:
- `id`
- `bank_account_id`
- `period_start`
- `period_end`
- `statement_balance`
- `system_balance`
- `difference`
- `status`
- `matched_transaction_count`
- `unmatched_transaction_count`
- `completed_by (optional)`
- `completed_at (optional)`
- `approved_by (optional)`
- `approved_at (optional)`

### Unit

Fields:
- `id`
- `property_id`
- `unit_number`
- `unit_type`
- `status`
- `square_footage`
- `bedrooms (optional)`
- `bathrooms (optional)`
- `floor (optional)`
- `amenities (optional)`
- `floor_plan (optional)`
- `ada_accessible`
- `pet_friendly`
- `furnished`
- `market_rent (optional)`
- `active_lease_id (optional)`
- `ami_restriction (optional)`

## Relationships

- **Portfolio → Property** (O2M): Portfolio contains Properties
- **Portfolio → Organization** (M2O): Portfolio is owned by Organization
- **Portfolio → BankAccount** (O2O): Portfolio uses BankAccount for trust funds
- **Property → Unit** (O2M): Property contains Units
- **Property → BankAccount** (M2O): Property uses BankAccount
- **Unit → Lease** (O2M): Unit has Leases over time
- **Unit → Lease** (O2O): Unit has at most one active Lease
- **Lease → PersonRole** (M2M): Lease is held by tenant PersonRoles
- **Lease → PersonRole** (M2M): Lease is guaranteed by guarantor PersonRoles
- **Lease → LedgerEntry** (O2M): Lease generates LedgerEntries
- **Lease → Application** (O2O): Lease originated from Application
- **Person → PersonRole** (O2M): Person has Roles in various contexts
- **Person → Organization** (M2M): Person is affiliated with Organizations
- **Organization → Organization** (O2M): Organization has subsidiary Organizations
- **Account → Account** (O2M): Account has sub-Accounts
- **LedgerEntry → JournalEntry** (M2O): LedgerEntry belongs to JournalEntry
- **LedgerEntry → Account** (M2O): LedgerEntry posts to Account
- **LedgerEntry → Property** (M2O): LedgerEntry relates to Property
- **LedgerEntry → Person** (M2O): LedgerEntry relates to Person
- **BankAccount → Account** (M2O): BankAccount is tracked via GL Account
- **Reconciliation → BankAccount** (M2O): Reconciliation is for BankAccount
- **Application → Person** (M2O): Application was submitted by Person
- **Application → Property** (M2O): Application is for Property
- **Application → Unit** (M2O): Application is for specific Unit

