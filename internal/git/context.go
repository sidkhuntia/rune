package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Context is repository metadata that helps a model write a commit message
// that fits the project, beyond what the raw diff shows.
type Context struct {
	Branch string   // current branch name, empty when detached or unborn
	Recent []string // most recent commit subjects, newest first
	Stat   string   // `git diff --stat` summary of the changes
}

const recentCommitCount = 8

// noisePathspecs excludes files whose diffs carry no signal about intent.
var noisePathspecs = []string{
	":(exclude,glob)**/go.sum",
	":(exclude,glob)**/package-lock.json",
	":(exclude,glob)**/pnpm-lock.yaml",
	":(exclude,glob)**/yarn.lock",
	":(exclude,glob)**/*.lock",
	":(exclude,glob)**/*.min.js",
	":(exclude,glob)**/*.min.css",
	":(exclude,glob)**/*.map",
}

func diffBase(staged bool) []string {
	if staged {
		return []string{"diff", "--cached"}
	}
	return []string{"diff", "HEAD"}
}

// gitOutput runs git and returns trimmed stdout, or "" on any failure.
// Context is best-effort: a brand-new repo has no branch or log yet.
func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GatherContext collects branch, recent history and a change summary.
func GatherContext(staged bool) Context {
	ctx := Context{Stat: gitOutput(append(diffBase(staged), "--stat")...)}

	if branch := gitOutput("rev-parse", "--abbrev-ref", "HEAD"); branch != "HEAD" {
		ctx.Branch = branch
	}
	if log := gitOutput("log", fmt.Sprintf("-n%d", recentCommitCount), "--format=%s"); log != "" {
		ctx.Recent = strings.Split(log, "\n")
	}
	return ctx
}

// SmartDiff returns the diff with lockfiles and generated files removed and
// each file bounded so one huge file cannot crowd out the rest. If nothing but
// noise changed, it falls back to the full diff.
func SmartDiff(staged bool, budget int) (string, error) {
	args := append(diffBase(staged), "--", ".")
	args = append(args, noisePathspecs...)

	out, err := exec.Command("git", args...).Output()
	diff := strings.TrimSpace(string(out))
	if err != nil || diff == "" {
		full, fullErr := ExtractDiff(staged)
		if fullErr != nil {
			return "", fullErr
		}
		diff = full
	}
	return TruncateDiff(diff, budget), nil
}

// TruncateDiff shrinks a diff to roughly budget characters by trimming the
// largest files first, keeping every file's header so none disappears.
func TruncateDiff(diff string, budget int) string {
	if len(diff) <= budget || budget <= 0 {
		return diff
	}

	files := splitFiles(diff)
	per := budget / len(files)
	if per < 1500 {
		per = 1500
	}

	var b strings.Builder
	for _, f := range files {
		if len(f) <= per {
			b.WriteString(f)
			continue
		}
		cut := strings.LastIndex(f[:per], "\n")
		if cut <= 0 {
			cut = per
		}
		omitted := strings.Count(f[cut:], "\n")
		b.WriteString(f[:cut])
		fmt.Fprintf(&b, "\n... (%d more lines in this file omitted)\n", omitted)
	}
	return strings.TrimSpace(b.String())
}

// splitFiles splits a unified diff into one chunk per file.
func splitFiles(diff string) []string {
	const marker = "diff --git "
	var files []string
	for _, part := range strings.Split(diff, "\n"+marker) {
		if !strings.HasPrefix(part, marker) {
			part = marker + part
		}
		files = append(files, strings.TrimRight(part, "\n")+"\n")
	}
	return files
}
