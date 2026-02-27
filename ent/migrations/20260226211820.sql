-- Create "base_entities" table
CREATE TABLE `base_entities` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  PRIMARY KEY (`id`)
);
-- Create "immutable_entities" table
CREATE TABLE `immutable_entities` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  PRIMARY KEY (`id`)
);
-- Create "stateful_entities" table
CREATE TABLE `stateful_entities` (
  `id` uuid NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_by` text NOT NULL,
  `updated_by` text NOT NULL,
  `source` text NOT NULL,
  `correlation_id` text NULL,
  `agent_goal_id` text NULL,
  `status` text NOT NULL,
  PRIMARY KEY (`id`)
);
