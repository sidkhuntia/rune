package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Colors for terminal output
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
	ColorBlue   = "\033[34m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

// PreviewCommitMessage displays a nicely formatted commit message preview
func PreviewCommitMessage(message string) {
	lines := strings.Split(message, "\n")
	if len(lines) == 0 {
		return
	}

	// Header
	fmt.Printf("\n%s┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓%s\n", ColorBold, ColorReset)
	fmt.Printf("%s┃%s                    %sGenerated Commit Message%s                  %s┃%s\n", ColorBold, ColorReset, ColorCyan+ColorBold, ColorReset, ColorBold, ColorReset)
	fmt.Printf("%s┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛%s\n", ColorBold, ColorReset)

	// Subject line (first line)
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		subject := strings.TrimSpace(lines[0])
		fmt.Printf("\n%s%s%s%s\n", ColorBold, ColorGreen, subject, ColorReset)

		// Show length indicator for subject
		subjectLen := utf8.RuneCountInString(subject)
		if subjectLen > 50 {
			fmt.Printf("%s(⚠️  %d chars - consider shortening to 50 chars or less)%s\n", ColorYellow, subjectLen, ColorReset)
		} else {
			fmt.Printf("%s(%d chars)%s\n", ColorDim, subjectLen, ColorReset)
		}
	}

	// Body (remaining lines)
	if len(lines) > 1 {
		bodyLines := lines[1:]
		hasBody := false

		for _, line := range bodyLines {
			if strings.TrimSpace(line) != "" {
				hasBody = true
				break
			}
		}

		if hasBody {
			fmt.Printf("\n%sDescription:%s\n", ColorBold, ColorReset)
			for _, line := range bodyLines {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("%s%s%s\n", ColorBlue, line, ColorReset)
				} else {
					fmt.Println()
				}
			}
		}
	}

	// Footer separator
	fmt.Printf("\n%s%s%s\n", ColorDim, strings.Repeat("─", 60), ColorReset)
}

// ShowCommitOptions displays the interactive menu with better formatting
func ShowCommitOptions() {
	fmt.Println("\n" + ColorBold + "What would you like to do?" + ColorReset)
	fmt.Printf("  %s1.%s 🔄 Re-generate commit message\n", ColorBold, ColorReset)
	fmt.Printf("  %s2.%s ✅ Commit as-is\n", ColorBold, ColorReset)
	fmt.Printf("  %s3.%s 📝 Edit and commit\n", ColorBold, ColorReset)
	fmt.Printf("  %s4.%s 🚫 Quit (cleanup staged files)\n", ColorBold, ColorReset)
	fmt.Printf("\n%sEnter your choice (1-4): %s", ColorBold, ColorReset)
}

// ShowSetupWelcome displays a welcome message for setup
func ShowSetupWelcome() {
	fmt.Printf("\n%s%s🚀 Welcome to Rune!%s%s\n", ColorBold, ColorCyan, ColorReset, ColorReset)
	fmt.Printf("%sLet's set up your AI provider for generating commit messages.%s\n\n", ColorDim, ColorReset)
}
