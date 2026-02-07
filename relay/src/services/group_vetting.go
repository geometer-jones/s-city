package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"s-city/src/storage"
)

// GroupVettingService enforces vetted-group membership behavior.
type GroupVettingService struct {
	repo *storage.GroupRepo
}

func NewGroupVettingService(repo *storage.GroupRepo) *GroupVettingService {
	return &GroupVettingService{repo: repo}
}

func (s *GroupVettingService) JoinRequiresApproval(ctx context.Context, groupID string) (bool, error) {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return group.IsVetted, nil
}

func (s *GroupVettingService) CanAutoApprove(ctx context.Context, groupID, pubKey string) (bool, error) {
	requiresApproval, err := s.JoinRequiresApproval(ctx, groupID)
	if err != nil {
		return false, err
	}
	if requiresApproval {
		return false, nil
	}

	isBanned, err := s.repo.IsBanned(ctx, groupID, pubKey)
	if err != nil {
		return false, err
	}
	return !isBanned, nil
}
