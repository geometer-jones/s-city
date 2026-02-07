package lib

import "testing"

func TestMetricsIncAndSnapshot(t *testing.T) {
	m := NewMetrics()
	m.Inc("events")
	m.Inc("events")

	snap := m.Snapshot()
	if snap["events"] != 2 {
		t.Fatalf("events counter = %d, want 2", snap["events"])
	}

	snap["events"] = 100
	snap2 := m.Snapshot()
	if snap2["events"] != 2 {
		t.Fatalf("snapshot should be a copy; got %d", snap2["events"])
	}
}
