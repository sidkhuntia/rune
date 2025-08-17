package git

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// gitMutex ensures git operations are atomic
var gitMutex sync.Mutex

// WithGitLock executes a function with git operation lock
func WithGitLock(fn func() error) error {
	gitMutex.Lock()
	defer gitMutex.Unlock()
	return fn()
}

// ExtractDiff extracts the diff from git.
// If staged is true, it returns the staged changes (--cached).
// If staged is false, it returns all changes including unstaged.
func ExtractDiff(staged bool) (string, error) {
	var cmd *exec.Cmd

	if staged {
		// Get only staged changes
		cmd = exec.Command("git", "diff", "--cached")
	} else {
		// Get all changes (staged + unstaged) relative to HEAD
		cmd = exec.Command("git", "diff", "HEAD")
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute git diff: %w", err)
	}

	diff := strings.TrimSpace(string(output))
	if diff == "" {
		return "", fmt.Errorf("no changes found %s\n %s", cmd.String(), string(output))
	}

	return diff, nil
}

// ListStagedFiles returns a slice of file paths that are currently staged for commit.
func ListStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list staged files: %w", err)
	}
	files := strings.Fields(string(output))
	return files, nil
}

// UnstageFiles unstages the given files from the index (staging area).
func UnstageFiles(files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"reset", "HEAD", "--"}, files...)
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unstage files: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// AtomicStageResult represents the result of an atomic staging operation
type AtomicStageResult struct {
	PreviouslyStaged []string
	NewlyStaged      []string
	TotalStaged      []string
}

// AtomicStageAll performs atomic staging of all changes with proper locking
func AtomicStageAll() (*AtomicStageResult, error) {
	var result AtomicStageResult

	err := WithGitLock(func() error {
		// Get current staged files
		previousStaged, err := ListStagedFiles()
		if err != nil {
			return fmt.Errorf("failed to list previously staged files: %w", err)
		}
		result.PreviouslyStaged = previousStaged

		// Stage all changes
		cmd := exec.Command("git", "add", ".")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to stage changes: %w\nOutput: %s", err, string(output))
		}

		// Get newly staged files
		totalStaged, err := ListStagedFiles()
		if err != nil {
			return fmt.Errorf("failed to list total staged files: %w", err)
		}
		result.TotalStaged = totalStaged

		// Calculate newly staged files
		for _, file := range totalStaged {
			found := false
			for _, prev := range previousStaged {
				if file == prev {
					found = true
					break
				}
			}
			if !found {
				result.NewlyStaged = append(result.NewlyStaged, file)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}
