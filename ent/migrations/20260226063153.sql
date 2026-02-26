-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_applications" table
CREATE TABLE `new_applications` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `status` text NOT NULL,
  `desired_move_in` datetime NOT NULL,
  `desired_lease_term_months` integer NOT NULL,
  `screening_request_id` text NULL,
  `screening_completed` datetime NULL,
  `credit_score` integer NULL,
  `background_clear` bool NOT NULL DEFAULT false,
  `income_verified` bool NOT NULL DEFAULT false,
  `income_to_rent_ratio` real NULL,
  `decision_by` text NULL,
  `decision_at` datetime NULL,
  `decision_reason` text NULL,
  `conditions` json NULL,
  `application_fee_amount_cents` integer NOT NULL,
  `application_fee_currency` text NOT NULL DEFAULT 'USD',
  `fee_paid` bool NOT NULL DEFAULT false,
  `applicant_person_id` uuid NOT NULL,
  `lease_application` uuid NULL,
  `person_applications` uuid NULL,
  `property_applications` uuid NOT NULL,
  `space_applications` uuid NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `applications_spaces_applications` FOREIGN KEY (`space_applications`) REFERENCES `spaces` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `applications_properties_applications` FOREIGN KEY (`property_applications`) REFERENCES `properties` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `applications_persons_applications` FOREIGN KEY (`person_applications`) REFERENCES `persons` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `applications_leases_application` FOREIGN KEY (`lease_application`) REFERENCES `leases` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `applications_persons_applicant` FOREIGN KEY (`applicant_person_id`) REFERENCES `persons` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "applications" to new temporary table "new_applications"
INSERT INTO `new_applications` (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `status`, `desired_move_in`, `desired_lease_term_months`, `screening_request_id`, `screening_completed`, `credit_score`, `background_clear`, `income_verified`, `income_to_rent_ratio`, `decision_by`, `decision_at`, `decision_reason`, `conditions`, `application_fee_amount_cents`, `application_fee_currency`, `fee_paid`, `applicant_person_id`, `lease_application`, `person_applications`, `property_applications`, `space_applications`) SELECT `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `status`, `desired_move_in`, `desired_lease_term_months`, `screening_request_id`, `screening_completed`, `credit_score`, `background_clear`, `income_verified`, `income_to_rent_ratio`, `decision_by`, `decision_at`, `decision_reason`, `conditions`, `application_fee_amount_cents`, `application_fee_currency`, `fee_paid`, `applicant_person_id`, `lease_application`, `person_applications`, `property_applications`, `space_applications` FROM `applications`;
-- Drop "applications" table after copying rows
DROP TABLE `applications`;
-- Rename temporary table "new_applications" to "applications"
ALTER TABLE `new_applications` RENAME TO `applications`;
-- Create index "applications_lease_application_key" to table: "applications"
CREATE UNIQUE INDEX `applications_lease_application_key` ON `applications` (`lease_application`);
-- Create "new_buildings" table
CREATE TABLE `new_buildings` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `name` text NOT NULL,
  `building_type` text NOT NULL,
  `address` json NULL,
  `description` text NULL,
  `status` text NOT NULL,
  `floors` integer NULL,
  `year_built` integer NULL,
  `total_square_footage` real NULL,
  `total_rentable_square_footage` real NULL,
  `property_buildings` uuid NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `buildings_properties_buildings` FOREIGN KEY (`property_buildings`) REFERENCES `properties` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "buildings" to new temporary table "new_buildings"
INSERT INTO `new_buildings` (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `building_type`, `address`, `description`, `status`, `floors`, `year_built`, `total_square_footage`, `total_rentable_square_footage`, `property_buildings`) SELECT `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `building_type`, `address`, `description`, `status`, `floors`, `year_built`, `total_square_footage`, `total_rentable_square_footage`, `property_buildings` FROM `buildings`;
-- Drop "buildings" table after copying rows
DROP TABLE `buildings`;
-- Rename temporary table "new_buildings" to "buildings"
ALTER TABLE `new_buildings` RENAME TO `buildings`;
-- Create "new_person_roles" table
CREATE TABLE `new_person_roles` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `role_type` text NOT NULL,
  `scope_type` text NOT NULL,
  `scope_id` text NOT NULL,
  `status` text NOT NULL,
  `effective` json NOT NULL,
  `attributes` json NULL,
  `person_roles` uuid NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `person_roles_persons_roles` FOREIGN KEY (`person_roles`) REFERENCES `persons` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "person_roles" to new temporary table "new_person_roles"
