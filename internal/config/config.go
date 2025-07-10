package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

// Config represents the application configuration
type Config struct {
	Provider     string `json:"provider"` // "novita" or "gemini"
	Model        string `json:"model"`
	StagedOnly   bool   `json:"staged_only"`    // true for staged only, false for all changes
	AutoStageAll bool   `json:"auto_stage_all"` // if true, automatically stage all changes when staged_only=false
	TimeoutSeconds int  `json:"timeout_seconds,omitempty"` // configurable timeout, defaults to 60
}

// Provider constants
const (
	ProviderNovita    = "novita"
	ProviderGemini    = "gemini"
	ProviderOpenRouter = "openrouter"
)

// Default models for each provider
var DefaultModels = map[string]string{
	ProviderNovita:    "qwen/qwen2.5-7b-instruct",
	ProviderGemini:    "gemini-1.5-flash",
	ProviderOpenRouter: "deepseek/deepseek-chat",
}

// OpenRouter model definitions with metadata
type ModelInfo struct {
	ID           string
	Name         string
	Company      string
	ContextWindow int
	Description  string
}

var OpenRouterModels = map[string]ModelInfo{
	"deepseek/deepseek-chat": {
		ID:           "deepseek/deepseek-chat",
		Name:         "DeepSeek V3",
		Company:      "DeepSeek",
		ContextWindow: 163840,
		Description:  "Large context window, excellent code understanding",
	},
	"qwen/qwen-3-32b": {
		ID:           "qwen/qwen-3-32b",
		Name:         "Qwen 3 32B",
		Company:      "Alibaba",
		ContextWindow: 40960,
		Description:  "Excellent code comprehension, fast inference",
	},
	"mistral/mistral-small-3.2-24b": {
		ID:           "mistral/mistral-small-3.2-24b",
		Name:         "Mistral Small 3.2 24B",
		Company:      "Mistral AI",
		ContextWindow: 96000,
		Description:  "Fast inference, good code understanding",
	},
	"qwen/qwen-2.5-coder-32b": {
		ID:           "qwen/qwen-2.5-coder-32b",
		Name:         "Qwen 2.5 Coder 32B",
		Company:      "Alibaba",
		ContextWindow: 128000,
		Description:  "Specialized for code-related tasks",
	},
	"google/gemma-3-27b": {
		ID:           "google/gemma-3-27b",
		Name:         "Google Gemma 3 27B",
		Company:      "Google",
		ContextWindow: 96000,
		Description:  "Well-rounded performance, good general use",
	},
	"google/gemini-2.0-flash-exp:free": {
		ID:           "google/gemini-2.0-flash-exp:free",
		Name:         "Google Gemini 2.0 Flash Exp (Free)",
		Company:      "Google",
		ContextWindow: 1000000,
		Description:  "Free tier, experimental model with large context",
	},
}

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "rune")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.json"), nil
}

// Load loads the configuration from file
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil // Config doesn't exist yet
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// InteractiveSetup guides the user through initial setup
func InteractiveSetup() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🚀 Welcome to Rune!")
	fmt.Println("Let's set up your AI provider for generating commit messages.")

	// Choose provider
	fmt.Println("Choose your AI provider:")
	fmt.Println("1. Novita AI (Qwen models) - https://novita.ai/")
	fmt.Println("2. Google Gemini Flash")
	fmt.Println("3. OpenRouter (Multiple models) - https://openrouter.ai/")
	fmt.Print("\nEnter your choice (1, 2, or 3): ")

	choice, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	choice = strings.TrimSpace(choice)

	var provider string
	var model string
	var apiKeyPrompt string
	var setupURL string

	switch choice {
	case "1":
		provider = ProviderNovita
		model = DefaultModels[ProviderNovita]
		apiKeyPrompt = "Please enter your Novita AI API key"
		setupURL = "Get your API key at: https://novita.ai/settings/key-management"
	case "2":
		provider = ProviderGemini
		model = DefaultModels[ProviderGemini]
		apiKeyPrompt = "Please enter your Google Gemini API key"
		setupURL = "Get your API key at: https://makersuite.google.com/app/apikey"
	case "3":
		provider = ProviderOpenRouter
		model = setupOpenRouterModel(reader)
		apiKeyPrompt = "Please enter your OpenRouter API key"
		setupURL = "Get your API key at: https://openrouter.ai/keys"
	default:
		return nil, fmt.Errorf("invalid choice: %s", choice)
	}

	fmt.Printf("\n%s\n", setupURL)
	fmt.Printf("%s: ", apiKeyPrompt)

	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}

	// Ask about commit scope preference (mutually exclusive)
	fmt.Println("\nHow do you want to generate commit messages by default?")
	fmt.Println("1. Only for staged changes (recommended for precise commits)")
	fmt.Println("2. For all tracked changes (stages everything before committing)")
	fmt.Print("Enter your choice (1 or 2): ")
	commitScopeChoice, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read commit scope preference: %w", err)
	}
	commitScopeChoice = strings.TrimSpace(commitScopeChoice)

	var stagedOnly bool
	var autoStageAll bool

	switch commitScopeChoice {
	case "1":
		stagedOnly = true
		autoStageAll = false
	case "2":
		stagedOnly = false
		autoStageAll = true
		fmt.Println("\n⚠️  All tracked changes will be staged and included in every commit by default.")
	default:
		return nil, fmt.Errorf("invalid choice: %s", commitScopeChoice)
	}

	config := &Config{
		Provider:     provider,
		Model:        model,
		StagedOnly:   stagedOnly,
		AutoStageAll: autoStageAll,
	}

	// Store API key securely
	if err := config.SetAPIKey(apiKey); err != nil {
		return nil, fmt.Errorf("failed to store API key securely: %w", err)
	}

	if err := config.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\n✅ Configuration saved! Using %s with model %s\n\n", provider, model)

	return config, nil
}

