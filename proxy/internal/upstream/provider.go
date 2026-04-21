package upstream

import "io"

// Provider is the contract every upstream adapter must satisfy.
// The router calls these methods without knowing which provider it's talking to.
type Provider interface {
	// Complete sends a non-streaming request and returns the full response.
	Complete(req *ChatRequest) (*ChatResponse, error)

	// Stream sends a streaming request and writes SSE lines to w, calling flush after each chunk.
	Stream(req *ChatRequest, w io.Writer, flush func()) error

	// Name returns the provider identifier (openai, anthropic, google, groq).
	Name() string
}
