package activity

import (
	"context"
	"testing"
	"time"

	"github.com/matthewbaird/ontology/internal/types"
)

func testEntry(entityType, entityID, category, weight, polarity, summary string, daysAgo int) types.ActivityEntry {
	return types.ActivityEntry{
		EventID:           "test-" + summary,
		EventType:         "TestEvent",
		OccurredAt:        time.Now().AddDate(0, 0, -daysAgo),
		IndexedEntityType: entityType,
		IndexedEntityID:   entityID,
		EntityRole:        "subject",
		Summary:           summary,
		Category:          category,
		Weight:            weight,
		Polarity:          polarity,
	}
}

func TestMemoryStore_WriteAndQuery(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	entries := []types.ActivityEntry{
		testEntry("person", "alice", "financial", "info", "positive", "Payment on time", 10),
		testEntry("person", "alice", "maintenance", "moderate", "negative", "Complaint filed", 5),
		testEntry("person", "bob", "financial", "info", "positive", "Payment on time", 10),
	}

	if err := store.WriteEntries(ctx, entries); err != nil {
		t.Fatalf("WriteEntries: %v", err)
	}

	results, _, total, err := store.QueryByEntity(ctx, "person", "alice", DefaultQueryOptions())
	if err != nil {
		t.Fatalf("QueryByEntity: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(results) != 2 {
		t.Errorf("results = %d, want 2", len(results))
	}
}

func TestMemoryStore_QueryByEntity_FilterCategory(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	entries := []types.ActivityEntry{
		testEntry("person", "alice", "financial", "info", "positive", "Payment", 10),
		testEntry("person", "alice", "maintenance", "moderate", "negative", "Complaint", 5),
	}
	store.WriteEntries(ctx, entries)

	opts := DefaultQueryOptions()
	opts.Categories = []string{"financial"}
	results, _, total, err := store.QueryByEntity(ctx, "person", "alice", opts)
	if err != nil {
		t.Fatalf("QueryByEntity: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(results) != 1 {
		t.Errorf("results = %d, want 1", len(results))
	}
	if results[0].Category != "financial" {
		t.Errorf("category = %q, want financial", results[0].Category)
	}
}

func TestMemoryStore_QueryByEntity_TimeWindow(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	entries := []types.ActivityEntry{
		testEntry("person", "alice", "financial", "info", "positive", "Recent", 5),
		testEntry("person", "alice", "financial", "info", "positive", "Old", 200),
	}
	store.WriteEntries(ctx, entries)

	since := time.Now().AddDate(0, 0, -30)
	opts := DefaultQueryOptions()
	opts.Since = &since
	results, _, total, err := store.QueryByEntity(ctx, "person", "alice", opts)
	if err != nil {
		t.Fatalf("QueryByEntity: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(results) != 1 || results[0].Summary != "Recent" {
		t.Errorf("expected only 'Recent' entry")
	}
}

func TestMemoryStore_QueryByEntity_MinWeight(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	entries := []types.ActivityEntry{
		testEntry("person", "alice", "financial", "info", "positive", "Info level", 5),
		testEntry("person", "alice", "maintenance", "strong", "negative", "Strong level", 5),
	}
	store.WriteEntries(ctx, entries)

	opts := DefaultQueryOptions()
	opts.MinWeight = "moderate"
	results, _, total, err := store.QueryByEntity(ctx, "person", "alice", opts)
	if err != nil {
		t.Fatalf("QueryByEntity: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(results) != 1 || results[0].Weight != "strong" {
		t.Errorf("expected only 'strong' entry")
	}
}

func TestMemoryStore_Search(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	entries := []types.ActivityEntry{
		testEntry("person", "alice", "maintenance", "moderate", "negative", "Noise complaint from neighbor", 5),
		testEntry("person", "alice", "financial", "info", "positive", "Payment received on time", 10),
		testEntry("person", "bob", "maintenance", "moderate", "negative", "Noise complaint party", 3),
	}
	store.WriteEntries(ctx, entries)

	results, total, err := store.Search(ctx, "noise", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(results) != 2 {
		t.Errorf("results = %d, want 2", len(results))
	}
}

func TestMemoryStore_Search_EntityTypeFilter(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	entries := []types.ActivityEntry{
		testEntry("person", "alice", "maintenance", "moderate", "negative", "Noise complaint", 5),
		testEntry("lease", "lease-1", "maintenance", "moderate", "negative", "Noise complaint", 5),
	}
	store.WriteEntries(ctx, entries)

	opts := DefaultSearchOptions()
	opts.EntityType = "person"
	results, total, err := store.Search(ctx, "noise", opts)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(results) != 1 || results[0].IndexedEntityType != "person" {
		t.Errorf("expected only person entity")
	}
}

func TestMemoryStore_Search_NoMatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	entries := []types.ActivityEntry{
		testEntry("person", "alice", "financial", "info", "positive", "Payment received", 5),
	}
	store.WriteEntries(ctx, entries)

	results, total, err := store.Search(ctx, "zzzznotfound", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Errorf("expected no results, got %d", total)
	}
}

func TestMemoryStore_EmptyStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	results, _, total, err := store.QueryByEntity(ctx, "person", "nobody", DefaultQueryOptions())
	if err != nil {
		t.Fatalf("QueryByEntity: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Errorf("expected empty results from empty store")
	}
}
