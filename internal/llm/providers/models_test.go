package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/config"
)

func TestIsConfiguredOllamaFalseByDefault(t *testing.T) {
	store := auth.NewFileStore(filepath.Join(t.TempDir(), "auth.json"))
	manager := auth.NewManager(store, nil, nil)
	provider, err := Resolve(config.Config{Provider: "ollama"})
	if err != nil {
		t.Fatal(err)
	}
	if IsConfigured(context.Background(), provider, config.Config{}, manager) {
		t.Fatal("ollama should not be configured by default")
	}
}

func TestIsConfiguredAPIKeyProvider(t *testing.T) {
	store := auth.NewFileStore(filepath.Join(t.TempDir(), "auth.json"))
	manager := auth.NewManager(store, nil, nil)
	provider, err := Resolve(config.Config{Provider: "openrouter"})
	if err != nil {
		t.Fatal(err)
	}
	if IsConfigured(context.Background(), provider, config.Config{}, manager) {
		t.Fatal("openrouter should not be configured before storing key")
	}
	if err := manager.StoreAPIKey(context.Background(), "openrouter", "sk-test"); err != nil {
		t.Fatal(err)
	}
	if !IsConfigured(context.Background(), provider, config.Config{}, manager) {
		t.Fatal("openrouter should be configured after storing key")
	}
}

func TestOpenAICompatibleListerFetchesRemoteModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("missing auth header %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"z-model"},{"id":"a-model"}]}`))
	}))
	defer server.Close()

	provider, err := Resolve(config.Config{Provider: "openrouter", BaseURL: server.URL + "/v1"})
	if err != nil {
		t.Fatal(err)
	}
	models, err := (&OpenAICompatibleLister{}).ListModels(context.Background(), provider, Credential{APIKey: "sk-test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "a-model" || models[1].ID != "z-model" {
		t.Fatalf("unexpected models %+v", models)
	}
}

func TestOpenAICompatibleListerUnsupportedWhenProviderDisablesListing(t *testing.T) {
	provider, err := Resolve(config.Config{Provider: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&OpenAICompatibleLister{}).ListModels(context.Background(), provider, Credential{APIKey: "sk-test"})
	if err == nil || !IsModelListUnsupported(err) {
		t.Fatalf("expected unsupported model listing error, got %v", err)
	}
}

func TestOpenAICompatibleListerUnsupportedStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Forbidden"))
	}))
	defer server.Close()

	provider, err := Resolve(config.Config{Provider: "openrouter", BaseURL: server.URL + "/v1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&OpenAICompatibleLister{}).ListModels(context.Background(), provider, Credential{APIKey: "sk-test"})
	if err == nil || !IsModelListUnsupported(err) {
		t.Fatalf("expected unsupported model listing error, got %v", err)
	}
}
