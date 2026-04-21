package cache

import (
	"context"
	"log/slog"

	"github.com/rakshit9/llmrelay/internal/upstream"
)

type HitType string

const (
	HitExact    HitType = "exact"
	HitSemantic HitType = "semantic"
	HitMiss     HitType = ""
)

type Result struct {
	Response *upstream.ChatResponse
	HitType  HitType
}

// Cache is the unified cache layer: exact (Redis) → semantic (pgvector).
type Cache struct {
	exact    *ExactCache
	semantic *SemanticCache
}

func New(exact *ExactCache, semantic *SemanticCache) *Cache {
	return &Cache{exact: exact, semantic: semantic}
}

// Get checks exact cache first, then semantic cache.
func (c *Cache) Get(ctx context.Context, req *upstream.ChatRequest) Result {
	// 1. Exact cache
	if resp, ok := c.exact.Get(ctx, req); ok {
		slog.Info("exact cache hit", "model", req.Model)
		return Result{Response: resp, HitType: HitExact}
	}

	// 2. Semantic cache
	if resp, ok := c.semantic.Get(ctx, req); ok {
		return Result{Response: resp, HitType: HitSemantic}
	}

	return Result{HitType: HitMiss}
}

// Set stores the response in both exact and semantic caches.
func (c *Cache) Set(ctx context.Context, req *upstream.ChatRequest, resp *upstream.ChatResponse) {
	c.exact.Set(ctx, req, resp)
	c.semantic.Set(ctx, req, resp)
}
