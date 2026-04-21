package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/rakshit9/llmrelay/internal/upstream"
)

// RequestKey produces a stable SHA256 hash for a chat request.
// Same model + same messages → same key, every time.
func RequestKey(req *upstream.ChatRequest) string {
	payload := struct {
		Model    string             `json:"model"`
		Messages []upstream.Message `json:"messages"`
	}{
		Model:    req.Model,
		Messages: req.Messages,
	}
	b, _ := json.Marshal(payload)
	return fmt.Sprintf("%x", sha256.Sum256(b))
}
