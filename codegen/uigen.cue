// codegen/uigen.cue
// UI generation overrides: display names, hidden fields, enum groupings.
// These are values not derivable from the ontology alone.
package codegen

#UIEntityOverride: {
	display_name:              string
	display_name_plural:       string
	primary_display_template?: string // e.g., "{space_number} — {tenant_name}"
	hidden_fields?: [...string]
	field_overrides?: [string]: #UIFieldOverride
}

#UIFieldOverride: {
	label?:          string
	help_text?:      string
	show_in_create?: bool
	show_in_update?: bool
	show_in_list?:   bool
	show_in_detail?: bool
}

#UIEnumGrouping: {
	values: [...#UIEnumValue]
	groups?: [...#UIEnumGroup]
}

#UIEnumValue: {
	value: string
	label: string
}

#UIEnumGroup: {
	label:  string
	values: [...string]
}

// Per-entity UI overrides
ui_entity_overrides: [string]: #UIEntityOverride
ui_entity_overrides: {
	Person: {
		display_name:             "Person"
		display_name_plural:      "People"
		primary_display_template: "{first_name} {last_name}"
	}
	Organization: {
		display_name:             "Organization"
		display_name_plural:      "Organizations"
		primary_display_template: "{legal_name}"
	}
	PersonRole: {
		display_name:             "Person Role"
		display_name_plural:      "Person Roles"
		primary_display_template: "{role_type}"
	}
	Portfolio: {
		display_name:             "Portfolio"
		display_name_plural:      "Portfolios"
		primary_display_template: "{name}"
	}
	Property: {
		display_name:             "Property"
		display_name_plural:      "Properties"
		primary_display_template: "{name}"
	}
	Building: {
		display_name:             "Building"
		display_name_plural:      "Buildings"
		primary_display_template: "{name}"
	}
	Space: {
		display_name:             "Space"
		display_name_plural:      "Spaces"
		primary_display_template: "{space_number}"
	}
	Lease: {
		display_name:             "Lease"
		display_name_plural:      "Leases"
		primary_display_template: "{lease_type}"
	}
	LeaseSpace: {
		display_name:             "Lease Space"
		display_name_plural:      "Lease Spaces"
		primary_display_template: "{relationship}"
	}
	Application: {
		display_name:             "Application"
		display_name_plural:      "Applications"
		primary_display_template: "{status}"
	}
	Account: {
		display_name:             "Account"
		display_name_plural:      "Accounts"
		primary_display_template: "{account_number} — {name}"
	}
	LedgerEntry: {
		display_name:             "Ledger Entry"
		display_name_plural:      "Ledger Entries"
		primary_display_template: "{entry_type} — {description}"
	}
	JournalEntry: {
		display_name:             "Journal Entry"
		display_name_plural:      "Journal Entries"
		primary_display_template: "{source_type} — {description}"
	}
	BankAccount: {
		display_name:             "Bank Account"
		display_name_plural:      "Bank Accounts"
		primary_display_template: "{name}"
	}
	Reconciliation: {
		display_name:             "Reconciliation"
		display_name_plural:      "Reconciliations"
		primary_display_template: "{status}"
	}
	Jurisdiction: {
		display_name:             "Jurisdiction"
		display_name_plural:      "Jurisdictions"
		primary_display_template: "{name}"
	}
	PropertyJurisdiction: {
		display_name:             "Property Jurisdiction"
		display_name_plural:      "Property Jurisdictions"
		primary_display_template: "{jurisdiction_id}"
	}
	JurisdictionRule: {
		display_name:             "Jurisdiction Rule"
		display_name_plural:      "Jurisdiction Rules"
		primary_display_template: "{rule_type}"
	}
}

// Enum grouping overrides (for enums that benefit from grouped display)
ui_enum_groupings: [string]: #UIEnumGrouping
ui_enum_groupings: {
	LeaseType: {
		values: [
			{value: "fixed_term", label:               "Fixed Term"},
			{value: "month_to_month", label:            "Month to Month"},
			{value: "commercial_nnn", label:             "Triple Net (NNN)"},
			{value: "commercial_nn", label:              "Double Net (NN)"},
			{value: "commercial_n", label:               "Single Net (N)"},
			{value: "commercial_gross", label:           "Gross / Full Service"},
			{value: "commercial_modified_gross", label:  "Modified Gross"},
			{value: "affordable", label:                 "Affordable Housing"},
			{value: "section_8", label:                  "Section 8"},
			{value: "student", label:                    "Student"},
			{value: "ground_lease", label:               "Ground Lease"},
			{value: "short_term", label:                 "Short Term"},
			{value: "membership", label:                 "Membership"},
		]
		groups: [
			{label: "Residential", values: ["fixed_term", "month_to_month", "affordable", "section_8", "student", "short_term"]},
			{label: "Commercial", values: ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross", "ground_lease"]},
			{label: "Other", values: ["membership"]},
		]
	}
	SpaceType: {
		values: [
			{value: "residential_unit", label:  "Residential Unit"},
			{value: "commercial_office", label: "Commercial Office"},
			{value: "commercial_retail", label: "Commercial Retail"},
			{value: "storage", label:           "Storage"},
			{value: "parking", label:           "Parking"},
			{value: "common_area", label:       "Common Area"},
			{value: "industrial", label:        "Industrial"},
			{value: "lot_pad", label:           "Lot / Pad"},
			{value: "bed_space", label:         "Bed Space"},
			{value: "desk_space", label:        "Desk Space"},
		]
		groups: [
			{label: "Residential", values: ["residential_unit", "bed_space"]},
			{label: "Commercial", values: ["commercial_office", "commercial_retail", "industrial", "desk_space"]},
			{label: "Utility", values: ["storage", "parking", "common_area", "lot_pad"]},
		]
	}
	PropertyType: {
		values: [
			{value: "single_family", label:      "Single Family"},
			{value: "multi_family", label:       "Multi-Family"},
			{value: "commercial_office", label:  "Commercial Office"},
			{value: "commercial_retail", label:  "Commercial Retail"},
			{value: "mixed_use", label:          "Mixed Use"},
			{value: "industrial", label:         "Industrial"},
			{value: "affordable_housing", label: "Affordable Housing"},
			{value: "student_housing", label:    "Student Housing"},
			{value: "senior_living", label:      "Senior Living"},
			{value: "vacation_rental", label:    "Vacation Rental"},
			{value: "mobile_home_park", label:   "Mobile Home Park"},
			{value: "self_storage", label:       "Self Storage"},
			{value: "coworking", label:          "Coworking"},
			{value: "data_center", label:        "Data Center"},
			{value: "medical_office", label:     "Medical Office"},
		]
		groups: [
			{label: "Residential", values: ["single_family", "multi_family", "affordable_housing", "student_housing", "senior_living", "vacation_rental", "mobile_home_park"]},
			{label: "Commercial", values: ["commercial_office", "commercial_retail", "mixed_use", "industrial", "self_storage", "coworking", "data_center", "medical_office"]},
		]
	}
	AccountType: {
		values: [
			{value: "asset", label:     "Asset"},
			{value: "liability", label: "Liability"},
			{value: "equity", label:    "Equity"},
			{value: "revenue", label:   "Revenue"},
			{value: "expense", label:   "Expense"},
		]
		groups: [
			{label: "Balance Sheet", values: ["asset", "liability", "equity"]},
			{label: "Income Statement", values: ["revenue", "expense"]},
		]
	}
}
