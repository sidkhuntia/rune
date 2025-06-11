package commitmsg

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/siddhartha/commitgen/internal/commit"
	"github.com/siddhartha/commitgen/internal/config"
	"github.com/siddhartha/commitgen/internal/git"
	"github.com/siddhartha/commitgen/internal/llm"
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
	Use:   "commitmsg",
	Short: "Generate AI-powered Git commit messages",
	Long: `CommitGen is a CLI tool that generates descriptive Git commit messages
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
		fmt.Println("Setup completed! You can now run commitmsg to generate commit messages.")
		return nil
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Run interactive setup if not configured
	if cfg == nil || !config.IsConfigured() {
		fmt.Println("🔧 CommitGen is not configured yet.")
		cfg, err = config.InteractiveSetup()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
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

	// Extract the git diff
	diff, err := git.ExtractDiff(!allFlag) // staged only by default, unless --all is specified
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

	if verboseFlag {
		fmt.Println("✅ Generated commit message:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(message.Format())
		fmt.Println(strings.Repeat("-", 50))
	}

	// Handle dry run
	if dryRunFlag {
		fmt.Println("Generated commit message:")
		fmt.Println(message.Format())
		return nil
	}

	// Edit the message if requested
	finalMessage := message.Format()
	if editFlag {
		editedMessage, err := openEditor(finalMessage)
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}
		finalMessage = editedMessage
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
	tmpFile, err := ioutil.TempFile("", "commitmsg-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write the initial message to the temp file
	if _, err := tmpFile.WriteString(initialMessage); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

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
	content, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// commitWithMessage commits the changes with the given message
func commitWithMessage(message string) error {
	// Create a temporary file for the commit message
	tmpFile, err := ioutil.TempFile("", "commit-msg-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp commit file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write the message to the temp file
	if _, err := tmpFile.WriteString(message); err != nil {
		return fmt.Errorf("failed to write commit message: %w", err)
	}
	tmpFile.Close()

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

// hasGitChanges checks if there are any changes to commit
func hasGitChanges(staged bool) bool {
	var cmd *exec.Cmd
	if staged {
		cmd = exec.Command("git", "diff", "--cached", "--quiet")
	} else {
		cmd = exec.Command("git", "diff", "--quiet")
	}

	err := cmd.Run()
	// git diff --quiet returns non-zero exit code if there are changes
	return err != nil
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
