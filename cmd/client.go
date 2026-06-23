package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/auth/oauth"
	authsources "github.com/cheetahbyte/apex/internal/auth/sources"
	"github.com/cheetahbyte/apex/internal/config"
	"github.com/cheetahbyte/apex/internal/llm"
)

type credentialSourceTokenSource struct {
	manager  *auth.Manager
	sourceID auth.CredentialSourceID
}

func (s credentialSourceTokenSource) Token(ctx context.Context) (string, error) {
	return s.manager.Token(ctx, s.sourceID)
}

func (s credentialSourceTokenSource) Refresh(ctx context.Context) (string, error) {
	return s.manager.Refresh(ctx, s.sourceID)
}

func newLLMClient(cfg config.Config) (llm.Client, error) {
	if strings.TrimSpace(cfg.CredentialSource) == "" {
		return llm.NewOpenAIClient(cfg.Model, cfg.BaseURL, cfg.APIKey), nil
	}
	manager, err := newAuthManager()
	if err != nil {
		return nil, err
	}
	sourceID := auth.CredentialSourceID(cfg.CredentialSource)
	if _, ok := manager.Source(sourceID); !ok {
		return nil, fmt.Errorf("unknown APEX_CREDENTIAL_SOURCE %q", cfg.CredentialSource)
	}
	return llm.NewOpenAIClientWithTokenSource(cfg.Model, cfg.BaseURL, credentialSourceTokenSource{
		manager:  manager,
		sourceID: sourceID,
	}), nil
}

func newAuthManager() (*auth.Manager, error) {
	store, err := auth.DefaultFileStore()
	if err != nil {
		return nil, err
	}
	return auth.NewManager(store, authsources.Builtins(), oauth.NewClient(nil)), nil
}
