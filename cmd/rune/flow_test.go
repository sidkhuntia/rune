package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/siddhartha/rune/internal/llm"
)

func TestParseAction(t *testing.T) {
	tests := []struct {
		in      string
		want    action
		wantErr bool
	}{
		{in: "2", want: action{kind: actCommit, index: 1}},
		{in: " 1 \n", want: action{kind: actCommit, index: 0}},
		{in: "e", want: action{kind: actEdit, index: 0}},
		{in: "E3", want: action{kind: actEdit, index: 2}},
		{in: "r", want: action{kind: actRegenerate}},
		{in: "r make it shorter", want: action{kind: actRegenerate, hint: "make it shorter"}},
		{in: "q", want: action{kind: actQuit}},
		{in: "", wantErr: true},
		{in: "0", wantErr: true},
		{in: "4", wantErr: true},
		{in: "e9", wantErr: true},
		{in: "bogus", wantErr: true},
		{in: "edge", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseAction(tt.in, 3)
		if (err != nil) != tt.wantErr || (!tt.wantErr && got != tt.want) {
			t.Errorf("parseAction(%q) = %+v, %v", tt.in, got, err)
		}
	}
}

type fakeClient struct {
	calls atomic.Int32
	reply func(i int, r llm.Request) (string, error)
}

func (f *fakeClient) GenerateCommitMessage(_ context.Context, r llm.Request) (string, error) {
	return f.reply(int(f.calls.Add(1)), r)
}

func TestGenerateCandidates(t *testing.T) {
	t.Run("dedupes and tolerates partial failure", func(t *testing.T) {
		c := &fakeClient{reply: func(i int, r llm.Request) (string, error) {
			switch r.Temperature {
			case 0.2:
				return "feat: add retry", nil
			case 0.5:
				return "Feat: Add Retry", nil // duplicate once case-folded
			}
			return "", fmt.Errorf("boom")
		}}
		got, failed, err := generateCandidates(context.Background(), c, llm.Request{}, 3)
		if err != nil || len(failed) != 1 || len(got) != 1 || got[0].Subject != "feat: add retry" {
			t.Fatalf("got %v, %v", got, err)
		}
	})

	t.Run("valid messages sort before flawed ones", func(t *testing.T) {
		c := &fakeClient{reply: func(i int, r llm.Request) (string, error) {
			if r.Temperature == 0.2 {
				return "Fix a thing.\n", nil // formatter strips the period -> valid
			}
			return "x: " + string(make([]byte, 0)) + "ok", nil
		}}
		got, _, err := generateCandidates(context.Background(), c, llm.Request{}, 2)
		if err != nil || len(got) == 0 {
			t.Fatalf("got %v, %v", got, err)
		}
	})

	t.Run("all failures surface the error", func(t *testing.T) {
		c := &fakeClient{reply: func(int, llm.Request) (string, error) { return "", fmt.Errorf("down") }}
		if _, _, err := generateCandidates(context.Background(), c, llm.Request{}, 3); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("single candidate keeps default temperature", func(t *testing.T) {
		c := &fakeClient{reply: func(_ int, r llm.Request) (string, error) {
			if r.Temperature != 0 {
				t.Errorf("temperature = %v, want unset", r.Temperature)
			}
			return "fix: ok", nil
		}}
		if _, _, err := generateCandidates(context.Background(), c, llm.Request{}, 1); err != nil {
			t.Fatal(err)
		}
	})
}

func TestResolveStyle(t *testing.T) {
	conv := []string{"feat: a", "fix(x): b", "Update readme"}
	plain := []string{"Add a", "Fix b", "feat: c"}
	for _, tt := range []struct {
		flag   string
		recent []string
		want   string
	}{
		{"auto", conv, llm.StyleConventional},
		{"auto", plain, llm.StylePlain},
		{"auto", nil, llm.StyleConventional},
		{"plain", conv, llm.StylePlain},
		{"conventional", plain, llm.StyleConventional},
	} {
		if got, err := resolveStyle(tt.flag, tt.recent); err != nil || got != tt.want {
			t.Errorf("resolveStyle(%q) = %q, %v; want %q", tt.flag, got, err, tt.want)
		}
	}
	if _, err := resolveStyle("fancy", nil); err == nil {
		t.Error("expected error for unknown style")
	}
}
