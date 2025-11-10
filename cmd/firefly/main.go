package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seggev/firefly/internal/server"
	"github.com/seggev/firefly/pkg/version"
)

func main() {
	port := getEnv("PORT", "8080")

	mux := http.NewServeMux()
	srv := server.New(mux, ":"+port)

	log.Printf("starting firefly version=%s commit=%s built_at=%s on port %s", version.Version, version.Commit, version.BuiltAt, port)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	// Wait for shutdown signal or server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("received signal: %s, shutting down...", sig)
	case err := <-errCh:
		// http.ErrServerClosed is expected on graceful shutdown
		if err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
