package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/siddhartha/rune/internal/commit"
	"github.com/siddhartha/rune/internal/config"
	"github.com/siddhartha/rune/internal/git"
	"github.com/siddhartha/rune/internal/llm"
	"github.com/siddhartha/rune/internal/ui"
)

var (
	// Command line flags
	editFlag    bool
	allFlag     bool
	modelFlag   string
	dryRunFlag  bool
	verboseFlag bool
	setupFlag   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rune",
	Short: "Generate AI-powered Git commit messages",
	Long: `Rune is a CLI tool that generates descriptive Git commit messages
by analyzing staged diffs using Qwen AI models.

The tool follows GitHub commit message conventions and allows you to edit
the generated message before committing.`,
	RunE: generateCommitMessage,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		ui.HandleError(err)
		os.Exit(1)
	}
}

func init() {
	// Define flags
	rootCmd.Flags().BoolVarP(&editFlag, "edit", "e", true, "Open editor to edit the generated commit message")
	rootCmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Include unstaged changes in addition to staged changes")
	rootCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Override the configured model")
	rootCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Generate commit message without actually committing")
	rootCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVar(&setupFlag, "setup", false, "Run interactive setup to configure AI provider")
}

// generateCommitMessage is the main function that orchestrates the commit message generation
func generateCommitMessage(cmd *cobra.Command, args []string) error {

	// Handle setup flag
	if setupFlag {
		_, err := config.InteractiveSetup()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		ui.Success("Setup completed! You can now run rune to generate commit messages.")
		return nil
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Run interactive setup if not configured
	if cfg == nil || !config.IsConfigured() {
		ui.Info("Rune is not configured yet.")
		cfg, err = config.InteractiveSetup()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
	}

	// Use configurable timeout, default to 60 seconds
	timeout := 60 * time.Second
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// check if the current directory is a git repository
	if !isGitRepository() {
		return fmt.Errorf("not a git repository")
	}

	// go to the root of the git repository
	rootDir, err := getGitRootDir()
	if err != nil {
		return fmt.Errorf("failed to get git root directory: %w", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		return fmt.Errorf("failed to change to git root directory: %w", err)
	}

	if verboseFlag {
		providerName := llm.GetProviderDisplayName(cfg.Provider)
		model := cfg.Model
		if modelFlag != "" {
			model = modelFlag
		}
		fmt.Printf("🤖 Using %s with model %s\n", providerName, model)
		fmt.Println("🔍 Extracting git diff...")
	}

	// Determine what changes to include based on config and flags
	includeAll := allFlag || !cfg.StagedOnly

	// Track files staged by the tool (for possible unstage on quit)
	var stagedByTool []string
	var commitSuccessful bool

	totalStagedFiles := 0

	// If we're including all changes and config allows auto-staging
	if includeAll && cfg.AutoStageAll {
		spinner := ui.NewSpinner("Staging all changes...")
		spinner.Start()
		
		stageResult, err := git.AtomicStageAll()
		spinner.Stop()
		
		if err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}
		
		stagedByTool = stageResult.NewlyStaged
		totalStagedFiles = len(stageResult.TotalStaged)

		if len(stagedByTool) > 0 {
			ui.Success("All changes staged successfully")
		}
	}

	// Cleanup staged files if commit fails or user quits
	defer func() {
		if !commitSuccessful && len(stagedByTool) > 0 {
			if verboseFlag {
				ui.Info("Cleaning up staged files...")
			}
			if unstageErr := git.UnstageFiles(stagedByTool); unstageErr != nil {
				ui.Warning(fmt.Sprintf("Failed to cleanup staged files: %v", unstageErr))
			}
		}
	}()

	if totalStagedFiles == 0 {
		ui.Info("No changes to commit")
		return nil
	}

	// Extract the git diff
	spinner := ui.NewSpinner("Analyzing changes...")
	spinner.Start()
	
	diff, err := git.ExtractDiff(true)
	spinner.Stop()
	
	if err != nil {
		return fmt.Errorf("failed to extract git diff: %w", err)
	}

	if verboseFlag {
		ui.Info(fmt.Sprintf("Found %d characters of changes", len(diff)))
	}

	// Initialize the LLM client based on configuration
	client, err := llm.NewLLMClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	var finalMessage string
	for {
		spinner := ui.NewSpinner("Generating commit message...")
		spinner.Start()
		
		// Generate the commit message
		rawMessage, err := client.GenerateCommitMessage(ctx, diff)
		spinner.UpdateMessage("Formatting commit message...")
		
		if err != nil {
			spinner.Stop()
			return fmt.Errorf("failed to generate commit message: %w", err)
		}

		// Format the commit message
		message, err := commit.FormatCommitMessage(rawMessage)
		spinner.Stop()
		
		if err != nil {
			return fmt.Errorf("failed to format commit message: %w", err)
		}

		// Validate the message
		if err := commit.ValidateMessage(message); err != nil {
			ui.Warning(err.Error())
		}

		ui.PreviewCommitMessage(message.Format())
		ui.ShowCommitOptions()
		var choice string
		if _, err := fmt.Scanln(&choice); err != nil {
			ui.Warning(fmt.Sprintf("Failed to read input: %v", err))
		}

		switch choice {
		case "1":
			continue // re-generate
		case "2":
			finalMessage = message.Format()
		case "3":
			editedMessage, err := openEditor(message.Format())
			if err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}
			if strings.TrimSpace(editedMessage) == "" {
				ui.Info("No changes made. Returning to options.")
				continue
			}
			finalMessage = editedMessage
		case "4":
			ui.Info("Aborted. No commit was made.")
			return nil // defer will handle cleanup
		default:
			ui.Warning("Invalid choice. Please enter 1, 2, 3, or 4.")
			continue
		}
		break
	}

	// Commit with the final message
	if err := commitWithMessage(finalMessage); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	commitSuccessful = true
	ui.Success("Successfully committed changes!")
	return nil
}

