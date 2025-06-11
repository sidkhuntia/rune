package git

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractDiff(t *testing.T) {
	// Create a temporary git repository for testing
	tempDir, err := ioutil.TempDir("", "commitgen-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Initialize git repo
	exec.Command("git", "init").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Create initial commit
	ioutil.WriteFile("main.go", []byte(`package main

func main() {
    // TODO: implement
}
`), 0644)
	exec.Command("git", "add", "main.go").Run()
	exec.Command("git", "commit", "-m", "Initial commit").Run()

	t.Run("staged changes", func(t *testing.T) {
		// Modify file and stage changes
		ioutil.WriteFile("main.go", []byte(`package main

import "fmt"

func main() {
    fmt.Println("Hello, CommitGen!")
    // Added implementation
}
`), 0644)
		exec.Command("git", "add", "main.go").Run()

		// Test ExtractDiff with staged=true
		diff, err := ExtractDiff(true)
		if err != nil {
			t.Errorf("ExtractDiff(true) returned error: %v", err)
		}

		if !strings.Contains(diff, "import \"fmt\"") {
			t.Errorf("Expected diff to contain import statement")
		}
		if !strings.Contains(diff, "fmt.Println") {
			t.Errorf("Expected diff to contain fmt.Println")
		}
	})

	t.Run("no staged changes", func(t *testing.T) {
		// Reset to clean state
		exec.Command("git", "reset", "--hard", "HEAD").Run()

		// Test ExtractDiff with no staged changes
		_, err := ExtractDiff(true)
		if err == nil {
			t.Errorf("Expected error when no staged changes, got nil")
		}
		if !strings.Contains(err.Error(), "no changes found") {
			t.Errorf("Expected 'no changes found' error, got: %v", err)
		}
	})

	t.Run("all changes including unstaged", func(t *testing.T) {
		// Create and add a file, then modify it to create unstaged changes
		ioutil.WriteFile("README.md", []byte("# CommitGen\n"), 0644)
		exec.Command("git", "add", "README.md").Run()
		exec.Command("git", "commit", "-m", "Add README").Run()

		// Now modify the file to create unstaged changes
		ioutil.WriteFile("README.md", []byte("# CommitGen\n\nA CLI tool for generating commit messages using AI.\n"), 0644)

		// Test ExtractDiff with staged=false
		diff, err := ExtractDiff(false)
		if err != nil {
			t.Errorf("ExtractDiff(false) returned error: %v", err)
		}

		if !strings.Contains(diff, "README.md") {
			t.Errorf("Expected diff to contain README.md")
		}
		if !strings.Contains(diff, "+A CLI tool for generating commit messages using AI.") {
			t.Errorf("Expected diff to contain the added line")
		}
	})
}

func TestExtractDiffWithSampleData(t *testing.T) {
	// This test verifies that our sample diff file is valid
	sampleDiffPath := filepath.Join("..", "..", "testdata", "sample.diff")
	content, err := ioutil.ReadFile(sampleDiffPath)
	if err != nil {
		t.Errorf("Failed to read sample diff file: %v", err)
		return
	}

	sampleDiff := string(content)

	// Verify the sample diff contains expected elements
	if !strings.Contains(sampleDiff, "diff --git") {
		t.Errorf("Sample diff should contain 'diff --git'")
	}
	if !strings.Contains(sampleDiff, "main.go") {
		t.Errorf("Sample diff should contain 'main.go'")
	}
	if !strings.Contains(sampleDiff, "README.md") {
		t.Errorf("Sample diff should contain 'README.md'")
	}
	if !strings.Contains(sampleDiff, "+import \"fmt\"") {
		t.Errorf("Sample diff should contain '+import \"fmt\"'")
	}
}
