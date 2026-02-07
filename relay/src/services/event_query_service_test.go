package services

import (
	"context"
	"testing"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/models"
	"s-city/src/storage"
)

type captureEventQueryRepo struct {
	lastFilter storage.EventFilter
	events     []models.Event
}

func (r *captureEventQueryRepo) QueryEvents(_ context.Context, filter storage.EventFilter) ([]models.Event, error) {
	r.lastFilter = filter
	return r.events, nil
}

func TestQueryEventsIncludeDeletedFlag(t *testing.T) {
	repo := &captureEventQueryRepo{}
	svc := NewEventQueryService(repo)

	if _, err := svc.QueryEvents(context.Background(), storage.EventFilter{Author: "pub"}); err != nil {
		t.Fatalf("QueryEvents returned error: %v", err)
	}
	if repo.lastFilter.IncludeDeleted {
		t.Fatalf("expected QueryEvents to force IncludeDeleted=false")
	}

	if _, err := svc.QueryEventsIncludingDeleted(context.Background(), storage.EventFilter{Author: "pub"}); err != nil {
		t.Fatalf("QueryEventsIncludingDeleted returned error: %v", err)
	}
	if !repo.lastFilter.IncludeDeleted {
		t.Fatalf("expected QueryEventsIncludingDeleted to force IncludeDeleted=true")
	}
}

func TestMatchesNostrFilterWithTagValues(t *testing.T) {
	event := models.Event{
		ID:        "id-1",
		PubKey:    "pub-1",
		CreatedAt: 100,
		Kind:      1,
		Tags: [][]string{
			{"e", "root-id"},
			{"p", "pub-2"},
		},
	}

	matchFilter := nostr.Filter{
		IDs:     []string{"id-1"},
		Authors: []string{"pub-1"},
		Kinds:   []int{1},
		Since:   ptrTimestamp(99),
		Until:   ptrTimestamp(101),
		Tags: map[string][]string{
			"#e": {"root-id", "other"},
			"#p": {"pub-2"},
		},
	}
	if !matchesNostrFilter(event, matchFilter) {
		t.Fatalf("expected event to match full filter")
	}

	mismatchFilter := nostr.Filter{
		Tags: map[string][]string{
			"#e": {"not-present"},
		},
	}
	if matchesNostrFilter(event, mismatchFilter) {
		t.Fatalf("did not expect event to match mismatched tag filter")
	}
}

func ptrTimestamp(v int64) *nostr.Timestamp {
	ts := nostr.Timestamp(v)
	return &ts
}
