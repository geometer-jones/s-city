package services

import (
	"strings"
	"testing"

	"s-city/src/models"
)

func TestRequiredPowBits(t *testing.T) {
	controls := NewAbuseControls(5, 60, 11)

	if got := controls.RequiredPowBits(9007); got != 28 {
		t.Fatalf("RequiredPowBits(9007) = %d, want 28", got)
	}
	if got := controls.RequiredPowBits(42); got != 11 {
		t.Fatalf("RequiredPowBits(42) = %d, want 11", got)
	}
}

func TestValidatePowBranches(t *testing.T) {
	controls := NewAbuseControls(5, 60, 0)

	t.Run("disabled requirement allows invalid id", func(t *testing.T) {
		err := controls.ValidatePow(models.Event{ID: "zz"}, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("invalid id is rejected when pow is required", func(t *testing.T) {
		err := controls.ValidatePow(models.Event{ID: "zz"}, 4)
		if err == nil || !strings.Contains(err.Error(), "invalid event id for pow") {
			t.Fatalf("expected invalid pow id error, got %v", err)
		}
	})

	t.Run("insufficient leading bits is rejected", func(t *testing.T) {
		err := controls.ValidatePow(models.Event{ID: "0f"}, 8)
		if err == nil || !strings.Contains(err.Error(), "insufficient pow") {
			t.Fatalf("expected insufficient pow error, got %v", err)
		}
	})

	t.Run("sufficient leading bits with matching nonce target succeeds", func(t *testing.T) {
		err := controls.ValidatePow(models.Event{
			ID:   "0f",
			Tags: [][]string{{"nonce", "123", "4"}},
		}, 4)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})
}
