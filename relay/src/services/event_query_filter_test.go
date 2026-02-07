package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/models"
	"s-city/src/storage"
)

type fakeEventQueryRepo struct {
	events []models.Event
	calls  int
}

func (r *fakeEventQueryRepo) QueryEvents(_ context.Context, filter storage.EventFilter) ([]models.Event, error) {
	r.calls++
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	out := make([]models.Event, 0, limit)
	for _, event := range r.events {
		if filter.Author != "" && event.PubKey != filter.Author {
			continue
		}
		if filter.Kind != nil && event.Kind != *filter.Kind {
			continue
		}
		if filter.Since != nil && event.CreatedAt < *filter.Since {
			continue
		}
		if filter.Until != nil {
			if event.CreatedAt > *filter.Until {
				continue
			}
			if filter.UntilID != "" && event.CreatedAt == *filter.Until && event.ID <= filter.UntilID {
				continue
			}
		}
		if filter.Tag != "" && !eventMatchesTagFilter(event, filter.Tag) {
			continue
		}
		out = append(out, event)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func eventMatchesTagFilter(event models.Event, raw string) bool {
	parts := strings.SplitN(raw, ":", 2)
	tagName := ""
	tagValue := strings.TrimSpace(raw)
	if len(parts) == 2 {
		tagName = strings.TrimSpace(parts[0])
		tagValue = strings.TrimSpace(parts[1])
	}

	for _, tag := range event.Tags {
		if len(tag) < 2 {
			continue
		}
		if tagName != "" && tag[0] != tagName {
			continue
		}
		if tag[1] == tagValue {
			return true
		}
	}
	return false
}

func TestQueryNostrFilterPaginatesForSparseMultiValueFilters(t *testing.T) {
	events := make([]models.Event, 0, 600)
	createdAt := int64(5000)
	for i := 0; i < 520; i++ {
		events = append(events, models.Event{
			ID:        fmt.Sprintf("nonmatch-%d", i),
			PubKey:    "x",
			CreatedAt: createdAt - int64(i),
			Kind:      1,
			Tags:      [][]string{},
			Content:   "",
			Sig:       "",
		})
	}
	for i := 0; i < 80; i++ {
		pubKey := "a"
		if i%2 == 1 {
			pubKey = "b"
		}
		events = append(events, models.Event{
			ID:        fmt.Sprintf("match-%d", i),
			PubKey:    pubKey,
			CreatedAt: createdAt - int64(520+i),
			Kind:      2,
			Tags:      [][]string{{"t", "group"}},
			Content:   "",
			Sig:       "",
		})
	}

	repo := &fakeEventQueryRepo{events: events}
	svc := NewEventQueryService(repo)

	filter := nostr.Filter{
		Authors: []string{"a", "b"},
		Kinds:   []int{1, 2},
		Limit:   10,
	}
	got, err := svc.QueryNostrFilter(context.Background(), filter)
	if err != nil {
		t.Fatalf("QueryNostrFilter returned error: %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("expected 10 events, got %d", len(got))
	}
	if repo.calls < 2 {
		t.Fatalf("expected paginated querying, got %d calls", repo.calls)
	}
	for _, event := range got {
		if event.PubKey != "a" && event.PubKey != "b" {
			t.Fatalf("unexpected pubkey in filtered result: %q", event.PubKey)
		}
	}
}
