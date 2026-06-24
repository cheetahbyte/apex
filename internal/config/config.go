package config

import "os"

// ToolMode controls how tools are exposed to the model.
//
//	native — provider-native tool API (OpenAI-style tools field)
//	text   — JSON text protocol injected as system prompt (works with any provider)
//	none   — tools disabled
type ToolMode string

const (
	ToolModeAuto   ToolMode = "auto"
	ToolModeNative ToolMode = "native"
	ToolModeText   ToolMode = "text"
	ToolModeNone   ToolMode = "none"
)

// Config holds runtime settings for the LLM provider. Precedence:
// env var > default. CLI flags can be layered on top later via cobra.
type Config struct {
	Provider         string
	Model            string
	BaseURL          string
	APIKey           string
	ToolMode         ToolMode
	CredentialSource string
}

// Default returns a Config populated from env vars with sensible fallbacks
// for local Ollama development.
func Default() Config {
	legacyCredentialSource := envOr("APEX_CREDENTIAL_SOURCE", os.Getenv("APEX_AUTH_PROVIDER"))
	return Config{
		Provider:         envOr("APEX_PROVIDER", envOr("APEX_LLM_PROVIDER", legacyCredentialSource)),
		Model:            os.Getenv("APEX_MODEL"),
		BaseURL:          os.Getenv("APEX_BASE_URL"),
		APIKey:           os.Getenv("APEX_API_KEY"),
		ToolMode:         ToolMode(envOr("APEX_TOOL_MODE", "auto")),
		CredentialSource: legacyCredentialSource,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
