package providers

import (
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/config"
)

func Builtins() map[string]Provider {
	return map[string]Provider{
		"ollama": {
			ID:           "ollama",
			DisplayName:  "Ollama",
			DefaultModel: "gemma4:12b",
			Client: ClientSpec{
				Type:    ClientTypeOpenAICompatible,
				BaseURL: "http://localhost:11434/v1",
			},
			Auth: AuthSpec{
				Type:       AuthTypeAPIKey,
				DefaultKey: "ollama",
			},
			ToolMode: config.ToolModeAuto,
		},
		"openrouter": {
			ID:           "openrouter",
			DisplayName:  "OpenRouter",
			DefaultModel: "anthropic/claude-sonnet-4",
			Client: ClientSpec{
				Type:    ClientTypeOpenAICompatible,
				BaseURL: "https://openrouter.ai/api/v1",
			},
			Auth: AuthSpec{
				Type: AuthTypeAPIKey,
				Prompts: []PromptSpec{{
					Name:     "key",
					Label:    "API Key",
					Secret:   true,
					Required: true,
				}},
			},
			ToolMode: config.ToolModeAuto,
		},
		"codex": {
			ID:           "codex",
			DisplayName:  "Codex",
			Aliases:      []string{"openai", "openai-codex", "chatgpt"},
			DefaultModel: "openai/gpt-5.5-fast",
			Client: ClientSpec{
				Type:    ClientTypeOpenAICompatible,
				BaseURL: "https://api.openai.com/v1",
			},
			Auth: AuthSpec{
				Type: AuthTypeOAuthPKCE,
				OAuth: &OAuthSpec{
					Issuer:        "https://auth.openai.com",
					ClientID:      "app_EMoamEEZ73f0CkXaXp7hrann",
					Scopes:        []string{"openid", "email", "profile", "offline_access"},
					RedirectPath:  "/auth/callback",
					DefaultPort:   1455,
					AuthEndpoint:  "https://auth.openai.com/oauth/authorize",
					TokenEndpoint: "https://auth.openai.com/oauth/token",
				},
			},
			ToolMode: config.ToolModeAuto,
		},
		"opencode-go": {
			ID:           "opencode-go",
			DisplayName:  "opencode-go",
			DefaultModel: "deepseek-v4-flash",
			Client: ClientSpec{
				Type:    ClientTypeOpenAICompatible,
				BaseURL: "https://api.opencode.ai/v1",
			},
			Auth: AuthSpec{
				Type: AuthTypeAPIKey,
				Prompts: []PromptSpec{{
					Name:     "key",
					Label:    "API Key",
					Secret:   true,
					Required: true,
				}},
			},
			ToolMode: config.ToolModeAuto,
		},
	}
}

func Resolve(cfg config.Config) (Provider, error) {
	providerID := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if providerID == "" {
		providerID = "ollama"
	}
	providers := Builtins()
	if canonicalID, ok := aliasIndex(providers)[providerID]; ok {
		providerID = canonicalID
	}
	provider, ok := providers[providerID]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q", cfg.Provider)
	}
	if strings.TrimSpace(cfg.BaseURL) != "" {
		provider.Client.BaseURL = strings.TrimSpace(cfg.BaseURL)
	}
	if strings.TrimSpace(cfg.Model) != "" {
		provider.DefaultModel = strings.TrimSpace(cfg.Model)
	}
	if cfg.ToolMode != "" {
		provider.ToolMode = cfg.ToolMode
	}
	return provider, nil
}

func aliasIndex(providers map[string]Provider) map[string]string {
	aliases := map[string]string{}
	for id, provider := range providers {
		aliases[strings.ToLower(strings.TrimSpace(id))] = id
		for _, alias := range provider.Aliases {
			aliases[strings.ToLower(strings.TrimSpace(alias))] = id
		}
	}
	return aliases
}
