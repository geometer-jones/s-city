package tests

import (
	"context"
	"testing"

	"s-city/src/lib"
	"s-city/src/models"
	"s-city/src/services"
	"s-city/src/storage"
)

func TestGroupProjectionModerationLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)

	eventsRepo := storage.NewEventsRepo(pool, storage.NewEventTagsRepo())
	groupRepo := storage.NewGroupRepo(pool)
	metrics := lib.NewMetrics()
	vetting := services.NewGroupVettingService(groupRepo)
	projection := services.NewGroupProjectionService(groupRepo, eventsRepo, "", "", vetting, metrics)

	groupID := "group-moderation"
	owner := "owner-pub"
	member := "member-pub"

	mustInsertAndApply := func(event models.Event) {
		t.Helper()
		if err := eventsRepo.InsertEvent(ctx, event); err != nil {
			t.Fatalf("insert event %s: %v", event.ID, err)
		}
		if err := projection.ApplyEvent(ctx, event); err != nil {
			t.Fatalf("apply event %s: %v", event.ID, err)
		}
	}

	mustInsertAndApply(models.Event{
		ID:        "evt-create-moderation",
		PubKey:    owner,
		CreatedAt: 100,
		Kind:      9007,
		Tags:      [][]string{{"h", groupID}, {"name", "Initial Name"}},
		Content:   "",
		Sig:       "sig",
	})

	mustInsertAndApply(models.Event{
		ID:        "evt-edit-metadata",
		PubKey:    owner,
		CreatedAt: 110,
		Kind:      9002,
		Tags: [][]string{
			{"h", groupID},
			{"name", "Updated Name"},
			{"about", "updated about"},
			{"picture", "https://example.com/group.png"},
			{"g", "abcdef123"},
			{"private", "true"},
			{"restricted"},
			{"vetted", "1"},
			{"hidden", "yes"},
		},
		Content: "",
		Sig:     "sig",
	})

	group, err := groupRepo.GetGroup(ctx, groupID)
	if err != nil {
		t.Fatalf("GetGroup after metadata update: %v", err)
	}
	if group.Name != "Updated Name" || group.About != "updated about" || group.Geohash != "abcdef" || !group.IsPrivate || !group.IsRestricted || !group.IsVetted || !group.IsHidden {
		t.Fatalf("unexpected group metadata after edit: %+v", group)
	}

	mustInsertAndApply(models.Event{
		ID:        "evt-create-role",
		PubKey:    owner,
		CreatedAt: 120,
		Kind:      9003,
		Tags:      [][]string{{"h", groupID}, {"role", "moderator"}, {"permissions", "delete-event,remove-user,admin"}, {"description", "Moderation team"}},
		Content:   "",
		Sig:       "sig",
	})
	roles, err := groupRepo.ListRoles(ctx, groupID)
	if err != nil {
		t.Fatalf("ListRoles after create-role: %v", err)
	}
	if len(roles) < 2 {
		t.Fatalf("expected owner + moderator roles, got %v", roles)
	}

	mustInsertAndApply(models.Event{
		ID:        "evt-put-user",
		PubKey:    owner,
		CreatedAt: 130,
		Kind:      9000,
		Tags:      [][]string{{"h", groupID}, {"p", member, "moderator"}},
		Content:   "",
		Sig:       "sig",
	})
	roleName, exists, err := groupRepo.GetMemberRole(ctx, groupID, member)
	if err != nil || !exists || roleName != "moderator" {
		t.Fatalf("GetMemberRole after put-user = (%q,%v,%v), want (moderator,true,nil)", roleName, exists, err)
	}

	targetEvent := models.Event{
		ID:        "evt-target-delete",
		PubKey:    member,
		CreatedAt: 140,
		Kind:      1,
		Tags:      [][]string{{"h", groupID}},
		Content:   "to be deleted",
		Sig:       "sig",
	}
	mustInsertAndApply(targetEvent)

	mustInsertAndApply(models.Event{
		ID:        "evt-delete-event",
		PubKey:    owner,
		CreatedAt: 141,
		Kind:      9005,
		Tags:      [][]string{{"h", groupID}, {"e", targetEvent.ID}, {"reason", "moderation"}},
		Content:   "",
		Sig:       "sig",
	})
	kindOne := 1
	eventsVisible, err := eventsRepo.QueryEvents(ctx, storage.EventFilter{Kind: &kindOne, Author: member, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents after delete-event: %v", err)
	}
	if len(eventsVisible) != 0 {
		t.Fatalf("expected deleted event hidden from default query, got %v", eventsVisible)
	}
	eventsIncludingDeleted, err := eventsRepo.QueryEvents(ctx, storage.EventFilter{Kind: &kindOne, Author: member, IncludeDeleted: true, Limit: 10})
	if err != nil {
		t.Fatalf("QueryEvents include deleted after delete-event: %v", err)
	}
	if len(eventsIncludingDeleted) != 1 || eventsIncludingDeleted[0].ID != targetEvent.ID {
		t.Fatalf("expected deleted target event still present in include-deleted query, got %v", eventsIncludingDeleted)
	}
	var targetMappingCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM group_events WHERE event_id = $1`, targetEvent.ID).Scan(&targetMappingCount); err != nil {
		t.Fatalf("count target group mapping: %v", err)
	}
	if targetMappingCount != 0 {
		t.Fatalf("expected target mapping removed after 9005, got %d", targetMappingCount)
	}

	mustInsertAndApply(models.Event{
		ID:        "evt-create-invite",
		PubKey:    owner,
		CreatedAt: 150,
		Kind:      9009,
		Tags:      [][]string{{"h", groupID}, {"code", "invite-code"}, {"max_usage_count", "5"}},
		Content:   "",
		Sig:       "sig",
	})
	invites, err := groupRepo.ListInvites(ctx, groupID)
	if err != nil {
		t.Fatalf("ListInvites after create-invite: %v", err)
	}
	if len(invites) != 1 || invites[0].Code != "invite-code" {
		t.Fatalf("unexpected invites after create-invite: %v", invites)
	}

	mustInsertAndApply(models.Event{
		ID:        "evt-remove-user",
		PubKey:    owner,
		CreatedAt: 160,
		Kind:      9001,
		Tags:      [][]string{{"h", groupID}, {"p", member}, {"ban"}, {"reason", "spam"}},
		Content:   "",
		Sig:       "sig",
	})
	if isMember, err := groupRepo.IsMember(ctx, groupID, member); err != nil || isMember {
		t.Fatalf("expected member removed after 9001, got isMember=%v err=%v", isMember, err)
	}
	if isBanned, err := groupRepo.IsBanned(ctx, groupID, member); err != nil || !isBanned {
		t.Fatalf("expected member banned after 9001, got isBanned=%v err=%v", isBanned, err)
	}

	mustInsertAndApply(models.Event{
		ID:        "evt-put-user-again",
		PubKey:    owner,
		CreatedAt: 170,
		Kind:      9000,
		Tags:      [][]string{{"h", groupID}, {"p", member}},
		Content:   "",
		Sig:       "sig",
	})
	mustInsertAndApply(models.Event{
		ID:        "evt-leave",
		PubKey:    member,
		CreatedAt: 171,
		Kind:      9022,
		Tags:      [][]string{{"h", groupID}},
		Content:   "",
		Sig:       "sig",
	})
	if isMember, err := groupRepo.IsMember(ctx, groupID, member); err != nil || isMember {
		t.Fatalf("expected member removed by leave request, got isMember=%v err=%v", isMember, err)
	}

	mustInsertAndApply(models.Event{
		ID:        "evt-delete-role",
		PubKey:    owner,
		CreatedAt: 180,
		Kind:      9004,
		Tags:      [][]string{{"h", groupID}, {"role", "moderator"}},
		Content:   "",
		Sig:       "sig",
	})
	roles, err = groupRepo.ListRoles(ctx, groupID)
	if err != nil {
		t.Fatalf("ListRoles after delete-role: %v", err)
	}
	for _, role := range roles {
		if role.RoleName == "moderator" {
			t.Fatalf("expected moderator role deleted, roles=%v", roles)
		}
	}

	mustInsertAndApply(models.Event{
		ID:        "evt-delete-group",
		PubKey:    owner,
		CreatedAt: 190,
		Kind:      9008,
		Tags:      [][]string{{"h", groupID}},
		Content:   "",
		Sig:       "sig",
	})
	group, err = groupRepo.GetGroup(ctx, groupID)
	if err != nil {
		t.Fatalf("GetGroup after delete-group: %v", err)
	}
	if !group.IsClosed || !group.IsHidden {
		t.Fatalf("expected closed+hidden after 9008, got %+v", group)
	}

	snapshot := metrics.Snapshot()
	if snapshot["group_projection_applied_total"] < 10 {
		t.Fatalf("expected projection metrics to increment, got %v", snapshot)
	}
}

func TestGroupProjectionRejectsUnauthorizedModeration(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)

	eventsRepo := storage.NewEventsRepo(pool, storage.NewEventTagsRepo())
	groupRepo := storage.NewGroupRepo(pool)
	projection := services.NewGroupProjectionService(groupRepo, eventsRepo, "", "", services.NewGroupVettingService(groupRepo), lib.NewMetrics())

	create := models.Event{
		ID:        "evt-unauth-create",
		PubKey:    "owner-pub",
		CreatedAt: 100,
		Kind:      9007,
		Tags:      [][]string{{"h", "group-unauth"}},
		Content:   "",
		Sig:       "sig",
	}
	if err := eventsRepo.InsertEvent(ctx, create); err != nil {
		t.Fatalf("insert create event: %v", err)
	}
	if err := projection.ApplyEvent(ctx, create); err != nil {
		t.Fatalf("apply create event: %v", err)
	}

	unauthorized := models.Event{
		ID:        "evt-unauth-edit",
		PubKey:    "stranger-pub",
		CreatedAt: 110,
		Kind:      9002,
		Tags:      [][]string{{"h", "group-unauth"}, {"name", "oops"}},
		Content:   "",
		Sig:       "sig",
	}
	if err := eventsRepo.InsertEvent(ctx, unauthorized); err != nil {
		t.Fatalf("insert unauthorized event: %v", err)
	}
	if err := projection.ApplyEvent(ctx, unauthorized); err == nil {
		t.Fatalf("expected unauthorized metadata edit to fail")
	}
}
