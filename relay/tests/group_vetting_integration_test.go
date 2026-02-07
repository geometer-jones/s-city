package tests

import (
	"context"
	"testing"
	"time"

	"s-city/src/models"
	"s-city/src/services"
	"s-city/src/storage"
)

func TestGroupVettingService(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)
	repo := storage.NewGroupRepo(pool)
	vetting := services.NewGroupVettingService(repo)

	requiresApproval, err := vetting.JoinRequiresApproval(ctx, "missing-group")
	if err != nil {
		t.Fatalf("JoinRequiresApproval missing group: %v", err)
	}
	if !requiresApproval {
		t.Fatalf("expected missing group to require approval")
	}

	group := models.Group{
		GroupID:      "group-vetting",
		Name:         "Vetting",
		CreatedAt:    100,
		CreatedBy:    "owner",
		UpdatedAt:    100,
		UpdatedBy:    "owner",
		IsVetted:     false,
		IsRestricted: true,
	}
	if err := repo.UpsertGroup(ctx, group); err != nil {
		t.Fatalf("UpsertGroup: %v", err)
	}

	requiresApproval, err = vetting.JoinRequiresApproval(ctx, group.GroupID)
	if err != nil {
		t.Fatalf("JoinRequiresApproval existing group: %v", err)
	}
	if requiresApproval {
		t.Fatalf("expected non-vetted group to allow auto-approval path")
	}

	autoApprove, err := vetting.CanAutoApprove(ctx, group.GroupID, "user-1")
	if err != nil {
		t.Fatalf("CanAutoApprove non-banned: %v", err)
	}
	if !autoApprove {
		t.Fatalf("expected non-banned user to auto-approve in non-vetted group")
	}

	if err := repo.UpsertBan(ctx, models.GroupBan{
		GroupID:   group.GroupID,
		PubKey:    "user-1",
		Reason:    "spam",
		BannedAt:  120,
		BannedBy:  "owner",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}); err != nil {
		t.Fatalf("UpsertBan: %v", err)
	}

	autoApprove, err = vetting.CanAutoApprove(ctx, group.GroupID, "user-1")
	if err != nil {
		t.Fatalf("CanAutoApprove banned: %v", err)
	}
	if autoApprove {
		t.Fatalf("expected banned user not to auto-approve")
	}
}
