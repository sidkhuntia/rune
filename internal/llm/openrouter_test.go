package llm

import (
	"testing"
)

func TestNewOpenRouterClient(t *testing.T) {
	// Test with no API key
	client, err := NewOpenRouterClient("")
	if err == nil {
		t.Error("Expected error when OPENROUTER_API_KEY is not set")
	}
	if client != nil {
		t.Error("Expected nil client when API key is not set")
	}

	// Test with default model
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	client, err = NewOpenRouterClient("")
	if err != nil {
		t.Errorf("Expected no error with API key set, got: %v", err)
	}
	if client == nil {
		t.Error("Expected non-nil client")
	}
	if client.model != "deepseek/deepseek-chat" {
		t.Errorf("Expected default model 'deepseek/deepseek-chat', got: %s", client.model)
	}

	// Test with custom model
	client, err = NewOpenRouterClient("qwen/qwen-3-32b")
	if err != nil {
		t.Errorf("Expected no error with custom model, got: %v", err)
	}
	if client.model != "qwen/qwen-3-32b" {
		t.Errorf("Expected custom model 'qwen/qwen-3-32b', got: %s", client.model)
	}
}