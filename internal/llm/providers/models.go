package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/config"
)

type ModelLister interface {
	ListModels(ctx context.Context, provider Provider, credential Credential) ([]ModelSpec, error)
}

func ContextWindowForModel(model string) int {
	if override := strings.TrimSpace(os.Getenv("APEX_CONTEXT_WINDOW")); override != "" {
		if n, err := strconv.Atoi(override); err == nil && n > 0 {
			return n
		}
	}

	model = strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(model, "gpt-5.5"):
		return 400000
	case strings.HasPrefix(model, "gpt-5.4"):
		return 400000
	case strings.HasPrefix(model, "gpt-4.1"), strings.HasPrefix(model, "gpt-4o"):
		return 128000
	default:
		return 0
	}
}

type OpenAICompatibleLister struct {
}

type ModelListUnsupportedError struct {
	Provider string
	Status   string
}

func (e ModelListUnsupportedError) Error() string {
	if e.Status == "" {
		return fmt.Sprintf("model listing unsupported for %s", e.Provider)
	}
	return fmt.Sprintf("model listing unsupported for %s: %s", e.Provider, e.Status)
}

func IsModelListUnsupported(err error) bool {
	var unsupported ModelListUnsupportedError
	return errors.As(err, &unsupported)
}

func (l *OpenAICompatibleLister) ListModels(ctx context.Context, provider Provider, credential Credential) ([]ModelSpec, error) {
	if !provider.Client.SupportsModelListing {
		return nil, ModelListUnsupportedError{Provider: provider.ID}
	}
	modelsPath := strings.TrimSpace(provider.Client.ModelsPath)
	if modelsPath == "" {
		modelsPath = "/models"
	}
	endpoint := strings.TrimRight(provider.Client.BaseURL, "/") + "/" + strings.TrimLeft(modelsPath, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if credential.TokenSource != nil {
		token, err := credential.TokenSource.Token(ctx)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	} else if strings.TrimSpace(credential.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(credential.APIKey))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
			return nil, ModelListUnsupportedError{Provider: provider.ID, Status: resp.Status}
		}
		return nil, fmt.Errorf("list models for %s: %s", provider.ID, resp.Status)
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	models := make([]ModelSpec, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		models = append(models, ModelSpec{ID: id, DisplayName: id})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

type StaticLister struct{}

func (l StaticLister) ListModels(ctx context.Context, provider Provider, credential Credential) ([]ModelSpec, error) {
	return []ModelSpec{
		{ID: "gpt-5.5", DisplayName: "gpt-5.5", Context: 400000},
		{ID: "gpt-5.4", DisplayName: "gpt-5.4"},
		{ID: "gpt-5.4-mini", DisplayName: "gpt-5.4-mini"},
		{ID: "gpt-5.3-codex-spark", DisplayName: "gpt-5.3-codex-spark"},
	}, nil
}

func ModelListers() map[ClientType]ModelLister {
	return map[ClientType]ModelLister{
		ClientTypeOpenAICompatible: &OpenAICompatibleLister{},
		ClientTypeCodex:            StaticLister{},
	}
}

func ListModels(ctx context.Context, provider Provider, cfg config.Config, manager *auth.Manager) ([]ModelSpec, error) {
	credential, err := CredentialForProvider(ctx, provider, cfg, manager)
	if err != nil {
		return nil, err
	}
	lister, ok := ModelListers()[provider.Client.Type]
	if !ok {
		return nil, fmt.Errorf("provider %q uses unsupported client type %q", provider.ID, provider.Client.Type)
	}

	return lister.ListModels(ctx, provider, credential)
}
