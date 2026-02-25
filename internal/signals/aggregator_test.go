package signals

import (
	"testing"
	"time"

	"github.com/matthewbaird/ontology/internal/types"
)

func makeActivity(eventType, category, weight, polarity string, daysAgo int) types.ActivityEntry {
	return types.ActivityEntry{
		EventID:           "test-" + category,
		EventType:         eventType,
		OccurredAt:        time.Now().AddDate(0, 0, -daysAgo),
		IndexedEntityType: "person",
		IndexedEntityID:   "test-person",
		EntityRole:        "subject",
		Summary:           "test entry",
		Category:          category,
		Weight:            weight,
		Polarity:          polarity,
	}
}

func TestAggregate_CategoryCounts(t *testing.T) {
	entries := []types.ActivityEntry{
		makeActivity("PaymentRecorded", "financial", "info", "positive", 10),
		makeActivity("PaymentRecorded", "financial", "info", "positive", 20),
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 5),
	}

	since := time.Now().AddDate(0, -6, 0)
	until := time.Now()
	summary := Aggregate(entries, "person", "test-person", since, until)

	if len(summary.Categories) != 2 {
		t.Errorf("got %d categories, want 2", len(summary.Categories))
	}
	if summary.Categories["financial"].SignalCount != 2 {
		t.Errorf("financial count = %d, want 2", summary.Categories["financial"].SignalCount)
	}
	if summary.Categories["maintenance"].SignalCount != 1 {
		t.Errorf("maintenance count = %d, want 1", summary.Categories["maintenance"].SignalCount)
	}
}

func TestAggregate_DominantPolarity(t *testing.T) {
	entries := []types.ActivityEntry{
		makeActivity("PaymentRecorded", "financial", "info", "positive", 10),
		makeActivity("PaymentRecorded", "financial", "info", "positive", 20),
		makeActivity("PaymentRecorded", "financial", "moderate", "negative", 15),
	}

	since := time.Now().AddDate(0, -6, 0)
	until := time.Now()
	summary := Aggregate(entries, "person", "test-person", since, until)

	if summary.Categories["financial"].DominantPolarity != "positive" {
		t.Errorf("dominant polarity = %q, want positive", summary.Categories["financial"].DominantPolarity)
	}
}

func TestAggregate_PositiveSentiment(t *testing.T) {
	entries := []types.ActivityEntry{
		makeActivity("PaymentRecorded", "financial", "info", "positive", 10),
		makeActivity("PaymentRecorded", "financial", "info", "positive", 20),
		makeActivity("PaymentRecorded", "financial", "info", "positive", 30),
	}

	since := time.Now().AddDate(0, -6, 0)
	until := time.Now()
	summary := Aggregate(entries, "person", "test-person", since, until)

	if summary.OverallSentiment != "positive" {
		t.Errorf("sentiment = %q, want positive", summary.OverallSentiment)
	}
}

func TestAggregate_ConcerningSentiment(t *testing.T) {
	entries := []types.ActivityEntry{
		makeActivity("ComplaintCreated", "maintenance", "strong", "negative", 5),
		makeActivity("ComplaintCreated", "maintenance", "strong", "negative", 10),
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 15),
	}

	since := time.Now().AddDate(0, -6, 0)
	until := time.Now()
	summary := Aggregate(entries, "person", "test-person", since, until)

	if summary.OverallSentiment != "concerning" {
		t.Errorf("sentiment = %q, want concerning", summary.OverallSentiment)
	}
}

func TestComputeTrend_Declining(t *testing.T) {
	since := time.Now().AddDate(0, -6, 0)
	until := time.Now()
	entries := []types.ActivityEntry{
		// 1 in first half, 3 in second half → declining
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 150),
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 30),
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 20),
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 10),
	}

	trend := computeTrend(entries, "maintenance", since, until)
	if trend != "declining" {
		t.Errorf("trend = %q, want declining", trend)
	}
}

func TestComputeTrend_Stable(t *testing.T) {
	since := time.Now().AddDate(0, -6, 0)
	until := time.Now()
	entries := []types.ActivityEntry{
		makeActivity("PaymentRecorded", "financial", "info", "positive", 150),
		makeActivity("PaymentRecorded", "financial", "info", "positive", 30),
	}

	trend := computeTrend(entries, "financial", since, until)
	if trend != "stable" {
		t.Errorf("trend = %q, want stable", trend)
	}
}

func TestEvaluateEscalations_CountRule(t *testing.T) {
	// 3 complaints within 180 days should trigger maint_complaint_pattern.
	entries := []types.ActivityEntry{
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 10),
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 30),
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 60),
	}

	escalations := EvaluateEscalations(entries)

	found := false
	for _, e := range escalations {
		if e.Rule.ID == "maint_complaint_pattern" {
			found = true
			if e.TriggeringCount < 3 {
				t.Errorf("triggering count = %d, want >= 3", e.TriggeringCount)
			}
		}
	}
	if !found {
		t.Error("expected maint_complaint_pattern escalation to fire")
	}
}

func TestEvaluateEscalations_NotEnoughForTrigger(t *testing.T) {
	// Only 1 complaint — should not trigger count-based escalation.
	entries := []types.ActivityEntry{
		makeActivity("ComplaintCreated", "maintenance", "moderate", "negative", 10),
	}

	escalations := EvaluateEscalations(entries)

	for _, e := range escalations {
		if e.Rule.ID == "maint_complaint_pattern" {
			t.Error("expected maint_complaint_pattern to NOT fire with only 1 complaint")
		}
	}
}

func TestWeightSeverity(t *testing.T) {
	if WeightSeverity("critical") != 1 {
		t.Error("critical should be 1")
	}
	if WeightSeverity("info") != 5 {
		t.Error("info should be 5")
	}
	if WeightSeverity("unknown") != 6 {
		t.Error("unknown should be 6")
	}
}

func TestIsAtLeastWeight(t *testing.T) {
	if !IsAtLeastWeight("critical", "strong") {
		t.Error("critical should be at least strong")
	}
	if !IsAtLeastWeight("strong", "strong") {
		t.Error("strong should be at least strong")
	}
	if IsAtLeastWeight("info", "strong") {
		t.Error("info should NOT be at least strong")
	}
}
