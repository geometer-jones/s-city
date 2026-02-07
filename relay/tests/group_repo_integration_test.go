package tests

import (
	"context"
	"testing"
	"time"

	"s-city/src/models"
	"s-city/src/storage"
)

func TestGroupRepoLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)
	repo := storage.NewGroupRepo(pool)

	group := models.Group{
		GroupID:      "group-1",
		Name:         "Group One",
		About:        "about",
		Picture:      "https://example.com/p.png",
		Geohash:      "abc123",
		IsPrivate:    true,
		IsRestricted: false,
		IsVetted:     false,
		IsHidden:     false,
		IsClosed:     false,
		CreatedAt:    100,
		CreatedBy:    "owner",
		UpdatedAt:    100,
		UpdatedBy:    "owner",
	}
	if err := repo.UpsertGroup(ctx, group); err != nil {
		t.Fatalf("UpsertGroup: %v", err)
	}

	loaded, err := repo.GetGroup(ctx, group.GroupID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if loaded.Name != group.Name || loaded.Geohash != group.Geohash {
		t.Fatalf("unexpected group: %+v", loaded)
	}

	if err := repo.UpsertRole(ctx, models.GroupRole{
		GroupID:     group.GroupID,
		RoleName:    "admin",
		Description: "admins",
		Permissions: []string{},
		CreatedAt:   101,
		CreatedBy:   "owner",
		UpdatedAt:   101,
		UpdatedBy:   "owner",
	}); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}

	if err := repo.UpsertMember(ctx, models.GroupMember{
		GroupID:    group.GroupID,
		PubKey:     "member-1",
		AddedAt:    102,
		AddedBy:    "owner",
		RoleName:   "admin",
		PromotedAt: 102,
		PromotedBy: "owner",
	}); err != nil {
		t.Fatalf("UpsertMember: %v", err)
	}

	if isMember, err := repo.IsMember(ctx, group.GroupID, "member-1"); err != nil || !isMember {
		t.Fatalf("IsMember = %v, %v; want true, nil", isMember, err)
	}
	roleName, exists, err := repo.GetMemberRole(ctx, group.GroupID, "member-1")
	if err != nil || !exists || roleName != "admin" {
		t.Fatalf("GetMemberRole = (%q, %v, %v), want (admin, true, nil)", roleName, exists, err)
	}

	hasPerm, err := repo.HasPermission(ctx, group.GroupID, "member-1", models.PermissionDeleteGroup)
	if err != nil || !hasPerm {
		t.Fatalf("HasPermission = %v, %v; want true, nil", hasPerm, err)
	}
	isAdmin, err := repo.IsAdmin(ctx, group.GroupID, "member-1")
	if err != nil || !isAdmin {
		t.Fatalf("IsAdmin = %v, %v; want true, nil", isAdmin, err)
	}

	if err := repo.UpsertBan(ctx, models.GroupBan{
		GroupID:   group.GroupID,
		PubKey:    "member-1",
		Reason:    "spam",
		BannedAt:  103,
		BannedBy:  "owner",
		ExpiresAt: 0,
	}); err != nil {
		t.Fatalf("UpsertBan: %v", err)
	}
	if banned, err := repo.IsBanned(ctx, group.GroupID, "member-1"); err != nil || !banned {
		t.Fatalf("IsBanned permanent = %v, %v; want true, nil", banned, err)
	}
	if err := repo.UpsertBan(ctx, models.GroupBan{
		GroupID:   group.GroupID,
		PubKey:    "member-1",
		Reason:    "expired",
		BannedAt:  104,
		BannedBy:  "owner",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}); err != nil {
		t.Fatalf("UpsertBan expired: %v", err)
	}
	if banned, err := repo.IsBanned(ctx, group.GroupID, "member-1"); err != nil || banned {
		t.Fatalf("IsBanned expired = %v, %v; want false, nil", banned, err)
	}

	if err := repo.UpsertInvite(ctx, models.GroupInvite{
		GroupID:       group.GroupID,
		Code:          "invite-1",
		ExpiresAt:     0,
		MaxUsageCount: 10,
		UsageCount:    0,
		CreatedAt:     105,
		CreatedBy:     "owner",
	}); err != nil {
		t.Fatalf("UpsertInvite: %v", err)
	}
	if err := repo.UpsertJoinRequest(ctx, models.GroupJoinRequest{GroupID: group.GroupID, PubKey: "joiner", CreatedAt: 106}); err != nil {
		t.Fatalf("UpsertJoinRequest: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO events (id, pubkey, created_at, kind, tags, content, sig)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7)
	`, "event-1", "owner", int64(200), 1, `[]`, "", "sig"); err != nil {
		t.Fatalf("insert source event for group event mapping: %v", err)
	}
	if err := repo.AddGroupEvent(ctx, models.GroupEvent{GroupID: group.GroupID, EventID: "event-1", CreatedAt: 200}); err != nil {
		t.Fatalf("AddGroupEvent: %v", err)
	}
	if err := repo.RemoveGroupEventByEventID(ctx, "event-1"); err != nil {
		t.Fatalf("RemoveGroupEventByEventID: %v", err)
	}

	if members, err := repo.ListMembers(ctx, group.GroupID); err != nil || len(members) != 1 {
		t.Fatalf("ListMembers len = %d, err = %v; want 1, nil", len(members), err)
	}
	if roles, err := repo.ListRoles(ctx, group.GroupID); err != nil || len(roles) != 1 {
		t.Fatalf("ListRoles len = %d, err = %v; want 1, nil", len(roles), err)
	}
	if bans, err := repo.ListBans(ctx, group.GroupID); err != nil || len(bans) != 1 {
		t.Fatalf("ListBans len = %d, err = %v; want 1, nil", len(bans), err)
	}
	if invites, err := repo.ListInvites(ctx, group.GroupID); err != nil || len(invites) != 1 {
		t.Fatalf("ListInvites len = %d, err = %v; want 1, nil", len(invites), err)
	}
	if groups, err := repo.ListGroups(ctx, storage.GroupFilter{Limit: 10}); err != nil || len(groups) != 1 {
		t.Fatalf("ListGroups len = %d, err = %v; want 1, nil", len(groups), err)
	}

	if err := repo.DeleteJoinRequest(ctx, group.GroupID, "joiner"); err != nil {
		t.Fatalf("DeleteJoinRequest: %v", err)
	}
	if err := repo.RemoveMember(ctx, group.GroupID, "member-1"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if err := repo.DeleteRole(ctx, group.GroupID, "admin"); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if err := repo.CloseGroup(ctx, group.GroupID, 300, "owner"); err != nil {
		t.Fatalf("CloseGroup: %v", err)
	}
	closedGroup, err := repo.GetGroup(ctx, group.GroupID)
	if err != nil {
		t.Fatalf("GetGroup after close: %v", err)
	}
	if !closedGroup.IsClosed || !closedGroup.IsHidden {
		t.Fatalf("expected closed group to be hidden+closed, got %+v", closedGroup)
	}
}
