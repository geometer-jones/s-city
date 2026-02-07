package services

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"s-city/src/models"
)

type rateBucket struct {
	tokens     float64
	lastRefill time.Time
}

// AbuseControls enforces per-pubkey rate limits and PoW minimums.
type AbuseControls struct {
	burst              int
	sustainedPerMinute int
	defaultPowBits     int

	mu      sync.Mutex
	buckets map[string]*rateBucket
}

func NewAbuseControls(burst, sustainedPerMinute, defaultPowBits int) *AbuseControls {
	return &AbuseControls{
		burst:              burst,
		sustainedPerMinute: sustainedPerMinute,
		defaultPowBits:     defaultPowBits,
		buckets:            make(map[string]*rateBucket),
	}
}

func (a *AbuseControls) Allow(pubKey string, now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	bucket, ok := a.buckets[pubKey]
	if !ok {
		bucket = &rateBucket{tokens: float64(a.burst), lastRefill: now}
		a.buckets[pubKey] = bucket
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	if elapsed > 0 {
		refillRate := float64(a.sustainedPerMinute) / 60.0
		bucket.tokens = minFloat(float64(a.burst), bucket.tokens+elapsed*refillRate)
		bucket.lastRefill = now
	}

	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

func (a *AbuseControls) RequiredPowBits(kind int) int {
	if bits, ok := kindPowBits[kind]; ok {
		return bits
	}
	return a.defaultPowBits
}

func (a *AbuseControls) ValidatePow(event models.Event, requiredBits int) error {
	if requiredBits <= 0 {
		return nil
	}

	powBits, err := leadingZeroBits(event.ID)
	if err != nil {
		return fmt.Errorf("invalid event id for pow: %w", err)
	}
	if powBits < requiredBits {
		return fmt.Errorf("insufficient pow: have %d bits, need %d", powBits, requiredBits)
	}

	tagDifficulty := extractNonceDifficulty(event.Tags)
	if tagDifficulty > 0 && tagDifficulty < requiredBits {
		return fmt.Errorf("pow nonce tag difficulty below required target")
	}
	return nil
}

var kindPowBits = map[int]int{
	9007:  28,
	1020:  24,
	0:     20,
	30022: 16,
	20002: 12,
	10006: 12,
	20011: 8,
	20012: 8,
}

func extractNonceDifficulty(tags [][]string) int {
	for _, tag := range tags {
		if len(tag) < 3 {
			continue
		}
		if tag[0] != "nonce" {
			continue
		}
		bits, err := strconv.Atoi(tag[2])
		if err == nil && bits > 0 {
			return bits
		}
	}
	return 0
}

func leadingZeroBits(hexID string) (int, error) {
	hexID = strings.TrimSpace(hexID)
	if len(hexID)%2 != 0 {
		return 0, fmt.Errorf("odd hex length")
	}

	bytes := make([]byte, len(hexID)/2)
	for i := 0; i < len(hexID); i += 2 {
		v, err := strconv.ParseUint(hexID[i:i+2], 16, 8)
		if err != nil {
			return 0, err
		}
		bytes[i/2] = byte(v)
	}

	bits := 0
	for _, b := range bytes {
		if b == 0 {
			bits += 8
			continue
		}
		for i := 7; i >= 0; i-- {
			if (b>>i)&1 == 0 {
				bits++
				continue
			}
			return bits, nil
		}
	}
	return bits, nil
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
