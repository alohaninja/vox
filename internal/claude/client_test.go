package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers.
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("x-api-key = %q, want %q", got, "test-key")
		}
		if got := r.Header.Get("anthropic-version"); got != apiVersion {
			t.Errorf("anthropic-version = %q, want %q", got, apiVersion)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		// Verify request body.
		var reqBody messagesRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if reqBody.System == "" {
			t.Error("expected system prompt")
		}
		if len(reqBody.Messages) != 1 || reqBody.Messages[0].Role != "user" {
			t.Error("expected single user message")
		}
		if reqBody.Model != "test-model" {
			t.Errorf("model = %q, want %q", reqBody.Model, "test-model")
		}
		if reqBody.MaxTokens != defaultMaxTokens {
			t.Errorf("max_tokens = %d, want %d", reqBody.MaxTokens, defaultMaxTokens)
		}

		resp := messagesResponse{
			Content:    []contentBlock{{Type: "text", Text: "Hello, world!"}},
			StopReason: "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("test-key", "test-model")
	c.apiURL = srv.URL

	got, err := c.Complete(context.Background(), "system prompt", "hello world")
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
	if got != "Hello, world!" {
		t.Errorf("Complete() = %q, want %q", got, "Hello, world!")
	}
}

func TestCompleteNoAPIKey(t *testing.T) {
	c := NewClient("", "")
	_, err := c.Complete(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
	if !strings.Contains(err.Error(), "API key is empty") {
		t.Errorf("error = %q, want it to mention 'API key is empty'", err)
	}
}

func TestCompleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"type":"invalid_request","message":"bad request"}}`, http.StatusBadRequest)
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	_, err := c.Complete(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Errorf("error = %q, want it to mention status 400", err)
	}
}

func TestCompleteAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{
			Error: &apiError{Type: "overloaded_error", Message: "overloaded"},
		})
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	_, err := c.Complete(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "claude API error") {
		t.Errorf("error = %q, want it to contain 'claude API error'", err)
	}
	if !strings.Contains(err.Error(), "overloaded_error") {
		t.Errorf("error = %q, want it to contain 'overloaded_error'", err)
	}
}

func TestCompleteEmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{
			Content:    []contentBlock{},
			StopReason: "end_turn",
		})
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	_, err := c.Complete(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestCompleteEmptyTextBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{
			Content:    []contentBlock{{Type: "text", Text: ""}},
			StopReason: "end_turn",
		})
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	_, err := c.Complete(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for empty text block")
	}
	if !strings.Contains(err.Error(), "empty text content") {
		t.Errorf("error = %q, want it to mention 'empty text content'", err)
	}
}

func TestCompleteWhitespaceOnlyTextBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{
			Content:    []contentBlock{{Type: "text", Text: "  \n\t  "}},
			StopReason: "end_turn",
		})
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	_, err := c.Complete(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for whitespace-only text block")
	}
}

func TestCompleteTruncatedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{
			Content:    []contentBlock{{Type: "text", Text: "partial output that was cut"}},
			StopReason: "max_tokens",
		})
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	_, err := c.Complete(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for truncated response")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Errorf("error = %q, want it to mention 'truncated'", err)
	}
}

func TestCompleteInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	_, err := c.Complete(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse response") {
		t.Errorf("error = %q, want it to mention 'parse response'", err)
	}
}

func TestCompleteCancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.Complete(ctx, "", "hello")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestCompleteTrimWhitespace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{
			Content:    []contentBlock{{Type: "text", Text: "  Hello, world!  \n"}},
			StopReason: "end_turn",
		})
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	got, err := c.Complete(context.Background(), "", "hello")
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
	if got != "Hello, world!" {
		t.Errorf("Complete() = %q, want %q (whitespace should be trimmed)", got, "Hello, world!")
	}
}

func TestSetMaxTokens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody messagesRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if reqBody.MaxTokens != 2048 {
			t.Errorf("max_tokens = %d, want 2048", reqBody.MaxTokens)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{
			Content:    []contentBlock{{Type: "text", Text: "ok"}},
			StopReason: "end_turn",
		})
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL
	c.SetMaxTokens(2048)

	_, err := c.Complete(context.Background(), "", "hello")
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
}

func TestDefaultModel(t *testing.T) {
	c := NewClient("key", "")
	if c.model != defaultModel {
		t.Errorf("default model = %q, want %q", c.model, defaultModel)
	}
}

func TestCustomModel(t *testing.T) {
	c := NewClient("key", "claude-sonnet-4-20250514")
	if c.model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want custom model", c.model)
	}
}

func TestPostProcessSystemPromptNotEmpty(t *testing.T) {
	if PostProcessSystemPrompt == "" {
		t.Fatal("PostProcessSystemPrompt should not be empty")
	}
}
