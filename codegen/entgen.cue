// codegen/entgen.cue
// Ontology -> Ent schema mapping configuration.
// This documents the tight coupling between ontology and data store.
package codegen

// EntgenConfig defines how ontology types map to Ent fields.
#EntgenConfig: {
	// Type mapping: CUE -> Ent field types
	type_mappings: {
		"string":           "String"
		"int":              "Int"
		"float":            "Float64"
		"bool":             "Bool"
		"time.Time":        "Time"
		"enum":             "Enum"
		"#Money":           "JSON" // flattened to _amount_cents + _currency
		"#Address":         "JSON"
		"#ContactMethod":   "JSON"
		"#DateRange":       "JSON"
	}

	// Constraint mapping: CUE constraint -> Ent validator
	constraint_mappings: {
		"!= \"\"":           ".NotEmpty()"
		"strings.MinRunes":  ".NotEmpty()"
		">= 0":             ".NonNegative()"
		"> 0":              ".Positive()"
		"=~ pattern":        ".Match(pattern)"
	}

	// Base types skipped in entity detection
	skip_definitions: ["#BaseEntity", "#StatefulEntity", "#ImmutableEntity"]

	// Output directory
	output_dir: "ent/schema"
}

entgen_config: #EntgenConfig
