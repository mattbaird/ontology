// Package types provides Go structs for CUE value types used in Ent JSON fields.
// These types are the Go representation of shared ontology types that are stored
// as JSON columns in Postgres when embedded in entities.
package types

import (
	"encoding/json"
	"time"
)

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
	EffectivePeriod DateRange       `json:"effective_period"`
	FixedAmount     *Money          `json:"fixed_amount,omitempty"`
	Adjustment      *RentAdjustment `json:"adjustment,omitempty"`
	Description     string          `json:"description"`
	ChargeCode      string          `json:"charge_code"`
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
	SpaceID         string    `json:"space_id,omitempty"`
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
	BaseYearExpenses      *Money   `json:"base_year_expenses,omitempty"`
	ExpenseStop           *Money   `json:"expense_stop,omitempty"`
	CategoryTerms         []CAMCategoryTerms `json:"category_terms,omitempty"`
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
	CPIFloor           *float64 `json:"cpi_floor,omitempty"`
	CPICeiling         *float64 `json:"cpi_ceiling,omitempty"`
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
	OccupancyStatus string     `json:"occupancy_status"`
	LiabilityStatus string     `json:"liability_status"`
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

// UsageBasedCharge defines metered utility charges.
type UsageBasedCharge struct {
	ID               string `json:"id"`
	ChargeCode       string `json:"charge_code"`
	Description      string `json:"description"`
	UnitOfMeasure    string `json:"unit_of_measure"`    // "kwh", "gallon", "cubic_foot", "therm", "hour", "gb"
	RatePerUnit      Money  `json:"rate_per_unit"`       // #PositiveMoney
	MeterID          string `json:"meter_id,omitempty"`
	BillingFrequency string `json:"billing_frequency"`   // "monthly", "quarterly"
	Cap              *Money `json:"cap,omitempty"`
	SpaceID          string `json:"space_id,omitempty"`
}

// PercentageRent defines retail percentage rent terms.
type PercentageRent struct {
	Rate                  float64 `json:"rate"`
	BreakpointType        string  `json:"breakpoint_type"`
	NaturalBreakpoint     *Money  `json:"natural_breakpoint,omitempty"`
	ArtificialBreakpoint  *Money  `json:"artificial_breakpoint,omitempty"`
	ReportingFrequency    string  `json:"reporting_frequency"`
	AuditRights           bool    `json:"audit_rights"`
}

// RentAdjustment defines formula-based rent escalations.
type RentAdjustment struct {
	Method                 string   `json:"method"`      // "cpi", "fixed_percent", "fixed_amount_increase", "market_review"
	BaseAmount             Money    `json:"base_amount"`
	CPIIndex               string   `json:"cpi_index,omitempty"`    // "CPI-U", "CPI-W", "regional"
	CPIFloor               *float64 `json:"cpi_floor,omitempty"`
	CPICeiling             *float64 `json:"cpi_ceiling,omitempty"`
	PercentIncrease        *float64 `json:"percent_increase,omitempty"`
	AmountIncrease         *Money   `json:"amount_increase,omitempty"`
	MarketReviewMechanism  string   `json:"market_review_mechanism,omitempty"`
}

// ExpansionRight defines commercial expansion options.
type ExpansionRight struct {
	Type               string     `json:"type"` // "first_right_of_refusal", "first_right_to_negotiate", "must_take", "option"
	TargetSpaceIDs     []string   `json:"target_space_ids"`
	ExerciseDeadline   *time.Time `json:"exercise_deadline,omitempty"`
	Terms              string     `json:"terms,omitempty"`
	NoticeRequiredDays int        `json:"notice_required_days"`
}

// ContractionRight defines commercial contraction options.
type ContractionRight struct {
	MinimumRetainedSqft  float64    `json:"minimum_retained_sqft"`
	EarliestExerciseDate time.Time  `json:"earliest_exercise_date"`
	Penalty              *Money     `json:"penalty,omitempty"`
	NoticeRequiredDays   int        `json:"notice_required_days"`
}

// CAMCategoryTerms defines per-category CAM controls.
type CAMCategoryTerms struct {
	Category    string   `json:"category"` // "property_tax", "insurance", "utilities", etc.
	TenantPays  bool     `json:"tenant_pays"`
	LandlordCap *Money   `json:"landlord_cap,omitempty"`
	TenantCap   *Money   `json:"tenant_cap,omitempty"`
	Escalation  *float64 `json:"escalation,omitempty"`
}

// RoleAttributes is a union type for role-specific attributes.
// In Go, we use json.RawMessage at the Ent level and unmarshal to the
// specific type based on the _type field.
type RoleAttributes struct {
	Type string `json:"_type"`
	// Raw holds the full JSON for type-specific unmarshaling
	Raw []byte `json:"-"`
}

