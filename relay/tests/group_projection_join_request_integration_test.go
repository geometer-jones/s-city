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

func TestGroupProjectionRejectsDuplicateJoinRequestForExistingMember(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)

	eventsRepo := storage.NewEventsRepo(pool, storage.NewEventTagsRepo())
	groupRepo := storage.NewGroupRepo(pool)
	vetting := services.NewGroupVettingService(groupRepo)
	projection := services.NewGroupProjectionService(groupRepo, nil, "", "", vetting, lib.NewMetrics())

	createEvent := models.Event{
		ID:        "evt-create-dup",
		PubKey:    "owner-pub",
		CreatedAt: 100,
		Kind:      9007,
		Tags:      [][]string{{"h", "group-dup"}, {"name", "Dup Group"}},
		Content:   "",
		Sig:       "sig",
	}
	if err := eventsRepo.InsertEvent(ctx, createEvent); err != nil {
		t.Fatalf("insert create event: %v", err)
	}
	if err := projection.ApplyEvent(ctx, createEvent); err != nil {
		t.Fatalf("apply create event: %v", err)
	}

	putUserEvent := models.Event{
		ID:        "evt-put-user",
		PubKey:    "owner-pub",
		CreatedAt: 110,
		Kind:      9000,
		Tags:      [][]string{{"h", "group-dup"}, {"p", "member-pub"}},
		Content:   "",
		Sig:       "sig",
	}
	if err := eventsRepo.InsertEvent(ctx, putUserEvent); err != nil {
		t.Fatalf("insert put-user event: %v", err)
	}
	if err := projection.ApplyEvent(ctx, putUserEvent); err != nil {
		t.Fatalf("apply put-user event: %v", err)
	}

	joinEvent := models.Event{
		ID:        "evt-join-duplicate",
		PubKey:    "member-pub",
		CreatedAt: 120,
		Kind:      9021,
		Tags:      [][]string{{"h", "group-dup"}},
		Content:   "",
		Sig:       "sig",
	}
	if err := eventsRepo.InsertEvent(ctx, joinEvent); err != nil {
		t.Fatalf("insert join event: %v", err)
	}
	err := projection.ApplyEvent(ctx, joinEvent)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}

	var joinReqCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM group_join_requests
		WHERE group_id = $1 AND pubkey = $2
	`, "group-dup", "member-pub").Scan(&joinReqCount); err != nil {
		t.Fatalf("count join requests: %v", err)
	}
	if joinReqCount != 0 {
		t.Fatalf("expected no pending join request for existing member, got %d", joinReqCount)
	}
}
