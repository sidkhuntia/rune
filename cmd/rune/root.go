package main

import (
	"bufio"
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
	"github.com/siddhartha/rune/internal/models"
	"github.com/siddhartha/rune/internal/ui"
)

const (
	// Default timeout for API requests
	defaultTimeoutSeconds = 60

	// defaultCandidates is how many messages are proposed per round.
	defaultCandidates = 3

	// diffBudget caps diff characters sent to the model.
	diffBudget = 60_000
)

var (
	// Command line flags
	editFlag       bool
	allFlag        bool
	stagedOnlyFlag bool
	modelFlag      string
	setDefaultFlag string
	listModelsFlag bool
	dryRunFlag     bool
	verboseFlag    bool
	setupFlag      bool
	yesFlag        bool
	updateFlag     bool
	hintFlag       string
	styleFlag      string
	candidatesFlag int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rune",
	Short: "Generate AI-powered Git commit messages",
	Long: `Rune is a CLI tool that generates descriptive Git commit messages
by analyzing staged diffs using AI models.

The tool follows GitHub commit message conventions and allows you to edit
the generated message before committing.` + models.FormatModelsHelp(),
	RunE: generateCommitMessage,

	Version: currentVersion(),

	// Runtime failures are reported once by ui.HandleError; usage text is noise there.
	SilenceUsage:  true,
	SilenceErrors: true,
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
	rootCmd.Flags().BoolVarP(&stagedOnlyFlag, "staged-only", "s", false, "Generate commit message only for manually staged changes (ignores config)")
	rootCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Use specific model (short name or full ID)")
	rootCmd.Flags().StringVar(&setDefaultFlag, "set-default-model", "", "Set default model for future use")
	rootCmd.Flags().BoolVar(&listModelsFlag, "list-models", false, "List all available models and exit")
	rootCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Generate commit message without actually committing")
	rootCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Commit with the generated message without prompting (non-interactive)")
	rootCmd.Flags().BoolVar(&updateFlag, "update", false, "Update rune to the latest release and exit")
	rootCmd.Flags().StringVar(&hintFlag, "hint", "", "Guide the message (e.g. \"focus on the auth change\")")
	rootCmd.Flags().StringVar(&styleFlag, "style", "auto", "Message style: auto (match repo history), conventional, or plain")
	rootCmd.Flags().IntVarP(&candidatesFlag, "candidates", "n", defaultCandidates, "Number of candidate messages to generate (1-5)")
	rootCmd.Flags().BoolVar(&setupFlag, "setup", false, "Run interactive setup to configure AI provider")
}

