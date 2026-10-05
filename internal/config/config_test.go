package config

import "testing"

func TestFromEnvAndAPIKeyPrecedence(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	if FromEnv() != nil {
		t.Fatal("expected nil config with no env keys")
	}

	t.Setenv("OPENROUTER_API_KEY", "or-key")
	c := FromEnv()
	if c == nil || c.Provider != ProviderOpenRouter || c.Model != DefaultModels[ProviderOpenRouter] || !c.StagedOnly {
		t.Fatalf("unexpected config: %+v", c)
	}
	if key, err := c.GetAPIKey(); err != nil || key != "or-key" || !c.HasAPIKey() {
		t.Fatalf("GetAPIKey = %q, %v", key, err)
	}

	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "g-key")
	if c := FromEnv(); c == nil || c.Provider != ProviderGemini {
		t.Fatalf("expected gemini fallback, got %+v", c)
	}
}
