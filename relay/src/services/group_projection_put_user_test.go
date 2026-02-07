package services

import (
	"testing"

	"s-city/src/models"
)

func TestParsePutUserTag(t *testing.T) {
	tests := []struct {
		name      string
		tags      [][]string
		wantPub   string
		wantRole  string
		wantError bool
	}{
		{
			name:     "parses p tag with role",
			tags:     [][]string{{"h", "group-1"}, {"p", "pubkey-1", "moderator"}},
			wantPub:  "pubkey-1",
			wantRole: "moderator",
		},
		{
			name:     "parses p tag without role",
			tags:     [][]string{{"p", "pubkey-2"}},
			wantPub:  "pubkey-2",
			wantRole: "",
		},
		{
			name:      "fails when p tag is missing",
			tags:      [][]string{{"h", "group-1"}},
			wantError: true,
		},
		{
			name:      "fails when p pubkey is empty",
			tags:      [][]string{{"p", "   ", "member"}},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPub, gotRole, err := parsePutUserTag(tc.tags)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePutUserTag returned error: %v", err)
			}
			if gotPub != tc.wantPub {
				t.Fatalf("pubkey = %q, want %q", gotPub, tc.wantPub)
			}
			if gotRole != tc.wantRole {
				t.Fatalf("role = %q, want %q", gotRole, tc.wantRole)
			}
		})
	}
}

func TestHasTag(t *testing.T) {
	tags := [][]string{
		{"h", "group-1"},
		{"ban"},
		{"p", "pubkey-1", "member"},
	}

	if !hasTag(tags, "ban") {
		t.Fatalf("expected ban tag to be detected")
	}
	if hasTag(tags, "missing") {
		t.Fatalf("did not expect missing tag to be detected")
	}
}

func TestJoinRequestPubKey(t *testing.T) {
	tests := []struct {
		name      string
		event     models.Event
		wantKey   string
		wantError bool
	}{
		{
			name: "defaults to author pubkey when p is absent",
			event: models.Event{
				PubKey: "pubkey-1",
				Tags:   [][]string{{"h", "group-1"}},
			},
			wantKey: "pubkey-1",
		},
		{
			name: "allows matching p tag",
			event: models.Event{
				PubKey: "pubkey-1",
				Tags:   [][]string{{"h", "group-1"}, {"p", "pubkey-1"}},
			},
			wantKey: "pubkey-1",
		},
		{
			name: "rejects mismatched p tag",
			event: models.Event{
				PubKey: "pubkey-1",
				Tags:   [][]string{{"h", "group-1"}, {"p", "pubkey-2"}},
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := joinRequestPubKey(tc.event)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("joinRequestPubKey returned error: %v", err)
			}
			if got != tc.wantKey {
				t.Fatalf("joinRequestPubKey = %q, want %q", got, tc.wantKey)
			}
		})
	}
}

func TestTagBoolValue(t *testing.T) {
	tests := []struct {
		name      string
		tags      [][]string
		tagName   string
		wantValue bool
		wantFound bool
	}{
		{
			name:      "presence tag is true",
			tags:      [][]string{{"private"}},
			tagName:   "private",
			wantValue: true,
			wantFound: true,
		},
		{
			name:      "empty value is true",
			tags:      [][]string{{"private", ""}},
			tagName:   "private",
			wantValue: true,
			wantFound: true,
		},
		{
			name:      "explicit false is false",
			tags:      [][]string{{"private", "false"}},
			tagName:   "private",
			wantValue: false,
			wantFound: true,
		},
		{
			name:      "missing tag is not found",
			tags:      [][]string{{"other", "true"}},
			tagName:   "private",
			wantValue: false,
			wantFound: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotValue, gotFound := tagBoolValue(tc.tags, tc.tagName)
			if gotFound != tc.wantFound || gotValue != tc.wantValue {
				t.Fatalf("tagBoolValue(%v, %q) = (%v, %v), want (%v, %v)", tc.tags, tc.tagName, gotValue, gotFound, tc.wantValue, tc.wantFound)
			}
		})
	}
}
