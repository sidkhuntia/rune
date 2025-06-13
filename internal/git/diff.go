package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExtractDiff extracts the diff from git.
// If staged is true, it returns the staged changes (--cached).
// If staged is false, it returns all changes including unstaged.
func ExtractDiff(staged bool) (string, error) {
	var cmd *exec.Cmd

	if staged {
		// Get only staged changes
		cmd = exec.Command("git", "diff", "--cached")
	} else {
		// Get all changes (staged + unstaged + untracked)
		// Use git status --porcelain to detect untracked files and git diff for tracked files
		cmd = exec.Command("git", "diff")
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
