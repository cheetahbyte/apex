package providers

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/config"
)

func TestBuildMissingAPIKeyReturnsError(t *testing.T) {
	store := auth.NewFileStore(filepath.Join(t.TempDir(), "auth.json"))
	manager := auth.NewManager(store, nil, nil)
	provider, err := Resolve(config.Config{Provider: "openrouter"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), provider, config.Config{Model: "test-model"}, manager); err == nil {
		t.Fatal("expected missing api key error")
	}
}

func TestBuildMissingModelReturnsError(t *testing.T) {
	store := auth.NewFileStore(filepath.Join(t.TempDir(), "auth.json"))
	manager := auth.NewManager(store, nil, nil)
	if err := manager.StoreAPIKey(context.Background(), "openrouter", "sk-test"); err != nil {
		t.Fatal(err)
	}
	provider, err := Resolve(config.Config{Provider: "openrouter"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), provider, config.Config{}, manager); err == nil {
		t.Fatal("expected missing model error")
	}
}

func TestBuildUsesStoredAPIKey(t *testing.T) {
	store := auth.NewFileStore(filepath.Join(t.TempDir(), "auth.json"))
	manager := auth.NewManager(store, nil, nil)
	if err := manager.StoreAPIKey(context.Background(), "openrouter", "sk-test"); err != nil {
		t.Fatal(err)
	}
	provider, err := Resolve(config.Config{Provider: "openrouter"})
	if err != nil {
		t.Fatal(err)
	}
	client, err := Build(context.Background(), provider, config.Config{Model: "test-model"}, manager)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}
