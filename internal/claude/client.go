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
	defaultAPIURL = "https://api.anthropic.com/v1/messages"
	apiVersion    = "2023-06-01"
	defaultModel  = "claude-haiku-4-5-20251001"
)

// Client is a lightweight Claude Messages API client using net/http directly.
// Consistent with the existing transcribe.Client pattern in this codebase.
type Client struct {
	apiKey     string
	model      string
	apiURL     string
	httpClient *http.Client
}

// NewClient creates a Claude API client. Model defaults to claude-haiku-4-5-20251001
// if empty (optimized for speed in the dictation pipeline).
func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = defaultModel
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
		apiURL: defaultAPIURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// message types for the Claude Messages API.
type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []contentBlock `json:"content"`
	Error   *apiError      `json:"error,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Complete sends a user message with an optional system prompt to Claude
// and returns the text response.
func (c *Client) Complete(ctx context.Context, system, userMessage string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	reqBody := messagesRequest{
		Model:     c.model,
		MaxTokens: 1024,
		System:    system,
		Messages: []message{
			{Role: "user", Content: userMessage},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude API returned status %d: %s", resp.StatusCode, truncate(respBytes, 512))
	}

	var result messagesResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse response: %w (body: %s)", err, truncate(respBytes, 256))
	}

	if result.Error != nil {
		return "", fmt.Errorf("claude API error: %s: %s", result.Error.Type, result.Error.Message)
	}

	for _, block := range result.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

// PostProcessSystemPrompt is the system prompt for cleaning up raw speech-to-text output.
const PostProcessSystemPrompt = `You are a transcription correction assistant. Fix grammar, punctuation, capitalization, and technical terms in the following speech-to-text output. Preserve the speaker's intent, tone, and meaning exactly. Return ONLY the corrected text with no explanation, preamble, or commentary.`
