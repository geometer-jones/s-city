package services

import (
	"strings"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/models"
)

func TestValidateEventVerifiesSchnorrSignature(t *testing.T) {
	privKey := nostr.GeneratePrivateKey()
	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		t.Fatalf("derive public key: %v", err)
	}

	createdAt := time.Now().Unix()
	nostrEvent := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Timestamp(createdAt),
		Kind:      1,
		Tags: nostr.Tags{
			nostr.Tag{"t", "nostr"},
		},
		Content: "hello world",
	}
	if err := nostrEvent.Sign(privKey); err != nil {
		t.Fatalf("sign event: %v", err)
	}

	event := models.Event{
		ID:        nostrEvent.ID,
		PubKey:    nostrEvent.PubKey,
		CreatedAt: createdAt,
		Kind:      nostrEvent.Kind,
		Tags:      [][]string{{"t", "nostr"}},
		Content:   nostrEvent.Content,
		Sig:       nostrEvent.Sig,
	}

	validator := NewValidator(5 * time.Minute)
	if err := validator.ValidateEvent(event); err != nil {
		t.Fatalf("expected valid event, got: %v", err)
	}

	event.Sig = strings.Repeat("0", 128)
	if err := validator.ValidateEvent(event); err == nil {
		t.Fatalf("expected invalid signature error, got nil")
	}
}