// ─── Signal Discovery Types ────────────────────────────────────────────────────
// These types support the signal discovery system (Layers 1-3).
// ActivityEntry is NOT an Ent entity — it's stored in a separate partitioned
// Postgres table outside the ORM.

// SourceRef identifies an entity referenced by a domain event.
type SourceRef struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Role       string `json:"role"` // "subject", "target", "related", "context"
}

// ActivityEntry is a secondary index entry over the domain event log,
// keyed by a referenced entity. One event produces multiple entries.
type ActivityEntry struct {
	EventID           string            `json:"event_id"`
	EventType         string            `json:"event_type"`
	OccurredAt        time.Time         `json:"occurred_at"`
	IndexedEntityType string            `json:"indexed_entity_type"`
	IndexedEntityID   string            `json:"indexed_entity_id"`
	EntityRole        string            `json:"entity_role"` // "subject", "target", "related", "context"
	SourceRefs        []SourceRef       `json:"source_refs"`
	Summary           string            `json:"summary"`
	Category          string            `json:"category"` // #SignalCategory
	Weight            string            `json:"weight"`   // #SignalWeight
	Polarity          string            `json:"polarity"` // #SignalPolarity
	Payload           json.RawMessage   `json:"payload"`
}

// SignalRegistration maps an event type to a signal classification.
type SignalRegistration struct {
	ID                     string           `json:"id"`
	EventType              string           `json:"event_type"`
	Condition              string           `json:"condition,omitempty"`
	Category               string           `json:"category"`
	Weight                 string           `json:"weight"`
	Polarity               string           `json:"polarity"`
	Description            string           `json:"description"`
	InterpretationGuidance string           `json:"interpretation_guidance,omitempty"`
	EscalationRules        []EscalationRule `json:"escalation_rules,omitempty"`
}

// EscalationRule defines when repeated or combined signals escalate in severity.
type EscalationRule struct {
	ID                     string                 `json:"id"`
	Description            string                 `json:"description"`
	TriggerType            string                 `json:"trigger_type"` // "count", "cross_category", "absence", "trend"
	SignalCategory         string                 `json:"signal_category,omitempty"`
	SignalPolarity         string                 `json:"signal_polarity,omitempty"`
	Count                  int                    `json:"count,omitempty"`
	WithinDays             int                    `json:"within_days,omitempty"`
	RequiredCategories     []CategoryRequirement  `json:"required_categories,omitempty"`
	ExpectedSignalCategory string                 `json:"expected_signal_category,omitempty"`
	AbsentForDays          int                    `json:"absent_for_days,omitempty"`
	AppliesToCondition     string                 `json:"applies_to_condition,omitempty"`
	TrendDirection         string                 `json:"trend_direction,omitempty"`
	TrendMetric            string                 `json:"trend_metric,omitempty"`
	TrendWindowDays        int                    `json:"trend_window_days,omitempty"`
	EscalatedWeight        string                 `json:"escalated_weight"`
	EscalatedDescription   string                 `json:"escalated_description"`
	RecommendedAction      string                 `json:"recommended_action,omitempty"`
}

// CategoryRequirement is used in cross-category escalation rules.
type CategoryRequirement struct {
	Category string `json:"category"`
	Polarity string `json:"polarity,omitempty"`
	MinCount int    `json:"min_count"`
}

// EscalatedSignal is a triggered escalation rule with context.
type EscalatedSignal struct {
	Rule             EscalationRule `json:"rule"`
	TriggeringCount  int            `json:"triggering_count"`
	EarliestOccurred time.Time      `json:"earliest_occurred"`
	LatestOccurred   time.Time      `json:"latest_occurred"`
}

// CategorySummary aggregates signals within a single category.
type CategorySummary struct {
	Category        string            `json:"category"`
	SignalCount     int               `json:"signal_count"`
	ByWeight        map[string]int    `json:"by_weight"`
	ByPolarity      map[string]int    `json:"by_polarity"`
	DominantPolarity string           `json:"dominant_polarity"`
	TopSignals      []ActivityEntry   `json:"top_signals,omitempty"`
	Trend           string            `json:"trend"` // "improving", "stable", "declining"
}

// SignalSummary is the pre-aggregated signal overview for an entity.
type SignalSummary struct {
	EntityType          string                     `json:"entity_type"`
	EntityID            string                     `json:"entity_id"`
	Since               time.Time                  `json:"since"`
	Until               time.Time                  `json:"until"`
	Categories          map[string]CategorySummary  `json:"categories"`
	OverallSentiment    string                     `json:"overall_sentiment"` // "positive", "mixed", "concerning", "critical"
	SentimentReason     string                     `json:"sentiment_reason"`
	Escalations         []EscalatedSignal          `json:"escalations"`
}
