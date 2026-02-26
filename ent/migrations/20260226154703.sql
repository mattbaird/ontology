-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
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
  `requires_trust_accounting` bool NOT NULL,
  `trust_bank_account_id` text NULL,
  `status` text NOT NULL,
  `default_payment_methods` json NULL,
  `fiscal_year_start_month` integer NOT NULL,
  `organization_owned_portfolios` uuid NULL,
  `portfolio_owner` uuid NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `portfolios_organizations_owner` FOREIGN KEY (`portfolio_owner`) REFERENCES `organizations` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `portfolios_organizations_owned_portfolios` FOREIGN KEY (`organization_owned_portfolios`) REFERENCES `organizations` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Copy rows from old table "portfolios" to new temporary table "new_portfolios"
INSERT INTO `new_portfolios` (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `management_type`, `requires_trust_accounting`, `trust_bank_account_id`, `status`, `default_payment_methods`, `fiscal_year_start_month`, `organization_owned_portfolios`, `portfolio_owner`) SELECT `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `management_type`, `requires_trust_accounting`, `trust_bank_account_id`, `status`, `default_payment_methods`, `fiscal_year_start_month`, `organization_owned_portfolios`, `portfolio_owner` FROM `portfolios`;
-- Drop "portfolios" table after copying rows
DROP TABLE `portfolios`;
-- Rename temporary table "new_portfolios" to "portfolios"
ALTER TABLE `new_portfolios` RENAME TO `portfolios`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
