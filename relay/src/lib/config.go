package lib

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// Config contains runtime configuration loaded from environment variables.
type Config struct {
	DatabaseURL        string
	RelayPubKey        string
	RelayPrivKey       string
	HTTPAddr           string
	LogLevel           string
	RateLimitBurst     int
	RateLimitPerMinute int
	DefaultPowBits     int
	MaxEventSkew       time.Duration
}

func LoadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		RelayPubKey:        strings.ToLower(strings.TrimSpace(os.Getenv("RELAY_PUBKEY"))),
		RelayPrivKey:       strings.ToLower(strings.TrimSpace(os.Getenv("RELAY_PRIVKEY"))),
		HTTPAddr:           getOrDefault("HTTP_ADDR", ":8080"),
		LogLevel:           getOrDefault("LOG_LEVEL", "INFO"),
		RateLimitBurst:     getIntOrDefault("RATE_LIMIT_BURST", 30),
		RateLimitPerMinute: getIntOrDefault("RATE_LIMIT_PER_MIN", 120),
		DefaultPowBits:     getIntOrDefault("DEFAULT_POW_BITS", 0),
		MaxEventSkew:       time.Duration(getIntOrDefault("MAX_EVENT_SKEW_SECONDS", 300)) * time.Second,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.RelayPrivKey == "" {
		return Config{}, fmt.Errorf("RELAY_PRIVKEY is required")
	}

	derivedPubKey, err := nostr.GetPublicKey(cfg.RelayPrivKey)
	if err != nil {
		return Config{}, fmt.Errorf("RELAY_PRIVKEY is invalid: %w", err)
	}
	derivedPubKey = strings.ToLower(strings.TrimSpace(derivedPubKey))
	if cfg.RelayPubKey == "" {
		cfg.RelayPubKey = derivedPubKey
	}
	if !strings.EqualFold(cfg.RelayPubKey, derivedPubKey) {
		return Config{}, fmt.Errorf("RELAY_PUBKEY does not match RELAY_PRIVKEY")
	}
	if cfg.RateLimitBurst <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_BURST must be > 0")
	}
	if cfg.RateLimitPerMinute <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_PER_MIN must be > 0")
	}
	if cfg.MaxEventSkew <= 0 {
		return Config{}, fmt.Errorf("MAX_EVENT_SKEW_SECONDS must be > 0")
	}

	return cfg, nil
}

func getOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getIntOrDefault(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}
