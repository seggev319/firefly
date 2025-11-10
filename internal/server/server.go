package server

import (
	"net/http"
	"time"

	"github.com/seggev/firefly/internal/handlers"
)

// New constructs an http.Server with sane timeouts and routes registered.
func New(mux *http.ServeMux, addr string) *http.Server {
	registerRoutes(mux)

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("firefly is running\n"))
	})
}
