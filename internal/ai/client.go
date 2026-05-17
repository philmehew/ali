// Package ai provides an OpenAI-compatible chat completions client.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/philmehew/ali/internal/models"
)

const maxRetries = 2

// Message role constants.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message represents a chat message in the OpenAI API format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Client is an OpenAI-compatible chat completions client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

// NewClient creates a new client from an AIConfig.
func NewClient(cfg *models.AIConfig) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 0}, // no overall timeout; context handles cancellation
		baseURL:    strings.TrimRight(cfg.URL, "/"),
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
	}
}

// chatRequest is the request body for the chat completions API.
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

// chatResponse is the non-streaming response.
type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// streamChunk is a single SSE chunk in a streaming response.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// errorResponse represents an API error response.
type errorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// isRetryable reports whether an HTTP status code indicates a transient error
// that may succeed on retry (429 Too Many Requests or 5xx server errors).
func isRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

// Complete sends a non-streaming chat completion request.
// Returns the assistant's response content.
// Retries up to maxRetries times on transient errors (429, 5xx) with exponential backoff.
func (c *Client) Complete(ctx context.Context, messages []Message) (string, error) {
	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("could not marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Second * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("could not create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("could not connect to AI service at %s: %w", c.baseURL, err)
		}

		if resp.StatusCode == http.StatusOK {
			var chatResp chatResponse
			if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
				_ = resp.Body.Close()
				return "", fmt.Errorf("could not parse AI response: %w", err)
			}
			_ = resp.Body.Close()

			if len(chatResp.Choices) == 0 {
				return "", fmt.Errorf("AI returned an empty response")
			}
			return chatResp.Choices[0].Message.Content, nil
		}

		err = c.parseError(resp)
		_ = resp.Body.Close()
		if !isRetryable(resp.StatusCode) {
			return "", err
		}
		lastErr = err
	}
	return "", lastErr
}

// CompleteStream sends a streaming chat completion request.
// Calls onToken for each content delta received.
// Returns the full assembled response when done.
// Retries up to maxRetries times on transient errors (429, 5xx) with exponential backoff.
func (c *Client) CompleteStream(ctx context.Context, messages []Message, onToken func(string)) (string, error) {
	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("could not marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Second * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("could not create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("could not connect to AI service at %s: %w", c.baseURL, err)
		}

		if resp.StatusCode != http.StatusOK {
			err := c.parseError(resp)
			_ = resp.Body.Close()
			if !isRetryable(resp.StatusCode) {
				return "", err
			}
			lastErr = err
			continue
		}

		result, err := c.readStream(resp, onToken)
		if err != nil {
			return "", err
		}
		return result, nil
	}
	return "", lastErr
}

// readStream reads the SSE response body and assembles the full response.
func (c *Client) readStream(resp *http.Response, onToken func(string)) (string, error) {
	defer func() { _ = resp.Body.Close() }()

	var builder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line size

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}
		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				builder.WriteString(content)
				if onToken != nil {
					onToken(content)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stream: %w", err)
	}

	result := builder.String()
	if result == "" {
		return "", fmt.Errorf("AI returned an empty response")
	}

	return result, nil
}

// parseError extracts a human-readable error from an API error response.
func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp errorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed: check your AI API key")
		case http.StatusTooManyRequests:
			return fmt.Errorf("rate limited by AI service: please try again")
		default:
			return fmt.Errorf("AI service error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed: check your AI API key")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limited by AI service: please try again")
	default:
		return fmt.Errorf("AI service error (status %d)", resp.StatusCode)
	}
}
