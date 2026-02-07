package lib

import "testing"

func TestNewLogger(t *testing.T) {
	levels := []string{"DEBUG", "WARN", "ERROR", "UNKNOWN"}
	for _, level := range levels {
		if logger := NewLogger(level); logger == nil {
			t.Fatalf("NewLogger(%q) returned nil", level)
		}
	}
}
