package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/config"
	"github.com/cheetahbyte/apex/internal/llm"
)

type Credential struct {
	APIKey      string
	TokenSource llm.BearerTokenSource
}

type tokenSource struct {
	manager *auth.Manager
	id      auth.CredentialSourceID
}

func CredentialForProvider(ctx context.Context, provider Provider, cfg config.Config, manager *auth.Manager) (Credential, error) {
	if apiKey := strings.TrimSpace(cfg.APIKey); apiKey != "" {
		return Credential{APIKey: apiKey}, nil
	}

	switch provider.Auth.Type {
	case AuthTypeNone:
		return Credential{}, nil

	case AuthTypeAPIKey:
		if apiKey := strings.TrimSpace(provider.Auth.DefaultKey); apiKey != "" {
			return Credential{APIKey: apiKey}, nil
		}
		if manager == nil {
			return Credential{}, fmt.Errorf("auth manager is required for provider %q", provider.ID)
		}
		apiKey, err := manager.APIKey(ctx, auth.CredentialSourceID(provider.ID))
		if err != nil {
			return Credential{}, err
		}
		return Credential{APIKey: apiKey}, nil

	case AuthTypeOAuthPKCE:
		if manager == nil {
			return Credential{}, fmt.Errorf("auth manager is required for provider %q", provider.ID)
		}
		if sourceAuth, ok, err := manager.Status(ctx, auth.CredentialSourceID(provider.ID)); err != nil {
			return Credential{}, err
		} else if !ok || strings.TrimSpace(sourceAuth.AccessToken) == "" || strings.TrimSpace(sourceAuth.RefreshToken) == "" {
			return Credential{}, fmt.Errorf("provider %q is not authenticated", provider.ID)
		}
		return Credential{
			TokenSource: tokenSource{
				manager: manager,
				id:      auth.CredentialSourceID(provider.ID),
			},
		}, nil

	default:
		return Credential{}, fmt.Errorf("provider %q uses unsupported auth type %q", provider.ID, provider.Auth.Type)
	}
}

func IsConfigured(ctx context.Context, provider Provider, cfg config.Config, manager *auth.Manager) bool {
	if provider.Auth.Type == AuthTypeNone {
		return true
	}
	if strings.TrimSpace(cfg.APIKey) != "" && selectedProviderMatches(cfg, provider) {
		return true
	}
	if manager == nil {
		return false
	}
	sourceAuth, ok, err := manager.Status(ctx, auth.CredentialSourceID(provider.ID))
	if err != nil || !ok {
		return false
	}
	switch provider.Auth.Type {
	case AuthTypeAPIKey:
		return sourceAuth.Type == auth.AuthKindAPIKey && strings.TrimSpace(sourceAuth.Key) != ""
	case AuthTypeOAuthPKCE:
		return sourceAuth.Type == auth.AuthKindOAuth2 && strings.TrimSpace(sourceAuth.AccessToken) != "" && strings.TrimSpace(sourceAuth.RefreshToken) != ""
	default:
		return false
	}
}

func selectedProviderMatches(cfg config.Config, provider Provider) bool {
	if strings.TrimSpace(cfg.Provider) == "" {
		return false
	}
	selected, err := Resolve(config.Config{Provider: cfg.Provider})
	return err == nil && selected.ID == provider.ID
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

func (s tokenSource) AccountID(ctx context.Context) (string, error) {
	sourceAuth, ok, err := s.manager.Status(ctx, s.id)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("no credential stored for %s", s.id)
	}
	if strings.TrimSpace(sourceAuth.AccountID) == "" {
		return "", fmt.Errorf("credential for %s has no ChatGPT account id", s.id)
	}
	return strings.TrimSpace(sourceAuth.AccountID), nil
}
