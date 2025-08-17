package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/siddhartha/rune/internal/models"
	"github.com/zalando/go-keyring"
)

// Config represents the application configuration
type Config struct {
	Provider       string `json:"provider"` // "novita" or "gemini"
	Model          string `json:"model"`
	StagedOnly     bool   `json:"staged_only"`               // true for staged only, false for all changes
	AutoStageAll   bool   `json:"auto_stage_all"`            // if true, automatically stage all changes when staged_only=false
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"` // configurable timeout, defaults to 60
}

// Provider constants
const (
	ProviderGemini     = "gemini"
	ProviderOpenRouter = "openrouter"

	// File permissions
	configDirPerm  = 0755
	configFilePerm = 0600
)

// Default models for each provider
var DefaultModels = map[string]string{
	ProviderGemini:     "gemini-2.0-flash-exp",
	ProviderOpenRouter: "deepseek/deepseek-chat",
}

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "rune")
	if err := os.MkdirAll(configDir, configDirPerm); err != nil {
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

	if err := os.WriteFile(configPath, data, configFilePerm); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// InteractiveSetup guides the user through initial setup
func InteractiveSetup() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)

	// Use UI package for better formatting
	fmt.Printf("\n\033[1m\033[36m🚀 Welcome to Rune!\033[0m\033[0m\n")
	fmt.Printf("\033[2mLet's set up your AI provider for generating commit messages.\033[0m\n\n")

	// Choose provider
	fmt.Println("Choose your AI provider:")
	fmt.Println("1. Google Gemini")
	fmt.Println("2. OpenRouter (Multiple models) - https://openrouter.ai/")
	fmt.Print("\nEnter your choice (1 or 2): ")

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
		provider = ProviderGemini
		model = DefaultModels[ProviderGemini]
		apiKeyPrompt = "Please enter your Google Gemini API key"
		setupURL = "Get your API key at: https://makersuite.google.com/app/apikey"
	case "2":
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

// ResolveModel resolves a model string to full model info and updates config if needed
func (c *Config) ResolveModel(modelInput string) (*models.ModelInfo, error) {
	if modelInput == "" {
		// Use configured model
		if c.Model == "" {
			// Get default for current provider
			return models.GetDefaultModel(c.Provider)
		}
		return models.FindModel(c.Model)
	}

	// User specified a model
	model, err := models.FindModel(modelInput)
	if err != nil {
		return nil, err
	}

	// Check if we need to switch providers
	if model.Provider != c.Provider {
		// We'll need API key for the new provider
		return model, nil
	}

	return model, nil
}

// SetDefaultModel sets the default model in config
func (c *Config) SetDefaultModel(modelInput string) error {
	model, err := models.FindModel(modelInput)
	if err != nil {
		return err
	}

	c.Model = model.ID
	c.Provider = model.Provider

	return c.Save()
}

// EnsureAPIKeyForProvider ensures API key exists for the given provider
func (c *Config) EnsureAPIKeyForProvider(provider string) error {
	// Temporarily switch provider to check API key
	originalProvider := c.Provider
	c.Provider = provider

	_, err := c.GetAPIKey()
	if err != nil {
		// API key doesn't exist, prompt user
		fmt.Printf("\n🔑 API key required for %s\n", provider)
		apiKey, err := c.promptForAPIKey(provider)
		if err != nil {
			c.Provider = originalProvider // restore
			return err
		}

		if err := c.SetAPIKey(apiKey); err != nil {
			c.Provider = originalProvider // restore
			return err
		}
	}

	// Don't restore provider - we want to keep the new one
	return nil
}

// promptForAPIKey prompts user to enter API key for a provider
func (c *Config) promptForAPIKey(provider string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	var setupURL string
	switch provider {
	case ProviderGemini:
		setupURL = "Get your API key at: https://makersuite.google.com/app/apikey"
	case ProviderOpenRouter:
		setupURL = "Get your API key at: https://openrouter.ai/keys"
	default:
		return "", fmt.Errorf("unknown provider: %s", provider)
	}

	fmt.Printf("%s\n", setupURL)
	fmt.Printf("Please enter your %s API key: ", provider)

	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read API key: %w", err)
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	return apiKey, nil
}

// setupOpenRouterModel allows user to select from available OpenRouter models
func setupOpenRouterModel(reader *bufio.Reader) string {
	fmt.Println("\nAvailable OpenRouter models:")

	openRouterModels := models.GetModelsByProvider("openrouter")

	for i, model := range openRouterModels {
		fmt.Printf("%d. %s (%s) - %dk context - %s\n",
			i+1, model.Name, model.Company, model.ContextSize/1000, model.Description)
	}

	fmt.Printf("\nEnter your choice (1-%d): ", len(openRouterModels))
	modelChoice, err := reader.ReadString('\n')
	if err != nil {
		return DefaultModels[ProviderOpenRouter] // fallback to default
	}

	modelChoice = strings.TrimSpace(modelChoice)

	// Parse choice
	if choice := parseInt(modelChoice); choice > 0 && choice <= len(openRouterModels) {
		return openRouterModels[choice-1].ID
	}

	fmt.Printf("Invalid choice, using default: %s\n", DefaultModels[ProviderOpenRouter])
	return DefaultModels[ProviderOpenRouter]
}

// parseInt safely parses an integer string
func parseInt(s string) int {
	if s == "" {
		return 0
	}

	value, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}

	return value
}
