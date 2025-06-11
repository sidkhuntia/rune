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
	// Default Qwen API endpoint - using Novita.ai's OpenAI-compatible API
	defaultQwenAPIURL = "https://api.novita.ai/v3/openai/chat/completions"
	defaultModel      = "qwen/qwen2.5-7b-instruct"
	defaultTimeout    = 30 * time.Second
)

// QwenClient implements the LLMClient interface for Qwen models
type QwenClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewQwenClient creates a new QwenClient with the API key from environment
func NewQwenClient() (*QwenClient, error) {
	apiKey := os.Getenv("NOVITA_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("NOVITA_API_KEY environment variable is required")
	}

	return &QwenClient{
		apiKey:  apiKey,
		baseURL: defaultQwenAPIURL,
		model:   defaultModel,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}, nil
}

// NewQwenClientWithConfig creates a new QwenClient with custom configuration
func NewQwenClientWithConfig(apiKey, baseURL, model string) *QwenClient {
	if baseURL == "" {
		baseURL = defaultQwenAPIURL
	}
	if model == "" {
		model = defaultModel
	}

	return &QwenClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// GenerateCommitMessage generates a commit message based on the provided diff
func (c *QwenClient) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	prompt := BuildCommitPrompt(diff)

	// Create the request payload using OpenAI-compatible format for Novita.ai
	reqBody := ChatCompletionRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant that generates concise, descriptive Git commit messages following GitHub conventions.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
		MaxTokens:   150,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
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
