package models

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ModelInfo represents information about a model
type ModelInfo struct {
	ID          string // Full model ID (e.g., "deepseek/deepseek-chat")
	ShortName   string // Short name (e.g., "deepseek", "qwen")
	Name        string // Display name
	Provider    string // Provider (novita, gemini, openrouter)
	Company     string // Company that created the model
	Description string // Brief description
	ContextSize int    // Context window size
	IsDefault   bool   // Whether this is the default for the provider
}

// ModelRegistry holds all available models
var ModelRegistry = map[string]*ModelInfo{
	// Gemini models (direct Google provider)
	"gemini-1.5-flash": {
		ID:          "gemini-1.5-flash",
		ShortName:   "g15",
		Name:        "Gemini 1.5 Flash",
		Provider:    "gemini",
		Company:     "Google",
		Description: "Fast and efficient model for quick tasks",
		ContextSize: 1000000,
		IsDefault:   false,
	},
	"gemini-1.5-pro": {
		ID:          "gemini-1.5-pro",
		ShortName:   "gp",
		Name:        "Gemini 1.5 Pro",
		Provider:    "gemini",
		Company:     "Google",
		Description: "More capable model for complex reasoning",
		ContextSize: 2000000,
		IsDefault:   false,
	},
	"gemini-2.0-flash-exp": {
		ID:          "gemini-2.0-flash-exp",
		ShortName:   "g2",
		Name:        "Gemini 2.0 Flash Experimental",
		Provider:    "gemini",
		Company:     "Google",
		Description: "Latest experimental model with improved capabilities",
		ContextSize: 1000000,
		IsDefault:   true,
	},

	// OpenRouter models
	"deepseek/deepseek-v3": {
		ID:          "deepseek/deepseek-chat-v3:free",
		ShortName:   "dv3",
		Name:        "DeepSeek V3",
		Provider:    "openrouter",
		Company:     "DeepSeek",
		Description: "Large context window, excellent code understanding",
		ContextSize: 163840,
		IsDefault:   true,
	},
	"deepseek/r1": {
		ID:          "deepseek/deepseek-r1-0528:free",
		ShortName:   "dr1",
		Name:        "DeepSeek R1",
		Provider:    "openrouter",
		Company:     "DeepSeek",
		Description: "Advanced reasoning and code generation",
		ContextSize: 163840,
		IsDefault:   false,
	},
	"google/gemini-2.0-flash-exp:free": {
		ID:          "google/gemini-2.0-flash-exp:free",
		ShortName:   "g2f",
		Name:        "Gemini 2.0 Flash Experimental",
		Provider:    "openrouter",
		Company:     "Google",
		Description: "Free tier, experimental model with large context",
		ContextSize: 1048576,
		IsDefault:   false,
	},
	"mistralai/mistral-7b-instruct": {
		ID:          "mistralai/mistral-7b-instruct",
		ShortName:   "m7",
		Name:        "Mistral 7B Instruct",
		Provider:    "openrouter",
		Company:     "Mistral AI",
		Description: "Lightweight and efficient, good for quick code tasks",
		ContextSize: 32768,
		IsDefault:   false,
	},
	"meta-llama/llama-3.1-70b-instruct": {
		ID:          "meta-llama/llama-3.3-70b-instruct:free",
		ShortName:   "l3",
		Name:        "Llama 3.1 70B Instruct",
		Provider:    "openrouter",
		Company:     "Meta",
		Description: "Strong programming capabilities, excellent code understanding",
		ContextSize: 65536,
		IsDefault:   false,
	},
	"gryphe/mythomax-l2-13b": {
		ID:          "gryphe/mythomax-l2-13b",
		ShortName:   "mx",
		Name:        "MythoMax L2 13B",
		Provider:    "openrouter",
		Company:     "Gryphe",
		Description: "Top-ranked programming model, good balance of performance",
		ContextSize: 4096,
		IsDefault:   false,
	},
	"qwen/qwq-32b-preview": {
		ID:          "qwen/qwq-32b-preview",
		ShortName:   "qwq",
		Name:        "Qwen QwQ 32B Preview",
		Provider:    "openrouter",
		Company:     "Alibaba",
		Description: "Excellent for coding tasks, up to 450 tokens/sec",
		ContextSize: 32768,
		IsDefault:   false,
	},
}

// Model aliases for even easier typing
var ModelAliases = map[string]string{
	// Ultra-short aliases (1-2 chars)
	"d": "d",  // DeepSeek (default for OpenRouter)
	"g": "g2", // Gemini 2.0 (default for Google)
	"m": "m7", // Mistral 7B
	"l": "l3", // Llama 3.1

	// Descriptive aliases
	"deep":    "d",   // DeepSeek
	"gemini":  "g2",  // Gemini 2.0
	"mistral": "m7",  // Mistral 7B
	"llama":   "l3",  // Llama 3.1
	"mytho":   "mx",  // MythoMax
	"qwen":    "qwq", // Qwen QwQ
	"pro":     "gp",  // Gemini Pro

	// Version-specific aliases
	"g1":  "g15", // Gemini 1.5
	"g15": "g15", // Gemini 1.5 Flash
	"g2":  "g2",  // Gemini 2.0
	"m7":  "m7",  // Mistral 7B
	"l3":  "l3",  // Llama 3.1

	// Provider shortcuts
	"google":     "g2", // Default Google model
	"openrouter": "d",  // Default OpenRouter model
}