// generateCommitMessage is the main function that orchestrates the commit message generation
func generateCommitMessage(cmd *cobra.Command, args []string) error {

	if updateFlag {
		return runUpdate()
	}

	// Handle list-models flag
	if listModelsFlag {
		printAllModels()
		return nil
	}

	// Handle setup flag
	if setupFlag {
		_, err := config.InteractiveSetup()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		ui.Success("Setup completed! You can now run rune to generate commit messages.")
		return nil
	}

	// Handle set-default-model flag
	if setDefaultFlag != "" {
		return handleSetDefaultModel(setDefaultFlag)
	}

	// Validate flag combinations
	if candidatesFlag < 1 || candidatesFlag > maxCandidates {
		return fmt.Errorf("--candidates must be between 1 and %d", maxCandidates)
	}
	if yesFlag && dryRunFlag {
		return fmt.Errorf("cannot use both --yes and --dry-run flags together")
	}
	if allFlag && stagedOnlyFlag {
		return fmt.Errorf("cannot use both --all and --staged-only flags together")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Fall back to API key environment variables (CI, scripts)
	if cfg == nil {
		cfg = config.FromEnv()
	}

	// Run interactive setup if not configured
	if cfg == nil || !cfg.HasAPIKey() {
		ui.Info("Rune is not configured yet.")
		cfg, err = config.InteractiveSetup()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
	}

	// Use configurable timeout with constant default
	timeout := defaultTimeoutSeconds * time.Second
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
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

	// Resolve model (this may require switching providers)
	selectedModel, err := cfg.ResolveModel(modelFlag)
	if err != nil {
		return fmt.Errorf("failed to resolve model: %w", err)
	}

	// Check if we need to switch providers
	if selectedModel.Provider != cfg.Provider {
		ui.Info(fmt.Sprintf("Switching to %s provider for model %s", selectedModel.Provider, selectedModel.Name))

		// Ensure API key exists for the new provider
		if err := cfg.EnsureAPIKeyForProvider(selectedModel.Provider); err != nil {
			return fmt.Errorf("failed to setup provider %s: %w", selectedModel.Provider, err)
		}

		// Update config temporarily (don't save unless user wants to set as default)
		cfg.Provider = selectedModel.Provider
	}

	if verboseFlag {
		providerName := llm.GetProviderDisplayName(cfg.Provider)
		ui.Info(fmt.Sprintf("Using %s with model %s", providerName, selectedModel.Name))
	}

	// Determine what changes to include based on config and flags
	// Priority: --staged-only flag > --all flag > config setting
	var includeAll bool
	if stagedOnlyFlag {
		includeAll = false // Force staged only, ignoring config
		if verboseFlag {
			ui.Info("Using --staged-only: only manually staged changes will be included")
		}
	} else {
		includeAll = allFlag || !cfg.StagedOnly
	}

	// Track files staged by the tool (for possible unstage on quit)
	var stagedByTool []string
	var commitSuccessful bool

	totalStagedFiles := 0

	// If we're including all changes and config allows auto-staging (but not when --staged-only is used)
	if includeAll && cfg.AutoStageAll && !stagedOnlyFlag {
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
	} else {
		// Get count of currently staged files
		stagedFiles, err := git.ListStagedFiles()
		if err != nil {
			return fmt.Errorf("failed to list staged files: %w", err)
		}
		totalStagedFiles = len(stagedFiles)
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

	// Extract the diff (noise filtered, size bounded) and repository context
	spinner := ui.NewSpinner("Analyzing changes...")
	spinner.Start()

	// Always get staged diff when --staged-only is used, otherwise follow existing logic
	getStagedDiff := stagedOnlyFlag || !includeAll
	diff, err := git.SmartDiff(getStagedDiff, diffBudget)
	repoCtx := git.GatherContext(getStagedDiff)
	spinner.Stop()

	if err != nil {
		return fmt.Errorf("failed to extract git diff: %w", err)
	}

	style, err := resolveStyle(styleFlag, repoCtx.Recent)
	if err != nil {
		return err
	}

	if verboseFlag {
		ui.Info(fmt.Sprintf("Found %d characters of changes (style: %s, branch: %s)", len(diff), style, repoCtx.Branch))
	}

	// Initialize the LLM client with selected model
	cfg.Model = selectedModel.ID // Update model for client creation
	client, err := llm.NewLLMClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	stdin := bufio.NewReader(os.Stdin)
	req := llm.Request{
		Diff:   diff,
		Stat:   repoCtx.Stat,
		Branch: repoCtx.Branch,
		Recent: repoCtx.Recent,
		Style:  style,
		Hint:   hintFlag,
	}

	// --yes has no one to choose between candidates, so don't pay for extras.
	count := candidatesFlag
	if yesFlag {
		count = 1
	}

	var finalMessage string
	for {
		spinner := ui.NewSpinner(generatingLabel(count))
		spinner.Start()
		// The timeout bounds one round of API calls, not time spent at the prompt.
		genCtx, cancelGen := context.WithTimeout(context.Background(), timeout)
		candidates, failures, err := generateCandidates(genCtx, client, req, count)
		cancelGen()
		spinner.Stop()
		if err != nil {
			return fmt.Errorf("failed to generate commit message: %w", err)
		}
		if len(failures) > 0 {
			ui.Warning(fmt.Sprintf("%d of %d candidates failed: %v", len(failures), count, failures[0]))
		}

		texts := make([]string, len(candidates))
		for i, c := range candidates {
			texts[i] = c.Format()
			if verr := commit.ValidateMessage(c); verr != nil {
				ui.Warning(fmt.Sprintf("Message %d: %v", i+1, verr))
			}
		}
		ui.PreviewCandidates(texts)

		if dryRunFlag {
			ui.Info("Dry run: no commit was made.")
			return nil // defer will handle cleanup
		}
		if yesFlag {
			finalMessage = texts[0]
			break
		}

		// Remember what was shown so a regenerate does not repeat itself.
		for _, c := range candidates {
			req.Avoid = append(req.Avoid, c.Subject)
		}

		act, ok := promptAction(stdin, len(candidates))
		if !ok {
			return fmt.Errorf("no input available to choose an action (use --yes to commit non-interactively)")
		}

		switch act.kind {
		case actRegenerate:
			if act.hint != "" {
				req.Hint = act.hint
			}
			continue
		case actQuit:
			ui.Info("Aborted. No commit was made.")
			return nil // defer will handle cleanup
		case actEdit:
			edited, err := openEditor(texts[act.index])
			if err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}
			if strings.TrimSpace(edited) == "" {
				ui.Info("Empty message. Returning to options.")
				continue
			}
			finalMessage = edited
		case actCommit:
			finalMessage = texts[act.index]
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

// printAllModels prints all available models in a formatted table
func printAllModels() {
	allModels := models.GetAllModels()

	fmt.Printf("\n%sAvailable Models:%s\n", "\033[1m", "\033[0m")
	fmt.Printf("%-15s %-25s %-12s %-15s %s\n", "SHORT NAME", "MODEL NAME", "PROVIDER", "COMPANY", "DESCRIPTION")
	fmt.Printf("%s\n", strings.Repeat("-", 100))

	for _, model := range allModels {
		defaultMarker := ""
		if model.IsDefault {
			defaultMarker = " *"
		}

		fmt.Printf("%-15s %-25s %-12s %-15s %s%s\n",
			model.ShortName,
			model.Name,
			model.Provider,
			model.Company,
			model.Description,
			defaultMarker)
	}

	fmt.Printf("\n%sUsage:%s\n", "\033[1m", "\033[0m")
	fmt.Printf("  rune --model <short-name>   # Use short name\n")
	fmt.Printf("  rune --model <full-id>      # Use full model ID\n")
	fmt.Printf("  rune --set-default-model <model>  # Set as default\n")
	fmt.Printf("\n%sExamples:%s\n", "\033[1m", "\033[0m")
	fmt.Printf("  rune --model d         # DeepSeek (1 char!)\n")
	fmt.Printf("  rune --model g         # Gemini 2.0\n")
	fmt.Printf("  rune --model q         # Qwen\n")
	fmt.Printf("  rune --model deep      # DeepSeek (full name)\n")
	fmt.Printf("  rune --set-default-model m    # Set Mistral as default\n")
	fmt.Printf("\n* = Default model for provider\n")
}

// handleSetDefaultModel handles the --set-default-model flag
func handleSetDefaultModel(modelInput string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg == nil {
		ui.Info("No configuration found. Run 'rune --setup' first.")
		return nil
	}

	model, err := models.FindModel(modelInput)
	if err != nil {
		return fmt.Errorf("model not found: %w", err)
	}

	// Check if we need API key for this provider
	if model.Provider != cfg.Provider {
		ui.Info(fmt.Sprintf("Model %s requires %s provider", model.Name, model.Provider))
		if err := cfg.EnsureAPIKeyForProvider(model.Provider); err != nil {
			return fmt.Errorf("failed to setup provider %s: %w", model.Provider, err)
		}
	}

	// Set as default
	if err := cfg.SetDefaultModel(modelInput); err != nil {
		return fmt.Errorf("failed to set default model: %w", err)
	}

	ui.Success(fmt.Sprintf("Default model set to %s (%s)", model.Name, model.Provider))
	return nil
}

// resolveStyle maps the --style flag, detecting from history when "auto".
func resolveStyle(flag string, recent []string) (string, error) {
	switch flag {
	case "conventional":
		return llm.StyleConventional, nil
	case "plain":
		return llm.StylePlain, nil
	case "auto", "":
		if commit.DetectConventional(recent) {
			return llm.StyleConventional, nil
		}
		return llm.StylePlain, nil
	}
	return "", fmt.Errorf("invalid --style %q (use auto, conventional or plain)", flag)
}

func generatingLabel(count int) string {
	if count > 1 {
		return fmt.Sprintf("Generating %d commit messages...", count)
	}
	return "Generating commit message..."
}

// promptAction reads choices until one parses. ok is false when input ends.
func promptAction(in *bufio.Reader, count int) (action, bool) {
	for {
		ui.ShowCandidateOptions(count)
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			return action{}, false
		}
		act, perr := parseAction(line, count)
		if perr != nil {
			ui.Warning(perr.Error())
			continue
		}
		return act, true
	}
}
