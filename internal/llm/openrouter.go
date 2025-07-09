package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// OpenRouter API endpoint
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	openRouterTimeout = 60 * time.Second
)

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
		model = "deepseek/deepseek-chat" // default model
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
func (c *OpenRouterClient) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	prompt := BuildCommitPrompt(diff)

	// Create the request payload using OpenAI-compatible format
	reqBody := ChatCompletionRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant that generates concise, descriptive Git commit messages following conventional commit format. Focus on the primary change and keep it under 50 characters for the subject line.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
		MaxTokens:   512,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers for OpenRouter
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/siddhartha/rune")
	req.Header.Set("X-Title", "Rune Git Commit Generator")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenRouter API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract the message from OpenAI-compatible response format
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	commitMsg := strings.TrimSpace(response.Choices[0].Message.Content)
	if commitMsg == "" {
		return "", fmt.Errorf("empty commit message received")
	}

	return commitMsg, nil
}