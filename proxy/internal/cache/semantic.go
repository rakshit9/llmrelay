package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakshit9/llmrelay/internal/upstream"
)

const semanticCacheTTL = 24 * time.Hour
const similarityThreshold = 0.92

type SemanticCache struct {
	pool    *pgxpool.Pool
	embedder *Embedder
}

func NewSemanticCache(pool *pgxpool.Pool) *SemanticCache {
	return &SemanticCache{
		pool:     pool,
		embedder: NewEmbedder(),
	}
}

// Get searches for a semantically similar cached response.
func (c *SemanticCache) Get(ctx context.Context, req *upstream.ChatRequest) (*upstream.ChatResponse, bool) {
	text := requestText(req)
	vec := c.embedder.Embed(text)
	pgVec := toPgVector(vec)

	var response string
	err := c.pool.QueryRow(ctx, `
		SELECT response
		FROM cache_vectors
		WHERE expires_at > NOW()
		  AND model_id = $1
		  AND 1 - (embedding <=> $2::vector) > $3
		ORDER BY embedding <=> $2::vector
		LIMIT 1
	`, req.Model, pgVec, similarityThreshold).Scan(&response)

	if err != nil {
		return nil, false // miss or error — fail open
	}

	var resp upstream.ChatResponse
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		return nil, false
	}

	slog.Info("semantic cache hit", "model", req.Model)
	return &resp, true
}

// Set stores a response with its embedding in pgvector.
func (c *SemanticCache) Set(ctx context.Context, req *upstream.ChatRequest, resp *upstream.ChatResponse) {
	text := requestText(req)
	vec := c.embedder.Embed(text)
	pgVec := toPgVector(vec)

	b, err := json.Marshal(resp)
	if err != nil {
		return
	}

	_, err = c.pool.Exec(ctx, `
		INSERT INTO cache_vectors (embedding, request_hash, response, model_id, expires_at)
		VALUES ($1::vector, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
	`, pgVec, RequestKey(req), string(b), req.Model, time.Now().Add(semanticCacheTTL))

	if err != nil {
		slog.Warn("semantic cache set error", "err", err)
	}
}

// requestText converts a chat request to a single string for embedding.
func requestText(req *upstream.ChatRequest) string {
	var parts []string
	for _, m := range req.Messages {
		parts = append(parts, m.Content)
	}
	return strings.Join(parts, " ")
}

// toPgVector formats a float32 slice as a pgvector literal: '[0.1,0.2,...]'
func toPgVector(vec []float32) string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%f", v))
	}
	sb.WriteString("]")
	return sb.String()
}
