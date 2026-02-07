package services

import (
	"testing"

	"s-city/src/models"
)

func TestCanonicalStateKindsForSource(t *testing.T) {
	tests := []struct {
		name              string
		sourceKind        int
		membershipChanged bool
		adminsChanged     bool
		want              []int
	}{
		{
			name:              "create group emits metadata members roles admins",
			sourceKind:        9007,
			membershipChanged: true,
			adminsChanged:     true,
			want:              []int{39000, 39002, 39003, 39001},
		},
		{
			name:       "edit metadata emits group metadata",
			sourceKind: 9002,
			want:       []int{39000},
		},
		{
			name:       "create role emits roles",
			sourceKind: 9003,
			want:       []int{39003},
		},
		{
			name:              "put user emits members",
			sourceKind:        9000,
			membershipChanged: true,
			adminsChanged:     false,
			want:              []int{39002},
		},
		{
			name:              "put user admin change emits members and admins",
			sourceKind:        9000,
			membershipChanged: true,
			adminsChanged:     true,
			want:              []int{39002, 39001},
		},
		{
			name:              "join request without auto-approval emits nothing",
			sourceKind:        9021,
			membershipChanged: false,
			want:              []int{},
		},
		{
			name:              "join request with auto-approval emits members",
			sourceKind:        9021,
			membershipChanged: true,
			want:              []int{39002},
		},
		{
			name:              "remove user emits members",
			sourceKind:        9001,
			membershipChanged: true,
			want:              []int{39002},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := canonicalStateKindsForSource(tc.sourceKind, tc.membershipChanged, tc.adminsChanged)
			if len(got) != len(tc.want) {
				t.Fatalf("kinds length = %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("kinds[%d] = %d, want %d (%v)", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}

func TestAdminAssignmentChanged(t *testing.T) {
	rolePermissions := map[string][]string{
		"member":      nil,
		"moderator":   nil,
		"ops-admin":   {"read", "admin"},
		"ops-admin-2": {"admin"},
	}

	tests := []struct {
		name          string
		previousRole  string
		requestedRole string
		want          bool
	}{
		{
			name:          "member to member no admin change",
			previousRole:  "member",
			requestedRole: "member",
			want:          false,
		},
		{
			name:          "member to admin role triggers change",
			previousRole:  "member",
			requestedRole: "admin",
			want:          true,
		},
		{
			name:          "admin role to member triggers change",
			previousRole:  "admin",
			requestedRole: "member",
			want:          true,
		},
		{
			name:          "admin role to owner triggers change",
			previousRole:  "admin",
			requestedRole: "owner",
			want:          true,
		},
		{
			name:          "custom admin-permission role assignment triggers change",
			previousRole:  "member",
			requestedRole: "ops-admin",
			want:          true,
		},
		{
			name:          "swap between custom admin-permission roles triggers change",
			previousRole:  "ops-admin",
			requestedRole: "ops-admin-2",
			want:          true,
		},
		{
			name:          "non-admin role change does not trigger change",
			previousRole:  "member",
			requestedRole: "moderator",
			want:          false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := adminAssignmentChanged(tc.previousRole, tc.requestedRole, rolePermissions)
			if got != tc.want {
				t.Fatalf("adminAssignmentChanged(%q, %q) = %v, want %v", tc.previousRole, tc.requestedRole, got, tc.want)
			}
		})
	}
}

func TestGroupMetadataStateTagsUsePresenceBooleans(t *testing.T) {
	group := models.Group{
		GroupID:      "group-1",
		Name:         "Name",
		About:        "About",
		Picture:      "https://example.com/p.png",
		Geohash:      "abcdef",
		IsPrivate:    true,
		IsRestricted: true,
		IsVetted:     true,
		IsHidden:     true,
		IsClosed:     true,
	}

	tags := groupMetadataStateTags(group)
	assertTagPresent(t, tags, []string{"d", "group-1"})
	assertTagPresent(t, tags, []string{"name", "Name"})
	assertTagPresent(t, tags, []string{"about", "About"})
	assertTagPresent(t, tags, []string{"picture", "https://example.com/p.png"})
	assertTagPresent(t, tags, []string{"g", "abcdef"})
	assertTagPresent(t, tags, []string{"private"})
	assertTagPresent(t, tags, []string{"restricted"})
	assertTagPresent(t, tags, []string{"vetted"})
	assertTagPresent(t, tags, []string{"hidden"})
	assertTagPresent(t, tags, []string{"closed"})

	for _, tag := range tags {
		switch tag[0] {
		case "private", "restricted", "vetted", "hidden", "closed":
			if len(tag) != 1 {
				t.Fatalf("expected %q to be presence-style tag, got %v", tag[0], tag)
			}
		}
	}
}

func TestGroupMembersStateTagsContainOnlyPubkeys(t *testing.T) {
	tags := groupMembersStateTags("group-1", []models.GroupMember{
		{PubKey: "pubkey-1", RoleName: "admin"},
		{PubKey: "pubkey-2", RoleName: "member"},
	})

	assertTagPresent(t, tags, []string{"d", "group-1"})
	assertTagPresent(t, tags, []string{"p", "pubkey-1"})
	assertTagPresent(t, tags, []string{"p", "pubkey-2"})
	for _, tag := range tags {
		if len(tag) > 0 && tag[0] == "p" && len(tag) != 2 {
			t.Fatalf("expected member p tag to contain only pubkey, got %v", tag)
		}
	}
}

func TestGroupRolesStateTagsContainNameAndOptionalDescription(t *testing.T) {
	tags := groupRolesStateTags("group-1", []models.GroupRole{
		{RoleName: "admin", Description: "administrators", Permissions: []string{"admin", "delete-event"}},
		{RoleName: "helper", Description: ""},
	})

	assertTagPresent(t, tags, []string{"d", "group-1"})
	assertTagPresent(t, tags, []string{"role", "admin", "administrators"})
	assertTagPresent(t, tags, []string{"role", "helper"})
	for _, tag := range tags {
		if len(tag) > 0 && tag[0] == "role" && len(tag) > 3 {
			t.Fatalf("expected role tag to contain only name/description, got %v", tag)
		}
	}
}

func assertTagPresent(t *testing.T, tags [][]string, want []string) {
	t.Helper()
	for _, tag := range tags {
		if len(tag) != len(want) {
			continue
		}
		matches := true
		for i := range want {
			if tag[i] != want[i] {
				matches = false
				break
			}
		}
		if matches {
			return
		}
	}
	t.Fatalf("expected tag %v not found in %v", want, tags)
}
