// policies/field_policies.cue
// Per-attribute visibility rules. Some roles can see a lease but not the security
// deposit amount. This is defined here, not on the entity.
package policies

#FieldPolicy: close({
	visible_to:  [...string] // permission groups that can see this field
	hidden_from: [...string] // permission groups that cannot see this field ("*" = default deny)
})

field_policies: {
	person: {
		ssn_last_four: #FieldPolicy & {
			visible_to:  ["organization_admin", "portfolio_admin", "accountant"]
			hidden_from: ["*"] // default deny
		}
		date_of_birth: #FieldPolicy & {
			visible_to:  ["organization_admin", "portfolio_admin", "leasing_agent"]
			hidden_from: ["maintenance_coordinator", "viewer"]
		}
	}

	lease: {
		security_deposit: #FieldPolicy & {
			visible_to:  ["organization_admin", "portfolio_admin", "property_manager", "accountant"]
			hidden_from: ["maintenance_coordinator"]
		}
	}

	bank_account: {
		routing_number: #FieldPolicy & {
			visible_to:  ["organization_admin", "accountant"]
			hidden_from: ["*"]
		}
		account_number_encrypted: #FieldPolicy & {
			visible_to:  []    // nobody sees this in UI; system-only
			hidden_from: ["*"]
		}
	}
}
