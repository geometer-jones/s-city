package storage

import "testing"

func TestCompareReplaceableVersion(t *testing.T) {
	tests := []struct {
		name       string
		createdAtA int64
		idA        string
		createdAtB int64
		idB        string
		want       int
	}{
		{
			name:       "newer timestamp wins",
			createdAtA: 200,
			idA:        "bbbb",
			createdAtB: 100,
			idB:        "aaaa",
			want:       1,
		},
		{
			name:       "older timestamp loses",
			createdAtA: 100,
			idA:        "aaaa",
			createdAtB: 200,
			idB:        "bbbb",
			want:       -1,
		},
		{
			name:       "equal timestamp lower id wins",
			createdAtA: 100,
			idA:        "0001",
			createdAtB: 100,
			idB:        "0002",
			want:       1,
		},
		{
			name:       "equal timestamp higher id loses",
			createdAtA: 100,
			idA:        "0002",
			createdAtB: 100,
			idB:        "0001",
			want:       -1,
		},
		{
			name:       "equal timestamp equal id is tie",
			createdAtA: 100,
			idA:        "ABCD",
			createdAtB: 100,
			idB:        "abcd",
			want:       0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := compareReplaceableVersion(tc.createdAtA, tc.idA, tc.createdAtB, tc.idB)
			if got != tc.want {
				t.Fatalf(
					"compareReplaceableVersion(%d,%q,%d,%q) = %d, want %d",
					tc.createdAtA, tc.idA, tc.createdAtB, tc.idB, got, tc.want,
				)
			}
		})
	}
}
