package ui

import (
	"fmt"
	"sync"
	"time"
)

// Spinner provides a simple text-based loading indicator
type Spinner struct {
	message string
	active  bool
	mu      sync.Mutex
	done    chan bool
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go s.spin()
}

// Stop ends the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	s.done <- true
	// Clear the spinner line
	fmt.Print("\r" + clearLine() + "\r")
}

// UpdateMessage changes the spinner message while it's running
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// spin handles the actual animation
func (s *Spinner) spin() {
	chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.active {
				fmt.Printf("\r%s %s", chars[i%len(chars)], s.message)
			}
			s.mu.Unlock()
			i++
		}
	}
}

// clearLine returns a string that clears the current line
func clearLine() string {
	return "\033[2K"
}

// Success prints a success message with a checkmark
func Success(message string) {
	fmt.Printf("✅ %s\n", message)
}

// Warning prints a warning message with a warning icon
func Warning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

// Error prints an error message with an error icon
func Error(message string) {
	fmt.Printf("❌ %s\n", message)
}

// Info prints an info message with an info icon
func Info(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}