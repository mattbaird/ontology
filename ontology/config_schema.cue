// ontology/config_schema.cue
package propeller

// ─── Configuration Scope ────────────────────────────────────────────────────
// Lower levels override higher. Jurisdiction overrides everything
// (legal requirements trump preferences).

#ConfigurationScope: close({
	level: "platform" | "organization" | "portfolio" | "property" | "lease_type" | "jurisdiction"
})

// ─── Lease Configuration ────────────────────────────────────────────────────
// Configurable defaults for lease-related operations.

#LeaseConfiguration: close({
	notice_required_days:       *30 | int & >=0 & <=365
	late_fee_grace_period_days: *5 | int & >=0 & <=30
	late_fee_type:              *"flat" | "flat" | "percent" | "per_day"
	late_fee_flat_amount?:      #NonNegativeMoney
	late_fee_percent?:          float & >0 & <=25
	security_deposit_max_months: *2.0 | float & >0 & <=6
	auto_renewal_enabled:       bool | *false
	auto_renewal_notice_days:   *60 | int & >=30 & <=180
	allow_partial_payments:     bool | *true
	minimum_payment_percent?:   float & >0 & <=100
})

// ─── Property Configuration ─────────────────────────────────────────────────
// Configurable defaults for property-level operations.

#PropertyConfiguration: close({
	maintenance_auto_approve_limit: #NonNegativeMoney
	screening_required:             bool | *true
	screening_income_ratio:         *3.0 | float & >=1 & <=10
	pet_policy:                     *"allowed" | "allowed" | "restricted" | "prohibited"
	max_pets?:                      int & >=0 & <=10
	pet_deposit?:                   #NonNegativeMoney
	pet_rent?:                      #NonNegativeMoney
})
