package openai

import "github.com/cheetahbyte/apex/internal/auth"

const (
	CodexProviderID   auth.ProviderID = "openai-codex"
	CodexIssuer                       = "https://auth.openai.com"
	CodexClientID                     = "app_EMoamEEZ73f0CkXaXp7hrann"
	CodexRedirectPath                 = "/auth/callback"
	CodexDefaultPort                  = 1455
)

type CodexProvider struct{}

func (CodexProvider) ID() auth.ProviderID { return CodexProviderID }

func (CodexProvider) DisplayName() string { return "OpenAI Codex" }

func (CodexProvider) AuthKind() auth.AuthKind { return auth.AuthKindOAuth2 }

func (CodexProvider) Issuer() string { return CodexIssuer }

func (CodexProvider) ClientID() string { return CodexClientID }

func (CodexProvider) Scopes() []string {
	return []string{"openid", "email", "profile", "offline_access"}
}

func (CodexProvider) RedirectPath() string { return CodexRedirectPath }

func (CodexProvider) DefaultPort() int { return CodexDefaultPort }

func (CodexProvider) AuthEndpoint() string { return CodexIssuer + "/oauth/authorize" }

func (CodexProvider) TokenEndpoint() string { return CodexIssuer + "/oauth/token" }
