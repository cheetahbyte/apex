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

type Credential struct {
	APIKey      string
	TokenSource llm.BearerTokenSource
}

type ClientBuilder interface {
	Build(ctx context.Context, provider Provider, model string, credential Credential) (llm.Client, error)
}

type OpenAICompatibleBuilder struct{}

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

func Builders() map[ClientType]ClientBuilder {
	return map[ClientType]ClientBuilder{
		ClientTypeOpenAICompatible: OpenAICompatibleBuilder{},
	}
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
	builder, ok := Builders()[provider.Client.Type]
	if !ok {
		return nil, fmt.Errorf("provider %q uses unsupported client type %q", provider.ID, provider.Client.Type)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.DefaultModel
	}
	if model == "" {
		return nil, fmt.Errorf("provider %q has no model configured", provider.ID)
	}
	if apiKey := strings.TrimSpace(cfg.APIKey); apiKey != "" {
		return builder.Build(ctx, provider, model, Credential{APIKey: apiKey})
	}
	if manager == nil {
		return nil, fmt.Errorf("auth manager is required for provider %q", provider.ID)
	}
	sourceID := auth.CredentialSourceID(provider.ID)
	switch provider.Auth.Type {
	case AuthTypeNone:
		return builder.Build(ctx, provider, model, Credential{})
	case AuthTypeAPIKey:
		if strings.TrimSpace(provider.Auth.DefaultKey) != "" {
			return builder.Build(ctx, provider, model, Credential{APIKey: strings.TrimSpace(provider.Auth.DefaultKey)})
		}
		apiKey, err := manager.APIKey(ctx, sourceID)
		if err != nil {
			return nil, err
		}
		return builder.Build(ctx, provider, model, Credential{APIKey: apiKey})
	case AuthTypeOAuthPKCE:
		return builder.Build(ctx, provider, model, Credential{TokenSource: tokenSource{manager: manager, id: sourceID}})
	default:
		return nil, fmt.Errorf("provider %q uses unsupported auth type %q", provider.ID, provider.Auth.Type)
	}
}