INSERT INTO `new_person_roles` (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `role_type`, `scope_type`, `scope_id`, `status`, `effective`, `attributes`, `person_roles`) SELECT `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `role_type`, `scope_type`, `scope_id`, `status`, `effective`, `attributes`, `person_roles` FROM `person_roles`;
-- Drop "person_roles" table after copying rows
DROP TABLE `person_roles`;
-- Rename temporary table "new_person_roles" to "person_roles"
ALTER TABLE `new_person_roles` RENAME TO `person_roles`;
-- Create "new_properties" table
CREATE TABLE `new_properties` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `name` text NOT NULL,
  `address` json NOT NULL,
  `property_type` text NOT NULL,
  `status` text NOT NULL,
  `year_built` integer NOT NULL,
  `total_square_footage` real NOT NULL,
  `total_spaces` integer NOT NULL,
  `lot_size_sqft` real NULL,
  `stories` integer NULL,
  `parking_spaces` integer NULL,
  `jurisdiction_id` text NULL,
  `rent_controlled` bool NOT NULL DEFAULT false,
  `compliance_programs` json NULL,
  `requires_lead_disclosure` bool NOT NULL,
  `chart_of_accounts_id` text NULL,
  `insurance_policy_number` text NULL,
  `insurance_expiry` datetime NULL,
  `bank_account_properties` uuid NULL,
  `portfolio_properties` uuid NOT NULL,
  `property_bank_account` uuid NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `properties_bank_accounts_bank_account` FOREIGN KEY (`property_bank_account`) REFERENCES `bank_accounts` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `properties_portfolios_properties` FOREIGN KEY (`portfolio_properties`) REFERENCES `portfolios` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `properties_bank_accounts_properties` FOREIGN KEY (`bank_account_properties`) REFERENCES `bank_accounts` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Copy rows from old table "properties" to new temporary table "new_properties"
INSERT INTO `new_properties` (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `address`, `property_type`, `status`, `year_built`, `total_square_footage`, `total_spaces`, `lot_size_sqft`, `stories`, `parking_spaces`, `jurisdiction_id`, `rent_controlled`, `compliance_programs`, `requires_lead_disclosure`, `chart_of_accounts_id`, `insurance_policy_number`, `insurance_expiry`, `bank_account_properties`, `portfolio_properties`, `property_bank_account`) SELECT `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `name`, `address`, `property_type`, `status`, `year_built`, `total_square_footage`, `total_spaces`, `lot_size_sqft`, `stories`, `parking_spaces`, `jurisdiction_id`, `rent_controlled`, `compliance_programs`, `requires_lead_disclosure`, `chart_of_accounts_id`, `insurance_policy_number`, `insurance_expiry`, `bank_account_properties`, `portfolio_properties`, `property_bank_account` FROM `properties`;
-- Drop "properties" table after copying rows
DROP TABLE `properties`;
-- Rename temporary table "new_properties" to "properties"
ALTER TABLE `new_properties` RENAME TO `properties`;
-- Create "new_spaces" table
CREATE TABLE `new_spaces` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `space_number` text NOT NULL,
  `space_type` text NOT NULL,
  `status` text NOT NULL,
  `leasable` bool NOT NULL,
  `shared_with_parent` bool NOT NULL DEFAULT false,
  `square_footage` real NOT NULL,
  `bedrooms` integer NULL,
  `bathrooms` real NULL,
  `floor` integer NULL,
  `amenities` json NULL,
  `floor_plan` text NULL,
  `ada_accessible` bool NOT NULL DEFAULT false,
  `pet_friendly` bool NOT NULL DEFAULT true,
  `furnished` bool NOT NULL DEFAULT false,
  `specialized_infrastructure` json NULL,
  `market_rent_amount_cents` integer NULL,
  `market_rent_currency` text NULL DEFAULT 'USD',
  `ami_restriction` integer NULL,
  `active_lease_id` text NULL,
  `building_spaces` uuid NULL,
  `property_spaces` uuid NOT NULL,
  `space_children` uuid NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `spaces_spaces_children` FOREIGN KEY (`space_children`) REFERENCES `spaces` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `spaces_properties_spaces` FOREIGN KEY (`property_spaces`) REFERENCES `properties` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `spaces_buildings_spaces` FOREIGN KEY (`building_spaces`) REFERENCES `buildings` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Copy rows from old table "spaces" to new temporary table "new_spaces"
INSERT INTO `new_spaces` (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `space_number`, `space_type`, `status`, `leasable`, `shared_with_parent`, `square_footage`, `bedrooms`, `bathrooms`, `floor`, `amenities`, `floor_plan`, `ada_accessible`, `pet_friendly`, `furnished`, `specialized_infrastructure`, `market_rent_amount_cents`, `market_rent_currency`, `ami_restriction`, `active_lease_id`, `building_spaces`, `property_spaces`, `space_children`) SELECT `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `source`, `correlation_id`, `agent_goal_id`, `space_number`, `space_type`, `status`, `leasable`, `shared_with_parent`, `square_footage`, `bedrooms`, `bathrooms`, `floor`, `amenities`, `floor_plan`, `ada_accessible`, `pet_friendly`, `furnished`, `specialized_infrastructure`, `market_rent_amount_cents`, `market_rent_currency`, `ami_restriction`, `active_lease_id`, `building_spaces`, `property_spaces`, `space_children` FROM `spaces`;
-- Drop "spaces" table after copying rows
DROP TABLE `spaces`;
-- Rename temporary table "new_spaces" to "spaces"
ALTER TABLE `new_spaces` RENAME TO `spaces`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
