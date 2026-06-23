package config

import (
	"os"
	"testing"
)

func TestDefault_toolModeAutoByDefault(t *testing.T) {
	os.Unsetenv("APEX_TOOL_MODE")
	cfg := Default()
	if cfg.ToolMode != ToolModeAuto {
		t.Fatalf("expected default tool mode 'auto', got %q", cfg.ToolMode)
	}
}

func TestDefault_toolModeFromEnv(t *testing.T) {
	os.Setenv("APEX_TOOL_MODE", "native")
	defer os.Unsetenv("APEX_TOOL_MODE")
	cfg := Default()
	if cfg.ToolMode != ToolModeNative {
		t.Fatalf("expected 'native' from env, got %q", cfg.ToolMode)
	}
}

func TestDefault_toolModeText(t *testing.T) {
	os.Setenv("APEX_TOOL_MODE", "text")
	defer os.Unsetenv("APEX_TOOL_MODE")
	cfg := Default()
	if cfg.ToolMode != ToolModeText {
		t.Fatalf("expected 'text' from env, got %q", cfg.ToolMode)
	}
}

func TestDefault_toolModeNone(t *testing.T) {
	os.Setenv("APEX_TOOL_MODE", "none")
	defer os.Unsetenv("APEX_TOOL_MODE")
	cfg := Default()
	if cfg.ToolMode != ToolModeNone {
		t.Fatalf("expected 'none' from env, got %q", cfg.ToolMode)
	}
}

func TestDefault_modelFromEnv(t *testing.T) {
	os.Setenv("APEX_MODEL", "gpt-4o")
	defer os.Unsetenv("APEX_MODEL")
	cfg := Default()
	if cfg.Model != "gpt-4o" {
		t.Fatalf("expected 'gpt-4o' from env, got %q", cfg.Model)
	}
}

func TestDefault_baseURLFromEnv(t *testing.T) {
	os.Setenv("APEX_BASE_URL", "https://api.openai.com/v1")
	defer os.Unsetenv("APEX_BASE_URL")
	cfg := Default()
	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("expected custom base URL, got %q", cfg.BaseURL)
	}
}

func TestDefault_authProviderFromEnv(t *testing.T) {
	os.Setenv("APEX_AUTH_PROVIDER", "openai-codex")
	defer os.Unsetenv("APEX_AUTH_PROVIDER")
	cfg := Default()
	if cfg.AuthProvider != "openai-codex" {
		t.Fatalf("expected auth provider from env, got %q", cfg.AuthProvider)
	}
}

func TestToolMode_constants(t *testing.T) {
	if ToolModeAuto != "auto" {
		t.Fatal("ToolModeAuto should be 'auto'")
	}
	if ToolModeNative != "native" {
		t.Fatal("ToolModeNative should be 'native'")
	}
	if ToolModeText != "text" {
		t.Fatal("ToolModeText should be 'text'")
	}
	if ToolModeNone != "none" {
		t.Fatal("ToolModeNone should be 'none'")
	}
}
