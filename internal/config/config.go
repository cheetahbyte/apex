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
	Model        string
	BaseURL      string
	APIKey       string
	ToolMode     ToolMode
	AuthProvider string
}

// Default returns a Config populated from env vars with sensible fallbacks
// for local Ollama development.
func Default() Config {
	return Config{
		Model:        envOr("APEX_MODEL", "gemma4:12b"),
		BaseURL:      envOr("APEX_BASE_URL", "http://localhost:11434/v1"),
		APIKey:       envOr("APEX_API_KEY", "ollama"),
		ToolMode:     ToolMode(envOr("APEX_TOOL_MODE", "auto")),
		AuthProvider: os.Getenv("APEX_AUTH_PROVIDER"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
