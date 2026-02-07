package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"s-city/src/lib"
	"s-city/src/models"
	"s-city/src/storage"
)

var ErrDuplicateEvent = errors.New("duplicate event")

// EventIngestService validates, abuse-checks, stores, and projects events.
type EventIngestService struct {
	repo        *storage.EventsRepo
	validator   *Validator
	abuse       *AbuseControls
	projection  *GroupProjectionService
	metrics     *lib.Metrics
	relayPubKey string
}

func NewEventIngestService(
	repo *storage.EventsRepo,
	validator *Validator,
	abuse *AbuseControls,
	projection *GroupProjectionService,
	metrics *lib.Metrics,
	relayPubKey string,
) *EventIngestService {
	return &EventIngestService{
		repo:        repo,
		validator:   validator,
		abuse:       abuse,
		projection:  projection,
		metrics:     metrics,
		relayPubKey: strings.ToLower(strings.TrimSpace(relayPubKey)),
	}
}

func (s *EventIngestService) Ingest(ctx context.Context, event models.Event) error {
	if err := s.validator.ValidateEvent(event); err != nil {
		s.metrics.Inc("events_rejected_validation_total")
		return err
	}

	if !s.abuse.Allow(event.PubKey, time.Now()) {
		s.metrics.Inc("events_rejected_rate_limit_total")
		return fmt.Errorf("rate limit exceeded")
	}

	requiredPowBits := s.abuse.RequiredPowBits(event.Kind)
	if err := s.abuse.ValidatePow(event, requiredPowBits); err != nil {
		s.metrics.Inc("events_rejected_pow_total")
		return err
	}
	if relayOnlyKind(event.Kind) && !strings.EqualFold(event.PubKey, s.relayPubKey) {
		s.metrics.Inc("events_rejected_validation_total")
		return fmt.Errorf("kind %d events must be signed by relay", event.Kind)
	}

	switch eventStorageMode(event.Kind) {
	case storageModeEphemeral:
		// Ephemeral events are accepted and relayed but intentionally not persisted.
	case storageModeReplaceable:
		if err := s.repo.UpsertReplaceableEvent(ctx, event); err != nil {
			return err
		}
		s.metrics.Inc("events_ingested_total")
	case storageModeParameterizedReplaceable:
		if err := s.repo.UpsertParameterizedReplaceableEvent(ctx, event, dTagValue(event.Tags)); err != nil {
			return err
		}
		s.metrics.Inc("events_ingested_total")
	default:
		if err := s.repo.InsertEvent(ctx, event); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				s.metrics.Inc("events_duplicate_total")
				return ErrDuplicateEvent
			}
			return err
		}
		s.metrics.Inc("events_ingested_total")
	}

	if s.projection != nil {
		if err := s.projection.ApplyEvent(ctx, event); err != nil {
			s.metrics.Inc("group_projection_errors_total")
			return err
		}
	}

	return nil
}

type storageMode int

const (
	storageModeRegular storageMode = iota
	storageModeReplaceable
	storageModeEphemeral
	storageModeParameterizedReplaceable
)

func eventStorageMode(kind int) storageMode {
	if kind == 0 || kind == 3 || (kind >= 10000 && kind <= 19999) {
		return storageModeReplaceable
	}
	if kind >= 20000 && kind <= 29999 {
		return storageModeEphemeral
	}
	if kind >= 30000 && kind <= 39999 {
		return storageModeParameterizedReplaceable
	}
	return storageModeRegular
}

func dTagValue(tags [][]string) string {
	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}
		if strings.TrimSpace(tag[0]) == "d" {
			return strings.TrimSpace(tag[1])
		}
	}
	return ""
}

func relayOnlyKind(kind int) bool {
	switch kind {
	case 39000, 39001, 39002, 39003:
		return true
	default:
		return false
	}
}
