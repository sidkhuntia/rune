package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *OpenRouterClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	c, err := NewOpenRouterClient("")
	if err != nil {
		t.Fatal(err)
	}
	c.baseURL = srv.URL
	retryBackoff = time.Millisecond
	return c
}

func TestOpenRouterGenerate(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing auth header")
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<think>hmm</think>\nfeat: add thing"},"finish_reason":"stop"}]}`))
	})
	got, err := c.GenerateCommitMessage(context.Background(), Request{Diff: "diff"})
	if err != nil || got != "feat: add thing" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestOpenRouterRetriesTransientErrors(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"fix: ok"},"finish_reason":"stop"}]}`))
	})
	got, err := c.GenerateCommitMessage(context.Background(), Request{Diff: "diff"})
	if err != nil || got != "fix: ok" || calls.Load() != 3 {
		t.Fatalf("got %q, %v after %d calls", got, err, calls.Load())
	}
}

func TestOpenRouterDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid key"}}`))
	})
	_, err := c.GenerateCommitMessage(context.Background(), Request{Diff: "diff"})
	if err == nil || !strings.Contains(err.Error(), "invalid key") || calls.Load() != 1 {
		t.Fatalf("err = %v, calls = %d", err, calls.Load())
	}
}

func TestOpenRouterTokenLimitIsExplained(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""},"finish_reason":"length"}]}`))
	})
	_, err := c.GenerateCommitMessage(context.Background(), Request{Diff: "diff"})
	if err == nil || !strings.Contains(err.Error(), "token limit") {
		t.Fatalf("err = %v", err)
	}
}

func TestOpenRouterHonorsContextCancel(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.GenerateCommitMessage(ctx, Request{Diff: "diff"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewOpenRouterClient(t *testing.T) {
	// Test with no API key
	client, err := NewOpenRouterClient("")
	if err == nil {
		t.Error("Expected error when OPENROUTER_API_KEY is not set")
	}
	if client != nil {
		t.Error("Expected nil client when API key is not set")
	}

	// Test with default model
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	client, err = NewOpenRouterClient("")
	if err != nil {
		t.Errorf("Expected no error with API key set, got: %v", err)
	}
	if client == nil {
		t.Error("Expected non-nil client")
		return
	}
	if client.model != "nvidia/nemotron-3-ultra-550b-a55b:free" {
		t.Errorf("Expected default model 'nvidia/nemotron-3-ultra-550b-a55b:free', got: %s", client.model)
	}

	// Test with custom model
	client, err = NewOpenRouterClient("qwen/qwen-3-32b")
	if err != nil {
		t.Errorf("Expected no error with custom model, got: %v", err)
	}
	if client.model != "qwen/qwen-3-32b" {
		t.Errorf("Expected custom model 'qwen/qwen-3-32b', got: %s", client.model)
	}
}
