package tests

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"s-city/src/lib"
	relayserver "s-city/src/relay"
)

func TestServerLifecycle(t *testing.T) {
	withRelayRootCWD(t)

	relayPriv, relayPub := generateKeypair(t)
	addr := freeTCPAddr(t)
	cfg := lib.Config{
		DatabaseURL:        "postgres://s_city:s_city@localhost:5432/s_city?sslmode=disable",
		RelayPubKey:        relayPub,
		RelayPrivKey:       relayPriv,
		HTTPAddr:           addr,
		LogLevel:           "ERROR",
		RateLimitBurst:     30,
		RateLimitPerMinute: 120,
		DefaultPowBits:     0,
		MaxEventSkew:       5 * time.Minute,
	}

	srv, err := relayserver.NewServer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	baseURL := "http://" + addr
	waitForHTTP(t, baseURL+"/health")

	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var health map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("decode /health: %v", err)
	}
	if health["status"] != "ok" {
		t.Fatalf("unexpected health body: %v", health)
	}

	metricsResp, err := http.Get(baseURL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer metricsResp.Body.Close()
	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want %d", metricsResp.StatusCode, http.StatusOK)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start returned error after shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for Start to return after shutdown")
	}
}

func withRelayRootCWD(t *testing.T) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve current test path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir to relay root %q: %v", root, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prev)
	})
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close free-port listener: %v", err)
	}
	return addr
}

func waitForHTTP(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for server at %s", url)
}

func TestServerRejectsBadConfig(t *testing.T) {
	withRelayRootCWD(t)

	cfg := lib.Config{
		DatabaseURL:        "://bad-url",
		RelayPubKey:        "",
		RelayPrivKey:       "deadbeef",
		HTTPAddr:           "127.0.0.1:0",
		LogLevel:           "ERROR",
		RateLimitBurst:     30,
		RateLimitPerMinute: 120,
		DefaultPowBits:     0,
		MaxEventSkew:       5 * time.Minute,
	}

	if _, err := relayserver.NewServer(context.Background(), cfg); err == nil {
		t.Fatalf("expected NewServer to fail with invalid config")
	}
}
