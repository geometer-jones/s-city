package tests

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"s-city/src/lib"
	"s-city/src/models"
	"s-city/src/services"
	"s-city/src/storage"
)

func TestEventIngestServiceStorageModesAndValidation(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)

	tagsRepo := storage.NewEventTagsRepo()
	eventsRepo := storage.NewEventsRepo(pool, tagsRepo)
	metrics := lib.NewMetrics()

	_, relayPub := generateKeypair(t)
	validator := services.NewValidator(5 * time.Minute)
	abuse := services.NewAbuseControls(100, 600, 0)
	ingest := services.NewEventIngestService(eventsRepo, validator, abuse, nil, metrics, relayPub)

	userPriv, userPub := generateKeypair(t)
	baseTime := nowUnix()

	regular := signedModelEvent(t, userPriv, baseTime, 1, [][]string{{"t", "nostr"}}, "regular")
	if err := ingest.Ingest(ctx, regular); err != nil {
		t.Fatalf("ingest regular: %v", err)
	}
	if err := ingest.Ingest(ctx, regular); !errors.Is(err, services.ErrDuplicateEvent) {
		t.Fatalf("expected duplicate error, got %v", err)
	}

	replaceableOld := signedModelEvent(t, userPriv, baseTime+1, 10000, [][]string{}, "replaceable-old")
	replaceableNew := signedModelEvent(t, userPriv, baseTime+2, 10000, [][]string{}, "replaceable-new")
	if err := ingest.Ingest(ctx, replaceableOld); err != nil {
		t.Fatalf("ingest replaceable old: %v", err)
	}
	if err := ingest.Ingest(ctx, replaceableNew); err != nil {
		t.Fatalf("ingest replaceable new: %v", err)
	}

	ephemeral := signedModelEvent(t, userPriv, baseTime+3, 20000, [][]string{}, "ephemeral")
	if err := ingest.Ingest(ctx, ephemeral); err != nil {
		t.Fatalf("ingest ephemeral: %v", err)
	}

	addressOld := signedModelEvent(t, userPriv, baseTime+4, 30000, [][]string{{"d", "room-1"}}, "addr-old")
	addressNew := signedModelEvent(t, userPriv, baseTime+5, 30000, [][]string{{"d", "room-1"}}, "addr-new")
	addressOther := signedModelEvent(t, userPriv, baseTime+6, 30000, [][]string{{"d", "room-2"}}, "addr-other")
	if err := ingest.Ingest(ctx, addressOld); err != nil {
		t.Fatalf("ingest address old: %v", err)
	}
	if err := ingest.Ingest(ctx, addressNew); err != nil {
		t.Fatalf("ingest address new: %v", err)
	}
	if err := ingest.Ingest(ctx, addressOther); err != nil {
		t.Fatalf("ingest address other: %v", err)
	}

	kindRegular := 1
	got, err := eventsRepo.QueryEvents(ctx, storage.EventFilter{Kind: &kindRegular, Author: userPub, IncludeDeleted: true, Limit: 10})
	if err != nil {
		t.Fatalf("query regular events: %v", err)
	}
	if len(got) != 1 || got[0].ID != regular.ID {
		t.Fatalf("unexpected regular events: %v", got)
	}

	kindReplaceable := 10000
	got, err = eventsRepo.QueryEvents(ctx, storage.EventFilter{Kind: &kindReplaceable, Author: userPub, IncludeDeleted: true, Limit: 10})
	if err != nil {
		t.Fatalf("query replaceable events: %v", err)
	}
	if len(got) != 1 || got[0].ID != replaceableNew.ID {
		t.Fatalf("unexpected replaceable events: %v", got)
	}

	kindEphemeral := 20000
	got, err = eventsRepo.QueryEvents(ctx, storage.EventFilter{Kind: &kindEphemeral, Author: userPub, IncludeDeleted: true, Limit: 10})
	if err != nil {
		t.Fatalf("query ephemeral events: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no persisted ephemeral events, got %v", got)
	}

	kindAddressable := 30000
	got, err = eventsRepo.QueryEvents(ctx, storage.EventFilter{Kind: &kindAddressable, Author: userPub, IncludeDeleted: true, Limit: 10})
	if err != nil {
		t.Fatalf("query addressable events: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two persisted addressable events, got %v", got)
	}

	relayOnly := signedModelEvent(t, userPriv, baseTime+7, 39000, [][]string{{"d", "group-1"}}, "")
	err = ingest.Ingest(ctx, relayOnly)
	if err == nil || !strings.Contains(err.Error(), "must be signed by relay") {
		t.Fatalf("expected relay-only signer rejection, got %v", err)
	}
}

