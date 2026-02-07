package lib

import (
	"strings"
	"testing"
	"time"
)

func TestGetOrDefault(t *testing.T) {
	const key = "SCITY_TEST_STRING"
	t.Setenv(key, "")
	if got := getOrDefault(key, "fallback"); got != "fallback" {
		t.Fatalf("getOrDefault empty returned %q, want fallback", got)
	}
	t.Setenv(key, "value")
	if got := getOrDefault(key, "fallback"); got != "value" {
		t.Fatalf("getOrDefault returned %q, want value", got)
	}
}

func TestGetIntOrDefault(t *testing.T) {
	const key = "SCITY_TEST_INT"
	t.Setenv(key, "")
	if got := getIntOrDefault(key, 7); got != 7 {
		t.Fatalf("getIntOrDefault empty returned %d, want 7", got)
	}
	t.Setenv(key, "abc")
	if got := getIntOrDefault(key, 7); got != 7 {
		t.Fatalf("getIntOrDefault invalid returned %d, want 7", got)
	}
	t.Setenv(key, "42")
	if got := getIntOrDefault(key, 7); got != 42 {
		t.Fatalf("getIntOrDefault returned %d, want 42", got)
	}
}

func TestLoadConfig(t *testing.T) {
	relayPriv := "19f43f4ef72a9f5f1385d7caec9da7d769e5f7969b2f5b98d6af95f7ce0d4d95"
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("RELAY_PRIVKEY", relayPriv)
	t.Setenv("RATE_LIMIT_BURST", "9")
	t.Setenv("RATE_LIMIT_PER_MIN", "60")
	t.Setenv("MAX_EVENT_SKEW_SECONDS", "120")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.RelayPubKey == "" {
		t.Fatalf("expected derived relay pubkey")
	}
	if cfg.RateLimitBurst != 9 || cfg.RateLimitPerMinute != 60 {
		t.Fatalf("unexpected rate limits: %+v", cfg)
	}
	if cfg.MaxEventSkew != 120*time.Second {
		t.Fatalf("unexpected skew: %v", cfg.MaxEventSkew)
	}
}

func TestLoadConfigRejectsInvalidEnvironment(t *testing.T) {
	const relayPriv = "19f43f4ef72a9f5f1385d7caec9da7d769e5f7969b2f5b98d6af95f7ce0d4d95"

	tests := []struct {
		name    string
		mutate  func(t *testing.T)
		wantErr string
	}{
		{
			name: "missing database url",
			mutate: func(t *testing.T) {
				t.Setenv("DATABASE_URL", "")
			},
			wantErr: "DATABASE_URL is required",
		},
		{
			name: "missing relay private key",
			mutate: func(t *testing.T) {
				t.Setenv("RELAY_PRIVKEY", "")
			},
			wantErr: "RELAY_PRIVKEY is required",
		},
		{
			name: "invalid relay private key",
			mutate: func(t *testing.T) {
				t.Setenv("RELAY_PRIVKEY", "not-a-valid-private-key")
			},
			wantErr: "RELAY_PRIVKEY is invalid",
		},
		{
			name: "relay pubkey mismatch",
			mutate: func(t *testing.T) {
				t.Setenv("RELAY_PUBKEY", "deadbeef")
			},
			wantErr: "RELAY_PUBKEY does not match RELAY_PRIVKEY",
		},
		{
			name: "non-positive burst",
			mutate: func(t *testing.T) {
				t.Setenv("RATE_LIMIT_BURST", "0")
			},
			wantErr: "RATE_LIMIT_BURST must be > 0",
		},
		{
			name: "non-positive sustained limit",
			mutate: func(t *testing.T) {
				t.Setenv("RATE_LIMIT_PER_MIN", "0")
			},
			wantErr: "RATE_LIMIT_PER_MIN must be > 0",
		},
		{
			name: "non-positive max skew",
			mutate: func(t *testing.T) {
				t.Setenv("MAX_EVENT_SKEW_SECONDS", "0")
			},
			wantErr: "MAX_EVENT_SKEW_SECONDS must be > 0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
			t.Setenv("RELAY_PRIVKEY", relayPriv)
			t.Setenv("RELAY_PUBKEY", "")
			t.Setenv("RATE_LIMIT_BURST", "9")
			t.Setenv("RATE_LIMIT_PER_MIN", "60")
			t.Setenv("MAX_EVENT_SKEW_SECONDS", "120")

			tc.mutate(t)

			_, err := LoadConfig()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("LoadConfig error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}
