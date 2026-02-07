package tests

import (
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/models"
)

func generateKeypair(t *testing.T) (priv string, pub string) {
	t.Helper()
	priv = nostr.GeneratePrivateKey()
	var err error
	pub, err = nostr.GetPublicKey(priv)
	if err != nil {
		t.Fatalf("derive pubkey: %v", err)
	}
	return priv, pub
}

func signedModelEvent(t *testing.T, priv string, createdAt int64, kind int, tags [][]string, content string) models.Event {
	t.Helper()

	pub, err := nostr.GetPublicKey(priv)
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
		PubKey:    pub,
		CreatedAt: nostr.Timestamp(createdAt),
		Kind:      kind,
		Tags:      nostrTags,
		Content:   content,
	}
	if err := evt.Sign(priv); err != nil {
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

func nowUnix() int64 {
	return time.Now().Unix()
}
