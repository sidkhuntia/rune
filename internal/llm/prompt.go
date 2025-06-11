package llm

import (
	"fmt"
	"strings"
)

const commitPromptTemplate = `Generate a concise Git commit message for the following diff. Follow these GitHub conventions:

1. Subject line (first line):
   - Use imperative mood (e.g., "Add", "Fix", "Update", "Remove")
   - Keep it under 50 characters
   - No period at the end
   - Be descriptive but concise

2. If needed, add a blank line followed by a body that:
   - Explains the "what" and "why" (not the "how")
   - Wraps at 72 characters per line
   - Uses present tense

3. Common prefixes to use:
   - feat: new feature
   - fix: bug fix
   - docs: documentation changes
   - style: formatting changes
   - refactor: code refactoring
   - test: adding or updating tests
   - chore: maintenance tasks

Examples of good commit messages:
- "Add user authentication middleware"
- "Fix memory leak in image processing"
- "Update README with installation instructions"
- "Remove deprecated API endpoints"

Git diff:
%s

Generate ONLY the commit message (no quotes, no explanations):
`

// BuildCommitPrompt creates a prompt for generating commit messages from a git diff
func BuildCommitPrompt(diff string) string {
	// Truncate very long diffs to avoid token limits
	const maxDiffLength = 4000
	if len(diff) > maxDiffLength {
		diff = diff[:maxDiffLength] + "\n... (diff truncated)"
	}

	return fmt.Sprintf(commitPromptTemplate, diff)
}

// ParseCommitMessage parses and validates a commit message
func ParseCommitMessage(message string) (subject, body string, err error) {
	lines := strings.Split(strings.TrimSpace(message), "\n")
	if len(lines) == 0 {
		return "", "", fmt.Errorf("empty commit message")
	}

	subject = strings.TrimSpace(lines[0])
	if subject == "" {
		return "", "", fmt.Errorf("empty subject line")
	}

	// Check subject line length
	if len(subject) > 50 {
		return "", "", fmt.Errorf("subject line too long: %d characters (max 50)", len(subject))
	}

	// Check if subject ends with period
	if strings.HasSuffix(subject, ".") {
		return "", "", fmt.Errorf("subject line should not end with a period")
	}

	// Extract body if present
	if len(lines) > 2 {
		// Skip the blank line after subject
		if strings.TrimSpace(lines[1]) != "" {
			return "", "", fmt.Errorf("second line should be blank")
		}
		body = strings.Join(lines[2:], "\n")
	} else if len(lines) == 2 {
		// If there's a second line, it should be blank or part of body
		if strings.TrimSpace(lines[1]) != "" {
			body = lines[1]
		}
	}

	return subject, body, nil
}

// ValidateCommitMessage validates a commit message against GitHub conventions
func ValidateCommitMessage(message string) error {
	_, _, err := ParseCommitMessage(message)
	return err
}
