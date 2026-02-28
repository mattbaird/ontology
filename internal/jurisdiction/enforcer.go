// Package jurisdiction provides runtime enforcement of jurisdiction rules.
// Rules are loaded from the database via PropertyJurisdiction → Jurisdiction → JurisdictionRule
// and validated against lease mutations.
package jurisdiction

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/ent/jurisdictionrule"
	"github.com/matthewbaird/ontology/ent/property"
	"github.com/matthewbaird/ontology/ent/propertyjurisdiction"
)

// SecurityDepositLimitDef is the JSON shape of a security_deposit_limit rule definition.
type SecurityDepositLimitDef struct {
	MaxMonths float64 `json:"max_months"` // e.g., 1.0 for CA
}

// RentIncreaseCapDef is the JSON shape of a rent_increase_cap rule definition.
type RentIncreaseCapDef struct {
	MaxPercentIncrease float64 `json:"max_percent_increase"` // e.g., 5.0
	BasedOn            string  `json:"based_on,omitempty"`   // "cpi_plus" or "fixed"
	CPIPlusPercent     float64 `json:"cpi_plus_percent,omitempty"`
}

// NoticePeriodDef is the JSON shape of a notice_period rule definition.
type NoticePeriodDef struct {
	MinDays       int    `json:"min_days"`
	TenancyOver   int    `json:"tenancy_over_months,omitempty"`   // 0 means any tenancy
	MinDaysExtend int    `json:"min_days_extended,omitempty"`     // longer notice for longer tenancy
	Condition     string `json:"condition,omitempty"`             // e.g., "tenancy > 12 months"
}

// RequiredDisclosureDef is the JSON shape of a required_disclosure rule definition.
type RequiredDisclosureDef struct {
	DisclosureType string `json:"disclosure_type"` // e.g., "lead_paint", "mold", "bed_bug"
	RequiredBefore string `json:"required_before"`  // "lease_signing", "move_in"
}

// LateFeeCapDef is the JSON shape of a late_fee_cap rule definition.
type LateFeeCapDef struct {
	MaxPercentOfRent float64 `json:"max_percent_of_rent,omitempty"`
	MaxFixedAmount   int64   `json:"max_fixed_amount_cents,omitempty"`
	GracePeriodDays  int     `json:"grace_period_days,omitempty"`
}

// Violation represents a jurisdiction rule violation found during enforcement.
type Violation struct {
	RuleType        string `json:"rule_type"`
	Jurisdiction    string `json:"jurisdiction"`
	StatuteRef      string `json:"statute_reference,omitempty"`
	Description     string `json:"description"`
}

func (v Violation) Error() string {
	if v.StatuteRef != "" {
		return fmt.Sprintf("jurisdiction violation (%s, %s): %s [%s]", v.Jurisdiction, v.RuleType, v.Description, v.StatuteRef)
	}
	return fmt.Sprintf("jurisdiction violation (%s, %s): %s", v.Jurisdiction, v.RuleType, v.Description)
}

// LeaseHook returns an Ent hook that enforces jurisdiction rules on lease mutations.
// The hook queries PropertyJurisdiction → Jurisdiction → JurisdictionRule for the
// lease's property_id and validates the mutation fields against active rules.
func LeaseHook(client *ent.Client) ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			// Only enforce on create and update.
			if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate|ent.OpUpdateOne) {
				return next.Mutate(ctx, m)
			}

			propertyID, ok := getStringField(ctx, m, "property_id")
			if !ok || propertyID == "" {
				return next.Mutate(ctx, m)
			}

			propUUID, err := uuid.Parse(propertyID)
			if err != nil {
				return next.Mutate(ctx, m)
			}

			leaseType, _ := getStringField(ctx, m, "lease_type")

			rules, err := loadActiveRules(ctx, client, propUUID, leaseType)
			if err != nil {
				log.Printf("jurisdiction: failed to load rules for property %s: %v", propertyID, err)
				return next.Mutate(ctx, m)
			}
			if len(rules) == 0 {
				return next.Mutate(ctx, m)
			}

			// Check security deposit limits.
			if depositCents, ok := getInt64Field(ctx, m, "security_deposit_amount_cents"); ok {
				if rentCents, ok := getInt64Field(ctx, m, "base_rent_amount_cents"); ok && rentCents > 0 {
					if err := checkSecurityDepositLimit(rules, depositCents, rentCents); err != nil {
						return nil, err
					}
				}
			}

			// Check notice period requirements.
			if noticeDays, ok := getIntField(ctx, m, "notice_required_days"); ok {
				if err := checkNoticePeriod(rules, noticeDays); err != nil {
					return nil, err
				}
			}

			return next.Mutate(ctx, m)
		})
	}
}

