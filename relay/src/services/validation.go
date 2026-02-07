package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"s-city/src/models"
)

var (
	hex64  = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	hex128 = regexp.MustCompile(`^[0-9a-fA-F]{128}$`)
)

// Validator enforces baseline Nostr event validity checks.
type Validator struct {
	maxSkew time.Duration
}

func NewValidator(maxSkew time.Duration) *Validator {
	return &Validator{maxSkew: maxSkew}
}

func (v *Validator) ValidateEvent(event models.Event) error {
	if !hex64.MatchString(event.ID) {
		return fmt.Errorf("invalid event id")
	}
	if !hex64.MatchString(event.PubKey) {
		return fmt.Errorf("invalid event pubkey")
	}
	if !hex128.MatchString(event.Sig) {
		return fmt.Errorf("invalid event signature format")
	}
	if event.CreatedAt == 0 {
		return fmt.Errorf("event created_at is required")
	}

	now := time.Now().Unix()
	skew := now - event.CreatedAt
	if skew < 0 {
		skew = -skew
	}
	if time.Duration(skew)*time.Second > v.maxSkew {
		return fmt.Errorf("event created_at out of allowed skew")
	}

	for i, tag := range event.Tags {
		if len(tag) == 0 {
			return fmt.Errorf("tag[%d] is empty", i)
		}
		if strings.TrimSpace(tag[0]) == "" {
			return fmt.Errorf("tag[%d] has empty name", i)
		}
	}

	if err := validateEventID(event); err != nil {
		return err
	}
	if err := validateSignatureFields(event); err != nil {
		return err
	}

	return nil
}

func validateEventID(event models.Event) error {
	expected, err := ComputeEventID(event.PubKey, event.CreatedAt, event.Kind, event.Tags, event.Content)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, event.ID) {
		return fmt.Errorf("event id does not match payload")
	}
	return nil
}

// ComputeEventID calculates the canonical Nostr event ID hash.
func ComputeEventID(pubKey string, createdAt int64, kind int, tags [][]string, content string) (string, error) {
	payload := []any{0, strings.ToLower(pubKey), createdAt, kind, tags, content}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal canonical event payload: %w", err)
	}

	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func validateSignatureFields(event models.Event) error {
	if !hex128.MatchString(event.Sig) {
		return fmt.Errorf("invalid signature")
	}

	nostrEvent := nostr.Event{
		ID:        strings.ToLower(event.ID),
		PubKey:    strings.ToLower(event.PubKey),
		CreatedAt: nostr.Timestamp(event.CreatedAt),
		Kind:      event.Kind,
		Content:   event.Content,
		Sig:       strings.ToLower(event.Sig),
	}
	nostrTags := make(nostr.Tags, 0, len(event.Tags))
	for _, tag := range event.Tags {
		nostrTag := make(nostr.Tag, len(tag))
		copy(nostrTag, tag)
		nostrTags = append(nostrTags, nostrTag)
	}
	nostrEvent.Tags = nostrTags

	ok, err := nostrEvent.CheckSignature()
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}
	if !ok {
		return fmt.Errorf("invalid signature")
	}
	return nil
}
