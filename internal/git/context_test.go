package git

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestTruncateDiffKeepsEveryFile(t *testing.T) {
	big := "diff --git a/big.go b/big.go\n" + strings.Repeat("+line\n", 5000)
	small := "diff --git a/small.go b/small.go\n+tiny\n"
	got := TruncateDiff(big+small, 4000)

	if len(got) > 6000 {
		t.Errorf("result too large: %d", len(got))
	}
	if !strings.Contains(got, "diff --git a/small.go") || !strings.Contains(got, "+tiny") {
		t.Error("small file was lost")
	}
	if !strings.Contains(got, "more lines in this file omitted") {
		t.Error("expected truncation marker")
	}
}

func TestTruncateDiffNoopUnderBudget(t *testing.T) {
	d := "diff --git a/a b/a\n+x\n"
	if TruncateDiff(d, 1000) != d {
		t.Error("diff under budget must be unchanged")
	}
}

func run(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestSmartDiffAndContext(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	run(t, "init", "-q", "-b", "trunk")
	run(t, "config", "user.email", "t@example.com")
	run(t, "config", "user.name", "T")
	run(t, "config", "commit.gpgsign", "false")
	_ = os.WriteFile("a.go", []byte("package a\n"), 0o644)
	_ = os.WriteFile("go.sum", []byte("x\n"), 0o644)
	run(t, "add", ".")
	run(t, "commit", "-qm", "feat: initial")

	_ = os.WriteFile("a.go", []byte("package a\n\nvar X = 1\n"), 0o644)
	_ = os.WriteFile("go.sum", []byte("x\ny\n"), 0o644)
	run(t, "add", ".")

	diff, err := SmartDiff(true, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "a.go") || strings.Contains(diff, "go.sum") {
		t.Errorf("noise not filtered:\n%s", diff)
	}

	c := GatherContext(true)
	if c.Branch != "trunk" || len(c.Recent) != 1 || c.Recent[0] != "feat: initial" || !strings.Contains(c.Stat, "a.go") {
		t.Errorf("unexpected context: %+v", c)
	}

	// Only noise changed: must fall back to the full diff rather than fail.
	run(t, "commit", "-qm", "chore: more")
	_ = os.WriteFile("go.sum", []byte("x\ny\nz\n"), 0o644)
	run(t, "add", ".")
	diff, err = SmartDiff(true, 10_000)
	if err != nil || !strings.Contains(diff, "go.sum") {
		t.Errorf("fallback failed: %v\n%s", err, diff)
	}
}
