package services

import (
	"strings"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/models"
)

func signedModelEvent(t *testing.T, createdAt int64, kind int, tags [][]string, content string) models.Event {
	t.Helper()

	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("derive pubkey: %v", err)
	}

	nostrTags := make(nostr.Tags, 0, len(tags))
	for _, tag := range tags {
		nostrTag := make(nostr.Tag, len(tag))
		copy(nostrTag, tag)
		nostrTags = append(nostrTags, nostrTag)
	}

	evt := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Timestamp(createdAt),
		Kind:      kind,
		Tags:      nostrTags,
		Content:   content,
	}
	if err := evt.Sign(privKey); err != nil {
		t.Fatalf("sign event: %v", err)
	}

	return models.Event{
		ID:        evt.ID,
		PubKey:    evt.PubKey,
		CreatedAt: createdAt,
		Kind:      evt.Kind,
		Tags:      tags,
		Content:   evt.Content,
		Sig:       evt.Sig,
	}
}

func TestValidateEventRejectsMalformedFields(t *testing.T) {
	now := time.Now().Unix()
	base := signedModelEvent(t, now, 1, [][]string{{"t", "nostr"}}, "hello")

	tests := []struct {
		name    string
		mutate  func(models.Event) models.Event
		wantErr string
	}{
		{
			name: "invalid id format",
			mutate: func(event models.Event) models.Event {
				event.ID = "not-hex"
				return event
			},
			wantErr: "invalid event id",
		},
		{
			name: "invalid pubkey format",
			mutate: func(event models.Event) models.Event {
				event.PubKey = "not-hex"
				return event
			},
			wantErr: "invalid event pubkey",
		},
		{
			name: "invalid signature format",
			mutate: func(event models.Event) models.Event {
				event.Sig = "too-short"
				return event
			},
			wantErr: "invalid event signature format",
		},
		{
			name: "missing created_at",
			mutate: func(event models.Event) models.Event {
				event.CreatedAt = 0
				return event
			},
			wantErr: "event created_at is required",
		},
		{
			name: "empty tag entry",
			mutate: func(event models.Event) models.Event {
				event.Tags = append(event.Tags, []string{})
				return event
			},
			wantErr: "is empty",
		},
		{
			name: "empty tag name",
			mutate: func(event models.Event) models.Event {
				event.Tags = [][]string{{" ", "nostr"}}
				return event
			},
			wantErr: "has empty name",
		},
	}

	validator := NewValidator(5 * time.Minute)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := tc.mutate(base)
			err := validator.ValidateEvent(event)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ValidateEvent error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidateEventRejectsOutOfAllowedSkew(t *testing.T) {
	validator := NewValidator(30 * time.Second)

	past := signedModelEvent(t, time.Now().Add(-2*time.Minute).Unix(), 1, nil, "")
	err := validator.ValidateEvent(past)
	if err == nil || !strings.Contains(err.Error(), "out of allowed skew") {
		t.Fatalf("expected skew error for old event, got %v", err)
	}

	future := signedModelEvent(t, time.Now().Add(2*time.Minute).Unix(), 1, nil, "")
	err = validator.ValidateEvent(future)
	if err == nil || !strings.Contains(err.Error(), "out of allowed skew") {
		t.Fatalf("expected skew error for future event, got %v", err)
	}
}

func TestValidateEventRejectsTamperedPayload(t *testing.T) {
	event := signedModelEvent(t, time.Now().Unix(), 1, [][]string{{"t", "nostr"}}, "hello")
	event.Content = "tampered"

	validator := NewValidator(5 * time.Minute)
	err := validator.ValidateEvent(event)
	if err == nil || !strings.Contains(err.Error(), "event id does not match payload") {
		t.Fatalf("expected event id mismatch error, got %v", err)
	}
}

func TestValidateEventAcceptsUppercaseHexFields(t *testing.T) {
	event := signedModelEvent(t, time.Now().Unix(), 1, [][]string{{"t", "nostr"}}, "hello")
	event.ID = strings.ToUpper(event.ID)
	event.PubKey = strings.ToUpper(event.PubKey)
	event.Sig = strings.ToUpper(event.Sig)

	validator := NewValidator(5 * time.Minute)
	if err := validator.ValidateEvent(event); err != nil {
		t.Fatalf("expected uppercase hex fields to validate, got %v", err)
	}
}

func TestComputeEventIDNormalizesPubkeyCase(t *testing.T) {
	pubLower := strings.Repeat("ab", 32)
	pubUpper := strings.ToUpper(pubLower)
	createdAt := int64(1700000000)
	tags := [][]string{{"t", "nostr"}}

	lowerID, err := ComputeEventID(pubLower, createdAt, 1, tags, "hello")
	if err != nil {
		t.Fatalf("ComputeEventID lower failed: %v", err)
	}
	upperID, err := ComputeEventID(pubUpper, createdAt, 1, tags, "hello")
	if err != nil {
		t.Fatalf("ComputeEventID upper failed: %v", err)
	}

	if lowerID != upperID {
		t.Fatalf("ComputeEventID mismatch: lower=%s upper=%s", lowerID, upperID)
	}
}
