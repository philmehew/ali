package ai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/philmehew/ali/internal/models"
)

func TestNewClient(t *testing.T) {
	cfg := &models.AIConfig{
		URL:    "https://api.example.com/v1",
		APIKey: "test-key",
		Model:  "gpt-4",
	}
	client := NewClient(cfg)
	if client.baseURL != "https://api.example.com/v1" {
		t.Errorf("expected baseURL %q, got %q", "https://api.example.com/v1", client.baseURL)
	}
	if client.apiKey != "test-key" {
		t.Errorf("expected apiKey %q, got %q", "test-key", client.apiKey)
	}
	if client.model != "gpt-4" {
		t.Errorf("expected model %q, got %q", "gpt-4", client.model)
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	cfg := &models.AIConfig{
		URL:    "https://api.example.com/v1/",
		APIKey: "key",
		Model:  "model",
	}
	client := NewClient(cfg)
	if client.baseURL != "https://api.example.com/v1" {
		t.Errorf("expected trailing slash trimmed, got %q", client.baseURL)
	}
}

func TestComplete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"choices":[{"message":{"role":"assistant","content":"df -h"}}]}`)
	}))
	defer server.Close()

	client := NewClient(&models.AIConfig{
		URL:    server.URL,
		APIKey: "test-key",
		Model:  "test-model",
	})

	result, err := client.Complete(context.Background(), []Message{
		{Role: "user", Content: "check disk size"},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result != "df -h" {
		t.Errorf("expected %q, got %q", "df -h", result)
	}
}

func TestComplete_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"Invalid API key"}}`)
	}))
	defer server.Close()

	client := NewClient(&models.AIConfig{
		URL:    server.URL,
		APIKey: "bad-key",
		Model:  "test-model",
	})

	_, err := client.Complete(context.Background(), []Message{
		{Role: "user", Content: "test"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected auth error, got %q", err.Error())
	}
}

func TestComplete_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"Service unavailable"}}`)
	}))
	defer server.Close()

	client := NewClient(&models.AIConfig{
		URL:    server.URL,
		APIKey: "key",
		Model:  "test-model",
	})

	_, err := client.Complete(context.Background(), []Message{
		{Role: "user", Content: "test"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AI service error") {
		t.Errorf("expected server error, got %q", err.Error())
	}
}

func TestComplete_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"Rate limit exceeded"}}`)
	}))
	defer server.Close()

	client := NewClient(&models.AIConfig{
		URL:    server.URL,
		APIKey: "key",
		Model:  "test-model",
	})

	_, err := client.Complete(context.Background(), []Message{
		{Role: "user", Content: "test"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("expected rate limit error, got %q", err.Error())
	}
}

func TestComplete_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"choices":[]}`)
	}))
	defer server.Close()

	client := NewClient(&models.AIConfig{
		URL:    server.URL,
		APIKey: "key",
		Model:  "test-model",
	})

	_, err := client.Complete(context.Background(), []Message{
		{Role: "user", Content: "test"},
	})
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("expected empty response error, got %q", err.Error())
	}
}

func TestComplete_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	client := NewClient(&models.AIConfig{
		URL:    server.URL,
		APIKey: "key",
		Model:  "test-model",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Complete(ctx, []Message{
		{Role: "user", Content: "test"},
	})
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestCompleteStream_Success(t *testing.T) {
	sseData := "data: {\"choices\":[{\"delta\":{\"content\":\"df\"}}]}\ndata: {\"choices\":[{\"delta\":{\"content\":\" -h\"}}]}\ndata: [DONE]\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	client := NewClient(&models.AIConfig{
		URL:    server.URL,
		APIKey: "test-key",
		Model:  "test-model",
	})

	var tokens []string
	result, err := client.CompleteStream(context.Background(), []Message{
		{Role: "user", Content: "check disk size"},
	}, func(token string) {
		tokens = append(tokens, token)
	})
	if err != nil {
		t.Fatalf("CompleteStream: %v", err)
	}
	if result != "df -h" {
		t.Errorf("expected %q, got %q", "df -h", result)
	}
	if len(tokens) != 2 || tokens[0] != "df" || tokens[1] != " -h" {
		t.Errorf("expected tokens [df, \" -h\"], got %v", tokens)
	}
}

func TestCompleteStream_EmptyDelta(t *testing.T) {
	sseData := "data: {\"choices\":[{\"delta\":{\"content\":\"\"}}]}\ndata: {\"choices\":[{\"delta\":{\"content\":\"ls\"}}]}\ndata: [DONE]\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	client := NewClient(&models.AIConfig{
		URL:    server.URL,
		APIKey: "test-key",
		Model:  "test-model",
	})

	result, err := client.CompleteStream(context.Background(), []Message{
		{Role: "user", Content: "test"},
	}, nil)
	if err != nil {
		t.Fatalf("CompleteStream: %v", err)
	}
	if result != "ls" {
		t.Errorf("expected %q, got %q", "ls", result)
	}
}

func TestCompleteStream_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"Invalid API key"}}`)
	}))
	defer server.Close()

	client := NewClient(&models.AIConfig{
		URL:    server.URL,
		APIKey: "bad-key",
		Model:  "test-model",
	})

	_, err := client.CompleteStream(context.Background(), []Message{
		{Role: "user", Content: "test"},
	}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected auth error, got %q", err.Error())
	}
}
