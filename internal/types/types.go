// Package types provides Go structs for CUE value types used in Ent JSON fields.
// These types are the Go representation of shared ontology types that are stored
// as JSON columns in Postgres when embedded in entities.
package types

import "time"

// Money represents a monetary amount using integer cents to eliminate
// floating-point errors in financial operations.
type Money struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"` // ISO 4217, e.g. "USD"
}

// NonNegativeMoney is Money with amount_cents >= 0.
// Constraint enforcement happens at the Ent hook level.
type NonNegativeMoney = Money

// PositiveMoney is Money with amount_cents > 0.
// Constraint enforcement happens at the Ent hook level.
type PositiveMoney = Money

// DateRange represents a time period with an optional end.
type DateRange struct {
	Start time.Time  `json:"start"`
	End   *time.Time `json:"end,omitempty"`
}

// Address represents a US postal address with optional geolocation.
type Address struct {
	Line1      string   `json:"line1"`
	Line2      string   `json:"line2,omitempty"`
	City       string   `json:"city"`
	State      string   `json:"state"`       // 2-letter state code
	PostalCode string   `json:"postal_code"` // ZIP or ZIP+4
	Country    string   `json:"country"`     // ISO 3166-1 alpha-2
	Latitude   *float64 `json:"latitude,omitempty"`
	Longitude  *float64 `json:"longitude,omitempty"`
	County     string   `json:"county,omitempty"`
}

// ContactMethod represents a way to contact a person or organization.
type ContactMethod struct {
	Type     string `json:"type"`     // "email", "phone", "sms", "mail", "portal"
	Value    string `json:"value"`    // The actual contact value
	Primary  bool   `json:"primary"`  // Whether this is the preferred method
	Verified bool   `json:"verified"` // Whether ownership has been verified
	OptOut   bool   `json:"opt_out"`  // Communication preference opt-out
	Label    string `json:"label,omitempty"` // "work", "home", "mobile", etc.
}

// EntityRef is the universal relationship primitive.
type EntityRef struct {
	EntityType   string `json:"entity_type"`
	EntityID     string `json:"entity_id"`
	Relationship string `json:"relationship"`
}

// RentScheduleEntry defines rent for a specific time period within a lease.
type RentScheduleEntry struct {
	EffectivePeriod DateRange `json:"effective_period"`
	Amount          Money     `json:"amount"`
	Description     string    `json:"description"`
	ChargeCode      string    `json:"charge_code"`
}

// RecurringCharge represents a recurring fee beyond base rent.
type RecurringCharge struct {
	ID              string    `json:"id"`
	ChargeCode      string    `json:"charge_code"`
	Description     string    `json:"description"`
	Amount          Money     `json:"amount"`
	Frequency       string    `json:"frequency"` // "monthly", "quarterly", "annually", "one_time"
	EffectivePeriod DateRange `json:"effective_period"`
	Taxable         bool      `json:"taxable"`
}

// LateFeePolicy defines how late fees are calculated.
type LateFeePolicy struct {
	GracePeriodDays int          `json:"grace_period_days"`
	FeeType         string       `json:"fee_type"` // "flat", "percent", "per_day", "tiered"
	FlatAmount      *Money       `json:"flat_amount,omitempty"`
	Percent         *float64     `json:"percent,omitempty"`
	PerDayAmount    *Money       `json:"per_day_amount,omitempty"`
	MaxFee          *Money       `json:"max_fee,omitempty"`
	Tiers           []LateFeeTier `json:"tiers,omitempty"`
}

// LateFeeTier defines a tier in a tiered late fee policy.
type LateFeeTier struct {
	DaysLateMin int   `json:"days_late_min"`
	DaysLateMax int   `json:"days_late_max"`
	Amount      Money `json:"amount"`
}

// CAMTerms defines Common Area Maintenance terms for commercial leases.
type CAMTerms struct {
	ReconciliationType    string  `json:"reconciliation_type"` // "estimated_with_annual_reconciliation", "fixed", "actual"
	ProRataSharePercent   float64 `json:"pro_rata_share_percent"`
	EstimatedMonthlyCAM   Money   `json:"estimated_monthly_cam"`
	AnnualCap             *Money  `json:"annual_cap,omitempty"`
	BaseYear              *int    `json:"base_year,omitempty"`
	IncludesPropertyTax   bool    `json:"includes_property_tax"`
	IncludesInsurance     bool    `json:"includes_insurance"`
	IncludesUtilities     bool    `json:"includes_utilities"`
	ExcludedCategories    []string `json:"excluded_categories,omitempty"`
}

