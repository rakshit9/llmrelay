package upstream

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const anthropicBaseURL = "https://api.anthropic.com/v1"
const anthropicVersion = "2023-06-01"

// Anthropic request shape
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Anthropic response shape
type anthropicResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Content []anthropicContent `json:"content"`
	Usage   anthropicUsage     `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicProvider struct {
	apiKey     string
	httpClient *http.Client
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Complete(req *ChatRequest) (*ChatResponse, error) {
	aReq := toAnthropicRequest(req, false)
	body, err := json.Marshal(aReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", anthropicBaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call anthropic: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &UpstreamError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var aResp anthropicResponse
	if err := json.Unmarshal(respBody, &aResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return fromAnthropicResponse(&aResp, req.Model), nil
}

func (p *AnthropicProvider) Stream(req *ChatRequest, w io.Writer, flush func()) error {
	aReq := toAnthropicRequest(req, true)
	body, err := json.Marshal(aReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", anthropicBaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	streamClient := &http.Client{Timeout: 0}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("call anthropic: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return &UpstreamError{StatusCode: resp.StatusCode, Body: string(b)}
	}

	// Anthropic SSE → translate to OpenAI SSE format
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flush()
			break
		}

		// Parse Anthropic event and emit OpenAI-compatible chunk
		chunk := translateAnthropicChunk(data)
		if chunk != "" {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flush()
		}
	}
	return scanner.Err()
}

// toAnthropicRequest converts our internal format to Anthropic's API shape.
func toAnthropicRequest(req *ChatRequest, stream bool) *anthropicRequest {
	aReq := &anthropicRequest{
		Model:     req.Model,
		MaxTokens: 4096,
		Stream:    stream,
	}

	for _, m := range req.Messages {
		if m.Role == "system" {
			aReq.System = m.Content
		} else {
			aReq.Messages = append(aReq.Messages, anthropicMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}
	return aReq
}

// fromAnthropicResponse converts Anthropic's response to our internal ChatResponse.
func fromAnthropicResponse(aResp *anthropicResponse, model string) *ChatResponse {
	content := ""
	if len(aResp.Content) > 0 {
		content = aResp.Content[0].Text
	}
	return &ChatResponse{
		ID:    aResp.ID,
		Model: model,
		Choices: []Choice{{
			Index:        0,
			Message:      Message{Role: "assistant", Content: content},
			FinishReason: "stop",
		}},
		Usage: Usage{
			PromptTokens:     aResp.Usage.InputTokens,
			CompletionTokens: aResp.Usage.OutputTokens,
			TotalTokens:      aResp.Usage.InputTokens + aResp.Usage.OutputTokens,
		},
	}
}

// translateAnthropicChunk converts an Anthropic SSE event to OpenAI chunk format.
func translateAnthropicChunk(data string) string {
	var event map[string]any
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return ""
	}

	eventType, _ := event["type"].(string)
	if eventType != "content_block_delta" {
		return ""
	}

	delta, _ := event["delta"].(map[string]any)
	text, _ := delta["text"].(string)
	if text == "" {
		return ""
	}

	chunk := map[string]any{
		"object": "chat.completion.chunk",
		"choices": []map[string]any{{
			"index": 0,
			"delta": map[string]any{
				"role":    "assistant",
				"content": text,
			},
		}},
	}
	b, _ := json.Marshal(chunk)
	return string(b)
}
