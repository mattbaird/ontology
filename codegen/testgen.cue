// codegen/testgen.cue
// Test generation configuration.
// Defines which constraints generate test cases and state machine test matrices.
package codegen

#TestgenConfig: {
	// State machine test generation
	state_machine_tests: {
		// Generate positive tests for every valid from -> to transition
		generate_positive: true
		// Generate negative tests for every invalid from -> to pair
		generate_negative: true
		// Output format
		output_format: "go_test"
	}

	// Constraint test generation
	constraint_tests: {
		// Generate tests for conditional block constraints
		generate_conditional: true
		// Test cross-field constraints (e.g., "NNN requires all three CAM flags")
		generate_cross_field: true
	}

	// Output
	output_dir: "gen/tests"
}

testgen_config: #TestgenConfig
