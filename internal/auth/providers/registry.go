package providers

import (
	"fmt"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/auth/providers/openai"
)

func Builtins() []auth.Provider {
	return []auth.Provider{openai.CodexProvider{}}
}

func ByID(id auth.ProviderID) (auth.Provider, error) {
	for _, provider := range Builtins() {
		if provider.ID() == id {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("unknown auth provider %q", id)
}
