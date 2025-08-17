package llm

import (
	"fmt"

	"github.com/siddhartha/rune/internal/config"
)

// NewLLMClient creates a new LLM client based on the configuration
func NewLLMClient(cfg *config.Config) (LLMClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Set the environment variable for the session
	if err := cfg.SetEnvVar(); err != nil {
		return nil, fmt.Errorf("failed to set environment variable: %w", err)
	}

	switch cfg.Provider {
	case config.ProviderGemini:
		return NewGeminiClient(cfg.Model)
	case config.ProviderOpenRouter:
		return NewOpenRouterClient(cfg.Model)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}

// GetProviderDisplayName returns a human-readable name for the provider
func GetProviderDisplayName(provider string) string {
	switch provider {
	case config.ProviderGemini:
		return "Google Gemini"
	case config.ProviderOpenRouter:
		return "OpenRouter"
	default:
		return "Unknown"
	}
}
