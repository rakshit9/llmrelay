package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/rakshit9/llmrelay/internal/auth"
	"github.com/rakshit9/llmrelay/internal/config"
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

	openai := upstream.NewOpenAIClient(cfg.OpenAIAPIKey)
	requireAuth := auth.Require(cfg.GatewayAPIKey)

	mux := http.NewServeMux()

	// Public — no auth needed
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Protected — clients must send Authorization: Bearer <GATEWAY_API_KEY>
	mux.Handle("POST /v1/chat/completions", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the incoming request body
		var req upstream.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Model == "" || len(req.Messages) == 0 {
			writeError(w, http.StatusBadRequest, "model and messages are required")
			return
		}

		slog.Info("chat request", "model", req.Model, "messages", len(req.Messages))

		// Forward to OpenAI
		resp, statusCode, err := openai.ChatCompletion(&req)
		if err != nil {
			slog.Error("upstream error", "err", err, "status", statusCode)
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		slog.Info("chat response",
			"model", resp.Model,
			"prompt_tokens", resp.Usage.PromptTokens,
			"completion_tokens", resp.Usage.CompletionTokens,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(resp)
	})))

	addr := ":" + cfg.Port
	slog.Info("proxy starting", "addr", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
