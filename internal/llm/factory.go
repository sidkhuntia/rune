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
	case config.ProviderNovita:
		return NewQwenClient()
	case config.ProviderGemini:
		return NewGeminiClient()
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}

// GetProviderDisplayName returns a human-readable name for the provider
func GetProviderDisplayName(provider string) string {
	switch provider {
	case config.ProviderNovita:
		return "Novita AI (Qwen)"
	case config.ProviderGemini:
		return "Google Gemini Flash"
	default:
		return "Unknown"
	}
}
