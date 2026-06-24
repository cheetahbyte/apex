package providers

import (
	"testing"

	"github.com/cheetahbyte/apex/internal/config"
)

func TestResolveDefaultsToOllama(t *testing.T) {
	provider, err := Resolve(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if provider.ID != "ollama" || provider.Client.BaseURL == "" {
		t.Fatalf("unexpected default provider %+v", provider)
	}
}

func TestResolveOpenRouter(t *testing.T) {
	provider, err := Resolve(config.Config{Provider: "openrouter"})
	if err != nil {
		t.Fatal(err)
	}
	if provider.ID != "openrouter" || provider.Auth.Type != AuthTypeAPIKey {
		t.Fatalf("unexpected openrouter provider %+v", provider)
	}
}

func TestResolveCodexAliases(t *testing.T) {
	for _, id := range []string{"openai", "openai-codex", "chatgpt", "codex"} {
		provider, err := Resolve(config.Config{Provider: id})
		if err != nil {
			t.Fatal(err)
		}
		if provider.ID != "codex" || provider.Auth.Type != AuthTypeOAuthPKCE {
			t.Fatalf("alias %q resolved to %+v", id, provider)
		}
	}
}

func TestResolveOverrides(t *testing.T) {
	provider, err := Resolve(config.Config{
		Provider: "opencode-go",
		BaseURL:  "http://example.test/v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.Client.BaseURL != "http://example.test/v1" {
		t.Fatalf("overrides not applied: %+v", provider)
	}
}

func TestBuiltinsDefineAliasesInProviderTable(t *testing.T) {
	provider := Builtins()["codex"]
	if len(provider.Aliases) == 0 {
		t.Fatal("expected codex aliases in provider definition")
	}
}
