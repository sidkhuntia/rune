package models

import "testing"

func TestFindModelResolution(t *testing.T) {
	const nemotron = "nvidia/nemotron-3-ultra-550b-a55b:free"
	for _, q := range []string{nemotron, "nm", "nemotron", "openrouter", "NM"} {
		m, err := FindModel(q)
		if err != nil || m.ID != nemotron {
			t.Errorf("FindModel(%q) = %v, %v", q, m, err)
		}
	}
	// Stored config holds the upstream ID, which can differ from the registry key.
	if m, err := FindModel("deepseek/deepseek-chat-v3:free"); err != nil || m.ShortName != "dv3" {
		t.Errorf("lookup by upstream ID failed: %v, %v", m, err)
	}
	if _, err := FindModel("nope"); err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestEveryAliasResolvesAndOneDefaultPerProvider(t *testing.T) {
	for alias := range ModelAliases {
		if _, err := FindModel(alias); err != nil {
			t.Errorf("alias %q does not resolve: %v", alias, err)
		}
	}
	for _, p := range []string{"gemini", "openrouter"} {
		n := 0
		for _, m := range GetModelsByProvider(p) {
			if m.IsDefault {
				n++
			}
		}
		if n != 1 {
			t.Errorf("provider %s has %d defaults, want 1", p, n)
		}
	}
}
