package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractDiff(t *testing.T) {
	// Create a temporary git repository for testing
	tempDir, err := os.MkdirTemp("", "commitgen-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore working dir: %v", err)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("Failed to set git user.email: %v", err)
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Fatalf("Failed to set git user.name: %v", err)
	}

	// Create initial commit
	if err := os.WriteFile("main.go", []byte(`package main

func main() {
    // TODO: implement
}
`), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}
	if err := exec.Command("git", "add", "main.go").Run(); err != nil {
		t.Fatalf("Failed to add main.go: %v", err)
	}
	if err := exec.Command("git", "commit", "-m", "Initial commit").Run(); err != nil {
		t.Fatalf("Failed to commit main.go: %v", err)
	}

	t.Run("staged changes", func(t *testing.T) {
		// Modify file and stage changes
		if err := os.WriteFile("main.go", []byte(`package main

import "fmt"

func main() {
    fmt.Println("Hello, CommitGen!")
    // Added implementation
}
`), 0644); err != nil {
			t.Fatalf("Failed to write main.go: %v", err)
		}
		if err := exec.Command("git", "add", "main.go").Run(); err != nil {
			t.Fatalf("Failed to add main.go: %v", err)
		}

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
		if err := exec.Command("git", "reset", "--hard", "HEAD").Run(); err != nil {
			t.Fatalf("Failed to reset git repo: %v", err)
		}

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
		if err := os.WriteFile("README.md", []byte("# CommitGen\n"), 0644); err != nil {
			t.Fatalf("Failed to write README.md: %v", err)
		}
		if err := exec.Command("git", "add", "README.md").Run(); err != nil {
			t.Fatalf("Failed to add README.md: %v", err)
		}
		if err := exec.Command("git", "commit", "-m", "Add README").Run(); err != nil {
			t.Fatalf("Failed to commit README.md: %v", err)
		}

		// Now modify the file to create unstaged changes
		if err := os.WriteFile("README.md", []byte("# CommitGen\n\nA CLI tool for generating commit messages using AI.\n"), 0644); err != nil {
			t.Fatalf("Failed to modify README.md: %v", err)
		}

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
	content, err := os.ReadFile(sampleDiffPath)
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
