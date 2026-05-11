package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

		resp := messagesResponse{
			Content: []contentBlock{
				{Type: "text", Text: "Hello, world!"},
			},
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
	t.Logf("got expected error: %v", err)
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
	t.Logf("got expected error: %v", err)
}

func TestCompleteEmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{Content: []contentBlock{}})
	}))
	defer srv.Close()

	c := NewClient("test-key", "")
	c.apiURL = srv.URL

	_, err := c.Complete(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for empty content")
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
