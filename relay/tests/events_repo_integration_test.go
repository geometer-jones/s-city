package tests

import (
	"context"
	"testing"

	"s-city/src/models"
	"s-city/src/storage"
)

func TestEventsRepoQueryAndDeletionLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)
	repo := storage.NewEventsRepo(pool, storage.NewEventTagsRepo())

	kindOne := 1
	inserted := []models.Event{
		{ID: "event-a", PubKey: "alice", CreatedAt: 100, Kind: kindOne, Tags: [][]string{{"t", "nostr"}, {"e", "root"}}, Content: "a", Sig: "sig"},
		{ID: "event-b", PubKey: "alice", CreatedAt: 101, Kind: kindOne, Tags: [][]string{{"t", "nostr"}}, Content: "b", Sig: "sig"},
		{ID: "event-d", PubKey: "alice", CreatedAt: 101, Kind: kindOne, Tags: [][]string{{"t", "nostr"}}, Content: "d", Sig: "sig"},
		{ID: "event-c", PubKey: "bob", CreatedAt: 101, Kind: 2, Tags: [][]string{{"t", "other"}}, Content: "c", Sig: "sig"},
	}
	for _, event := range inserted {
		if err := repo.InsertEvent(ctx, event); err != nil {
			t.Fatalf("InsertEvent(%s): %v", event.ID, err)
		}
	}

	got, err := repo.QueryEvents(ctx, storage.EventFilter{Tag: "t:nostr", Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents by tag: %v", err)
	}
	assertEventIDs(t, got, []string{"event-b", "event-d", "event-a"})

	until := int64(101)
	got, err = repo.QueryEvents(ctx, storage.EventFilter{
		Kind:    &kindOne,
		Until:   &until,
		UntilID: "event-b",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryEvents with until cursor: %v", err)
	}
	assertEventIDs(t, got, []string{"event-d", "event-a"})

	loaded, err := repo.GetEvent(ctx, "event-a")
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if len(loaded.Tags) != 2 || loaded.Tags[1][0] != "e" || loaded.Tags[1][1] != "root" {
		t.Fatalf("unexpected loaded tags: %v", loaded.Tags)
	}

	if err := repo.MarkDeleted(ctx, models.DeletedEvent{
		EventID:   "event-b",
		DeletedAt: 200,
		DeletedBy: "alice",
		Reason:    "test",
	}); err != nil {
		t.Fatalf("MarkDeleted: %v", err)
	}

	got, err = repo.QueryEvents(ctx, storage.EventFilter{Tag: "t:nostr", Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents excluding deleted: %v", err)
	}
	assertEventIDs(t, got, []string{"event-d", "event-a"})

	got, err = repo.QueryEvents(ctx, storage.EventFilter{Tag: "t:nostr", IncludeDeleted: true, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents including deleted: %v", err)
	}
	assertEventIDs(t, got, []string{"event-b", "event-d", "event-a"})
}

func TestEventsRepoUpsertReplaceableAndParameterized(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)
	repo := storage.NewEventsRepo(pool, storage.NewEventTagsRepo())

	kindReplaceable := 0
	kindAddressable := 30000

	if err := repo.UpsertReplaceableEvent(ctx, models.Event{
		ID: "r1", PubKey: "alice", CreatedAt: 100, Kind: kindReplaceable, Tags: [][]string{}, Content: "v1", Sig: "sig",
	}); err != nil {
		t.Fatalf("UpsertReplaceableEvent r1: %v", err)
	}
	if err := repo.UpsertReplaceableEvent(ctx, models.Event{
		ID: "r0-old", PubKey: "alice", CreatedAt: 99, Kind: kindReplaceable, Tags: [][]string{}, Content: "old", Sig: "sig",
	}); err != nil {
		t.Fatalf("UpsertReplaceableEvent old: %v", err)
	}
	got, err := repo.QueryEvents(ctx, storage.EventFilter{Author: "alice", Kind: &kindReplaceable, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents replaceable after old insert: %v", err)
	}
	assertEventIDs(t, got, []string{"r1"})

	if err := repo.UpsertReplaceableEvent(ctx, models.Event{
		ID: "r0", PubKey: "alice", CreatedAt: 100, Kind: kindReplaceable, Tags: [][]string{}, Content: "tie-lower-id", Sig: "sig",
	}); err != nil {
		t.Fatalf("UpsertReplaceableEvent tie lower id: %v", err)
	}
	got, err = repo.QueryEvents(ctx, storage.EventFilter{Author: "alice", Kind: &kindReplaceable, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents replaceable after tie: %v", err)
	}
	assertEventIDs(t, got, []string{"r0"})

	if err := repo.UpsertReplaceableEvent(ctx, models.Event{
		ID: "r2", PubKey: "alice", CreatedAt: 101, Kind: kindReplaceable, Tags: [][]string{}, Content: "newest", Sig: "sig",
	}); err != nil {
		t.Fatalf("UpsertReplaceableEvent newer: %v", err)
	}
	got, err = repo.QueryEvents(ctx, storage.EventFilter{Author: "alice", Kind: &kindReplaceable, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents replaceable after newer: %v", err)
	}
	assertEventIDs(t, got, []string{"r2"})

	if err := repo.UpsertParameterizedReplaceableEvent(ctx, models.Event{
		ID: "p1-old", PubKey: "alice", CreatedAt: 100, Kind: kindAddressable, Tags: [][]string{{"d", "room-1"}}, Content: "old", Sig: "sig",
	}, "room-1"); err != nil {
		t.Fatalf("UpsertParameterizedReplaceableEvent p1-old: %v", err)
	}
	if err := repo.UpsertParameterizedReplaceableEvent(ctx, models.Event{
		ID: "p1-older", PubKey: "alice", CreatedAt: 99, Kind: kindAddressable, Tags: [][]string{{"d", "room-1"}}, Content: "older", Sig: "sig",
	}, "room-1"); err != nil {
		t.Fatalf("UpsertParameterizedReplaceableEvent p1-older: %v", err)
	}
	if err := repo.UpsertParameterizedReplaceableEvent(ctx, models.Event{
		ID: "p1-new", PubKey: "alice", CreatedAt: 101, Kind: kindAddressable, Tags: [][]string{{"d", "room-1"}}, Content: "new", Sig: "sig",
	}, "room-1"); err != nil {
		t.Fatalf("UpsertParameterizedReplaceableEvent p1-new: %v", err)
	}
	if err := repo.UpsertParameterizedReplaceableEvent(ctx, models.Event{
		ID: "p2", PubKey: "alice", CreatedAt: 100, Kind: kindAddressable, Tags: [][]string{{"d", "room-2"}}, Content: "room-2", Sig: "sig",
	}, "room-2"); err != nil {
		t.Fatalf("UpsertParameterizedReplaceableEvent p2: %v", err)
	}

	got, err = repo.QueryEvents(ctx, storage.EventFilter{Author: "alice", Kind: &kindAddressable, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents parameterized all: %v", err)
	}
	assertEventIDs(t, got, []string{"p1-new", "p2"})

	got, err = repo.QueryEvents(ctx, storage.EventFilter{Author: "alice", Kind: &kindAddressable, Tag: "d:room-1", Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents parameterized by d tag: %v", err)
	}
	assertEventIDs(t, got, []string{"p1-new"})
}

func assertEventIDs(t *testing.T, events []models.Event, want []string) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("event count = %d, want %d (%v)", len(events), len(want), events)
	}
	for i := range want {
		if events[i].ID != want[i] {
			t.Fatalf("event[%d].ID = %q, want %q", i, events[i].ID, want[i])
		}
	}
}
