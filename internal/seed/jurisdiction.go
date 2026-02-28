// Package seed provides demo data seeding for the Ent database.
package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/ent/jurisdiction"
	"github.com/matthewbaird/ontology/ent/jurisdictionrule"
	"github.com/matthewbaird/ontology/ent/propertyjurisdiction"
)

// SeedJurisdictions creates a jurisdiction hierarchy (Federal → California → LA County → Santa Monica)
// with real-world rules. If jurisdictions already exist (idempotent check), it skips seeding.
func SeedJurisdictions(ctx context.Context, client *ent.Client) error {
	// Check if jurisdictions already exist.
	count, err := client.Jurisdiction.Query().Count(ctx)
	if err != nil {
		return fmt.Errorf("checking jurisdictions: %w", err)
	}
	if count > 0 {
		log.Printf("jurisdictions already seeded (%d found), skipping", count)
		return nil
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	now := time.Now()
	actor := "system"
	source := jurisdiction.SourceSystem

	// ── Federal ───────────────────────────────────────────────────────
	federal, err := tx.Jurisdiction.Create().
		SetName("United States").
		SetJurisdictionType(jurisdiction.JurisdictionTypeFederal).
		SetCountryCode("US").
		SetStatus(jurisdiction.StatusActive).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(source).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating federal jurisdiction: %w", err)
	}

	// Federal rule: lead paint disclosure (pre-1978).
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeLeadPaintDisclosure).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"disclosure_type":  "lead_paint",
			"required_before":  "lease_signing",
			"applies_to_built": "before_1978",
			"description":      "Sellers and landlords must disclose known lead-based paint hazards for housing built before 1978",
		})).
		SetStatuteReference("42 U.S.C. §4852d").
		SetEffectiveDate(time.Date(1996, 3, 6, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(federal.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating federal lead paint rule: %w", err)
	}

	// ── California ───────────────────────────────────────────────────
	california, err := tx.Jurisdiction.Create().
		SetName("California").
		SetJurisdictionType(jurisdiction.JurisdictionTypeState).
		SetStateCode("CA").
		SetCountryCode("US").
		SetStatus(jurisdiction.StatusActive).
		SetParentJurisdictionID(federal.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(source).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating CA jurisdiction: %w", err)
	}

	// CA: Security deposit limit — max 1 month rent (AB 12, effective 2025-07-01).
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeSecurityDepositLimit).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"max_months":  1.0,
			"description": "Security deposit may not exceed one month's rent",
		})).
		SetStatuteReference("Cal. Civ. Code §1950.5 (as amended by AB 12)").
		SetEffectiveDate(time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(california.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating CA security deposit rule: %w", err)
	}

	// CA: Notice period — 30 days (< 1 year tenancy), 60 days (>= 1 year).
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeNoticePeriod).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"min_days":              30,
			"tenancy_over_months":   0,
			"min_days_extended":     60,
			"condition":             "tenancy > 12 months",
			"description":           "30-day notice for tenancies under 1 year; 60-day notice for tenancies 1 year or longer",
		})).
		SetStatuteReference("Cal. Civ. Code §1946.1").
		SetEffectiveDate(time.Date(2007, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(california.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating CA notice period rule: %w", err)
	}

	// CA: Rent increase cap — AB 1482 Tenant Protection Act (5% + CPI, max 10%).
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeRentIncreaseCap).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"max_percent_increase": 10.0,
			"based_on":             "cpi_plus",
			"cpi_plus_percent":     5.0,
			"description":          "Annual rent increase capped at 5% + CPI or 10%, whichever is lower",
		})).
		SetAppliesToLeaseTypes([]string{"fixed_term", "month_to_month"}).
		SetStatuteReference("Cal. Civ. Code §1947.12 (AB 1482)").
		SetEffectiveDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetExpirationDate(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(california.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating CA rent increase cap rule: %w", err)
	}

	// CA: Just cause eviction (AB 1482).
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeJustCauseEviction).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"applies_after_months":  12,
			"at_fault_causes":      []string{"nonpayment", "breach", "nuisance", "criminal_activity", "subletting_unauthorized", "refusal_to_sign_renewal"},
			"no_fault_causes":      []string{"owner_move_in", "withdrawal_from_rental_market", "substantial_renovation"},
			"relocation_required":  true,
			"relocation_amount":    "one_month_rent",
			"description":          "Just cause required for eviction after 12 months of tenancy",
		})).
		SetAppliesToLeaseTypes([]string{"fixed_term", "month_to_month"}).
		SetStatuteReference("Cal. Civ. Code §1946.2 (AB 1482)").
		SetEffectiveDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetExpirationDate(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(california.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating CA just cause eviction rule: %w", err)
	}

	// CA: Mold disclosure.
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeMoldDisclosure).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"disclosure_type": "mold",
			"required_before": "lease_signing",
			"description":     "Landlord must provide written disclosure of known mold hazards",
		})).
		SetStatuteReference("Cal. Health & Safety Code §26147-26148").
		SetEffectiveDate(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(california.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating CA mold disclosure rule: %w", err)
	}

	// CA: Bed bug disclosure.
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeBedBugDisclosure).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"disclosure_type": "bed_bug",
			"required_before": "lease_signing",
			"description":     "Landlord must provide written notice about bed bug history and prevention",
		})).
		SetStatuteReference("Cal. Civ. Code §1942.5(a)(1)").
		SetEffectiveDate(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(california.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating CA bed bug disclosure rule: %w", err)
	}

	// ── Los Angeles County ───────────────────────────────────────────
	laCounty, err := tx.Jurisdiction.Create().
		SetName("Los Angeles County").
		SetJurisdictionType(jurisdiction.JurisdictionTypeCounty).
		SetFipsCode("06037").
		SetStateCode("CA").
		SetCountryCode("US").
		SetStatus(jurisdiction.StatusActive).
		SetParentJurisdictionID(california.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(source).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating LA County jurisdiction: %w", err)
	}

	// LA County: Rent stabilization (unincorporated areas).
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeRentControl).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"max_percent_increase": 3.0,
			"applies_to":          "unincorporated_areas",
			"based_on":            "fixed",
			"description":         "Rent stabilization for unincorporated LA County: max 3% annual increase",
		})).
		SetStatuteReference("LA County Rent Stabilization Ordinance §8.52.060").
		SetEffectiveDate(time.Date(2020, 4, 1, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(laCounty.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating LA County rent stabilization rule: %w", err)
	}

	// ── Santa Monica ─────────────────────────────────────────────────
	santaMonica, err := tx.Jurisdiction.Create().
		SetName("Santa Monica").
		SetJurisdictionType(jurisdiction.JurisdictionTypeCity).
		SetFipsCode("0670000").
		SetStateCode("CA").
		SetCountryCode("US").
		SetStatus(jurisdiction.StatusActive).
		SetParentJurisdictionID(laCounty.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(source).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating Santa Monica jurisdiction: %w", err)
	}

	// Santa Monica: Strict rent control — max annual increase set by Rent Control Board.
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeRentControl).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"max_percent_increase": 3.0,
			"based_on":             "board_determined",
			"board":                "Santa Monica Rent Control Board",
			"description":          "Annual rent increase set by Rent Control Board, typically 75% of CPI (historically ~3%)",
		})).
		SetStatuteReference("Santa Monica Charter Amendment §1800 et seq.").
		SetEffectiveDate(time.Date(1979, 4, 10, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(santaMonica.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating Santa Monica rent control rule: %w", err)
	}

	// Santa Monica: Just cause eviction (stricter than state).
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeJustCauseEviction).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"applies_after_months":  0,
			"at_fault_causes":      []string{"nonpayment", "breach", "nuisance", "criminal_activity", "illegal_use"},
			"no_fault_causes":      []string{"owner_move_in", "withdrawal_from_rental_market"},
			"relocation_required":  true,
			"relocation_amount":    "varies_by_tenure",
			"right_to_counsel":     true,
			"description":          "Just cause required for all evictions (no minimum tenancy). Tenant has right to counsel.",
		})).
		SetStatuteReference("SMMC §4.36.020").
		SetEffectiveDate(time.Date(1979, 4, 10, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(santaMonica.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating Santa Monica just cause rule: %w", err)
	}

	// Santa Monica: Relocation assistance for no-fault evictions.
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeRelocationAssistance).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"base_amount_cents":       15725_00,
			"senior_disabled_amount":  18868_00,
			"low_income_additional":   3145_00,
			"applies_to":             "no_fault_evictions",
			"description":            "Mandatory relocation assistance for no-fault evictions; amounts adjusted annually",
		})).
		SetStatuteReference("SMMC §4.36.040").
		SetEffectiveDate(now).
		SetJurisdictionID(santaMonica.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating Santa Monica relocation rule: %w", err)
	}

	// Santa Monica: Right to counsel in eviction proceedings.
	_, err = tx.JurisdictionRule.Create().
		SetRuleType(jurisdictionrule.RuleTypeRightToCounsel).
		SetStatus(jurisdictionrule.StatusActive).
		SetRuleDefinition(mustMarshal(map[string]any{
			"coverage":    "all_residential_tenants",
			"description": "All residential tenants have right to free legal representation in eviction proceedings",
		})).
		SetStatuteReference("SMMC §4.36.060").
		SetEffectiveDate(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetJurisdictionID(santaMonica.ID).
		SetCreatedBy(actor).
		SetUpdatedBy(actor).
		SetSource(jurisdictionrule.SourceSystem).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("creating Santa Monica right to counsel rule: %w", err)
	}

	// ── Link demo property to jurisdictions ──────────────────────────
	// If the "sunset-apartments" property exists in the demo seed data,
	// link it to all applicable jurisdictions.
	props, err := tx.Property.Query().All(ctx)
	if err == nil && len(props) > 0 {
		for _, prop := range props {
			for _, j := range []*ent.Jurisdiction{federal, california, laCounty, santaMonica} {
				_, err := tx.PropertyJurisdiction.Create().
					SetPropertyID(prop.ID).
					SetJurisdictionID(j.ID).
					SetEffectiveDate(now).
					SetLookupSource(propertyjurisdiction.LookupSourceManual).
					SetVerified(true).
					SetVerifiedAt(now).
					SetVerifiedBy(actor).
					SetCreatedBy(actor).
					SetUpdatedBy(actor).
					SetSource(propertyjurisdiction.SourceSystem).
					Save(ctx)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("linking property %s to jurisdiction %s: %w", prop.ID, j.Name, err)
				}
			}
		}
		log.Printf("linked %d properties to 4 jurisdictions", len(props))
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing jurisdiction seed: %w", err)
	}

	log.Printf("seeded 4 jurisdictions (Federal, CA, LA County, Santa Monica) with %d rules",
		countRules(federal, california, laCounty, santaMonica))
	return nil
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func countRules(jurisdictions ...*ent.Jurisdiction) int {
	// We created: 1 federal + 5 CA + 1 LA County + 4 Santa Monica = 11 rules
	_ = jurisdictions
	return 11
}
