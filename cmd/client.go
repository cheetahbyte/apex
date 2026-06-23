package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/auth/oauth"
	authproviders "github.com/cheetahbyte/apex/internal/auth/providers"
	"github.com/cheetahbyte/apex/internal/config"
	"github.com/cheetahbyte/apex/internal/llm"
)

type providerTokenSource struct {
	manager    *auth.Manager
	providerID auth.ProviderID
}

func (s providerTokenSource) Token(ctx context.Context) (string, error) {
	return s.manager.Token(ctx, s.providerID)
}

func (s providerTokenSource) Refresh(ctx context.Context) (string, error) {
	return s.manager.Refresh(ctx, s.providerID)
}

func newLLMClient(cfg config.Config) (llm.Client, error) {
	if strings.TrimSpace(cfg.AuthProvider) == "" {
		return llm.NewOpenAIClient(cfg.Model, cfg.BaseURL, cfg.APIKey), nil
	}
	manager, err := newAuthManager()
	if err != nil {
		return nil, err
	}
	providerID := auth.ProviderID(cfg.AuthProvider)
	if _, ok := manager.Provider(providerID); !ok {
		return nil, fmt.Errorf("unknown APEX_AUTH_PROVIDER %q", cfg.AuthProvider)
	}
	return llm.NewOpenAIClientWithTokenSource(cfg.Model, cfg.BaseURL, providerTokenSource{
		manager:    manager,
		providerID: providerID,
	}), nil
}

func newAuthManager() (*auth.Manager, error) {
	store, err := auth.DefaultFileStore()
	if err != nil {
		return nil, err
	}
	return auth.NewManager(store, authproviders.Builtins(), oauth.NewClient(nil)), nil
}
