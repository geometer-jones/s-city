package storage

import "s-city/src/models"

// EventTagsRepo contains deterministic tag normalization helpers.
type EventTagsRepo struct{}

func NewEventTagsRepo() *EventTagsRepo {
	return &EventTagsRepo{}
}

func (r *EventTagsRepo) Normalize(eventID string, tags [][]string) []models.EventTag {
	normalized := make([]models.EventTag, 0, len(tags))
	for idx, tag := range tags {
		if len(tag) == 0 || tag[0] == "" {
			continue
		}

		value := ""
		if len(tag) > 1 {
			value = tag[1]
		}

		normalized = append(normalized, models.EventTag{
			EventID:  eventID,
			TagIndex: idx,
			TagName:  tag[0],
			TagValue: value,
			TagArray: append([]string(nil), tag...),
		})
	}

	return normalized
}
