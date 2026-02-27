-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_bank_accounts" table
CREATE TABLE `new_bank_accounts` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `name` text NOT NULL,
  `account_type` text NOT NULL,
  `institution_name` text NOT NULL,
  `routing_number` text NOT NULL,
  `account_mask` text NOT NULL,
  `account_number_encrypted` text NULL,
  `plaid_account_id` text NULL,
  `plaid_access_token` text NULL,
  `property_id` text NULL,
  `entity_id` text NULL,
  `status` text NOT NULL,
  `is_default` bool NOT NULL DEFAULT false,
  `accepts_deposits` bool NOT NULL DEFAULT true,
  `accepts_payments` bool NOT NULL DEFAULT true,
  `current_balance_amount_cents` integer NULL,
  `current_balance_currency` text NULL DEFAULT 'USD',
  `last_statement_date` datetime NULL,
  `account_bank_accounts` uuid NULL,
  `bank_account_gl_account` uuid NOT NULL,
  `portfolio_trust_account` uuid NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `bank_accounts_portfolios_trust_account` FOREIGN KEY (`portfolio_trust_account`) REFERENCES `portfolios` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `bank_accounts_accounts_gl_account` FOREIGN KEY (`bank_account_gl_account`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `bank_accounts_accounts_bank_accounts` FOREIGN KEY (`account_bank_accounts`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Copy rows from old table "bank_accounts" to new temporary table "new_bank_accounts"
INSERT INTO `new_bank_accounts` (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `account_type`, `routing_number`, `property_id`, `entity_id`, `status`, `current_balance_amount_cents`, `current_balance_currency`, `account_bank_accounts`, `bank_account_gl_account`, `portfolio_trust_account`) SELECT `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `account_type`, `routing_number`, `property_id`, `entity_id`, `status`, `current_balance_amount_cents`, `current_balance_currency`, `account_bank_accounts`, `bank_account_gl_account`, `portfolio_trust_account` FROM `bank_accounts`;
-- Drop "bank_accounts" table after copying rows
DROP TABLE `bank_accounts`;
-- Rename temporary table "new_bank_accounts" to "bank_accounts"
ALTER TABLE `new_bank_accounts` RENAME TO `bank_accounts`;
-- Create index "bank_accounts_portfolio_trust_account_key" to table: "bank_accounts"
CREATE UNIQUE INDEX `bank_accounts_portfolio_trust_account_key` ON `bank_accounts` (`portfolio_trust_account`);
-- Add column "middle_name" to table: "persons"
ALTER TABLE `persons` ADD COLUMN `middle_name` text NULL;
-- Add column "record_source" to table: "persons"
ALTER TABLE `persons` ADD COLUMN `record_source` text NOT NULL DEFAULT 'user';
-- Create "new_reconciliations" table
CREATE TABLE `new_reconciliations` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `period_start` datetime NOT NULL,
  `period_end` datetime NOT NULL,
  `statement_date` datetime NOT NULL,
  `statement_balance_amount_cents` integer NOT NULL,
  `statement_balance_currency` text NOT NULL DEFAULT 'USD',
  `gl_balance_amount_cents` integer NOT NULL,
  `gl_balance_currency` text NOT NULL DEFAULT 'USD',
  `difference_amount_cents` integer NULL,
  `difference_currency` text NULL DEFAULT 'USD',
  `status` text NOT NULL,
  `unreconciled_items` integer NULL,
  `reconciled_by` text NULL,
  `reconciled_at` datetime NULL,
  `approved_by` text NULL,
  `approved_at` datetime NULL,
  `bank_account_reconciliations` uuid NULL,
  `reconciliation_bank_account` uuid NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `reconciliations_bank_accounts_bank_account` FOREIGN KEY (`reconciliation_bank_account`) REFERENCES `bank_accounts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `reconciliations_bank_accounts_reconciliations` FOREIGN KEY (`bank_account_reconciliations`) REFERENCES `bank_accounts` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Copy rows from old table "reconciliations" to new temporary table "new_reconciliations"
INSERT INTO `new_reconciliations` (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `period_start`, `period_end`, `statement_balance_amount_cents`, `statement_balance_currency`, `difference_amount_cents`, `difference_currency`, `status`, `approved_by`, `approved_at`, `bank_account_reconciliations`, `reconciliation_bank_account`) SELECT `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `period_start`, `period_end`, `statement_balance_amount_cents`, `statement_balance_currency`, `difference_amount_cents`, `difference_currency`, `status`, `approved_by`, `approved_at`, `bank_account_reconciliations`, `reconciliation_bank_account` FROM `reconciliations`;
-- Drop "reconciliations" table after copying rows
DROP TABLE `reconciliations`;
-- Rename temporary table "new_reconciliations" to "reconciliations"
ALTER TABLE `new_reconciliations` RENAME TO `reconciliations`;
-- Create "new_portfolios" table
CREATE TABLE `new_portfolios` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `name` text NOT NULL,
  `management_type` text NOT NULL,
  `description` text NULL,
  `status` text NOT NULL,
  `default_chart_of_accounts_id` text NULL,
  `default_bank_account_id` text NULL,
  `organization_owned_portfolios` uuid NULL,
  `portfolio_owner` uuid NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `portfolios_organizations_owner` FOREIGN KEY (`portfolio_owner`) REFERENCES `organizations` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `portfolios_organizations_owned_portfolios` FOREIGN KEY (`organization_owned_portfolios`) REFERENCES `organizations` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Copy rows from old table "portfolios" to new temporary table "new_portfolios"
INSERT INTO `new_portfolios` (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `management_type`, `status`, `organization_owned_portfolios`, `portfolio_owner`) SELECT `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `management_type`, `status`, `organization_owned_portfolios`, `portfolio_owner` FROM `portfolios`;
-- Drop "portfolios" table after copying rows
DROP TABLE `portfolios`;
-- Rename temporary table "new_portfolios" to "portfolios"
ALTER TABLE `new_portfolios` RENAME TO `portfolios`;
-- Create "jurisdictions" table
CREATE TABLE `jurisdictions` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `name` text NOT NULL,
  `jurisdiction_type` text NOT NULL,
  `fips_code` text NULL,
  `state_code` text NULL,
  `country_code` text NOT NULL,
  `status` text NOT NULL,
  `successor_jurisdiction_id` text NULL,
  `effective_date` datetime NULL,
  `dissolution_date` datetime NULL,
  `governing_body` text NULL,
  `regulatory_url` text NULL,
  `jurisdiction_children` uuid NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `jurisdictions_jurisdictions_children` FOREIGN KEY (`jurisdiction_children`) REFERENCES `jurisdictions` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Create "jurisdiction_rules" table
CREATE TABLE `jurisdiction_rules` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `rule_type` text NOT NULL,
  `status` text NOT NULL,
  `applies_to_lease_types` json NULL,
  `applies_to_property_types` json NULL,
  `applies_to_space_types` json NULL,
  `exemptions` json NULL,
  `rule_definition` json NOT NULL,
  `statute_reference` text NULL,
  `ordinance_number` text NULL,
  `statute_url` text NULL,
  `effective_date` datetime NOT NULL,
  `expiration_date` datetime NULL,
  `last_verified` datetime NULL,
  `verified_by` text NULL,
  `verification_source` text NULL,
  `jurisdiction_rules` uuid NOT NULL,
  `jurisdiction_rule_superseded_by` uuid NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `jurisdiction_rules_jurisdiction_rules_superseded_by` FOREIGN KEY (`jurisdiction_rule_superseded_by`) REFERENCES `jurisdiction_rules` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `jurisdiction_rules_jurisdictions_rules` FOREIGN KEY (`jurisdiction_rules`) REFERENCES `jurisdictions` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "jurisdiction_rules_jurisdiction_rule_superseded_by_key" to table: "jurisdiction_rules"
CREATE UNIQUE INDEX `jurisdiction_rules_jurisdiction_rule_superseded_by_key` ON `jurisdiction_rules` (`jurisdiction_rule_superseded_by`);
-- Create "property_jurisdictions" table
CREATE TABLE `property_jurisdictions` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `effective_date` datetime NOT NULL,
  `end_date` datetime NULL,
  `lookup_source` text NOT NULL,
  `verified` bool NOT NULL DEFAULT false,
  `verified_at` datetime NULL,
  `verified_by` text NULL,
  `jurisdiction_property_jurisdictions` uuid NULL,
  `property_property_jurisdictions` uuid NULL,
  `property_jurisdiction_property` uuid NOT NULL,
  `property_jurisdiction_jurisdiction` uuid NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `property_jurisdictions_jurisdictions_jurisdiction` FOREIGN KEY (`property_jurisdiction_jurisdiction`) REFERENCES `jurisdictions` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `property_jurisdictions_properties_property` FOREIGN KEY (`property_jurisdiction_property`) REFERENCES `properties` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `property_jurisdictions_properties_property_jurisdictions` FOREIGN KEY (`property_property_jurisdictions`) REFERENCES `properties` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `property_jurisdictions_jurisdictions_property_jurisdictions` FOREIGN KEY (`jurisdiction_property_jurisdictions`) REFERENCES `jurisdictions` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
