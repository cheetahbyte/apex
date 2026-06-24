package providers

import (
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/config"
)

func Builtins() map[string]Provider {
	return map[string]Provider{
		"ollama": {
			ID:           "ollama",
			DisplayName:  "Ollama",
			Protocol:     ProtocolOpenAICompatible,
			BaseURL:      "http://localhost:11434/v1",
			DefaultModel: "gemma4:12b",
			AuthKind:     auth.AuthKindAPIKey,
			ToolMode:     config.ToolModeAuto,
		},
		"openrouter": {
			ID:           "openrouter",
			DisplayName:  "OpenRouter",
			Protocol:     ProtocolOpenAICompatible,
			BaseURL:      "https://openrouter.ai/api/v1",
			DefaultModel: "anthropic/claude-sonnet-4",
			AuthKind:     auth.AuthKindAPIKey,
			ToolMode:     config.ToolModeAuto,
		},
		"codex": {
			ID:           "codex",
			DisplayName:  "Codex",
			Protocol:     ProtocolOpenAICompatible,
			BaseURL:      "https://api.openai.com/v1",
			DefaultModel: "openai/gpt-5.5-fast",
			AuthKind:     auth.AuthKindOAuth2,
			ToolMode:     config.ToolModeAuto,
		},
		"opencode-go": {
			ID:           "opencode-go",
			DisplayName:  "opencode-go",
			Protocol:     ProtocolOpenAICompatible,
			BaseURL:      "https://api.opencode.ai/v1",
			DefaultModel: "deepseek-v4-flash",
			AuthKind:     auth.AuthKindAPIKey,
			ToolMode:     config.ToolModeAuto,
		},
	}
}

func Resolve(cfg config.Config) (Provider, error) {
	providerID := canonicalProviderID(cfg.Provider)
	if providerID == "" {
		providerID = "ollama"
	}
	provider, ok := Builtins()[providerID]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q", cfg.Provider)
	}
	if strings.TrimSpace(cfg.BaseURL) != "" {
		provider.BaseURL = strings.TrimSpace(cfg.BaseURL)
	}
	if strings.TrimSpace(cfg.Model) != "" {
		provider.DefaultModel = strings.TrimSpace(cfg.Model)
	}
	if cfg.ToolMode != "" {
		provider.ToolMode = cfg.ToolMode
	}
	return provider, nil
}

func canonicalProviderID(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "", "ollama":
		return strings.ToLower(strings.TrimSpace(id))
	case "openai", "openai-codex", "chatgpt":
		return "codex"
	default:
		return strings.ToLower(strings.TrimSpace(id))
	}
}