// TenantImprovement defines tenant improvement allowance terms.
type TenantImprovement struct {
	Allowance              Money      `json:"allowance"`
	Amortized              bool       `json:"amortized"`
	AmortizationTermMonths *int       `json:"amortization_term_months,omitempty"`
	InterestRatePercent    *float64   `json:"interest_rate_percent,omitempty"`
	CompletionDeadline     *time.Time `json:"completion_deadline,omitempty"`
}

// RenewalOption defines a lease renewal option.
type RenewalOption struct {
	OptionNumber       int    `json:"option_number"`
	TermMonths         int    `json:"term_months"`
	RentAdjustment     string `json:"rent_adjustment"` // "fixed", "cpi", "percent_increase", "market"
	FixedRent          *Money   `json:"fixed_rent,omitempty"`
	PercentIncrease    *float64 `json:"percent_increase,omitempty"`
	NoticeRequiredDays int      `json:"notice_required_days"`
	MustExerciseBy     *time.Time `json:"must_exercise_by,omitempty"`
}

// SubsidyTerms defines affordable housing subsidy information.
type SubsidyTerms struct {
	Program               string `json:"program"` // "section_8", "pbv", "vash", "home", "lihtc"
	HousingAuthority      string `json:"housing_authority"`
	HAPContractID         string `json:"hap_contract_id,omitempty"`
	ContractRent          Money  `json:"contract_rent"`
	TenantPortion         Money  `json:"tenant_portion"`
	SubsidyPortion        Money  `json:"subsidy_portion"`
	UtilityAllowance      Money  `json:"utility_allowance"`
	AnnualRecertDate      *time.Time `json:"annual_recert_date,omitempty"`
	IncomeLimitAMIPercent int    `json:"income_limit_ami_percent"`
}

// AccountDimensions supports multi-dimensional accounting.
type AccountDimensions struct {
	EntityID   string `json:"entity_id,omitempty"`
	PropertyID string `json:"property_id,omitempty"`
	Dimension1 string `json:"dimension_1,omitempty"`
	Dimension2 string `json:"dimension_2,omitempty"`
	Dimension3 string `json:"dimension_3,omitempty"`
}

// JournalLine represents a single line in a journal entry.
type JournalLine struct {
	AccountID   string            `json:"account_id"`
	Debit       *Money            `json:"debit,omitempty"`
	Credit      *Money            `json:"credit,omitempty"`
	Description string            `json:"description,omitempty"`
	Dimensions  *AccountDimensions `json:"dimensions,omitempty"`
}

// TenantAttributes are role-specific attributes for tenants.
type TenantAttributes struct {
	Type            string     `json:"_type"` // always "tenant"
	Standing        string     `json:"standing"`
	ScreeningStatus string     `json:"screening_status"`
	ScreeningDate   *time.Time `json:"screening_date,omitempty"`
	CurrentBalance  *Money     `json:"current_balance,omitempty"`
	MoveInDate      *time.Time `json:"move_in_date,omitempty"`
	MoveOutDate     *time.Time `json:"move_out_date,omitempty"`
	PetCount        *int       `json:"pet_count,omitempty"`
	VehicleCount    *int       `json:"vehicle_count,omitempty"`
}

// OwnerAttributes are role-specific attributes for owners.
type OwnerAttributes struct {
	Type                 string   `json:"_type"` // always "owner"
	OwnershipPercent     float64  `json:"ownership_percent"`
	DistributionMethod   string   `json:"distribution_method"`
	ManagementFeePercent *float64 `json:"management_fee_percent,omitempty"`
	TaxReporting         string   `json:"tax_reporting"`
	ReserveAmount        *Money   `json:"reserve_amount,omitempty"`
}

// ManagerAttributes are role-specific attributes for property managers.
type ManagerAttributes struct {
	Type              string `json:"_type"` // always "manager"
	LicenseNumber     string `json:"license_number,omitempty"`
	LicenseState      string `json:"license_state,omitempty"`
	ApprovalLimit     *Money `json:"approval_limit,omitempty"`
	CanSignLeases     bool   `json:"can_sign_leases"`
	CanApproveExpenses bool  `json:"can_approve_expenses"`
}

// GuarantorAttributes are role-specific attributes for guarantors.
type GuarantorAttributes struct {
	Type            string     `json:"_type"` // always "guarantor"
	GuaranteeType   string     `json:"guarantee_type"` // "full", "partial", "conditional"
	GuaranteeAmount *Money     `json:"guarantee_amount,omitempty"`
	GuaranteeTerm   *DateRange `json:"guarantee_term,omitempty"`
	CreditScore     *int       `json:"credit_score,omitempty"`
}

// RoleAttributes is a union type for role-specific attributes.
// In Go, we use json.RawMessage at the Ent level and unmarshal to the
// specific type based on the _type field.
type RoleAttributes struct {
	Type string `json:"_type"`
	// Raw holds the full JSON for type-specific unmarshaling
	Raw []byte `json:"-"`
}
