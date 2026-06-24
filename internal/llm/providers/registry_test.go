package providers

import (
	"testing"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/config"
)

func TestResolveDefaultsToOllama(t *testing.T) {
	provider, err := Resolve(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if provider.ID != "ollama" || provider.BaseURL == "" || provider.DefaultModel == "" {
		t.Fatalf("unexpected default provider %+v", provider)
	}
}

func TestResolveOpenRouter(t *testing.T) {
	provider, err := Resolve(config.Config{Provider: "openrouter"})
	if err != nil {
		t.Fatal(err)
	}
	if provider.ID != "openrouter" || provider.AuthKind != auth.AuthKindAPIKey {
		t.Fatalf("unexpected openrouter provider %+v", provider)
	}
}

func TestResolveCodexAliases(t *testing.T) {
	for _, id := range []string{"openai", "openai-codex", "chatgpt", "codex"} {
		provider, err := Resolve(config.Config{Provider: id})
		if err != nil {
			t.Fatal(err)
		}
		if provider.ID != "codex" || provider.AuthKind != auth.AuthKindOAuth2 {
			t.Fatalf("alias %q resolved to %+v", id, provider)
		}
	}
}

func TestResolveOverrides(t *testing.T) {
	provider, err := Resolve(config.Config{
		Provider: "opencode-go",
		Model:    "custom-model",
		BaseURL:  "http://example.test/v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.DefaultModel != "custom-model" || provider.BaseURL != "http://example.test/v1" {
		t.Fatalf("overrides not applied: %+v", provider)
	}
}
