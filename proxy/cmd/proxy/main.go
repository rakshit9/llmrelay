package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	// Structured JSON logger — like Python's structlog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	mux := http.NewServeMux()

	// GET /health — returns {"status":"ok"}
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		slog.Info("health check", "method", r.Method, "path", r.URL.Path)
	})

	addr := ":8080"
	slog.Info("proxy starting", "addr", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
