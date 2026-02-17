// Package claude implements a ClaimEvaluator backed by the Anthropic Claude API.
// It wraps a Claude Max subscription for OpenClaw agents participating in
// Legible Money's Proof of Truth consensus.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL       = "https://api.anthropic.com"
	defaultModel         = "claude-sonnet-4-5-20250929"
	defaultMaxTokens     = 1024
	defaultTimeout       = 60 * time.Second
	anthropicAPIVersion  = "2023-06-01"
	messagesEndpoint     = "/v1/messages"
)

// AnthropicClient is a minimal HTTP client for the Claude Messages API.
// Uses only stdlib — no external dependencies.
type AnthropicClient struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// NewAnthropicClient creates a client with sensible defaults.
func NewAnthropicClient(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		APIKey:  apiKey,
		BaseURL: defaultBaseURL,
		Model:   defaultModel,
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Message represents a conversation message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MessagesRequest is the request body for the Messages API.
type MessagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
}

// ContentBlock represents a content block in the response.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// MessagesResponse is the response from the Messages API.
type MessagesResponse struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
	Model   string         `json:"model"`
	Usage   Usage          `json:"usage"`
}

// TextContent returns the concatenated text from all text content blocks.
func (r *MessagesResponse) TextContent() string {
	var text string
	for _, block := range r.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return text
}

// APIError represents an error response from the Anthropic API.
type APIError struct {
	StatusCode int
	Type       string `json:"type"`
	ErrorInfo  struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("anthropic API error (HTTP %d): %s: %s",
		e.StatusCode, e.ErrorInfo.Type, e.ErrorInfo.Message)
}

// IsRateLimited returns true if the error is a 429 rate limit response.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsOverloaded returns true if the API is overloaded (529).
func (e *APIError) IsOverloaded() bool {
	return e.StatusCode == 529
}

// CreateMessage sends a message to the Claude Messages API.
func (c *AnthropicClient) CreateMessage(ctx context.Context, req MessagesRequest) (*MessagesResponse, error) {
	if req.Model == "" {
		req.Model = c.Model
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = defaultMaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.BaseURL+messagesEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		_ = json.Unmarshal(respBody, apiErr)
		return nil, apiErr
	}

	var msgResp MessagesResponse
	if err := json.Unmarshal(respBody, &msgResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &msgResp, nil
}
