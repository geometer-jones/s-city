package storage

import "testing"

func TestRoleHasPermission(t *testing.T) {
	tests := []struct {
		name        string
		roleName    string
		permissions []string
		required    string
		want        bool
	}{
		{
			name:     "owner has all permissions",
			roleName: "owner",
			required: "remove-user",
			want:     true,
		},
		{
			name:     "admin has default permissions",
			roleName: "admin",
			required: "delete-group",
			want:     true,
		},
		{
			name:     "admin default denies unknown permissions",
			roleName: "admin",
			required: "foo",
			want:     false,
		},
		{
			name:        "custom role with explicit permission",
			roleName:    "moderator",
			permissions: []string{"remove-user"},
			required:    "remove-user",
			want:        true,
		},
		{
			name:        "custom role without permission",
			roleName:    "moderator",
			permissions: []string{"create-invite"},
			required:    "remove-user",
			want:        false,
		},
		{
			name:        "admin permission flag grants all",
			roleName:    "moderator",
			permissions: []string{"admin"},
			required:    "delete-event",
			want:        true,
		},
		{
			name:        "permission matching is normalized",
			roleName:    "moderator",
			permissions: []string{"  Remove-User  "},
			required:    "remove-user",
			want:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := roleHasPermission(tc.roleName, tc.permissions, tc.required)
			if got != tc.want {
				t.Fatalf("roleHasPermission(%q, %v, %q) = %v, want %v", tc.roleName, tc.permissions, tc.required, got, tc.want)
			}
		})
	}
}
