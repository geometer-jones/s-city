package tests

import (
	"context"
	"strings"
	"testing"

	"s-city/src/lib"
	"s-city/src/models"
	"s-city/src/services"
	"s-city/src/storage"
)

func TestEventDeleteServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)
	repo := storage.NewEventsRepo(pool, storage.NewEventTagsRepo())
	metrics := lib.NewMetrics()
	svc := services.NewEventDeleteService(repo, nil, metrics)

	source := models.Event{
		ID:        "evt-delete-1",
		PubKey:    "alice",
		CreatedAt: 100,
		Kind:      1,
		Tags:      [][]string{{"t", "nostr"}},
		Content:   "hello",
		Sig:       "sig",
	}
	if err := repo.InsertEvent(ctx, source); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	err := svc.DeleteEvent(ctx, models.DeletedEvent{EventID: source.ID, DeletedBy: "mallory", DeletedAt: 120})
	if err == nil || !strings.Contains(err.Error(), "authorized") {
		t.Fatalf("expected authorization error, got %v", err)
	}

	if err := svc.DeleteEvent(ctx, models.DeletedEvent{DeletedBy: "alice"}); err == nil {
		t.Fatalf("expected required field validation error")
	}

	if err := svc.DeleteEvent(ctx, models.DeletedEvent{
		EventID:   source.ID,
		DeletedBy: "alice",
		DeletedAt: 121,
		Reason:    "cleanup",
	}); err != nil {
		t.Fatalf("DeleteEvent success path: %v", err)
	}

	kind := 1
	got, err := repo.QueryEvents(ctx, storage.EventFilter{Kind: &kind, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents excluding deleted: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected deleted event to be excluded, got %v", got)
	}

	got, err = repo.QueryEvents(ctx, storage.EventFilter{Kind: &kind, IncludeDeleted: true, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents including deleted: %v", err)
	}
	if len(got) != 1 || got[0].ID != source.ID {
		t.Fatalf("expected deleted event in IncludeDeleted query, got %v", got)
	}

	snapshot := metrics.Snapshot()
	if snapshot["events_deleted_total"] != 1 {
		t.Fatalf("expected events_deleted_total=1, got %v", snapshot)
	}
}
