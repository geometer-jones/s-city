package tests

import (
	"context"
	"testing"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/lib"
	"s-city/src/models"
	"s-city/src/services"
	"s-city/src/storage"
)

func TestGroupProjectionFlow(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)

	tagsRepo := storage.NewEventTagsRepo()
	eventsRepo := storage.NewEventsRepo(pool, tagsRepo)
	groupRepo := storage.NewGroupRepo(pool)
	metrics := lib.NewMetrics()

	relayPriv := nostr.GeneratePrivateKey()
	relayPub, err := nostr.GetPublicKey(relayPriv)
	if err != nil {
		t.Fatalf("derive relay pubkey: %v", err)
	}
	ownerPriv := nostr.GeneratePrivateKey()
	ownerPub, err := nostr.GetPublicKey(ownerPriv)
	if err != nil {
		t.Fatalf("derive owner pubkey: %v", err)
	}

	vetting := services.NewGroupVettingService(groupRepo)
	projection := services.NewGroupProjectionService(groupRepo, eventsRepo, relayPub, relayPriv, vetting, metrics)

	createEvent := models.Event{
		ID:        "evt-create",
		PubKey:    ownerPub,
		CreatedAt: 100,
		Kind:      9007,
		Tags:      [][]string{{"h", "group-1"}, {"name", "Group One"}},
		Content:   "",
		Sig:       "sig",
	}
	if err := eventsRepo.InsertEvent(ctx, createEvent); err != nil {
		t.Fatalf("insert create event: %v", err)
	}
	if err := projection.ApplyEvent(ctx, createEvent); err != nil {
		t.Fatalf("apply create event: %v", err)
	}

	group, err := groupRepo.GetGroup(ctx, "group-1")
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if group.Name != "Group One" || group.CreatedBy != ownerPub {
		t.Fatalf("unexpected group state: %+v", group)
	}

	if members, err := groupRepo.ListMembers(ctx, "group-1"); err != nil || len(members) != 1 {
		t.Fatalf("ListMembers len = %d, err = %v; want 1, nil", len(members), err)
	}
	if roles, err := groupRepo.ListRoles(ctx, "group-1"); err != nil || len(roles) != 1 {
		t.Fatalf("ListRoles len = %d, err = %v; want 1, nil", len(roles), err)
	}

	var canonicalCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM events
		WHERE pubkey = $1 AND kind IN (39000, 39001, 39002, 39003)
	`, relayPub).Scan(&canonicalCount); err != nil {
		t.Fatalf("count canonical events: %v", err)
	}
	if canonicalCount != 4 {
		t.Fatalf("canonical event count = %d, want 4", canonicalCount)
	}

	metadataEvent := models.Event{
		ID:        "evt-metadata",
		PubKey:    ownerPub,
		CreatedAt: 110,
		Kind:      9002,
		Tags:      [][]string{{"h", "group-1"}, {"vetted", "true"}},
		Content:   "",
		Sig:       "sig",
	}
	if err := eventsRepo.InsertEvent(ctx, metadataEvent); err != nil {
		t.Fatalf("insert metadata event: %v", err)
	}
	if err := projection.ApplyEvent(ctx, metadataEvent); err != nil {
		t.Fatalf("apply metadata event: %v", err)
	}

	joinerPriv := nostr.GeneratePrivateKey()
	joinerPub, err := nostr.GetPublicKey(joinerPriv)
	if err != nil {
		t.Fatalf("derive joiner pubkey: %v", err)
	}
	joinEvent := models.Event{
		ID:        "evt-join",
		PubKey:    joinerPub,
		CreatedAt: 120,
		Kind:      9021,
		Tags:      [][]string{{"h", "group-1"}},
		Content:   "",
		Sig:       "sig",
	}
	if err := eventsRepo.InsertEvent(ctx, joinEvent); err != nil {
		t.Fatalf("insert join event: %v", err)
	}
	if err := projection.ApplyEvent(ctx, joinEvent); err != nil {
		t.Fatalf("apply join event: %v", err)
	}

	if _, memberExists, err := groupRepo.GetMemberRole(ctx, "group-1", joinerPub); err != nil || memberExists {
		t.Fatalf("expected pending join request, memberExists=%v, err=%v", memberExists, err)
	}
	var joinReqCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM group_join_requests WHERE group_id = $1 AND pubkey = $2`, "group-1", joinerPub).Scan(&joinReqCount); err != nil {
		t.Fatalf("count join requests: %v", err)
	}
	if joinReqCount != 1 {
		t.Fatalf("join request count = %d, want 1", joinReqCount)
	}

	if err := projection.ApproveJoinRequest(ctx, "group-1", joinerPub, ownerPub, 130); err != nil {
		t.Fatalf("ApproveJoinRequest: %v", err)
	}
	roleName, memberExists, err := groupRepo.GetMemberRole(ctx, "group-1", joinerPub)
	if err != nil || !memberExists || roleName != "member" {
		t.Fatalf("GetMemberRole after approval = (%q, %v, %v), want (member, true, nil)", roleName, memberExists, err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM group_join_requests WHERE group_id = $1 AND pubkey = $2`, "group-1", joinerPub).Scan(&joinReqCount); err != nil {
		t.Fatalf("count join requests after approval: %v", err)
	}
	if joinReqCount != 0 {
		t.Fatalf("join request count after approval = %d, want 0", joinReqCount)
	}

	if err := projection.ApplyDeletion(ctx, "evt-join"); err != nil {
		t.Fatalf("ApplyDeletion: %v", err)
	}
	var eventMappingCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM group_events WHERE event_id = $1`, "evt-join").Scan(&eventMappingCount); err != nil {
		t.Fatalf("count group event mappings: %v", err)
	}
	if eventMappingCount != 0 {
		t.Fatalf("group event mapping count = %d, want 0", eventMappingCount)
	}

	snapshot := metrics.Snapshot()
	if snapshot["group_projection_applied_total"] < 3 {
		t.Fatalf("expected projection applied metric increments, got %v", snapshot)
	}
	if snapshot["group_join_approved_total"] != 1 {
		t.Fatalf("expected one join-approved metric increment, got %v", snapshot)
	}
}
