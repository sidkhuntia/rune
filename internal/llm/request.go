package llm

import (
	"fmt"
	"strings"
)

// Commit message styles.
const (
	StyleConventional = "conventional"
	StylePlain        = "plain"
)

// Request carries everything a model needs to write one commit message.
type Request struct {
	Diff        string
	Stat        string   // `git diff --stat` summary
	Branch      string   // current branch
	Recent      []string // recent subjects, so the model can match house style
	Style       string   // StyleConventional or StylePlain
	Hint        string   // free-form steer from the user ("shorter", "mention auth")
	Avoid       []string // subjects already shown, so regeneration gives something new
	Temperature float64  // 0 means use the client default
}

const maxPromptDiff = 100_000

const systemPromptBase = `You write Git commit messages for a developer who is about to commit.

Rules:
- Output ONLY the commit message. No quotes, no code fences, no preamble.
- Subject line: imperative mood, at most 50 characters (hard limit 72), no trailing period.
- Describe the intent and effect of the change, not a file-by-file list.
- Add a body only when the change is non-trivial: blank line after the subject, wrapped at 72 columns, explaining what changed and why.
- Only state facts visible in the diff. Never invent motivation, ticket numbers or behavior.
- Ignore pure noise such as whitespace, formatting and generated files unless that is the whole change.
`

const conventionalRules = `- Use Conventional Commits: "type(scope): subject" where type is one of feat, fix, docs, style, refactor, perf, test, build, ci, chore.
- Keep the type and scope lowercase; start the description with a lowercase verb. Omit the scope if no single area fits.
- Append "!" after the type/scope for breaking changes and explain in the body.
`

const plainRules = `- Do not use a "type:" prefix. Start with a capitalized imperative verb (Add, Fix, Update, Remove, Refactor).
`

// SystemPrompt returns the role and rules for the request's style.
func SystemPrompt(style string) string {
	if style == StylePlain {
		return systemPromptBase + plainRules
	}
	return systemPromptBase + conventionalRules
}

// UserPrompt renders the context and diff for the request.
func UserPrompt(r Request) string {
	var b strings.Builder

	if r.Branch != "" {
		fmt.Fprintf(&b, "Branch: %s\n\n", r.Branch)
	}
	if len(r.Recent) > 0 {
		b.WriteString("Recent commits in this repository (match their tone and level of detail):\n")
		for _, s := range r.Recent {
			fmt.Fprintf(&b, "- %s\n", s)
		}
		b.WriteString("\n")
	}
	if r.Stat != "" {
		fmt.Fprintf(&b, "Files changed:\n%s\n\n", r.Stat)
	}
	if r.Hint != "" {
		fmt.Fprintf(&b, "The author asks: %s\n\n", r.Hint)
	}
	if len(r.Avoid) > 0 {
		b.WriteString("Already proposed (write something meaningfully different):\n")
		for _, s := range r.Avoid {
			fmt.Fprintf(&b, "- %s\n", s)
		}
		b.WriteString("\n")
	}

	diff := r.Diff
	if len(diff) > maxPromptDiff {
		diff = diff[:maxPromptDiff] + "\n... (diff truncated)"
	}
	fmt.Fprintf(&b, "Diff:\n%s\n\nWrite the commit message now.", diff)
	return b.String()
}

// temperatureOr returns the request temperature or a client default.
func temperatureOr(r Request, def float64) float64 {
	if r.Temperature > 0 {
		return r.Temperature
	}
	return def
}
