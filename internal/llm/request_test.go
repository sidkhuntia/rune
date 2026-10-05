package llm

import (
	"strings"
	"testing"
)

func TestUserPromptIncludesContext(t *testing.T) {
	p := UserPrompt(Request{
		Diff:   "diff --git a/x b/x",
		Stat:   " x | 2 +-",
		Branch: "feature/retry",
		Recent: []string{"feat: add client"},
		Hint:   "mention backoff",
		Avoid:  []string{"feat: add retry"},
	})
	for _, want := range []string{"feature/retry", "feat: add client", " x | 2 +-", "mention backoff", "feat: add retry", "diff --git a/x b/x"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestUserPromptOmitsEmptySections(t *testing.T) {
	p := UserPrompt(Request{Diff: "d"})
	for _, banned := range []string{"Branch:", "Recent commits", "Files changed", "author asks", "Already proposed"} {
		if strings.Contains(p, banned) {
			t.Errorf("prompt unexpectedly contains %q", banned)
		}
	}
}

func TestSystemPromptStyle(t *testing.T) {
	if !strings.Contains(SystemPrompt(StyleConventional), "Conventional Commits") {
		t.Error("conventional prompt missing rule")
	}
	if strings.Contains(SystemPrompt(StylePlain), "Conventional Commits") || !strings.Contains(SystemPrompt(StylePlain), "capitalized") {
		t.Error("plain prompt has wrong rules")
	}
}