func TestEventIngestServiceRateLimitPowAndProjectionErrors(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)

	tagsRepo := storage.NewEventTagsRepo()
	eventsRepo := storage.NewEventsRepo(pool, tagsRepo)
	groupRepo := storage.NewGroupRepo(pool)
	metrics := lib.NewMetrics()

	relayPriv, relayPub := generateKeypair(t)
	validator := services.NewValidator(5 * time.Minute)

	priv, pub := generateKeypair(t)
	baseTime := nowUnix()

	// Validation rejection branch.
	invalidIngest := services.NewEventIngestService(eventsRepo, validator, services.NewAbuseControls(10, 600, 0), nil, metrics, relayPub)
	invalid := signedModelEvent(t, priv, baseTime, 1, [][]string{}, "invalid")
	invalid.ID = "not-a-valid-id"
	if err := invalidIngest.Ingest(ctx, invalid); err == nil || !strings.Contains(err.Error(), "invalid event id") {
		t.Fatalf("expected validation failure, got %v", err)
	}

	// Rate-limit branch.
	rateLimited := services.NewEventIngestService(eventsRepo, validator, services.NewAbuseControls(1, 1, 0), nil, metrics, relayPub)
	first := signedModelEvent(t, priv, baseTime+1, 1, [][]string{}, "first")
	second := signedModelEvent(t, priv, baseTime+2, 1, [][]string{}, "second")
	if err := rateLimited.Ingest(ctx, first); err != nil {
		t.Fatalf("ingest first event under rate limit: %v", err)
	}
	if err := rateLimited.Ingest(ctx, second); err == nil || !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("expected rate-limit rejection, got %v", err)
	}

	// PoW rejection branch.
	powLimited := services.NewEventIngestService(eventsRepo, validator, services.NewAbuseControls(10, 600, 12), nil, metrics, relayPub)
	powEvent := signedModelEvent(t, priv, baseTime+3, 1, [][]string{}, "pow")
	if err := powLimited.Ingest(ctx, powEvent); err == nil || !strings.Contains(err.Error(), "insufficient pow") {
		t.Fatalf("expected pow rejection, got %v", err)
	}

	// Projection-error branch.
	if err := groupRepo.UpsertGroup(ctx, models.Group{
		GroupID:   "projection-group",
		CreatedAt: 10,
		CreatedBy: "owner-pub",
		UpdatedAt: 10,
		UpdatedBy: "owner-pub",
	}); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	projection := services.NewGroupProjectionService(groupRepo, nil, relayPub, relayPriv, nil, metrics)
	projectionIngest := services.NewEventIngestService(eventsRepo, validator, services.NewAbuseControls(10, 600, 0), projection, metrics, relayPub)
	unauthorized := signedModelEvent(t, priv, baseTime+4, 9003, [][]string{{"h", "projection-group"}, {"role", "mod"}}, "")
	err := projectionIngest.Ingest(ctx, unauthorized)
	if err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("expected projection authorization error, got %v", err)
	}

	if pub == "" {
		t.Fatalf("unexpected empty pubkey")
	}
	snapshot := metrics.Snapshot()
	if snapshot["events_rejected_validation_total"] == 0 {
		t.Fatalf("expected validation rejection metric increment, got %v", snapshot)
	}
	if snapshot["events_rejected_rate_limit_total"] == 0 {
		t.Fatalf("expected rate-limit metric increment, got %v", snapshot)
	}
	if snapshot["events_rejected_pow_total"] == 0 {
		t.Fatalf("expected pow rejection metric increment, got %v", snapshot)
	}
	if snapshot["group_projection_errors_total"] == 0 {
		t.Fatalf("expected projection error metric increment, got %v", snapshot)
	}
}