// loadActiveRules queries all active jurisdiction rules that apply to a property.
func loadActiveRules(ctx context.Context, client *ent.Client, propertyID uuid.UUID, leaseType string) ([]*ent.JurisdictionRule, error) {
	now := time.Now()

	// Find all jurisdictions linked to this property.
	pjs, err := client.PropertyJurisdiction.Query().
		Where(
			propertyjurisdiction.HasPropertyWith(property.IDEQ(propertyID)),
		).
		WithJurisdiction(func(jq *ent.JurisdictionQuery) {
			jq.WithRules(func(rq *ent.JurisdictionRuleQuery) {
				rq.Where(
					jurisdictionrule.StatusEQ(jurisdictionrule.StatusActive),
					jurisdictionrule.EffectiveDateLTE(now),
				)
			})
		}).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying property jurisdictions: %w", err)
	}

	var rules []*ent.JurisdictionRule
	for _, pj := range pjs {
		j := pj.Edges.Jurisdiction
		if j == nil {
			continue
		}
		for _, rule := range j.Edges.Rules {
			// Skip expired rules.
			if rule.ExpirationDate != nil && rule.ExpirationDate.Before(now) {
				continue
			}
			// If rule specifies lease types, check if our lease type matches.
			if leaseType != "" && len(rule.AppliesToLeaseTypes) > 0 {
				if !contains(rule.AppliesToLeaseTypes, leaseType) {
					continue
				}
			}
			// Set back-reference so violation messages include jurisdiction name.
			rule.Edges.Jurisdiction = j
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

// checkSecurityDepositLimit validates that the security deposit doesn't exceed
// jurisdiction-mandated limits (typically expressed as max months of rent).
func checkSecurityDepositLimit(rules []*ent.JurisdictionRule, depositCents, rentCents int64) error {
	for _, rule := range rules {
		if rule.RuleType != jurisdictionrule.RuleTypeSecurityDepositLimit {
			continue
		}
		var def SecurityDepositLimitDef
		if err := json.Unmarshal(rule.RuleDefinition, &def); err != nil {
			continue
		}
		if def.MaxMonths <= 0 {
			continue
		}
		maxDeposit := int64(float64(rentCents) * def.MaxMonths)
		if depositCents > maxDeposit {
			statute := ""
			if rule.StatuteReference != nil {
				statute = *rule.StatuteReference
			}
			jurisdictionName := "unknown"
			if j := rule.Edges.Jurisdiction; j != nil {
				jurisdictionName = j.Name
			}
			return Violation{
				RuleType:     "security_deposit_limit",
				Jurisdiction: jurisdictionName,
				StatuteRef:   statute,
				Description:  fmt.Sprintf("security deposit of %d cents exceeds maximum of %.1f months rent (%d cents)", depositCents, def.MaxMonths, maxDeposit),
			}
		}
	}
	return nil
}

// checkNoticePeriod validates that notice_required_days meets jurisdiction minimums.
func checkNoticePeriod(rules []*ent.JurisdictionRule, noticeDays int) error {
	for _, rule := range rules {
		if rule.RuleType != jurisdictionrule.RuleTypeNoticePeriod {
			continue
		}
		var def NoticePeriodDef
		if err := json.Unmarshal(rule.RuleDefinition, &def); err != nil {
			continue
		}
		if def.MinDays <= 0 {
			continue
		}
		if noticeDays < def.MinDays {
			statute := ""
			if rule.StatuteReference != nil {
				statute = *rule.StatuteReference
			}
			jurisdictionName := "unknown"
			if j := rule.Edges.Jurisdiction; j != nil {
				jurisdictionName = j.Name
			}
			return Violation{
				RuleType:     "notice_period",
				Jurisdiction: jurisdictionName,
				StatuteRef:   statute,
				Description:  fmt.Sprintf("notice period of %d days is below minimum of %d days", noticeDays, def.MinDays),
			}
		}
	}
	return nil
}

// ValidateRentIncrease checks if a rent increase from oldRent to newRent complies
// with jurisdiction-mandated caps. Called from RenewLease command handler.
func ValidateRentIncrease(ctx context.Context, client *ent.Client, propertyID uuid.UUID, leaseType string, oldRentCents, newRentCents int64) error {
	rules, err := loadActiveRules(ctx, client, propertyID, leaseType)
	if err != nil {
		log.Printf("jurisdiction: failed to load rules for rent increase check: %v", err)
		return nil
	}

	for _, rule := range rules {
		if rule.RuleType != jurisdictionrule.RuleTypeRentIncreaseCap {
			continue
		}
		var def RentIncreaseCapDef
		if err := json.Unmarshal(rule.RuleDefinition, &def); err != nil {
			continue
		}
		if def.MaxPercentIncrease <= 0 || oldRentCents <= 0 {
			continue
		}
		actualPct := float64(newRentCents-oldRentCents) / float64(oldRentCents) * 100
		if actualPct > def.MaxPercentIncrease {
			statute := ""
			if rule.StatuteReference != nil {
				statute = *rule.StatuteReference
			}
			jurisdictionName := "unknown"
			if j := rule.Edges.Jurisdiction; j != nil {
				jurisdictionName = j.Name
			}
			return Violation{
				RuleType:     "rent_increase_cap",
				Jurisdiction: jurisdictionName,
				StatuteRef:   statute,
				Description:  fmt.Sprintf("rent increase of %.1f%% exceeds cap of %.1f%%", actualPct, def.MaxPercentIncrease),
			}
		}
	}
	return nil
}

// GetRequiredDisclosures returns all required disclosures for a property's jurisdictions.
// Called by command handlers to include disclosure requirements in responses.
func GetRequiredDisclosures(ctx context.Context, client *ent.Client, propertyID uuid.UUID, leaseType string) ([]RequiredDisclosureDef, error) {
	rules, err := loadActiveRules(ctx, client, propertyID, leaseType)
	if err != nil {
		return nil, err
	}

	var disclosures []RequiredDisclosureDef
	for _, rule := range rules {
		if rule.RuleType != jurisdictionrule.RuleTypeRequiredDisclosure {
			continue
		}
		var def RequiredDisclosureDef
		if err := json.Unmarshal(rule.RuleDefinition, &def); err != nil {
			continue
		}
		disclosures = append(disclosures, def)
	}
	return disclosures, nil
}

// EvictionContext holds jurisdiction-derived eviction requirements.
type EvictionContext struct {
	JustCauseRequired  bool `json:"just_cause_required"`
	CurePeriodDays     int  `json:"cure_period_days"`
	RelocationRequired bool `json:"relocation_required"`
	RightToCounsel     bool `json:"right_to_counsel"`
}

// GetEvictionContext queries jurisdiction rules for eviction-related requirements
// applicable to a property. Returns zero-value context if no rules are found.
func GetEvictionContext(ctx context.Context, client *ent.Client, propertyID uuid.UUID, leaseType string) EvictionContext {
	rules, err := loadActiveRules(ctx, client, propertyID, leaseType)
	if err != nil {
		log.Printf("jurisdiction: failed to load eviction rules: %v", err)
		return EvictionContext{}
	}

	var ec EvictionContext
	for _, rule := range rules {
		switch rule.RuleType {
		case jurisdictionrule.RuleTypeJustCauseEviction:
			ec.JustCauseRequired = true
			var def map[string]any
			if json.Unmarshal(rule.RuleDefinition, &def) == nil {
				if reloc, ok := def["relocation_required"].(bool); ok && reloc {
					ec.RelocationRequired = true
				}
				if rtc, ok := def["right_to_counsel"].(bool); ok && rtc {
					ec.RightToCounsel = true
				}
			}
		case jurisdictionrule.RuleTypeRelocationAssistance:
			ec.RelocationRequired = true
		case jurisdictionrule.RuleTypeRightToCounsel:
			ec.RightToCounsel = true
		case jurisdictionrule.RuleTypeEvictionProcedure:
			var def map[string]any
			if json.Unmarshal(rule.RuleDefinition, &def) == nil {
				if days, ok := def["cure_period_days"].(float64); ok && int(days) > ec.CurePeriodDays {
					ec.CurePeriodDays = int(days)
				}
			}
		}
	}
	return ec
}

// helper to get a string field from mutation (current or old value).
func getStringField(ctx context.Context, m ent.Mutation, name string) (string, bool) {
	if v, ok := m.Field(name); ok {
		if s, isStr := v.(string); isStr {
			return s, true
		}
	}
	if v, err := m.OldField(ctx, name); err == nil {
		if s, isStr := v.(string); isStr {
			return s, true
		}
	}
	return "", false
}

func getInt64Field(ctx context.Context, m ent.Mutation, name string) (int64, bool) {
	if v, ok := m.Field(name); ok {
		if n, isInt := v.(int64); isInt {
			return n, true
		}
	}
	if v, err := m.OldField(ctx, name); err == nil {
		if n, isInt := v.(int64); isInt {
			return n, true
		}
	}
	return 0, false
}

func getIntField(ctx context.Context, m ent.Mutation, name string) (int, bool) {
	if v, ok := m.Field(name); ok {
		if n, isInt := v.(int); isInt {
			return n, true
		}
	}
	if v, err := m.OldField(ctx, name); err == nil {
		if n, isInt := v.(int); isInt {
			return n, true
		}
	}
	return 0, false
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
