package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rakshit9/llmrelay/internal/upstream"
)

const exactCacheTTL = time.Hour

type ExactCache struct {
	client *redis.Client
}

func NewExactCache(addr string) *ExactCache {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &ExactCache{client: client}
}

// Get returns a cached response for the request, or (nil, false) on miss.
// On Redis failure, returns miss — cache failing open, service stays up.
func (c *ExactCache) Get(ctx context.Context, req *upstream.ChatRequest) (*upstream.ChatResponse, bool) {
	key := "exact:" + RequestKey(req)

	val, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false // clean miss
	}
	if err != nil {
		slog.Warn("redis get error", "err", err)
		return nil, false // fail open
	}

	var resp upstream.ChatResponse
	if err := json.Unmarshal([]byte(val), &resp); err != nil {
		return nil, false
	}

	return &resp, true
}

// Set stores a response in Redis with a 1-hour TTL.
// On Redis failure, logs and continues — cache is not critical path.
func (c *ExactCache) Set(ctx context.Context, req *upstream.ChatRequest, resp *upstream.ChatResponse) {
	key := "exact:" + RequestKey(req)

	b, err := json.Marshal(resp)
	if err != nil {
		return
	}

	if err := c.client.Set(ctx, key, b, exactCacheTTL).Err(); err != nil {
		slog.Warn("redis set error", "err", err)
	}
}

// Ping checks Redis connectivity. Used at startup.
func (c *ExactCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
