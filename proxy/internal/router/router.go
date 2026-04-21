package router

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"time"

	"github.com/rakshit9/llmrelay/internal/config"
	"github.com/rakshit9/llmrelay/internal/upstream"
)

const maxRetries = 3

type Router struct {
	providers map[string]upstream.Provider
}

func New(providers map[string]upstream.Provider) *Router {
	return &Router{providers: providers}
}

// Complete routes a non-streaming request with failover.
func (r *Router) Complete(req *upstream.ChatRequest) (*upstream.ChatResponse, string, error) {
	providers, err := r.resolveChain(req.Model)
	if err != nil {
		return nil, "", err
	}

	var lastErr error
	attempts := 0

	for _, p := range providers {
		if attempts >= maxRetries {
			break
		}
		attempts++

		slog.Info("routing request", "provider", p.Name(), "model", req.Model, "attempt", attempts)

		resp, err := p.Complete(req)
		if err == nil {
			return resp, p.Name(), nil
		}

		lastErr = err
		if !upstream.IsRetryable(err) {
			return nil, "", err
		}

		wait := backoff(attempts)
		slog.Warn("upstream error, retrying",
			"provider", p.Name(),
			"err", err,
			"wait_ms", wait.Milliseconds(),
		)
		time.Sleep(wait)
	}

	return nil, "", fmt.Errorf("all providers failed: %w", lastErr)
}

// Stream routes a streaming request with failover.
func (r *Router) Stream(req *upstream.ChatRequest, w io.Writer, flush func()) (string, error) {
	providers, err := r.resolveChain(req.Model)
	if err != nil {
		return "", err
	}

	var lastErr error
	attempts := 0

	for _, p := range providers {
		if attempts >= maxRetries {
			break
		}
		attempts++

		slog.Info("routing stream", "provider", p.Name(), "model", req.Model, "attempt", attempts)

		err := p.Stream(req, w, flush)
		if err == nil {
			return p.Name(), nil
		}

		lastErr = err
		if !upstream.IsRetryable(err) {
			return "", err
		}

		wait := backoff(attempts)
		slog.Warn("stream error, retrying",
			"provider", p.Name(),
			"err", err,
			"wait_ms", wait.Milliseconds(),
		)
		time.Sleep(wait)
	}

	return "", fmt.Errorf("all providers failed: %w", lastErr)
}

// resolveChain returns the ordered list of providers to try for a model.
func (r *Router) resolveChain(model string) ([]upstream.Provider, error) {
	entry, ok := config.LookupModel(model)
	if !ok {
		return nil, fmt.Errorf("unknown model: %s", model)
	}

	primary, ok := r.providers[entry.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %q not configured", entry.Provider)
	}

	chain := []upstream.Provider{primary}
	for _, name := range entry.Failover {
		if p, ok := r.providers[name]; ok {
			chain = append(chain, p)
		}
	}

	return chain, nil
}

// backoff returns exponential wait with jitter: ~500ms, ~1s, ~2s.
func backoff(attempt int) time.Duration {
	base := time.Duration(500<<(attempt-1)) * time.Millisecond
	jitter := time.Duration(rand.Intn(200)) * time.Millisecond
	return base + jitter
}
