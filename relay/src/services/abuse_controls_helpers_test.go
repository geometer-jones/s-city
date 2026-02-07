package services

import (
	"testing"

	"s-city/src/models"
)

func TestExtractNonceDifficulty(t *testing.T) {
	tags := [][]string{{"nonce", "123", "20"}}
	if got := extractNonceDifficulty(tags); got != 20 {
		t.Fatalf("extractNonceDifficulty = %d, want 20", got)
	}

	tags = [][]string{{"nonce", "123", "bad"}}
	if got := extractNonceDifficulty(tags); got != 0 {
		t.Fatalf("extractNonceDifficulty invalid bits = %d, want 0", got)
	}
}

func TestLeadingZeroBits(t *testing.T) {
	tests := []struct {
		name      string
		hexID     string
		wantBits  int
		wantError bool
	}{
		{name: "all zeros", hexID: "0000", wantBits: 16},
		{name: "single leading zero nibble", hexID: "0f", wantBits: 4},
		{name: "odd length", hexID: "abc", wantError: true},
		{name: "invalid hex", hexID: "zz", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := leadingZeroBits(tc.hexID)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("leadingZeroBits returned error: %v", err)
			}
			if got != tc.wantBits {
				t.Fatalf("leadingZeroBits(%q) = %d, want %d", tc.hexID, got, tc.wantBits)
			}
		})
	}
}

func TestMinFloat(t *testing.T) {
	if got := minFloat(1.0, 2.0); got != 1.0 {
		t.Fatalf("minFloat returned %v, want 1", got)
	}
	if got := minFloat(2.0, 1.0); got != 1.0 {
		t.Fatalf("minFloat returned %v, want 1", got)
	}
}

func TestValidatePowChecksNonceDifficulty(t *testing.T) {
	controls := NewAbuseControls(1, 1, 0)
	event := models.Event{
		ID:   "000f",
		Tags: [][]string{{"nonce", "123", "4"}},
	}

	err := controls.ValidatePow(event, 8)
	if err == nil {
		t.Fatalf("expected nonce-difficulty validation error")
	}
}
