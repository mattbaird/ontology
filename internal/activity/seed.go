// Seed populates an activity store with demo data for signal discovery demos.
package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/matthewbaird/ontology/internal/types"
)

// SeedDemoData populates the store with realistic activity data for 6 demo tenants.
func SeedDemoData(ctx context.Context, store Store) error {
	var entries []types.ActivityEntry

	// ─── Marcus Johnson: flight risk scenario ───
	marcus := "marcus-johnson"
	marcusLease := "lease-mj-101"
	space101 := "space-101"
	property := "sunset-apartments"

	marcusRefs := func(role string) []types.SourceRef {
		return []types.SourceRef{
			{EntityType: "person", EntityID: marcus, Role: role},
			{EntityType: "lease", EntityID: marcusLease, Role: "related"},
			{EntityType: "space", EntityID: space101, Role: "context"},
			{EntityType: "property", EntityID: property, Role: "context"},
		}
	}

	// On-time payments (positive baseline)
	for _, month := range []string{"2025-09", "2025-10", "2025-11", "2025-12", "2026-01"} {
		entries = append(entries, makeEntry(
			"pay-mj-"+month, "PaymentRecorded",
			mustTime(month+"-01T10:00:00Z"),
			"person", marcus, "subject", marcusRefs("subject"),
			"Payment received on time", "financial", "info", "positive",
			map[string]any{"amount_cents": 185000, "currency": "USD", "days_past_due": 0},
		))
		entries = append(entries, makeEntry(
			"pay-mj-"+month, "PaymentRecorded",
			mustTime(month+"-01T10:00:00Z"),
			"lease", marcusLease, "related", marcusRefs("subject"),
			"Payment received on time", "financial", "info", "positive",
			map[string]any{"amount_cents": 185000, "currency": "USD", "days_past_due": 0},
		))
	}

	// Noise complaints (3 in 5 months — triggers escalation)
	complaints := []struct {
		id   string
		date string
		desc string
	}{
		{"comp-mj-1", "2025-10-14T16:45:00Z", "Noise complaint filed: loud music from Space 101, reported by neighbor Space 102"},
		{"comp-mj-2", "2025-11-22T11:10:00Z", "Noise complaint filed: late-night disturbance from Space 101, reported by neighbor Space 103"},
		{"comp-mj-3", "2026-01-18T14:30:00Z", "Noise complaint filed: party noise from Space 101, multiple neighbors reported"},
	}
	for _, c := range complaints {
		entries = append(entries, makeEntry(
			c.id, "ComplaintCreated", mustTime(c.date),
			"person", marcus, "subject", marcusRefs("subject"),
			c.desc, "maintenance", "moderate", "negative",
			map[string]any{"complaint_type": "noise", "space_id": space101},
		))
		entries = append(entries, makeEntry(
			c.id, "ComplaintCreated", mustTime(c.date),
			"space", space101, "target", marcusRefs("subject"),
			c.desc, "maintenance", "moderate", "negative",
			map[string]any{"complaint_type": "noise", "space_id": space101},
		))
	}

	// Outreach with no response (2 attempts)
	entries = append(entries, makeEntry(
		"outreach-mj-1", "OutreachAttempted", mustTime("2025-12-03T08:00:00Z"),
		"person", marcus, "subject", marcusRefs("subject"),
		"Outreach attempt with no response (email re: lease renewal)", "communication", "moderate", "negative",
		map[string]any{"channel": "email", "topic": "lease_renewal", "responded": false},
	))
	entries = append(entries, makeEntry(
		"outreach-mj-2", "OutreachAttempted", mustTime("2026-01-05T09:00:00Z"),
		"person", marcus, "subject", marcusRefs("subject"),
		"Outreach attempt with no response (phone call re: noise concerns)", "communication", "moderate", "negative",
		map[string]any{"channel": "phone", "topic": "noise_concerns", "responded": false},
	))

	// Roommate departure
	entries = append(entries, makeEntry(
		"roommate-mj-1", "RoommateDeparted", mustTime("2025-12-10T15:20:00Z"),
		"person", marcus, "subject", marcusRefs("subject"),
		"Roommate departed the lease (Kevin Park moved out)", "relationship", "strong", "negative",
		map[string]any{"departed_person": "kevin-park", "remaining_occupants": 1},
	))
	entries = append(entries, makeEntry(
		"roommate-mj-1", "RoommateDeparted", mustTime("2025-12-10T15:20:00Z"),
		"lease", marcusLease, "related", marcusRefs("subject"),
		"Roommate departed the lease (Kevin Park moved out)", "relationship", "strong", "negative",
		map[string]any{"departed_person": "kevin-park", "remaining_occupants": 1},
	))

	// Portal activity drop
	entries = append(entries, makeEntry(
		"portal-mj-1", "PortalActivityChanged", mustTime("2026-01-22T00:00:00Z"),
		"person", marcus, "subject", marcusRefs("subject"),
		"Portal login frequency decreased (last login 18 days ago, was weekly)", "communication", "weak", "negative",
		map[string]any{"previous_frequency": "weekly", "current_gap_days": 18},
	))

	// Lease expiring within 90 days
	entries = append(entries, makeEntry(
		"lifecycle-mj-1", "LeaseExpirationApproaching", mustTime("2026-01-25T00:00:00Z"),
		"person", marcus, "subject", marcusRefs("subject"),
		"Lease expiring within 90 days (expires 2026-04-30)", "lifecycle", "moderate", "neutral",
		map[string]any{"expires_at": "2026-04-30", "days_remaining": 95},
	))
	entries = append(entries, makeEntry(
		"lifecycle-mj-1", "LeaseExpirationApproaching", mustTime("2026-01-25T00:00:00Z"),
		"lease", marcusLease, "subject", marcusRefs("subject"),
		"Lease expiring within 90 days (expires 2026-04-30)", "lifecycle", "moderate", "neutral",
		map[string]any{"expires_at": "2026-04-30", "days_remaining": 95},
	))

	// ─── Jennifer Park: late payment pattern ───
	jennifer := "jennifer-park"
	jenniferRefs := []types.SourceRef{
		{EntityType: "person", EntityID: jennifer, Role: "subject"},
		{EntityType: "lease", EntityID: "lease-jp-204", Role: "related"},
		{EntityType: "property", EntityID: property, Role: "context"},
	}

	entries = append(entries, makeEntry(
		"pay-jp-nov", "PaymentRecorded", mustTime("2025-11-08T14:00:00Z"),
		"person", jennifer, "subject", jenniferRefs,
		"Payment received late (8 days past due)", "financial", "moderate", "negative",
		map[string]any{"amount_cents": 142000, "days_past_due": 8},
	))
	entries = append(entries, makeEntry(
		"pay-jp-dec", "PaymentRecorded", mustTime("2025-12-12T16:30:00Z"),
		"person", jennifer, "subject", jenniferRefs,
		"Payment received late (12 days past due)", "financial", "moderate", "negative",
		map[string]any{"amount_cents": 142000, "days_past_due": 12},
	))
	entries = append(entries, makeEntry(
		"pay-jp-jan", "PaymentRecorded", mustTime("2026-01-06T09:00:00Z"),
		"person", jennifer, "subject", jenniferRefs,
		"Payment received late (6 days past due)", "financial", "moderate", "negative",
		map[string]any{"amount_cents": 142000, "days_past_due": 6},
	))
	entries = append(entries, makeEntry(
		"fee-jp-dec", "LateFeeAssessed", mustTime("2025-12-15T00:00:00Z"),
		"person", jennifer, "subject", jenniferRefs,
		"Late fee assessed on account ($75)", "financial", "moderate", "negative",
		map[string]any{"fee_amount_cents": 7500},
	))

	// ─── David Kim: lease violation ───
	david := "david-kim"
	davidRefs := []types.SourceRef{
		{EntityType: "person", EntityID: david, Role: "subject"},
		{EntityType: "lease", EntityID: "lease-dk-305", Role: "related"},
		{EntityType: "property", EntityID: property, Role: "context"},
	}

	entries = append(entries, makeEntry(
		"viol-dk-1", "LeaseViolationRecorded", mustTime("2026-01-10T11:00:00Z"),
		"person", david, "subject", davidRefs,
		"Lease violation recorded: unauthorized pet (dog, ~40 lbs)", "compliance", "strong", "negative",
		map[string]any{"violation_type": "unauthorized_pet", "description": "Large dog observed by maintenance tech"},
	))
	entries = append(entries, makeEntry(
		"pay-dk-jan", "PaymentRecorded", mustTime("2026-01-01T09:00:00Z"),
		"person", david, "subject", davidRefs,
		"Payment received on time", "financial", "info", "positive",
		map[string]any{"amount_cents": 165000, "days_past_due": 0},
	))

	// ─── James Wright: model tenant ───
	james := "james-wright"
	jamesRefs := []types.SourceRef{
		{EntityType: "person", EntityID: james, Role: "subject"},
		{EntityType: "lease", EntityID: "lease-jw-410", Role: "related"},
		{EntityType: "property", EntityID: property, Role: "context"},
	}

	for _, month := range []string{"2025-09", "2025-10", "2025-11", "2025-12", "2026-01"} {
		entries = append(entries, makeEntry(
			"pay-jw-"+month, "PaymentRecorded", mustTime(month+"-01T08:00:00Z"),
			"person", james, "subject", jamesRefs,
			"Payment received on time", "financial", "info", "positive",
			map[string]any{"amount_cents": 195000, "days_past_due": 0},
		))
	}
	entries = append(entries, makeEntry(
		"contact-jw-1", "TenantContactReceived", mustTime("2026-01-15T10:00:00Z"),
		"person", james, "subject", jamesRefs,
		"Tenant initiated contact (asked about lease renewal options)", "communication", "info", "positive",
		map[string]any{"channel": "portal", "topic": "renewal_inquiry"},
	))

	// ─── Amy Torres: recently renewed ───
	amy := "amy-torres"
	amyRefs := []types.SourceRef{
		{EntityType: "person", EntityID: amy, Role: "subject"},
		{EntityType: "lease", EntityID: "lease-at-502", Role: "related"},
		{EntityType: "property", EntityID: property, Role: "context"},
	}

	entries = append(entries, makeEntry(
		"renewal-at-1", "RenewalSigned", mustTime("2026-01-20T14:00:00Z"),
		"person", amy, "subject", amyRefs,
		"Lease renewal signed (12-month term, 2% increase)", "lifecycle", "strong", "positive",
		map[string]any{"term_months": 12, "rent_change_pct": 2.0},
	))
	for _, month := range []string{"2025-11", "2025-12", "2026-01"} {
		entries = append(entries, makeEntry(
			"pay-at-"+month, "PaymentRecorded", mustTime(month+"-01T09:00:00Z"),
			"person", amy, "subject", amyRefs,
			"Payment received on time", "financial", "info", "positive",
			map[string]any{"amount_cents": 155000, "days_past_due": 0},
		))
	}

	// ─── Lisa Hernandez: guarantor removed ───
	lisa := "lisa-hernandez"
	lisaRefs := []types.SourceRef{
		{EntityType: "person", EntityID: lisa, Role: "subject"},
		{EntityType: "lease", EntityID: "lease-lh-108", Role: "related"},
		{EntityType: "property", EntityID: property, Role: "context"},
	}

	entries = append(entries, makeEntry(
		"guarantor-lh-1", "GuarantorChanged", mustTime("2026-01-08T16:00:00Z"),
		"person", lisa, "subject", lisaRefs,
		"Guarantor removed from lease (parent co-signer removed)", "relationship", "moderate", "contextual",
		map[string]any{"change_type": "removed", "previous_guarantor": "Maria Hernandez"},
	))
	entries = append(entries, makeEntry(
		"pay-lh-jan", "PaymentRecorded", mustTime("2026-01-01T10:00:00Z"),
		"person", lisa, "subject", lisaRefs,
		"Payment received on time", "financial", "info", "positive",
		map[string]any{"amount_cents": 175000, "days_past_due": 0},
	))
	entries = append(entries, makeEntry(
		"lifecycle-lh-1", "LeaseExpirationApproaching", mustTime("2026-01-28T00:00:00Z"),
		"person", lisa, "subject", lisaRefs,
		"Lease expiring within 90 days (expires 2026-04-15)", "lifecycle", "moderate", "neutral",
		map[string]any{"expires_at": "2026-04-15", "days_remaining": 77},
	))

	if err := store.WriteEntries(ctx, entries); err != nil {
		return fmt.Errorf("seeding activity store: %w", err)
	}
	log.Printf("seeded %d activity entries for 6 demo tenants", len(entries))
	return nil
}

func makeEntry(
	eventID, eventType string,
	occurredAt time.Time,
	indexedEntityType, indexedEntityID, entityRole string,
	sourceRefs []types.SourceRef,
	summary, category, weight, polarity string,
	payload map[string]any,
) types.ActivityEntry {
	p, _ := json.Marshal(payload)
	return types.ActivityEntry{
		EventID:           eventID,
		EventType:         eventType,
		OccurredAt:        occurredAt,
		IndexedEntityType: indexedEntityType,
		IndexedEntityID:   indexedEntityID,
		EntityRole:        entityRole,
		SourceRefs:        sourceRefs,
		Summary:           summary,
		Category:          category,
		Weight:            weight,
		Polarity:          polarity,
		Payload:           p,
	}
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		log.Fatalf("invalid time literal %q: %v", s, err)
	}
	return t
}
