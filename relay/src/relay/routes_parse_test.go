package relay

import (
	"net/http"
	"net/url"
	"testing"
)

func TestParseEventFilter(t *testing.T) {
	req := &http.Request{URL: &url.URL{RawQuery: "author=pubkey&kind=1&since=10&until=20&tag=t:nostr&limit=25"}}

	filter, err := parseEventFilter(req)
	if err != nil {
		t.Fatalf("parseEventFilter returned error: %v", err)
	}
	if filter.Author != "pubkey" || filter.Tag != "t:nostr" || filter.Limit != 25 {
		t.Fatalf("unexpected parsed filter: %+v", filter)
	}
	if filter.Kind == nil || *filter.Kind != 1 {
		t.Fatalf("expected kind=1, got %+v", filter.Kind)
	}
	if filter.Since == nil || *filter.Since != 10 {
		t.Fatalf("expected since=10, got %+v", filter.Since)
	}
	if filter.Until == nil || *filter.Until != 20 {
		t.Fatalf("expected until=20, got %+v", filter.Until)
	}
}

func TestParseEventFilterRejectsInvalidNumbers(t *testing.T) {
	tests := []string{
		"kind=not-a-number",
		"since=not-a-number",
		"until=not-a-number",
		"limit=not-a-number",
	}

	for _, rawQuery := range tests {
		req := &http.Request{URL: &url.URL{RawQuery: rawQuery}}
		if _, err := parseEventFilter(req); err == nil {
			t.Fatalf("expected parseEventFilter to return error for %q", rawQuery)
		}
	}
}

func TestParseGroupFilter(t *testing.T) {
	req := &http.Request{URL: &url.URL{RawQuery: "geohash_prefix=abc&is_private=true&is_vetted=false&updated_since=123&limit=9"}}

	filter, err := parseGroupFilter(req)
	if err != nil {
		t.Fatalf("parseGroupFilter returned error: %v", err)
	}
	if filter.GeohashPrefix != "abc" || filter.Limit != 9 {
		t.Fatalf("unexpected parsed filter: %+v", filter)
	}
	if filter.IsPrivate == nil || !*filter.IsPrivate {
		t.Fatalf("expected is_private=true, got %+v", filter.IsPrivate)
	}
	if filter.IsVetted == nil || *filter.IsVetted {
		t.Fatalf("expected is_vetted=false, got %+v", filter.IsVetted)
	}
	if filter.UpdatedSince == nil || *filter.UpdatedSince != 123 {
		t.Fatalf("expected updated_since=123, got %+v", filter.UpdatedSince)
	}
}

func TestParseGroupFilterRejectsInvalidNumbers(t *testing.T) {
	tests := []string{
		"is_private=not-a-bool",
		"is_vetted=not-a-bool",
		"updated_since=not-a-number",
		"limit=not-a-number",
	}

	for _, rawQuery := range tests {
		req := &http.Request{URL: &url.URL{RawQuery: rawQuery}}
		if _, err := parseGroupFilter(req); err == nil {
			t.Fatalf("expected parseGroupFilter to return error for %q", rawQuery)
		}
	}
}

func TestSplitPath(t *testing.T) {
	parts := splitPath("/group-1/join-requests/user/approve/")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts, got %d (%v)", len(parts), parts)
	}
	if parts[0] != "group-1" || parts[1] != "join-requests" || parts[2] != "user" || parts[3] != "approve" {
		t.Fatalf("unexpected split parts: %v", parts)
	}
}

func TestSplitPathEmptyInput(t *testing.T) {
	tests := []struct {
		path     string
		wantNil  bool
		wantSize int
	}{
		{path: "", wantNil: true},
		{path: "/", wantSize: 0},
		{path: "   ", wantNil: true},
	}

	for _, tc := range tests {
		parts := splitPath(tc.path)
		if tc.wantNil {
			if parts != nil {
				t.Fatalf("splitPath(%q) = %v, want nil", tc.path, parts)
			}
			continue
		}
		if len(parts) != tc.wantSize {
			t.Fatalf("splitPath(%q) length = %d, want %d", tc.path, len(parts), tc.wantSize)
		}
	}
}
