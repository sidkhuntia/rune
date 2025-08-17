package ui

import (
	"fmt"
	"strings"
)

// UserError represents a user-friendly error with suggestions
type UserError struct {
	Title          string
	Description    string
	Suggestions    []string
	TechnicalError error
}

// Error implements the error interface
func (e *UserError) Error() string {
	return e.Title
}

// Display shows a user-friendly error message with suggestions
func (e *UserError) Display() {
	Error(e.Title)

	if e.Description != "" {
		fmt.Printf("   %s\n", e.Description)
	}

	if len(e.Suggestions) > 0 {
		fmt.Println("\n💡 Suggestions:")
		for _, suggestion := range e.Suggestions {
			fmt.Printf("   • %s\n", suggestion)
		}
	}

	if e.TechnicalError != nil {
		fmt.Printf("\n🔧 Technical details: %v\n", e.TechnicalError)
	}
}

// TranslateError converts common errors to user-friendly messages
func TranslateError(err error) *UserError {
	errMsg := err.Error()

	// Git-related errors
	if strings.Contains(errMsg, "not a git repository") {
		return &UserError{
			Title:       "Not in a Git repository",
			Description: "Rune needs to be run inside a Git repository to analyze changes.",
			Suggestions: []string{
				"Navigate to your project directory",
				"Initialize a Git repository with 'git init'",
				"Clone an existing repository with 'git clone <url>'",
			},
			TechnicalError: err,
		}
	}

	if strings.Contains(errMsg, "no changes found") {
		return &UserError{
			Title:       "No changes to commit",
			Description: "There are no staged changes to generate a commit message for.",
			Suggestions: []string{
				"Stage your changes with 'git add <file>' or 'git add .'",
				"Use the --all flag to automatically stage all changes",
				"Check git status with 'git status'",
			},
			TechnicalError: err,
		}
	}

	// API Key related errors
	if strings.Contains(errMsg, "failed to retrieve API key") {
		return &UserError{
			Title:       "API key not found",
			Description: "Your AI provider API key is not configured or accessible.",
			Suggestions: []string{
				"Run 'rune --setup' to configure your API key",
				"Check that your keyring/keychain is accessible",
				"Verify your API key is still valid",
			},
			TechnicalError: err,
		}
	}

	// Network/LLM errors
	if strings.Contains(errMsg, "failed to generate commit message") {
		return &UserError{
			Title:       "AI service unavailable",
			Description: "Unable to generate commit message using the AI service.",
			Suggestions: []string{
				"Check your internet connection",
				"Verify your API key is valid and has sufficient credits",
				"Try again in a few moments",
				"Check the AI provider's status page",
			},
			TechnicalError: err,
		}
	}

	// Timeout errors
	if strings.Contains(errMsg, "context deadline exceeded") {
		return &UserError{
			Title:       "Request timed out",
			Description: "The AI service took too long to respond.",
			Suggestions: []string{
				"Try again - the service might be temporarily slow",
				"Consider reducing the size of your changes",
				"Configure a longer timeout in your settings",
			},
			TechnicalError: err,
		}
	}

	// Model-related errors
	if strings.Contains(errMsg, "model not found") {
		return &UserError{
			Title:       "Model not found",
			Description: "The specified model is not available.",
			Suggestions: []string{
				"Run 'rune --list-models' to see available models",
				"Use a model short name like 'deepseek' or 'qwen'",
				"Check spelling of the model name",
			},
			TechnicalError: err,
		}
	}

	if strings.Contains(errMsg, "failed to resolve model") {
		return &UserError{
			Title:       "Invalid model",
			Description: "Unable to use the specified model.",
			Suggestions: []string{
				"Run 'rune --list-models' to see available options",
				"Verify the model name is correct",
				"Try using a short name like 'deepseek' instead",
			},
			TechnicalError: err,
		}
	}

	// Configuration errors
	if strings.Contains(errMsg, "setup failed") || strings.Contains(errMsg, "failed to load config") {
		return &UserError{
			Title:       "Configuration error",
			Description: "There's an issue with your Rune configuration.",
			Suggestions: []string{
				"Run 'rune --setup' to reconfigure",
				"Check file permissions in ~/.config/rune/",
				"Try removing ~/.config/rune/ and running setup again",
			},
			TechnicalError: err,
		}
	}

	// Default case
	return &UserError{
		Title:       "An error occurred",
		Description: "Something went wrong while running Rune.",
		Suggestions: []string{
			"Try running the command again",
			"Check the technical details below for more information",
			"Report this issue if it persists",
		},
		TechnicalError: err,
	}
}

// HandleError translates and displays user-friendly error messages
func HandleError(err error) {
	if err == nil {
		return
	}

	userErr := TranslateError(err)
	userErr.Display()
}
