package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/lib"
	"s-city/src/models"
)

func TestEventStorageMode(t *testing.T) {
	tests := []struct {
		name string
		kind int
		want storageMode
	}{
		{name: "kind 0 is replaceable", kind: 0, want: storageModeReplaceable},
		{name: "kind 3 is replaceable", kind: 3, want: storageModeReplaceable},
		{name: "kind 10000 is replaceable", kind: 10000, want: storageModeReplaceable},
		{name: "kind 19999 is replaceable", kind: 19999, want: storageModeReplaceable},
		{name: "kind 1000 is regular", kind: 1000, want: storageModeRegular},
		{name: "kind 9999 is regular", kind: 9999, want: storageModeRegular},
		{name: "kind 1 is regular", kind: 1, want: storageModeRegular},
		{name: "kind 20000 is ephemeral", kind: 20000, want: storageModeEphemeral},
		{name: "kind 29999 is ephemeral", kind: 29999, want: storageModeEphemeral},
		{name: "kind 30000 is parameterized replaceable", kind: 30000, want: storageModeParameterizedReplaceable},
		{name: "kind 39999 is parameterized replaceable", kind: 39999, want: storageModeParameterizedReplaceable},
		{name: "kind 40000 is regular", kind: 40000, want: storageModeRegular},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := eventStorageMode(tc.kind)
			if got != tc.want {
				t.Fatalf("eventStorageMode(%d) = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}

func TestDTagValue(t *testing.T) {
	tests := []struct {
		name string
		tags [][]string
		want string
	}{
		{
			name: "returns d tag value",
			tags: [][]string{{"h", "group-a"}, {"d", "room-1"}},
			want: "room-1",
		},
		{
			name: "trims d tag value",
			tags: [][]string{{"d", "  room-2  "}},
			want: "room-2",
		},
		{
			name: "returns empty for missing d tag",
			tags: [][]string{{"h", "group-a"}, {"p", "pubkey"}},
			want: "",
		},
		{
			name: "returns empty for malformed d tag",
			tags: [][]string{{"d"}},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dTagValue(tc.tags)
			if got != tc.want {
				t.Fatalf("dTagValue(%v) = %q, want %q", tc.tags, got, tc.want)
			}
		})
	}
}

func TestRelayOnlyKind(t *testing.T) {
	if !relayOnlyKind(39000) || !relayOnlyKind(39001) || !relayOnlyKind(39002) || !relayOnlyKind(39003) {
		t.Fatalf("expected canonical state kinds to be relay-only")
	}
	if relayOnlyKind(39004) || relayOnlyKind(9007) {
		t.Fatalf("did not expect non-canonical kinds to be relay-only")
	}
}

func TestIngestRejectsNonRelaySignerForRelayOnlyKind(t *testing.T) {
	relayPriv := nostr.GeneratePrivateKey()
	relayPub, err := nostr.GetPublicKey(relayPriv)
	if err != nil {
		t.Fatalf("derive relay public key: %v", err)
	}

	userPriv := nostr.GeneratePrivateKey()
	userPub, err := nostr.GetPublicKey(userPriv)
	if err != nil {
		t.Fatalf("derive user public key: %v", err)
	}

	createdAt := time.Now().Unix()
	nostrEvent := nostr.Event{
		PubKey:    userPub,
		CreatedAt: nostr.Timestamp(createdAt),
		Kind:      39000,
		Tags: nostr.Tags{
			nostr.Tag{"d", "group-1"},
		},
		Content: "",
	}
	if err := nostrEvent.Sign(userPriv); err != nil {
		t.Fatalf("sign event: %v", err)
	}

	event := models.Event{
		ID:        nostrEvent.ID,
		PubKey:    nostrEvent.PubKey,
		CreatedAt: createdAt,
		Kind:      nostrEvent.Kind,
		Tags:      [][]string{{"d", "group-1"}},
		Content:   nostrEvent.Content,
		Sig:       nostrEvent.Sig,
	}

	svc := NewEventIngestService(
		nil,
		NewValidator(5*time.Minute),
		NewAbuseControls(10, 600, 0),
		nil,
		lib.NewMetrics(),
		relayPub,
	)

	err = svc.Ingest(context.Background(), event)
	if err == nil || !strings.Contains(err.Error(), "must be signed by relay") {
		t.Fatalf("expected relay-only rejection, got: %v", err)
	}
}
