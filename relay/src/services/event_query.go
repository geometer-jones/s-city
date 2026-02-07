package services

import (
	"context"
	"strings"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/models"
	"s-city/src/storage"
)

type eventQueryRepo interface {
	QueryEvents(ctx context.Context, filter storage.EventFilter) ([]models.Event, error)
}

// EventQueryService serves active event reads and deletion-aware filtering.
type EventQueryService struct {
	repo eventQueryRepo
}

func NewEventQueryService(repo eventQueryRepo) *EventQueryService {
	return &EventQueryService{repo: repo}
}

func (s *EventQueryService) QueryEvents(ctx context.Context, filter storage.EventFilter) ([]models.Event, error) {
	filter.IncludeDeleted = false
	return s.repo.QueryEvents(ctx, filter)
}

func (s *EventQueryService) QueryEventsIncludingDeleted(ctx context.Context, filter storage.EventFilter) ([]models.Event, error) {
	filter.IncludeDeleted = true
	return s.repo.QueryEvents(ctx, filter)
}

// QueryNostrFilter provides websocket REQ-compatible querying.
func (s *EventQueryService) QueryNostrFilter(ctx context.Context, filter nostr.Filter) ([]models.Event, error) {
	targetLimit := int(filter.Limit)
	if targetLimit <= 0 {
		targetLimit = 100
	}

	coarse := storage.EventFilter{
		Limit:          500,
		IncludeDeleted: false,
	}
	if len(filter.Authors) == 1 {
		coarse.Author = filter.Authors[0]
	}
	if len(filter.Kinds) == 1 {
		kind := filter.Kinds[0]
		coarse.Kind = &kind
	}
	if filter.Since != nil {
		since := int64(*filter.Since)
		coarse.Since = &since
	}
	for tagKey, tagValues := range filter.Tags {
		if len(tagValues) == 0 {
			continue
		}
		tagKey = strings.TrimPrefix(tagKey, "#")
		coarse.Tag = tagKey + ":" + tagValues[0]
		break
	}

	var untilCursor *int64
	if filter.Until != nil {
		u := int64(*filter.Until)
		untilCursor = &u
	}
	untilIDCursor := ""

	filtered := make([]models.Event, 0, targetLimit)
	seen := make(map[string]struct{}, targetLimit)
	for len(filtered) < targetLimit {
		query := coarse
		if untilCursor != nil {
			u := *untilCursor
			query.Until = &u
			query.UntilID = untilIDCursor
		}

		events, err := s.QueryEvents(ctx, query)
		if err != nil {
			return nil, err
		}
		if len(events) == 0 {
			break
		}

		for _, event := range events {
			if _, exists := seen[event.ID]; exists {
				continue
			}
			seen[event.ID] = struct{}{}
			if matchesNostrFilter(event, filter) {
				filtered = append(filtered, event)
				if len(filtered) >= targetLimit {
					break
				}
			}
		}
		if len(filtered) >= targetLimit || len(events) < query.Limit {
			break
		}

		oldest := events[len(events)-1]
		if coarse.Since != nil && oldest.CreatedAt < *coarse.Since {
			break
		}
		if untilCursor != nil && oldest.CreatedAt == *untilCursor && oldest.ID == untilIDCursor {
			break
		}
		nextUntil := oldest.CreatedAt
		untilCursor = &nextUntil
		untilIDCursor = oldest.ID
	}

	return filtered, nil
}

func matchesNostrFilter(event models.Event, filter nostr.Filter) bool {
	if len(filter.IDs) > 0 && !stringInSlice(event.ID, filter.IDs) {
		return false
	}
	if len(filter.Authors) > 0 && !stringInSlice(event.PubKey, filter.Authors) {
		return false
	}
	if len(filter.Kinds) > 0 && !intInSlice(event.Kind, filter.Kinds) {
		return false
	}
	if filter.Since != nil && event.CreatedAt < int64(*filter.Since) {
		return false
	}
	if filter.Until != nil && event.CreatedAt > int64(*filter.Until) {
		return false
	}

	for key, values := range filter.Tags {
		key = strings.TrimPrefix(key, "#")
		if !eventHasAnyTagValue(event.Tags, key, values) {
			return false
		}
	}
	return true
}

func eventHasAnyTagValue(tags [][]string, tagName string, values []string) bool {
	for _, tag := range tags {
		if len(tag) < 2 || tag[0] != tagName {
			continue
		}
		for _, value := range values {
			if tag[1] == value {
				return true
			}
		}
	}
	return false
}

func stringInSlice(value string, values []string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func intInSlice(value int, values []int) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
