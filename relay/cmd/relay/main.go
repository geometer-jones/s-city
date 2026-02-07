package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"s-city/src/lib"
	"s-city/src/relay"
)

func main() {
	cfg, err := lib.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	server, err := relay.NewServer(ctx, cfg)
	if err != nil {
		log.Fatalf("bootstrap server: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("received signal: %s", sig)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("graceful shutdown failed: %v", err)
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Fatalf("server exited with error: %v", err)
		}
	}
}
