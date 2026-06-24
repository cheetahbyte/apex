package sources

import (
	"fmt"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/auth/sources/openai"
)

func Builtins() []auth.CredentialSource {
	return []auth.CredentialSource{openai.CodexSource{}}
}

func ByID(id auth.CredentialSourceID) (auth.CredentialSource, error) {
	if id == "openai" || id == "openai-codex" || id == "chatgpt" {
		id = "codex"
	}
	for _, source := range Builtins() {
		if source.ID() == id {
			return source, nil
		}
	}
	return nil, fmt.Errorf("unknown provider %q", id)
}
