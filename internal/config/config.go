package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the application configuration
type Config struct {
	Provider string `json:"provider"` // "novita" or "gemini"
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

// Provider constants
const (
	ProviderNovita = "novita"
	ProviderGemini = "gemini"
)

// Default models for each provider
var DefaultModels = map[string]string{
	ProviderNovita: "qwen/qwen2.5-7b-instruct",
	ProviderGemini: "gemini-1.5-flash",
}

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "commitgen")
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

	data, err := ioutil.ReadFile(configPath)
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

	if err := ioutil.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// InteractiveSetup guides the user through initial setup
func InteractiveSetup() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🚀 Welcome to Gafu!")
	fmt.Println("Let's set up your AI provider for generating commit messages.")

	// Choose provider
	fmt.Println("Choose your AI provider:")
	fmt.Println("1. Novita AI (Qwen models) - https://novita.ai/")
	fmt.Println("2. Google Gemini Flash")
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
		provider = ProviderNovita
		model = DefaultModels[ProviderNovita]
		apiKeyPrompt = "Please enter your Novita AI API key"
		setupURL = "Get your API key at: https://novita.ai/settings/key-management"
	case "2":
		provider = ProviderGemini
		model = DefaultModels[ProviderGemini]
		apiKeyPrompt = "Please enter your Google Gemini API key"
		setupURL = "Get your API key at: https://makersuite.google.com/app/apikey"
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

	config := &Config{
		Provider: provider,
		APIKey:   apiKey,
		Model:    model,
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
	return err == nil && config != nil && config.Provider != "" && config.APIKey != ""
}

// GetEnvVarName returns the appropriate environment variable name for the provider
func (c *Config) GetEnvVarName() string {
	switch c.Provider {
	case ProviderNovita:
		return "NOVITA_API_KEY"
	case ProviderGemini:
		return "GEMINI_API_KEY"
	default:
		return ""
	}
}

// SetEnvVar sets the appropriate environment variable for the current session
func (c *Config) SetEnvVar() error {
	envVar := c.GetEnvVarName()
	if envVar == "" {
		return fmt.Errorf("unknown provider: %s", c.Provider)
	}

	return os.Setenv(envVar, c.APIKey)
}
