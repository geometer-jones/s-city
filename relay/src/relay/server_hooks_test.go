package relay

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fiatjaf/eventstore"
	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"

	"s-city/src/lib"
	"s-city/src/services"
	"s-city/src/storage"
)

func TestWireKhatruHooksStoreQueryDelete(t *testing.T) {
	ctx := context.Background()
	pool := openRelayIntegrationPool(t)

	tagsRepo := storage.NewEventTagsRepo()
	eventsRepo := storage.NewEventsRepo(pool, tagsRepo)
	groupRepo := storage.NewGroupRepo(pool)
	metrics := lib.NewMetrics()

	relayPriv := nostr.GeneratePrivateKey()
	relayPub, err := nostr.GetPublicKey(relayPriv)
	if err != nil {
		t.Fatalf("derive relay pubkey: %v", err)
	}

	validator := services.NewValidator(5 * time.Minute)
	abuse := services.NewAbuseControls(30, 120, 0)
	vetting := services.NewGroupVettingService(groupRepo)
	projection := services.NewGroupProjectionService(groupRepo, eventsRepo, relayPub, relayPriv, vetting, metrics)
	ingest := services.NewEventIngestService(eventsRepo, validator, abuse, projection, metrics, relayPub)
	query := services.NewEventQueryService(eventsRepo)
	del := services.NewEventDeleteService(eventsRepo, projection, metrics)

	r := khatru.NewRelay()
	wireKhatruHooks(r, ingest, query, del)

	if len(r.StoreEvent) == 0 || len(r.QueryEvents) == 0 || len(r.DeleteEvent) == 0 {
		t.Fatalf("expected khatru hooks to be registered")
	}

	userPriv := nostr.GeneratePrivateKey()
	userPub, err := nostr.GetPublicKey(userPriv)
	if err != nil {
		t.Fatalf("derive user pubkey: %v", err)
	}

	event := nostr.Event{
		PubKey:    userPub,
		CreatedAt: nostr.Now(),
		Kind:      1,
		Tags:      nostr.Tags{nostr.Tag{"t", "hooks"}},
		Content:   "hello hooks",
	}
	if err := event.Sign(userPriv); err != nil {
		t.Fatalf("sign event: %v", err)
	}

	if err := r.StoreEvent[len(r.StoreEvent)-1](ctx, &event); err != nil {
		t.Fatalf("StoreEvent first insert failed: %v", err)
	}
	if err := r.StoreEvent[len(r.StoreEvent)-1](ctx, &event); !errors.Is(err, eventstore.ErrDupEvent) {
		t.Fatalf("StoreEvent duplicate err = %v, want %v", err, eventstore.ErrDupEvent)
	}

	filter := nostr.Filter{
		Authors: []string{userPub},
		Kinds:   []int{1},
		Limit:   10,
	}
	ch, err := r.QueryEvents[len(r.QueryEvents)-1](ctx, filter)
	if err != nil {
		t.Fatalf("QueryEvents hook failed: %v", err)
	}

	results := make([]*nostr.Event, 0)
	for evt := range ch {
		results = append(results, evt)
	}
	if len(results) != 1 || results[0].ID != event.ID {
		t.Fatalf("unexpected query results: %v", results)
	}

	if err := r.DeleteEvent[len(r.DeleteEvent)-1](ctx, &nostr.Event{ID: event.ID, PubKey: userPub}); err != nil {
		t.Fatalf("DeleteEvent hook failed: %v", err)
	}

	ch, err = r.QueryEvents[len(r.QueryEvents)-1](ctx, filter)
	if err != nil {
		t.Fatalf("QueryEvents hook after delete failed: %v", err)
	}
	for evt := range ch {
		t.Fatalf("expected no events after delete, got %v", evt)
	}
}
