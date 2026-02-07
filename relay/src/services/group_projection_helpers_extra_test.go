package services

import (
	"context"
	"testing"

	"s-city/src/models"
)

func TestParseInt64Tag(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int64
	}{
		{name: "valid", raw: "42", want: 42},
		{name: "invalid", raw: "not-a-number", want: 0},
		{name: "empty", raw: "", want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseInt64Tag(tc.raw); got != tc.want {
				t.Fatalf("parseInt64Tag(%q) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}
}

func TestParseCSVTag(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "comma separated", raw: "a, b,c", want: []string{"a", "b", "c"}},
		{name: "empty", raw: "", want: nil},
		{name: "only whitespace entries", raw: " , ", want: []string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCSVTag(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("parseCSVTag(%q) length = %d, want %d (%v)", tc.raw, len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("parseCSVTag(%q)[%d] = %q, want %q", tc.raw, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestDefaultString(t *testing.T) {
	if got := defaultString("value", "fallback"); got != "value" {
		t.Fatalf("defaultString returned %q, want value", got)
	}
	if got := defaultString("   ", "fallback"); got != "fallback" {
		t.Fatalf("defaultString returned %q, want fallback", got)
	}
}

func TestOwnerRolePermissions(t *testing.T) {
	perms := ownerRolePermissions()
	if len(perms) == 0 {
		t.Fatalf("ownerRolePermissions returned empty permissions")
	}

	want := map[string]bool{
		models.PermissionAddUser:      false,
		models.PermissionPromoteUser:  false,
		models.PermissionRemoveUser:   false,
		models.PermissionEditMetadata: false,
		models.PermissionCreateRole:   false,
		models.PermissionDeleteRole:   false,
		models.PermissionDeleteEvent:  false,
		models.PermissionCreateGroup:  false,
		models.PermissionDeleteGroup:  false,
		models.PermissionCreateInvite: false,
	}

	for _, perm := range perms {
		if _, ok := want[perm]; ok {
			want[perm] = true
		}
	}

	for perm, found := range want {
		if !found {
			t.Fatalf("expected owner permission %q not found in %v", perm, perms)
		}
	}
}

func TestSyncCanonicalStateEventsSkipsWithoutRelaySigningContext(t *testing.T) {
	svc := &GroupProjectionService{}
	err := svc.syncCanonicalStateEvents(context.Background(), models.Event{Kind: 9007, CreatedAt: 1}, "group-1", true, true)
	if err != nil {
		t.Fatalf("expected no error when canonical sync is disabled, got %v", err)
	}
}

func TestNormalizeHelpers(t *testing.T) {
	if got := normalizeRoleName("  Admin "); got != "admin" {
		t.Fatalf("normalizeRoleName returned %q", got)
	}
	if got := normalizePermission("  Remove-User "); got != "remove-user" {
		t.Fatalf("normalizePermission returned %q", got)
	}
}
