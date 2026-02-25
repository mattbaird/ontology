package signals

import (
	"encoding/json"
	"testing"
)

func init() {
	Init()
}

func TestClassifyEvent_OnTimePayment(t *testing.T) {
	event := DomainEvent{
		EventID:   "pay-1",
		EventType: "PaymentRecorded",
		Payload:   json.RawMessage(`{"days_past_due": 0}`),
	}
	result, ok := ClassifyEvent(event)
	if !ok {
		t.Fatal("expected classification for PaymentRecorded")
	}
	if result.Category != "financial" {
		t.Errorf("category = %q, want financial", result.Category)
	}
	if result.Polarity != "positive" {
		t.Errorf("polarity = %q, want positive", result.Polarity)
	}
	if result.Weight != "info" {
		t.Errorf("weight = %q, want info", result.Weight)
	}
}

func TestClassifyEvent_LatePayment(t *testing.T) {
	event := DomainEvent{
		EventID:   "pay-2",
		EventType: "PaymentRecorded",
		Payload:   json.RawMessage(`{"days_past_due": 5}`),
	}
	result, ok := ClassifyEvent(event)
	if !ok {
		t.Fatal("expected classification for late PaymentRecorded")
	}
	if result.Polarity != "negative" {
		t.Errorf("polarity = %q, want negative", result.Polarity)
	}
}

func TestClassifyEvent_Complaint(t *testing.T) {
	event := DomainEvent{
		EventID:   "comp-1",
		EventType: "ComplaintCreated",
	}
	result, ok := ClassifyEvent(event)
	if !ok {
		t.Fatal("expected classification for ComplaintCreated")
	}
	if result.Category != "maintenance" {
		t.Errorf("category = %q, want maintenance", result.Category)
	}
	if result.Polarity != "negative" {
		t.Errorf("polarity = %q, want negative", result.Polarity)
	}
}

func TestClassifyEvent_UnknownType(t *testing.T) {
	event := DomainEvent{
		EventID:   "x-1",
		EventType: "TotallyMadeUpEvent",
	}
	_, ok := ClassifyEvent(event)
	if ok {
		t.Error("expected no classification for unknown event type")
	}
}

func TestClassifyEvent_NoPayload(t *testing.T) {
	// PaymentRecorded with no payload — both registrations have conditions
	// (days_past_due == 0, days_past_due > 0) so neither matches without payload.
	event := DomainEvent{
		EventID:   "pay-3",
		EventType: "PaymentRecorded",
	}
	_, ok := ClassifyEvent(event)
	if ok {
		t.Error("expected no classification when payload is missing and all registrations have conditions")
	}
}

func TestClassifyEvent_FallbackToUnconditional(t *testing.T) {
	// ComplaintCreated has no condition — should always classify.
	event := DomainEvent{
		EventID:   "comp-2",
		EventType: "ComplaintCreated",
	}
	result, ok := ClassifyEvent(event)
	if !ok {
		t.Fatal("expected classification for ComplaintCreated without payload")
	}
	if result.Category != "maintenance" {
		t.Errorf("category = %q, want maintenance", result.Category)
	}
}

func TestMatchCondition_Equals(t *testing.T) {
	payload := map[string]interface{}{"status": "active"}
	if !matchCondition("status == active", payload) {
		t.Error("expected status == active to match")
	}
	if matchCondition("status == inactive", payload) {
		t.Error("expected status == inactive to not match")
	}
}

func TestMatchCondition_NumericComparison(t *testing.T) {
	payload := map[string]interface{}{"days_past_due": float64(5)}
	if !matchCondition("days_past_due > 0", payload) {
		t.Error("expected days_past_due > 0 to match")
	}
	if matchCondition("days_past_due > 10", payload) {
		t.Error("expected days_past_due > 10 to not match")
	}
	if !matchCondition("days_past_due <= 5", payload) {
		t.Error("expected days_past_due <= 5 to match")
	}
	if !matchCondition("days_past_due >= 5", payload) {
		t.Error("expected days_past_due >= 5 to match")
	}
	if matchCondition("days_past_due < 5", payload) {
		t.Error("expected days_past_due < 5 to not match")
	}
}

func TestMatchCondition_MissingKey(t *testing.T) {
	payload := map[string]interface{}{"foo": "bar"}
	if matchCondition("missing_key == bar", payload) {
		t.Error("expected missing key to not match")
	}
}

func TestMatchCondition_NilPayload(t *testing.T) {
	if matchCondition("key == val", nil) {
		t.Error("expected nil payload to not match")
	}
}

func TestValueEquals_Bool(t *testing.T) {
	if !valueEquals(true, "true") {
		t.Error("expected true == 'true'")
	}
	if !valueEquals(false, "false") {
		t.Error("expected false == 'false'")
	}
	if valueEquals(true, "false") {
		t.Error("expected true != 'false'")
	}
}
