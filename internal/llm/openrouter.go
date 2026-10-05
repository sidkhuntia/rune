package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	// OpenRouter API endpoint
	openRouterAPIURL  = "https://openrouter.ai/api/v1/chat/completions"
	openRouterTimeout = 60 * time.Second

	// Reasoning models spend completion tokens thinking before they answer,
	// so the budget must cover both the reasoning and the commit message.
	maxCompletionTokens = 2048
	maxResponseBytes    = 4 << 20
	maxRetries          = 2
)

// retryBackoff is the base delay between retries (doubles each attempt).
var retryBackoff = time.Second

// thinkBlock matches inline reasoning that some models emit in the content.
var thinkBlock = regexp.MustCompile(`(?s)<think>.*?</think>`)

// OpenRouterClient implements the LLMClient interface for OpenRouter models
type OpenRouterClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenRouterClient creates a new OpenRouterClient with the API key from environment
func NewOpenRouterClient(model string) (*OpenRouterClient, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY environment variable is required")
	}

	if model == "" {
		model = "nvidia/nemotron-3-ultra-550b-a55b:free" // default model
	}

	return &OpenRouterClient{
		apiKey:  apiKey,
		baseURL: openRouterAPIURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: openRouterTimeout,
		},
	}, nil
}

// GenerateCommitMessage generates a commit message based on the provided diff
func (c *OpenRouterClient) GenerateCommitMessage(ctx context.Context, r Request) (string, error) {
	// Create the request payload using OpenAI-compatible format
	reqBody := ChatCompletionRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: SystemPrompt(r.Style),
			},
			{
				Role:    "user",
				Content: UserPrompt(r),
			},
		},
		Temperature: temperatureOr(r, 0.3),
		MaxTokens:   maxCompletionTokens,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryBackoff * time.Duration(1<<(attempt-1))):
			}
		}

		msg, retryable, err := c.doRequest(ctx, jsonBody)
		if err == nil {
			return msg, nil
		}
		lastErr = err
		if !retryable {
			break
		}
	}
	return "", lastErr
}

// doRequest performs a single API call. The bool reports whether the failure
// is transient (rate limit, upstream 5xx, network) and worth retrying.
func (c *OpenRouterClient) doRequest(ctx context.Context, jsonBody []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers for OpenRouter
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/siddhartha/rune")
	req.Header.Set("X-Title", "Rune Git Commit Generator")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", ctx.Err() == nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", true, fmt.Errorf("failed to read response: %w", err)
	}

	var response ChatCompletionResponse
	parseErr := json.Unmarshal(body, &response)

	if resp.StatusCode != http.StatusOK {
		detail := strings.TrimSpace(string(body))
		if parseErr == nil && response.Error != nil && response.Error.Message != "" {
			detail = response.Error.Message
		}
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return "", retryable, fmt.Errorf("OpenRouter API request failed with status %d: %s", resp.StatusCode, detail)
	}
	if parseErr != nil {
		return "", false, fmt.Errorf("failed to parse response: %w", parseErr)
	}

	// OpenRouter can report upstream failures with a 200 and an error object.
	if response.Error != nil {
		return "", true, fmt.Errorf("OpenRouter error: %s", response.Error.Message)
	}
	if len(response.Choices) == 0 {
		return "", true, fmt.Errorf("no choices in response")
	}

	choice := response.Choices[0]
	commitMsg := strings.TrimSpace(thinkBlock.ReplaceAllString(choice.Message.Content, ""))
	if commitMsg == "" {
		if choice.FinishReason == "length" {
			return "", false, fmt.Errorf("model hit the token limit before producing a message (reasoning models need headroom)")
		}
		return "", true, fmt.Errorf("empty commit message received")
	}

	return commitMsg, false, nil
}
