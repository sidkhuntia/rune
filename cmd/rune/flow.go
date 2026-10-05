package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/siddhartha/rune/internal/commit"
	"github.com/siddhartha/rune/internal/llm"
)

// maxCandidates bounds parallel API calls per round.
const maxCandidates = 5

// candidateTemperatures spreads candidates from conservative to creative.
var candidateTemperatures = []float64{0.2, 0.5, 0.8, 0.6, 0.4}

// generateCandidates asks the model for n distinct messages in parallel.
// Individual failures are tolerated and returned in failures; err is non-nil
// only if none succeed. Messages that pass validation are ordered first.
func generateCandidates(ctx context.Context, client llm.LLMClient, base llm.Request, n int) (msgs []*commit.Message, failures []error, err error) {
	if n < 1 {
		n = 1
	}
	if n > maxCandidates {
		n = maxCandidates
	}

	type result struct {
		msg *commit.Message
		err error
	}
	results := make([]result, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := base
			if n > 1 {
				req.Temperature = candidateTemperatures[i]
			}
			raw, err := client.GenerateCommitMessage(ctx, req)
			if err != nil {
				results[i].err = err
				return
			}
			results[i].msg, results[i].err = commit.FormatCommitMessage(raw)
		}(i)
	}
	wg.Wait()

	var valid, flawed []*commit.Message
	seen := map[string]bool{}
	for _, r := range results {
		if r.err != nil {
			failures = append(failures, r.err)
			continue
		}
		key := strings.ToLower(r.msg.Subject)
		if seen[key] {
			continue
		}
		seen[key] = true
		if commit.ValidateMessage(r.msg) == nil {
			valid = append(valid, r.msg)
		} else {
			flawed = append(flawed, r.msg)
		}
	}

	all := append(valid, flawed...)
	if len(all) == 0 {
		if len(failures) == 0 {
			return nil, nil, fmt.Errorf("model returned no usable message")
		}
		return nil, failures, failures[0]
	}
	return all, failures, nil
}

// actionKind is what the user chose at the prompt.
type actionKind int

const (
	actCommit actionKind = iota
	actEdit
	actRegenerate
	actQuit
)

// action is a parsed prompt response.
type action struct {
	kind  actionKind
	index int    // zero-based candidate for commit/edit
	hint  string // optional steer for regenerate
}

// parseAction interprets input such as "2", "e", "e3", "r", "r shorter", "q".
func parseAction(input string, count int) (action, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return action{}, fmt.Errorf("enter a number to commit, e to edit, r to regenerate, or q to quit")
	}

	if n, err := strconv.Atoi(input); err == nil {
		if n < 1 || n > count {
			return action{}, fmt.Errorf("choose a message between 1 and %d", count)
		}
		return action{kind: actCommit, index: n - 1}, nil
	}

	cmd, rest, _ := strings.Cut(input, " ")
	rest = strings.TrimSpace(rest)
	switch lower := strings.ToLower(cmd); {
	case lower == "q" || lower == "quit":
		return action{kind: actQuit}, nil
	case lower == "r" || lower == "regen":
		return action{kind: actRegenerate, hint: rest}, nil
	case strings.HasPrefix(lower, "e") && (lower == "e" || lower == "edit" || isDigits(lower[1:])):
		idx := 0
		if d := strings.TrimPrefix(lower, "e"); d != "" && isDigits(d) {
			n, _ := strconv.Atoi(d)
			if n < 1 || n > count {
				return action{}, fmt.Errorf("choose a message between 1 and %d", count)
			}
			idx = n - 1
		}
		return action{kind: actEdit, index: idx}, nil
	}
	return action{}, fmt.Errorf("unrecognized choice %q", input)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
