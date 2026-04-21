package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rakshit9/llmrelay/internal/auth"
	"github.com/rakshit9/llmrelay/internal/config"
	"github.com/rakshit9/llmrelay/internal/middleware"
	"github.com/rakshit9/llmrelay/internal/observability"
	"github.com/rakshit9/llmrelay/internal/router"
	"github.com/rakshit9/llmrelay/internal/upstream"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	// Register all configured providers
	providers := map[string]upstream.Provider{
		"openai": upstream.NewOpenAIProvider(cfg.OpenAIAPIKey),
	}
	if cfg.AnthropicAPIKey != "" {
		providers["anthropic"] = upstream.NewAnthropicProvider(cfg.AnthropicAPIKey)
	}
	if cfg.GoogleAPIKey != "" {
		providers["google"] = upstream.NewGoogleProvider(cfg.GoogleAPIKey)
	}
	if cfg.GroqAPIKey != "" {
		providers["groq"] = upstream.NewGroqProvider(cfg.GroqAPIKey)
	}

	rtr := router.New(providers)

	global := middleware.Chain(middleware.RequestID, middleware.Logger)
	requireAuth := auth.Require(cfg.GatewayAPIKey)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.Handle("GET /metrics", promhttp.Handler())

	mux.Handle("POST /v1/chat/completions", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := middleware.GetRequestID(r.Context())

		var req upstream.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Model == "" || len(req.Messages) == 0 {
			writeError(w, http.StatusBadRequest, "model and messages are required")
			return
		}

		// Streaming path
		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, ok := w.(http.Flusher)
			if !ok {
				writeError(w, http.StatusInternalServerError, "streaming not supported")
				return
			}

			provider, err := rtr.Stream(&req, w, flusher.Flush)
			latency := time.Since(start).Seconds()

			status := "200"
			if err != nil {
				slog.Error("stream error", "request_id", requestID, "err", err)
				status = "502"
			}

			observability.RequestsTotal.WithLabelValues(req.Model, provider, status).Inc()
			observability.RequestDuration.WithLabelValues(req.Model, provider).Observe(latency)
			return
		}

		// Non-streaming path
		resp, provider, err := rtr.Complete(&req)
		latency := time.Since(start).Seconds()

		if err != nil {
			slog.Error("upstream error", "request_id", requestID, "err", err)
			observability.RequestsTotal.WithLabelValues(req.Model, "", strconv.Itoa(http.StatusBadGateway)).Inc()
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		observability.RequestsTotal.WithLabelValues(req.Model, provider, "200").Inc()
		observability.RequestDuration.WithLabelValues(req.Model, provider).Observe(latency)
		observability.TokensTotal.WithLabelValues(req.Model, provider, "prompt").Add(float64(resp.Usage.PromptTokens))
		observability.TokensTotal.WithLabelValues(req.Model, provider, "completion").Add(float64(resp.Usage.CompletionTokens))

		slog.Info("chat complete",
			"request_id", requestID,
			"model", resp.Model,
			"provider", provider,
			"prompt_tokens", resp.Usage.PromptTokens,
			"completion_tokens", resp.Usage.CompletionTokens,
			"latency_ms", time.Since(start).Milliseconds(),
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})))

	addr := ":" + cfg.Port
	slog.Info("proxy starting", "addr", addr)

	if err := http.ListenAndServe(addr, global(mux)); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
