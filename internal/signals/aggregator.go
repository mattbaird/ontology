package signals

import (
	"sort"
	"time"

	"github.com/matthewbaird/ontology/internal/types"
)

// Aggregate produces a SignalSummary from a set of activity entries within a time window.
func Aggregate(entries []types.ActivityEntry, entityType, entityID string, since, until time.Time) types.SignalSummary {
	categories := make(map[string]*types.CategorySummary)

	for _, entry := range entries {
		cat := entry.Category
		cs, exists := categories[cat]
		if !exists {
			cs = &types.CategorySummary{
				Category:   cat,
				ByWeight:   make(map[string]int),
				ByPolarity: make(map[string]int),
			}
			categories[cat] = cs
		}
		cs.SignalCount++
		cs.ByWeight[entry.Weight]++
		cs.ByPolarity[entry.Polarity]++
	}

	// Compute dominant polarity and trend per category.
	result := make(map[string]types.CategorySummary, len(categories))
	for cat, cs := range categories {
		cs.DominantPolarity = dominantPolarity(cs.ByPolarity)
		cs.Trend = computeTrend(entries, cat, since, until)
		result[cat] = *cs
	}

	// Evaluate escalations.
	escalations := EvaluateEscalations(entries)

	// Compute overall sentiment.
	sentiment, reason := computeSentiment(result, escalations)

	return types.SignalSummary{
		EntityType:       entityType,
		EntityID:         entityID,
		Since:            since,
		Until:            until,
		Categories:       result,
		OverallSentiment: sentiment,
		SentimentReason:  reason,
		Escalations:      escalations,
	}
}

// EvaluateEscalations checks all count-based and cross-category escalation rules
// against the provided activity entries.
func EvaluateEscalations(entries []types.ActivityEntry) []types.EscalatedSignal {
	var escalated []types.EscalatedSignal

	// Collect all escalation rules from registrations.
	// Per-registration rules are scoped to entries matching the parent's EventType.
	for _, reg := range SignalRegistry {
		for _, rule := range reg.EscalationRules {
			if es, ok := evaluateRule(rule, entries, reg.EventType); ok {
				escalated = append(escalated, es)
			}
		}
	}

	// Check cross-category escalation rules (no event-type scoping).
	for _, rule := range GetCrossCategoryEscalations() {
		if es, ok := evaluateRule(rule, entries, ""); ok {
			escalated = append(escalated, es)
		}
	}

	return escalated
}

func evaluateRule(rule types.EscalationRule, entries []types.ActivityEntry, scopeEventType string) (types.EscalatedSignal, bool) {
	switch rule.TriggerType {
	case "count":
		return evaluateCountRule(rule, entries, scopeEventType)
	case "cross_category":
		return evaluateCrossCategoryRule(rule, entries)
	default:
		// Absence and trend rules require external state; not evaluated here.
		return types.EscalatedSignal{}, false
	}
}

func evaluateCountRule(rule types.EscalationRule, entries []types.ActivityEntry, scopeEventType string) (types.EscalatedSignal, bool) {
	now := time.Now()
	windowStart := now.AddDate(0, 0, -rule.WithinDays)

	var matching []types.ActivityEntry
	for _, e := range entries {
		if e.OccurredAt.Before(windowStart) {
			continue
		}
		// Scope to parent registration's event type when set.
		if scopeEventType != "" && e.EventType != scopeEventType {
			continue
		}
		if rule.SignalCategory != "" && e.Category != rule.SignalCategory {
			continue
		}
		if rule.SignalPolarity != "" && e.Polarity != rule.SignalPolarity {
			continue
		}
		matching = append(matching, e)
	}

	if len(matching) < rule.Count {
		return types.EscalatedSignal{}, false
	}

	// Sort by time to get earliest/latest.
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].OccurredAt.Before(matching[j].OccurredAt)
	})

	return types.EscalatedSignal{
		Rule:             rule,
		TriggeringCount:  len(matching),
		EarliestOccurred: matching[0].OccurredAt,
		LatestOccurred:   matching[len(matching)-1].OccurredAt,
	}, true
}

