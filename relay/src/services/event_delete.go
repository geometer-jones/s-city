package services

import (
	"context"
	"fmt"
	"time"

	"s-city/src/lib"
	"s-city/src/models"
	"s-city/src/storage"
)

// EventDeleteService processes deletion requests and projection side effects.
type EventDeleteService struct {
	repo       *storage.EventsRepo
	projection *GroupProjectionService
	metrics    *lib.Metrics
}

func NewEventDeleteService(repo *storage.EventsRepo, projection *GroupProjectionService, metrics *lib.Metrics) *EventDeleteService {
	return &EventDeleteService{repo: repo, projection: projection, metrics: metrics}
}

func (s *EventDeleteService) DeleteEvent(ctx context.Context, req models.DeletedEvent) error {
	if req.EventID == "" || req.DeletedBy == "" {
		return fmt.Errorf("event_id and deleted_by are required")
	}
	if req.DeletedAt == 0 {
		req.DeletedAt = time.Now().Unix()
	}

	event, err := s.repo.GetEvent(ctx, req.EventID)
	if err != nil {
		return err
	}
	if event.PubKey != req.DeletedBy {
		return fmt.Errorf("delete not authorized for this pubkey")
	}

	if err := s.repo.MarkDeleted(ctx, req); err != nil {
		return err
	}
	s.metrics.Inc("events_deleted_total")

	if s.projection != nil {
		if err := s.projection.ApplyDeletion(ctx, req.EventID); err != nil {
			return err
		}
	}

	return nil
}
