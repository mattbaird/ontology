package signals

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/matthewbaird/ontology/internal/types"
)

// DomainEvent represents a raw domain event to be classified.
type DomainEvent struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

// ClassificationResult holds the output of classifying a domain event.
type ClassificationResult struct {
	Category     string
	Weight       string
	Polarity     string
	Description  string
	Registration *types.SignalRegistration
}

// ClassifyEvent looks up the event type in the signal registry and returns
// the matching classification. If a registration has a condition, we attempt
// simple key==value matching against the event payload.
//
// Returns ok=false if no registration matches the event type.
func ClassifyEvent(event DomainEvent) (result ClassificationResult, ok bool) {
	registrations := LookupSignals(event.EventType)
	if len(registrations) == 0 {
		return ClassificationResult{}, false
	}

	// Parse payload once for condition matching.
	var payload map[string]interface{}
	if len(event.Payload) > 0 {
		_ = json.Unmarshal(event.Payload, &payload)
	}

	// Try registrations with conditions first (more specific), then without.
	var fallback *types.SignalRegistration
	for i := range registrations {
		reg := &registrations[i]
		if reg.Condition == "" {
			if fallback == nil {
				fallback = reg
			}
			continue
		}
		if matchCondition(reg.Condition, payload) {
			return ClassificationResult{
				Category:     reg.Category,
				Weight:       reg.Weight,
				Polarity:     reg.Polarity,
				Description:  reg.Description,
				Registration: reg,
			}, true
		}
	}

	// Fall back to unconditional registration.
	if fallback != nil {
		return ClassificationResult{
			Category:     fallback.Category,
			Weight:       fallback.Weight,
			Polarity:     fallback.Polarity,
			Description:  fallback.Description,
			Registration: fallback,
		}, true
	}

	return ClassificationResult{}, false
}

// matchCondition performs simple condition matching against payload fields.
// Supports: "field == value", "field > N", "field < N", "field <= N", "field >= N"
func matchCondition(condition string, payload map[string]interface{}) bool {
	if payload == nil {
		return false
	}

	// Order matters: check two-char operators before single-char.
	for _, op := range []string{"<=", ">=", "==", "<", ">"} {
		parts := strings.SplitN(condition, op, 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		expected := strings.TrimSpace(parts[1])
		actual, exists := payload[key]
		if !exists {
			return false
		}
		switch op {
		case "==":
			return valueEquals(actual, expected)
		case "<=":
			return valueCompare(actual, expected) <= 0
		case ">=":
			return valueCompare(actual, expected) >= 0
		case "<":
			return valueCompare(actual, expected) < 0
		case ">":
			return valueCompare(actual, expected) > 0
		}
	}

	return false
}

// valueEquals checks if a payload value matches the expected string.
func valueEquals(actual interface{}, expected string) bool {
	switch v := actual.(type) {
	case string:
		return v == expected
	case float64:
		ev, err := strconv.ParseFloat(expected, 64)
		if err != nil {
			return strconv.FormatFloat(v, 'f', -1, 64) == expected
		}
		return v == ev
	case bool:
		return (v && expected == "true") || (!v && expected == "false")
	default:
		return false
	}
}

// valueCompare returns -1, 0, or 1 comparing actual to threshold numerically.
func valueCompare(actual interface{}, threshold string) int {
	av, ok := actual.(float64)
	if !ok {
		return 0
	}
	tv, err := strconv.ParseFloat(threshold, 64)
	if err != nil {
		return 0
	}
	if av < tv {
		return -1
	}
	if av > tv {
		return 1
	}
	return 0
}