// FindModel finds a model by ID, short name, or alias
func FindModel(query string) (*ModelInfo, error) {
	query = strings.TrimSpace(strings.ToLower(query))

	// First try exact ID match
	for id, model := range ModelRegistry {
		if strings.ToLower(id) == query {
			return model, nil
		}
	}

	// Then try short name match
	for _, model := range ModelRegistry {
		if strings.ToLower(model.ShortName) == query {
			return model, nil
		}
	}

	// Finally try alias match
	if aliasTarget, exists := ModelAliases[query]; exists {
		// Recursively resolve alias
		return FindModel(aliasTarget)
	}

	return nil, fmt.Errorf("model not found: %s", query)
}

// GetModelsByProvider returns all models for a specific provider
func GetModelsByProvider(provider string) []*ModelInfo {
	var models []*ModelInfo
	for _, model := range ModelRegistry {
		if model.Provider == provider {
			models = append(models, model)
		}
	}

	// Sort by default first, then by name
	sort.Slice(models, func(i, j int) bool {
		if models[i].IsDefault != models[j].IsDefault {
			return models[i].IsDefault
		}
		return models[i].Name < models[j].Name
	})

	return models
}

// GetAllModels returns all models sorted by provider and name
func GetAllModels() []*ModelInfo {
	var models []*ModelInfo
	for _, model := range ModelRegistry {
		models = append(models, model)
	}

	// Sort by provider, then by default status, then by name
	sort.Slice(models, func(i, j int) bool {
		if models[i].Provider != models[j].Provider {
			return models[i].Provider < models[j].Provider
		}
		if models[i].IsDefault != models[j].IsDefault {
			return models[i].IsDefault
		}
		return models[i].Name < models[j].Name
	})

	return models
}

// GetDefaultModel returns the default model for a provider
func GetDefaultModel(provider string) (*ModelInfo, error) {
	for _, model := range ModelRegistry {
		if model.Provider == provider && model.IsDefault {
			return model, nil
		}
	}
	return nil, fmt.Errorf("no default model found for provider: %s", provider)
}

// FormatModelsHelp returns a formatted string for help text
func FormatModelsHelp() string {
	var help strings.Builder
	help.WriteString("\nAvailable models:\n")

	providers := []string{"gemini", "openrouter"}

	for _, provider := range providers {
		models := GetModelsByProvider(provider)
		if len(models) == 0 {
			continue
		}

		help.WriteString(fmt.Sprintf("\n  %s:\n", cases.Title(language.English).String(provider)))
		for _, model := range models {
			defaultMarker := ""
			if model.IsDefault {
				defaultMarker = " (default)"
			}

			// Show both short name and common aliases
			aliases := getCommonAliases(model.ShortName)
			aliasText := ""
			if len(aliases) > 0 {
				aliasText = fmt.Sprintf(" | %s", strings.Join(aliases, ", "))
			}

			help.WriteString(fmt.Sprintf("    %-6s%s %s%s\n",
				model.ShortName, aliasText, model.Name, defaultMarker))
		}
	}

	help.WriteString("\nQuick examples:\n")
	help.WriteString("  --model d      # DeepSeek (1 char!)\n")
	help.WriteString("  --model g      # Gemini 2.0\n")
	help.WriteString("  --model m      # Mistral 7B\n")
	help.WriteString("  --model l      # Llama 3.1\n")

	return help.String()
}

// getCommonAliases returns the most useful aliases for a short name
func getCommonAliases(shortName string) []string {
	var aliases []string

	for alias, target := range ModelAliases {
		if target == shortName && alias != shortName {
			// Only show the most useful aliases (avoid cluttering)
			if len(alias) <= 6 && !strings.Contains(alias, "router") {
				aliases = append(aliases, alias)
			}
		}
	}

	// Sort by length (shortest first)
	sort.Slice(aliases, func(i, j int) bool {
		if len(aliases[i]) != len(aliases[j]) {
			return len(aliases[i]) < len(aliases[j])
		}
		return aliases[i] < aliases[j]
	})

	// Limit to 3 most useful aliases
	if len(aliases) > 3 {
		aliases = aliases[:3]
	}

	return aliases
}

// ValidateModelForProvider checks if a model is valid for a provider
func ValidateModelForProvider(modelID, provider string) error {
	model, err := FindModel(modelID)
	if err != nil {
		return err
	}

	if model.Provider != provider {
		return fmt.Errorf("model %s belongs to provider %s, not %s",
			modelID, model.Provider, provider)
	}

	return nil
}
