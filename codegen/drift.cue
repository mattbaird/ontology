// codegen/drift.cue
// Cross-boundary validation rules for drift detection.
// Ensures commands, events, API contracts, and policies stay consistent with the ontology.
package codegen

#DriftCheckConfig: {
	// Packages to validate
	packages: [...string]

	// Drift checks to run
	checks: {
		// Check 1: Command _affects lists reference valid entity types
		command_entity_references: true
		// Check 2: Events using ontology enums stay in sync
		event_enum_sync: true
		// Check 3: API responses importing ontology enums stay in sync
		api_enum_sync: true
		// Check 5: Permission commands reference actual command keys
		permission_command_references: true
	}
}

drift_config: #DriftCheckConfig & {
	packages: [
		"./ontology/...",
		"./commands/...",
		"./events/...",
		"./api/...",
		"./policies/...",
		"./codegen/...",
	]
}
