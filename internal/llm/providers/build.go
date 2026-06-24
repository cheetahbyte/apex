package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/config"
	"github.com/cheetahbyte/apex/internal/llm"
)

type tokenSource struct {
	manager *auth.Manager
	id      auth.CredentialSourceID
}

func (s tokenSource) Token(ctx context.Context) (string, error) {
	return s.manager.BearerToken(ctx, s.id)
}

func (s tokenSource) Refresh(ctx context.Context) (string, error) {
	sourceAuth, ok, err := s.manager.Status(ctx, s.id)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("no credential stored for %s", s.id)
	}
	if sourceAuth.Type == auth.AuthKindAPIKey {
		key := strings.TrimSpace(sourceAuth.Key)
		if key == "" {
			return "", fmt.Errorf("no api key found for source %s", s.id)
		}
		return key, nil
	}
	return s.manager.Refresh(ctx, s.id)
}

func Build(ctx context.Context, provider Provider, cfg config.Config, manager *auth.Manager) (llm.Client, error) {
	if provider.Protocol != ProtocolOpenAICompatible {
		return nil, fmt.Errorf("provider %q uses unsupported protocol %q", provider.ID, provider.Protocol)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.DefaultModel
	}
	if model == "" {
		return nil, fmt.Errorf("provider %q has no model configured", provider.ID)
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = provider.BaseURL
	}
	if baseURL == "" {
		return nil, fmt.Errorf("provider %q has no base URL configured", provider.ID)
	}
	if apiKey := strings.TrimSpace(cfg.APIKey); apiKey != "" {
		return llm.NewOpenAIClient(model, baseURL, apiKey), nil
	}
	if provider.ID == "ollama" {
		return llm.NewOpenAIClient(model, baseURL, "ollama"), nil
	}
	if manager == nil {
		return nil, fmt.Errorf("auth manager is required for provider %q", provider.ID)
	}
	sourceID := auth.CredentialSourceID(provider.ID)
	if provider.AuthKind == auth.AuthKindAPIKey {
		apiKey, err := manager.APIKey(ctx, sourceID)
		if err != nil {
			return nil, err
		}
		return llm.NewOpenAIClient(model, baseURL, apiKey), nil
	}
	return llm.NewOpenAIClientWithTokenSource(model, baseURL, tokenSource{manager: manager, id: sourceID}), nil
}
