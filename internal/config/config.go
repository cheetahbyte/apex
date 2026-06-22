package config

import "os"

// Config holds runtime settings for the LLM provider. Precedence:
// env var > default. CLI flags can be layered on top later via cobra.
type Config struct {
	Model   string
	BaseURL string
	APIKey  string
}

// Default returns a Config populated from env vars with sensible fallbacks
// for local Ollama development.
func Default() Config {
	return Config{
		Model:   envOr("APEX_MODEL", "llama3.2"),
		BaseURL: envOr("APEX_BASE_URL", "http://localhost:11434/v1"),
		APIKey:  envOr("APEX_API_KEY", "ollama"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
