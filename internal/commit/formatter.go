package commit

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	// MaxSubjectLength is the maximum recommended length for commit subject lines
	MaxSubjectLength = 50
	// MaxBodyLineLength is the maximum recommended length for commit body lines
	MaxBodyLineLength = 72
)

// Message represents a structured commit message
type Message struct {
	Subject string
	Body    string
}

// Format formats a commit message according to GitHub conventions
func (m *Message) Format() string {
	if m.Subject == "" {
		return ""
	}

	result := m.Subject
	if m.Body != "" {
		result += "\n\n" + m.Body
	}
	return result
}

// String returns the formatted commit message
func (m *Message) String() string {
	return m.Format()
}

// FormatCommitMessage formats a raw commit message according to GitHub conventions
func FormatCommitMessage(rawMessage string) (*Message, error) {
	if rawMessage == "" {
		return nil, fmt.Errorf("empty commit message")
	}

	lines := strings.Split(strings.TrimSpace(rawMessage), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty commit message")
	}

	subject := strings.TrimSpace(lines[0])
	if subject == "" {
		return nil, fmt.Errorf("empty subject line")
	}

	// Format the subject line
	subject = formatSubject(subject)

	var body string
	if len(lines) > 1 {
		// Extract body lines (skip empty lines after subject)
		bodyLines := make([]string, 0)
		startBodyIndex := 1

		// Skip empty lines after subject
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) != "" {
				startBodyIndex = i
				break
			}
		}

		if startBodyIndex < len(lines) {
			bodyLines = lines[startBodyIndex:]
		}

		if len(bodyLines) > 0 {
			body = formatBody(strings.Join(bodyLines, "\n"))
		}
	}

	return &Message{
		Subject: subject,
		Body:    body,
	}, nil
}

// formatSubject formats the subject line according to conventions
func formatSubject(subject string) string {
	// Trim whitespace
	subject = strings.TrimSpace(subject)

	// Remove trailing period if present
	if strings.HasSuffix(subject, ".") {
		subject = strings.TrimSuffix(subject, ".")
	}

	// Ensure first letter is capitalized
	if len(subject) > 0 {
		runes := []rune(subject)
		runes[0] = unicode.ToUpper(runes[0])
		subject = string(runes)
	}

	// Truncate if too long
	if len(subject) > MaxSubjectLength {
		subject = subject[:MaxSubjectLength-3] + "..."
	}

	return subject
}

// formatBody formats the body text with proper line wrapping
func formatBody(body string) string {
	if body == "" {
		return ""
	}

	// Split into paragraphs
	paragraphs := strings.Split(strings.TrimSpace(body), "\n\n")
	formattedParagraphs := make([]string, 0, len(paragraphs))

	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph) == "" {
			continue
		}
		formattedParagraphs = append(formattedParagraphs, wrapText(paragraph, MaxBodyLineLength))
	}

	return strings.Join(formattedParagraphs, "\n\n")
}

// wrapText wraps text to the specified line length
func wrapText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return strings.TrimSpace(text)
	}

	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		// If adding this word would exceed the limit, start a new line
		if currentLine.Len() > 0 && currentLine.Len()+1+len(word) > maxLength {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		}

		// Add word to current line
		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)
	}

	// Add the last line if it has content
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}

// ValidateMessage validates a commit message against conventions
func ValidateMessage(msg *Message) error {
	if msg == nil {
		return fmt.Errorf("nil message")
	}

	if msg.Subject == "" {
		return fmt.Errorf("empty subject line")
	}

	if len(msg.Subject) > MaxSubjectLength {
		return fmt.Errorf("subject line too long: %d characters (max %d)", len(msg.Subject), MaxSubjectLength)
	}

	if strings.HasSuffix(msg.Subject, ".") {
		return fmt.Errorf("subject line should not end with a period")
	}

	// Check if subject starts with lowercase (should be capitalized)
	if len(msg.Subject) > 0 && unicode.IsLower([]rune(msg.Subject)[0]) {
		return fmt.Errorf("subject line should start with a capital letter")
	}

	return nil
}

// ParseMessage parses a formatted commit message into a Message struct
func ParseMessage(formatted string) (*Message, error) {
	if formatted == "" {
		return nil, fmt.Errorf("empty commit message")
	}

	lines := strings.Split(formatted, "\n")
	subject := strings.TrimSpace(lines[0])

	if subject == "" {
		return nil, fmt.Errorf("empty subject line")
	}

	msg := &Message{Subject: subject}

	// Parse body if present
	if len(lines) > 1 {
		if len(lines) == 2 {
			// If there's exactly 2 lines, the second line is the body
			msg.Body = strings.TrimSpace(lines[1])
		} else if len(lines) > 2 {
			// If there are more than 2 lines, check if second line is blank
			if strings.TrimSpace(lines[1]) == "" {
				// Proper format with blank line
				msg.Body = strings.Join(lines[2:], "\n")
			} else {
				// No blank line, treat everything after first line as body
				msg.Body = strings.Join(lines[1:], "\n")
			}
		}
	}

	return msg, nil
}