func evaluateCrossCategoryRule(rule types.EscalationRule, entries []types.ActivityEntry) (types.EscalatedSignal, bool) {
	now := time.Now()
	windowStart := now.AddDate(0, 0, -rule.WithinDays)

	// Count entries per required category within the time window.
	counts := make(map[string]int)
	var earliest, latest time.Time

	for _, e := range entries {
		if e.OccurredAt.Before(windowStart) {
			continue
		}
		for _, req := range rule.RequiredCategories {
			if e.Category == req.Category {
				if req.Polarity == "" || e.Polarity == req.Polarity {
					counts[req.Category]++
					if earliest.IsZero() || e.OccurredAt.Before(earliest) {
						earliest = e.OccurredAt
					}
					if e.OccurredAt.After(latest) {
						latest = e.OccurredAt
					}
				}
			}
		}
	}

	// Check all requirements are met.
	totalMatching := 0
	for _, req := range rule.RequiredCategories {
		if counts[req.Category] < req.MinCount {
			return types.EscalatedSignal{}, false
		}
		totalMatching += counts[req.Category]
	}

	return types.EscalatedSignal{
		Rule:             rule,
		TriggeringCount:  totalMatching,
		EarliestOccurred: earliest,
		LatestOccurred:   latest,
	}, true
}

// dominantPolarity returns the polarity with the highest count.
func dominantPolarity(byPolarity map[string]int) string {
	best := ""
	bestCount := 0
	for p, c := range byPolarity {
		if c > bestCount {
			best = p
			bestCount = c
		}
	}
	return best
}

// computeTrend compares signal volume in the first vs second half of the time window.
func computeTrend(entries []types.ActivityEntry, category string, since, until time.Time) string {
	mid := since.Add(until.Sub(since) / 2)
	var firstHalf, secondHalf int
	for _, e := range entries {
		if e.Category != category {
			continue
		}
		if e.OccurredAt.Before(mid) {
			firstHalf++
		} else {
			secondHalf++
		}
	}

	// For negative-polarity categories, more events = declining.
	// For simplicity, compare raw counts.
	if secondHalf > firstHalf+1 {
		return "declining"
	}
	if firstHalf > secondHalf+1 {
		return "improving"
	}
	return "stable"
}

// computeSentiment determines overall sentiment from category summaries and escalations.
func computeSentiment(categories map[string]types.CategorySummary, escalations []types.EscalatedSignal) (string, string) {
	// Check for critical escalations.
	for _, e := range escalations {
		if e.Rule.EscalatedWeight == "critical" {
			return "critical", "Critical escalation triggered: " + e.Rule.EscalatedDescription
		}
	}

	// Count signals by severity and polarity.
	var criticalCount, negativeCount, positiveCount int
	strongNegative := 0
	for _, cs := range categories {
		criticalCount += cs.ByWeight["critical"]
		negativeCount += cs.ByPolarity["negative"]
		positiveCount += cs.ByPolarity["positive"]
		// Strong signals only count toward concern if the category is predominantly negative.
		if cs.DominantPolarity == "negative" {
			strongNegative += cs.ByWeight["strong"]
		}
	}

	if criticalCount > 0 {
		return "critical", "Critical-weight signals present requiring immediate attention."
	}

	// Triggered escalations with strong+ weight indicate a pattern requiring attention.
	for _, e := range escalations {
		if IsAtLeastWeight(e.Rule.EscalatedWeight, "strong") {
			return "concerning", "Escalation triggered: " + e.Rule.EscalatedDescription
		}
	}

	if strongNegative >= 2 || negativeCount > positiveCount*2 {
		return "concerning", "Multiple strong negative signals or predominantly negative activity."
	}
	if strongNegative >= 1 || negativeCount > positiveCount {
		return "mixed", "Strong negative or unfavorable signals present that warrant monitoring."
	}
	return "positive", "Activity is predominantly positive or neutral."
}