// IsConfigured checks if the application is already configured
func IsConfigured() bool {
	config, err := Load()
	if err != nil || config == nil || config.Provider == "" {
		return false
	}
	
	// Check if API key exists in secure storage
	_, err = config.GetAPIKey()
	return err == nil
}

// GetEnvVarName returns the appropriate environment variable name for the provider
func (c *Config) GetEnvVarName() string {
	switch c.Provider {
	case ProviderNovita:
		return "NOVITA_API_KEY"
	case ProviderGemini:
		return "GEMINI_API_KEY"
	case ProviderOpenRouter:
		return "OPENROUTER_API_KEY"
	default:
		return ""
	}
}

// GetAPIKey retrieves the API key from secure storage
func (c *Config) GetAPIKey() (string, error) {
	service := "rune-cli"
	user := c.Provider
	
	apiKey, err := keyring.Get(service, user)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve API key from secure storage: %w", err)
	}
	
	return apiKey, nil
}

// SetAPIKey stores the API key in secure storage
func (c *Config) SetAPIKey(apiKey string) error {
	service := "rune-cli"
	user := c.Provider
	
	if err := keyring.Set(service, user, apiKey); err != nil {
		return fmt.Errorf("failed to store API key in secure storage: %w", err)
	}
	
	return nil
}

// DeleteAPIKey removes the API key from secure storage
func (c *Config) DeleteAPIKey() error {
	service := "rune-cli"
	user := c.Provider
	
	if err := keyring.Delete(service, user); err != nil {
		return fmt.Errorf("failed to delete API key from secure storage: %w", err)
	}
	
	return nil
}

// SetEnvVar sets the appropriate environment variable for the current session
func (c *Config) SetEnvVar() error {
	envVar := c.GetEnvVarName()
	if envVar == "" {
		return fmt.Errorf("unknown provider: %s", c.Provider)
	}

	apiKey, err := c.GetAPIKey()
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	return os.Setenv(envVar, apiKey)
}

// setupOpenRouterModel allows user to select from available OpenRouter models
func setupOpenRouterModel(reader *bufio.Reader) string {
	fmt.Println("\nAvailable OpenRouter models:")
	
	models := []string{
		"deepseek/deepseek-chat",
		"qwen/qwen-3-32b",
		"mistral/mistral-small-3.2-24b",
		"qwen/qwen-2.5-coder-32b",
		"google/gemma-3-27b",
		"google/gemini-2.0-flash-exp:free",
	}
	
	for i, modelID := range models {
		if info, exists := OpenRouterModels[modelID]; exists {
			fmt.Printf("%d. %s (%s) - %dk context - %s\n", 
				i+1, info.Name, info.Company, info.ContextWindow/1000, info.Description)
		}
	}
	
	fmt.Print("\nEnter your choice (1-6): ")
	modelChoice, err := reader.ReadString('\n')
	if err != nil {
		return DefaultModels[ProviderOpenRouter] // fallback to default
	}
	
	modelChoice = strings.TrimSpace(modelChoice)
	switch modelChoice {
	case "1":
		return "deepseek/deepseek-chat"
	case "2":
		return "qwen/qwen-3-32b"
	case "3":
		return "mistral/mistral-small-3.2-24b"
	case "4":
		return "qwen/qwen-2.5-coder-32b"
	case "5":
		return "google/gemma-3-27b"
	case "6":
		return "google/gemini-2.0-flash-exp:free"
	default:
		fmt.Printf("Invalid choice, using default: %s\n", DefaultModels[ProviderOpenRouter])
		return DefaultModels[ProviderOpenRouter]
	}
}
