package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestQwenClient_GenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name           string
		diff           string
		responseStatus int
		responseBody   string
		expectedMsg    string
		expectedError  string
	}{
		{
			name:           "successful response",
			diff:           "diff --git a/main.go b/main.go\n+fmt.Println(\"Hello\")",
			responseStatus: http.StatusOK,
			responseBody: `{
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": "Add Hello world print statement"
						},
						"finish_reason": "stop"
					}
				],
				"usage": {
					"total_tokens": 50
				}
			}`,
			expectedMsg: "Add Hello world print statement",
		},
		{
			name:           "API error response",
			diff:           "some diff",
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"error": "Invalid API key"}`,
			expectedError:  "API request failed with status 401",
		},
		{
			name:           "invalid JSON response",
			diff:           "some diff",
			responseStatus: http.StatusOK,
			responseBody:   `invalid json`,
			expectedError:  "failed to parse response",
		},
		{
			name:           "missing choices in response",
			diff:           "some diff",
			responseStatus: http.StatusOK,
			responseBody:   `{"usage": {"total_tokens": 50}}`,
			expectedError:  "no choices in response",
		},
		{
			name:           "empty choices in response",
			diff:           "some diff",
			responseStatus: http.StatusOK,
			responseBody:   `{"choices": []}`,
			expectedError:  "no choices in response",
		},
		{
			name:           "empty commit message",
			diff:           "some diff",
			responseStatus: http.StatusOK,
			responseBody:   `{"choices": [{"message": {"content": "   "}}]}`,
			expectedError:  "empty commit message received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and headers
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
				}

				authHeader := r.Header.Get("Authorization")
				if !strings.HasPrefix(authHeader, "Bearer ") {
					t.Errorf("Expected Authorization header with Bearer token, got %s", authHeader)
				}

				w.WriteHeader(tt.responseStatus)
				_, err := w.Write([]byte(tt.responseBody))
				if err != nil {
					t.Errorf("Failed to write response body: %v", err)
				}
			}))
			defer server.Close()

			// Create client with mock server URL
			client := NewQwenClientWithConfig("test-api-key", server.URL, "qwen2.5-7b-instruct")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := client.GenerateCommitMessage(ctx, tt.diff)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if result != tt.expectedMsg {
					t.Errorf("Expected message '%s', got '%s'", tt.expectedMsg, result)
				}
			}
		})
	}
}

func TestNewQwenClient(t *testing.T) {
	t.Run("with API key set", func(t *testing.T) {
		// Set environment variable
		t.Setenv("NOVITA_API_KEY", "test-key")

		client, err := NewQwenClient()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if client == nil {
			t.Error("Expected client to be created")
		}
	})

	t.Run("without API key", func(t *testing.T) {
		// Unset environment variable
		t.Setenv("NOVITA_API_KEY", "")

		client, err := NewQwenClient()
		if err == nil {
			t.Error("Expected error when API key is not set")
		}
		if client != nil {
			t.Error("Expected client to be nil when API key is not set")
		}
		if !strings.Contains(err.Error(), "NOVITA_API_KEY environment variable is required") {
			t.Errorf("Expected error about missing API key, got: %v", err)
		}
	})
}

func TestNewQwenClientWithConfig(t *testing.T) {
	t.Run("with all parameters", func(t *testing.T) {
		client := NewQwenClientWithConfig("api-key", "https://custom.url", "custom-model")
		if client.apiKey != "api-key" {
			t.Errorf("Expected API key 'api-key', got '%s'", client.apiKey)
		}
		if client.baseURL != "https://custom.url" {
			t.Errorf("Expected base URL 'https://custom.url', got '%s'", client.baseURL)
		}
		if client.model != "custom-model" {
			t.Errorf("Expected model 'custom-model', got '%s'", client.model)
		}
	})

	t.Run("with defaults", func(t *testing.T) {
		client := NewQwenClientWithConfig("api-key", "", "")
		if client.baseURL != defaultQwenAPIURL {
			t.Errorf("Expected default base URL, got '%s'", client.baseURL)
		}
		if client.model != defaultModel {
			t.Errorf("Expected default model, got '%s'", client.model)
		}
	})
}

func TestQwenClient_RequestTimeout(t *testing.T) {
	// Create a server that sleeps longer than client timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"choices": [{"message": {"content": "test"}}]}`))
		if err != nil {
			t.Errorf("Failed to write response body: %v", err)
		}
	}))
	defer server.Close()

	client := NewQwenClientWithConfig("test-key", server.URL, "test-model")
	// Set a very short timeout
	client.httpClient.Timeout = 100 * time.Millisecond

	ctx := context.Background()
	_, err := client.GenerateCommitMessage(ctx, "test diff")

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to make request") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestQwenClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"choices": [{"message": {"content": "test"}}]}`))
		if err != nil {
			t.Errorf("Failed to write response body: %v", err)
		}
	}))
	defer server.Close()

	client := NewQwenClientWithConfig("test-key", server.URL, "test-model")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.GenerateCommitMessage(ctx, "test diff")

	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
}
