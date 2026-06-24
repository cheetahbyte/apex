package providers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cheetahbyte/apex/internal/config"
)

func Builtins() map[string]Provider {
	return map[string]Provider{
		"ollama": {
			ID:          "ollama",
			DisplayName: "Ollama",
			Client: ClientSpec{
				Type:                 ClientTypeOpenAICompatible,
				BaseURL:              "http://localhost:11434/v1",
				SupportsModelListing: true,
				ModelsPath:           "/models",
			},
			Auth: AuthSpec{
				Type:       AuthTypeAPIKey,
				DefaultKey: "ollama",
			},
			ToolMode: config.ToolModeAuto,
		},
		"openrouter": {
			ID:          "openrouter",
			DisplayName: "OpenRouter",
			Client: ClientSpec{
				Type:                 ClientTypeOpenAICompatible,
				BaseURL:              "https://openrouter.ai/api/v1",
				SupportsModelListing: true,
				ModelsPath:           "/models",
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
			ID:          "codex",
			DisplayName: "Codex",
			Aliases:     []string{"openai", "openai-codex", "chatgpt"},
			Client: ClientSpec{
				Type:                 ClientTypeCodex,
				BaseURL:              "https://chatgpt.com/backend-api/codex",
				SupportsModelListing: false,
			},
			Auth: AuthSpec{
				Type: AuthTypeOAuthPKCE,
				OAuth: &OAuthSpec{
					Issuer:   "https://auth.openai.com",
					ClientID: "app_EMoamEEZ73f0CkXaXp7hrann",
					Scopes:   []string{"openid", "email", "profile", "offline_access"},
					AuthorizeParams: map[string]string{
						"id_token_add_organizations": "true",
						"codex_cli_simplified_flow":  "true",
						"originator":                 "apex",
					},
					RedirectPath:  "/auth/callback",
					DefaultPort:   1455,
					AuthEndpoint:  "https://auth.openai.com/oauth/authorize",
					TokenEndpoint: "https://auth.openai.com/oauth/token",
				},
			},
			ToolMode: config.ToolModeAuto,
		},
		"opencode-go": {
			ID:          "opencode-go",
			DisplayName: "opencode-go",
			Client: ClientSpec{
				Type:                 ClientTypeOpenAICompatible,
				BaseURL:              "https://opencode.ai/zen/go/v1",
				SupportsModelListing: true,
				ModelsPath:           "/models",
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
	if cfg.ToolMode != "" {
		provider.ToolMode = cfg.ToolMode
	}
	return provider, nil
}

func All() []Provider {
	providers := Builtins()
	out := make([]Provider, 0, len(providers))
	for _, provider := range providers {
		out = append(out, provider)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
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
