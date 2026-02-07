package lib

import "sync"

// Metrics is a tiny in-memory counter store for instrumentation hooks.
type Metrics struct {
	mu       sync.RWMutex
	counters map[string]uint64
}

func NewMetrics() *Metrics {
	return &Metrics{counters: make(map[string]uint64)}
}

func (m *Metrics) Inc(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name]++
}

func (m *Metrics) Snapshot() map[string]uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make(map[string]uint64, len(m.counters))
	for k, v := range m.counters {
		cp[k] = v
	}
	return cp
}
