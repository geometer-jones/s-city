package storage

import "testing"

func TestParseTagFilter(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantName  string
		wantValue string
	}{
		{name: "named tag", raw: "t:nostr", wantName: "t", wantValue: "nostr"},
		{name: "unnamed tag", raw: "nostr", wantName: "", wantValue: "nostr"},
		{name: "trims spaces", raw: " p : pubkey ", wantName: "p", wantValue: "pubkey"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotName, gotValue := parseTagFilter(tc.raw)
			if gotName != tc.wantName || gotValue != tc.wantValue {
				t.Fatalf("parseTagFilter(%q) = (%q, %q), want (%q, %q)", tc.raw, gotName, gotValue, tc.wantName, tc.wantValue)
			}
		})
	}
}
