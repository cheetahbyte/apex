package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/config"
	"github.com/cheetahbyte/apex/internal/llm"
)

type ClientBuilder interface {
	Build(ctx context.Context, provider Provider, model string, credential Credential) (llm.Client, error)
}

type OpenAICompatibleBuilder struct{}

type CodexBuilder struct{}

func (OpenAICompatibleBuilder) Build(ctx context.Context, provider Provider, model string, credential Credential) (llm.Client, error) {
	baseURL := strings.TrimSpace(provider.Client.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("provider %q has no base URL configured", provider.ID)
	}
	if credential.TokenSource != nil {
		return llm.NewOpenAIClientWithTokenSource(model, baseURL, credential.TokenSource), nil
	}
	if strings.TrimSpace(credential.APIKey) == "" {
		return nil, fmt.Errorf("provider %q has no API key configured", provider.ID)
	}
	return llm.NewOpenAIClient(model, baseURL, credential.APIKey), nil
}

func (CodexBuilder) Build(ctx context.Context, provider Provider, model string, credential Credential) (llm.Client, error) {
	baseURL := strings.TrimSpace(provider.Client.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("provider %q has no base URL configured", provider.ID)
	}
	if credential.TokenSource == nil {
		return nil, fmt.Errorf("provider %q requires OAuth credentials", provider.ID)
	}
	return llm.NewCodexClient(model, baseURL, credential.TokenSource), nil
}

func Builders() map[ClientType]ClientBuilder {
	return map[ClientType]ClientBuilder{
		ClientTypeOpenAICompatible: OpenAICompatibleBuilder{},
		ClientTypeCodex:            CodexBuilder{},
	}
}

func Build(ctx context.Context, provider Provider, cfg config.Config, manager *auth.Manager) (llm.Client, error) {
	builder, ok := Builders()[provider.Client.Type]
	if !ok {
		return nil, fmt.Errorf("provider %q uses unsupported client type %q", provider.ID, provider.Client.Type)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, fmt.Errorf("provider %q requires APEX_MODEL", provider.ID)
	}
	credential, err := CredentialForProvider(ctx, provider, cfg, manager)
	if err != nil {
		return nil, err
	}

	return builder.Build(ctx, provider, model, credential)
}
