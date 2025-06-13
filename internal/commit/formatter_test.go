package commit

import (
	"strings"
	"testing"
)

func TestFormatCommitMessage(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSubject string
		wantBody    string
		wantError   string
	}{
		{
			name:        "simple subject only",
			input:       "add user authentication",
			wantSubject: "Add user authentication",
			wantBody:    "",
		},
		{
			name:        "subject with trailing period",
			input:       "fix memory leak.",
			wantSubject: "Fix memory leak",
			wantBody:    "",
		},
		{
			name:        "subject with body",
			input:       "add user authentication\n\nImplement JWT-based authentication system",
			wantSubject: "Add user authentication",
			wantBody:    "Implement JWT-based authentication system",
		},
		{
			name:        "long subject gets truncated",
			input:       "this is a very long commit subject line that exceeds the seventy-two character limit and should be truncated",
			wantSubject: "This is a very long commit subject line that exceeds the seventy-two ...",
			wantBody:    "",
		},
		{
			name:        "subject with extra whitespace",
			input:       "  fix bug  ",
			wantSubject: "Fix bug",
			wantBody:    "",
		},
		{
			name:        "multiline body gets wrapped",
			input:       "add feature\n\nThis is a very long line that should be wrapped at seventy-two characters to follow proper Git commit conventions and make the message readable in various Git tools",
			wantSubject: "Add feature",
			wantBody:    "This is a very long line that should be wrapped at seventy-two\ncharacters to follow proper Git commit conventions and make the message\nreadable in various Git tools",
		},
		{
			name:        "unicode characters",
			input:       "añadir función",
			wantSubject: "Añadir función",
			wantBody:    "",
		},
		{
			name:      "empty input",
			input:     "",
			wantError: "empty commit message",
		},
		{
			name:        "empty subject with body",
			input:       "\n\nSome body text",
			wantSubject: "Some body text",
			wantBody:    "",
		},
		{
			name:        "body with multiple paragraphs",
			input:       "fix critical bug\n\nFirst paragraph explains the issue.\n\nSecond paragraph provides more context about the fix and why it was necessary.",
			wantSubject: "Fix critical bug",
			wantBody:    "First paragraph explains the issue.\n\nSecond paragraph provides more context about the fix and why it was\nnecessary.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatCommitMessage(tt.input)

			if tt.wantError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.wantError)
				} else if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.wantError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}

			if result.Subject != tt.wantSubject {
				t.Errorf("Subject = '%s', want '%s'", result.Subject, tt.wantSubject)
			}

			if result.Body != tt.wantBody {
				t.Errorf("Body = '%s', want '%s'", result.Body, tt.wantBody)
			}
		})
	}
}

func TestMessage_Format(t *testing.T) {
	tests := []struct {
		name     string
		message  *Message
		expected string
	}{
		{
			name:     "subject only",
			message:  &Message{Subject: "Add feature"},
			expected: "Add feature",
		},
		{
			name:     "subject with body",
			message:  &Message{Subject: "Add feature", Body: "Implement new functionality"},
			expected: "Add feature\n\nImplement new functionality",
		},
		{
			name:     "empty subject",
			message:  &Message{Subject: "", Body: "Some body"},
			expected: "",
		},
		{
			name:     "subject with multiline body",
			message:  &Message{Subject: "Fix bug", Body: "First line\nSecond line"},
			expected: "Fix bug\n\nFirst line\nSecond line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.message.Format()
			if result != tt.expected {
				t.Errorf("Format() = '%s', want '%s'", result, tt.expected)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "short text no wrapping",
			input:     "Short text",
			maxLength: 72,
			expected:  "Short text",
		},
		{
			name:      "exact length",
			input:     strings.Repeat("a", 72),
			maxLength: 72,
			expected:  strings.Repeat("a", 72),
		},
		{
			name:      "wrap long line",
			input:     "This is a very long line that should be wrapped at the specified length to ensure proper formatting",
			maxLength: 20,
			expected:  "This is a very long\nline that should be\nwrapped at the\nspecified length to\nensure proper\nformatting",
		},
		{
			name:      "preserve single word longer than limit",
			input:     "Supercalifragilisticexpialidocious short",
			maxLength: 10,
			expected:  "Supercalifragilisticexpialidocious\nshort",
		},
		{
			name:      "empty input",
			input:     "",
			maxLength: 72,
			expected:  "",
		},
		{
			name:      "whitespace only",
			input:     "   \n  \t  ",
			maxLength: 72,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.input, tt.maxLength)
			if result != tt.expected {
				t.Errorf("wrapText() = '%s', want '%s'", result, tt.expected)
			}
		})
	}
}

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name      string
		message   *Message
		wantError string
	}{
		{
			name:    "valid message",
			message: &Message{Subject: "Add feature", Body: "Some body"},
		},
		{
			name:      "nil message",
			message:   nil,
			wantError: "nil message",
		},
		{
			name:      "empty subject",
			message:   &Message{Subject: "", Body: "Some body"},
			wantError: "empty subject line",
		},
		{
			name:      "subject too long",
			message:   &Message{Subject: strings.Repeat("a", 73)},
			wantError: "subject line too long",
		},
		{
			name:      "subject with period",
			message:   &Message{Subject: "Add feature."},
			wantError: "subject line should not end with a period",
		},
		{
			name:      "subject starts with lowercase",
			message:   &Message{Subject: "add feature"},
			wantError: "subject line should start with a capital letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessage(tt.message)

			if tt.wantError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.wantError)
				} else if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.wantError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSubject string
		wantBody    string
		wantError   string
	}{
		{
			name:        "subject only",
			input:       "Add feature",
			wantSubject: "Add feature",
			wantBody:    "",
		},
		{
			name:        "subject with body",
			input:       "Add feature\n\nImplement new functionality",
			wantSubject: "Add feature",
			wantBody:    "Implement new functionality",
		},
		{
			name:        "subject with multiline body",
			input:       "Fix bug\n\nFirst line\nSecond line",
			wantSubject: "Fix bug",
			wantBody:    "First line\nSecond line",
		},
		{
			name:      "empty input",
			input:     "",
			wantError: "empty commit message",
		},
		{
			name:      "empty subject",
			input:     "\n\nBody text",
			wantError: "empty subject line",
		},
		{
			name:        "no blank line after subject",
			input:       "Subject\nImmediate body",
			wantSubject: "Subject",
			wantBody:    "Immediate body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseMessage(tt.input)

			if tt.wantError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.wantError)
				} else if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.wantError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}

			if result.Subject != tt.wantSubject {
				t.Errorf("Subject = '%s', want '%s'", result.Subject, tt.wantSubject)
			}

			if result.Body != tt.wantBody {
				t.Errorf("Body = '%s', want '%s'", result.Body, tt.wantBody)
			}
		})
	}
}

func TestFormatSubject(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic subject",
			input:    "add feature",
			expected: "Add feature",
		},
		{
			name:     "remove trailing period",
			input:    "Fix bug.",
			expected: "Fix bug",
		},
		{
			name:     "already capitalized",
			input:    "Update README",
			expected: "Update README",
		},
		{
			name:     "truncate long subject",
			input:    "This is a very long commit subject that exceeds seventy-two characters",
			expected: "This is a very long commit subject that exceeds seventy-two characters",
		},
		{
			name:     "whitespace handling",
			input:    "  fix bug  ",
			expected: "Fix bug",
		},
		{
			name:     "unicode",
			input:    "añadir función",
			expected: "Añadir función",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSubject(&tt.input)
			if result != tt.expected {
				t.Errorf("formatSubject(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
