package cmd

import (
	"context"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/auth/oauth"
	authsources "github.com/cheetahbyte/apex/internal/auth/sources"
	"github.com/cheetahbyte/apex/internal/config"
	"github.com/cheetahbyte/apex/internal/llm"
	llmproviders "github.com/cheetahbyte/apex/internal/llm/providers"
)

func newLLMClient(ctx context.Context, cfg config.Config) (llm.Client, llmproviders.Provider, error) {
	manager, err := newAuthManager()
	if err != nil {
		return nil, llmproviders.Provider{}, err
	}
	provider, err := llmproviders.Resolve(cfg)
	if err != nil {
		return nil, llmproviders.Provider{}, err
	}
	client, err := llmproviders.Build(ctx, provider, cfg, manager)
	if err != nil {
		return nil, llmproviders.Provider{}, err
	}
	return client, provider, nil
}

func newAuthManager() (*auth.Manager, error) {
	store, err := auth.DefaultFileStore()
	if err != nil {
		return nil, err
	}
	return auth.NewManager(store, authsources.Builtins(), oauth.NewClient(nil)), nil
}
