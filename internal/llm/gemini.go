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
)

const (
	// Gemini API base URL (model will be appended)
	geminiAPIBaseURL   = "https://generativelanguage.googleapis.com/v1beta/models"
	defaultGeminiModel = "gemini-2.0-flash-exp"
)

// GeminiClient implements the LLMClient interface for Google Gemini models
type GeminiClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// GeminiRequest represents the request structure for Gemini API
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

// GeminiContent represents content in Gemini API
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

// GeminiPart represents a part of content in Gemini API
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig represents generation configuration for Gemini API
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GeminiResponse represents the response structure from Gemini API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

// GeminiCandidate represents a candidate response from Gemini API
type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

// NewGeminiClient creates a new GeminiClient with the API key from environment
func NewGeminiClient(model string) (*GeminiClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}

	if model == "" {
		model = defaultGeminiModel
	}

	// Build the full URL with the model
	baseURL := fmt.Sprintf("%s/%s:generateContent", geminiAPIBaseURL, model)

	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}, nil

}

// NewGeminiClientWithConfig creates a new GeminiClient with custom configuration
func NewGeminiClientWithConfig(apiKey, baseURL, model string) *GeminiClient {
	if model == "" {

		model = defaultGeminiModel
	}

	if baseURL == "" {
		baseURL = fmt.Sprintf("%s/%s:generateContent", geminiAPIBaseURL, model)
	}

	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// GenerateCommitMessage generates a commit message based on the provided diff
func (c *GeminiClient) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	prompt := BuildCommitPrompt(diff)

	// Create the request payload using Gemini's format
	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: "You are a helpful assistant that generates concise, descriptive Git commit messages following GitHub conventions.\n\n" + prompt,
					},
				},
				Role: "user",
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			Temperature:     0.3,
			MaxOutputTokens: 1000,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Add API key as query parameter for Gemini
	url := c.baseURL + "?key=" + c.apiKey

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response GeminiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract the message from Gemini's response format
	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	candidate := response.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("no parts in candidate content")
	}

	commitMsg := strings.TrimSpace(candidate.Content.Parts[0].Text)
	if commitMsg == "" {
		return "", fmt.Errorf("empty commit message received")
	}

	return commitMsg, nil
}
