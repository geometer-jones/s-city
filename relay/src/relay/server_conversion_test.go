package relay

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/models"
)

func TestModelEventFromNostrCopiesTags(t *testing.T) {
	in := &nostr.Event{
		ID:        "id-1",
		PubKey:    "pub",
		CreatedAt: 123,
		Kind:      1,
		Tags:      nostr.Tags{nostr.Tag{"p", "key"}},
		Content:   "hello",
		Sig:       "sig",
	}

	out := modelEventFromNostr(in)
	if out.ID != in.ID || out.PubKey != in.PubKey || out.Kind != in.Kind {
		t.Fatalf("unexpected mapped event: %+v", out)
	}
	out.Tags[0][1] = "changed"
	if in.Tags[0][1] == "changed" {
		t.Fatalf("expected deep copy of tags")
	}
}

func TestNostrEventFromModelCopiesTags(t *testing.T) {
	in := models.Event{
		ID:        "id-2",
		PubKey:    "pub",
		CreatedAt: 456,
		Kind:      2,
		Tags:      [][]string{{"e", "id-1"}},
		Content:   "world",
		Sig:       "sig",
	}

	out := nostrEventFromModel(in)
	if out.ID != in.ID || out.PubKey != in.PubKey || out.Kind != in.Kind {
		t.Fatalf("unexpected mapped event: %+v", out)
	}
	out.Tags[0][1] = "changed"
	if in.Tags[0][1] == "changed" {
		t.Fatalf("expected deep copy of tags")
	}
}
