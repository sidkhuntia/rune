package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/siddhartha/rune/internal/commit"
	"github.com/siddhartha/rune/internal/config"
	"github.com/siddhartha/rune/internal/git"
	"github.com/siddhartha/rune/internal/llm"
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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Handle setup flag
	if setupFlag {
		_, err := config.InteractiveSetup()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		fmt.Println("Setup completed! You can now run rune to generate commit messages.")
		return nil
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Run interactive setup if not configured
	if cfg == nil || !config.IsConfigured() {
		fmt.Println("🔧 Rune is not configured yet.")
		cfg, err = config.InteractiveSetup()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
	}

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

	// If we're including all changes and config allows auto-staging
	if includeAll && cfg.AutoStageAll {

		previousStagedFiles, err := git.ListStagedFiles()
		if err != nil {
			return fmt.Errorf("failed to list staged files: %w", err)
		}

		if err := stageAllChanges(); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}
		fmt.Println("✅ All changes staged successfully")
		// Track what we just staged
		toBeStagedFiles, err := git.ListStagedFiles()
		if err != nil {
			return fmt.Errorf("failed to list staged files: %w", err)
		}

		for _, file := range toBeStagedFiles {
			if !slices.Contains(previousStagedFiles, file) {
				stagedByTool = append(stagedByTool, file)
			}
		}
	}

	// Extract the git diff
	diff, err := git.ExtractDiff(true)
	if err != nil {
		return fmt.Errorf("failed to extract git diff: %w", err)
	}

	if verboseFlag {
		fmt.Printf("📄 Found %d characters of changes\n", len(diff))
	}

	// Initialize the LLM client based on configuration
	client, err := llm.NewLLMClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	var finalMessage string
	for {
		if verboseFlag {
			fmt.Println("🤖 Generating commit message...")
		}
		// Generate the commit message
		rawMessage, err := client.GenerateCommitMessage(ctx, diff)
		if err != nil {
			return fmt.Errorf("failed to generate commit message: %w", err)
		}

		if verboseFlag {
			fmt.Println("📝 Formatting commit message...")
		}
		// Format the commit message
		message, err := commit.FormatCommitMessage(rawMessage)
		if err != nil {
			return fmt.Errorf("failed to format commit message: %w", err)
		}

		// Validate the message
		if err := commit.ValidateMessage(message); err != nil {
			fmt.Printf("⚠️  Warning: %v\n", err)
		}

		fmt.Println("\nGenerated commit message:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(message.Format())
		fmt.Println(strings.Repeat("-", 50))

		// Interactive panel
		fmt.Println("What would you like to do?")
		fmt.Println("1. 🔄 Re-generate commit message")
		fmt.Println("2. ✅ Commit as-is")
		fmt.Println("3. 📝 Edit and commit")
		fmt.Println("4. 🚫 Quit (unstage any files staged by this tool)")
		fmt.Print("Enter your choice (1-4): ")
		var choice string
		if _, err := fmt.Scanln(&choice); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read input: %v\n", err)
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
				fmt.Println("📝 No changes made. Returning to options.")
				continue
			}
			finalMessage = editedMessage
		case "4":
			if len(stagedByTool) > 0 {
				fmt.Println("Unstaging files staged by this tool...")
				_ = git.UnstageFiles(stagedByTool)
			}
			fmt.Println("Aborted. No commit was made.")
			return nil
		default:
			fmt.Println("Invalid choice. Please enter 1, 2, 3, or 4.")
			continue
		}
		break
	}

	// Commit with the final message
	if err := commitWithMessage(finalMessage); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	fmt.Println("✅ Successfully committed changes!")
	return nil
}

// openEditor opens the user's preferred editor to edit the commit message
func openEditor(initialMessage string) (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "commitmsg-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temp file: %v\n", err)
		}
	}()

	// Write the initial message to the temp file
	if _, err := tmpFile.WriteString(initialMessage); err != nil {
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

	// Read the edited content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
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
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temp file: %v\n", err)
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

// stageAllChanges stages all modified files
func stageAllChanges() error {
	cmd := exec.Command("git", "add", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stage changes: %w\nOutput: %s", err, string(output))
	}
	return nil
}