// openEditor opens the user's preferred editor to edit the commit message
func openEditor(initialMessage string) (string, error) {
	// Create a temporary file with .gitcommit extension for syntax highlighting
	tmpFile, err := os.CreateTemp("", "COMMIT_EDITMSG")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			ui.Warning(fmt.Sprintf("Failed to remove temp file: %v", err))
		}
	}()

	// Create enhanced commit message template
	template := buildCommitTemplate(initialMessage)
	
	// Write the template to the temp file
	if _, err := tmpFile.WriteString(template); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Get the editor from environment or use default
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to vi if no EDITOR is set
	}

	// Open the editor
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read and clean the edited content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	return cleanCommitMessage(string(content)), nil
}

// commitWithMessage commits the changes with the given message
func commitWithMessage(message string) error {
	// Create a temporary file for the commit message
	tmpFile, err := os.CreateTemp("", "commit-msg-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp commit file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			ui.Warning(fmt.Sprintf("Failed to remove temp file: %v", err))
		}
	}()

	// Write the message to the temp file
	if _, err := tmpFile.WriteString(message); err != nil {
		return fmt.Errorf("failed to write commit message: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp commit file: %w", err)
	}

	// Execute git commit
	cmd := exec.Command("git", "commit", "-F", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// isGitRepository checks if the current directory is a git repository
func isGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}


// getGitRootDir returns the root directory of the git repository
func getGitRootDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// buildCommitTemplate creates an enhanced commit message template
func buildCommitTemplate(initialMessage string) string {
	template := initialMessage + "\n\n"
	template += "# Please enter the commit message for your changes. Lines starting\n"
	template += "# with '#' will be ignored, and an empty message aborts the commit.\n"
	template += "#\n"
	template += "# Conventional Commit Format:\n"
	template += "# <type>[optional scope]: <description>\n"
	template += "#\n"
	template += "# [optional body]\n"
	template += "#\n"
	template += "# [optional footer(s)]\n"
	template += "#\n"
	template += "# Types: feat, fix, docs, style, refactor, test, chore\n"
	template += "# Example: feat(auth): add OAuth2 login support\n"
	template += "#\n"
	template += "# Tips:\n"
	template += "# - Use imperative mood (\"add\" not \"added\")\n"
	template += "# - Keep the first line under 50 characters\n"
	template += "# - Separate subject from body with a blank line\n"
	template += "# - Wrap body at 72 characters\n"
	
	return template
}

// cleanCommitMessage removes comments and cleans up the commit message
func cleanCommitMessage(content string) string {
	lines := strings.Split(content, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		// Remove comment lines (starting with #)
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			cleanLines = append(cleanLines, line)
		}
	}
	
	// Join lines and trim
	result := strings.Join(cleanLines, "\n")
	result = strings.TrimSpace(result)
	
	return result
}

