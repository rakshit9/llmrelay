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

const googleBaseURL = "https://generativelanguage.googleapis.com/v1/models"

type googleRequest struct {
	Contents []googleContent `json:"contents"`
}

type googleContent struct {
	Role  string        `json:"role"`
	Parts []googlePart  `json:"parts"`
}

type googlePart struct {
	Text string `json:"text"`
}

type googleResponse struct {
	Candidates []googleCandidate `json:"candidates"`
	UsageMetadata googleUsage    `json:"usageMetadata"`
}

type googleCandidate struct {
	Content googleContent `json:"content"`
}

type googleUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type GoogleProvider struct {
	apiKey     string
	httpClient *http.Client
}

func NewGoogleProvider(apiKey string) *GoogleProvider {
	return &GoogleProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *GoogleProvider) Name() string { return "google" }

func (p *GoogleProvider) Complete(req *ChatRequest) (*ChatResponse, error) {
	gReq := toGoogleRequest(req)
	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", googleBaseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call google: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &UpstreamError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var gResp googleResponse
	if err := json.Unmarshal(respBody, &gResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return fromGoogleResponse(&gResp, req.Model), nil
}

func (p *GoogleProvider) Stream(req *ChatRequest, w io.Writer, flush func()) error {
	gReq := toGoogleRequest(req)
	body, err := json.Marshal(gReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:streamGenerateContent?alt=sse&key=%s", googleBaseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	streamClient := &http.Client{Timeout: 0}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("call google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return &UpstreamError{StatusCode: resp.StatusCode, Body: string(b)}
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var gResp googleResponse
		if err := json.Unmarshal([]byte(data), &gResp); err != nil {
			continue
		}

		text := extractGoogleText(&gResp)
		if text == "" {
			continue
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
		fmt.Fprintf(w, "data: %s\n\n", b)
		flush()
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flush()
	return scanner.Err()
}

func toGoogleRequest(req *ChatRequest) *googleRequest {
	gReq := &googleRequest{}
	for _, m := range req.Messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			// Google doesn't have a system role — prepend as user turn
			gReq.Contents = append(gReq.Contents, googleContent{
				Role:  "user",
				Parts: []googlePart{{Text: m.Content}},
			})
			continue
		}
		gReq.Contents = append(gReq.Contents, googleContent{
			Role:  role,
			Parts: []googlePart{{Text: m.Content}},
		})
	}
	return gReq
}

func fromGoogleResponse(gResp *googleResponse, model string) *ChatResponse {
	content := extractGoogleText(gResp)
	return &ChatResponse{
		Model: model,
		Choices: []Choice{{
			Index:        0,
			Message:      Message{Role: "assistant", Content: content},
			FinishReason: "stop",
		}},
		Usage: Usage{
			PromptTokens:     gResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: gResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      gResp.UsageMetadata.TotalTokenCount,
		},
	}
}

func extractGoogleText(gResp *googleResponse) string {
	if len(gResp.Candidates) == 0 {
		return ""
	}
	parts := gResp.Candidates[0].Content.Parts
	if len(parts) == 0 {
		return ""
	}
	return parts[0].Text
}
