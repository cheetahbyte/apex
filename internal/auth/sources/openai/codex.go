package openai

import "github.com/cheetahbyte/apex/internal/auth"

const (
	CodexSourceID     auth.CredentialSourceID = "codex"
	CodexIssuer                               = "https://auth.openai.com"
	CodexClientID                             = "app_EMoamEEZ73f0CkXaXp7hrann"
	CodexRedirectPath                         = "/auth/callback"
	CodexDefaultPort                          = 1455
)

type CodexSource struct{}

func (CodexSource) ID() auth.CredentialSourceID { return CodexSourceID }

func (CodexSource) DisplayName() string { return "OpenAI Codex" }

func (CodexSource) AuthKind() auth.AuthKind { return auth.AuthKindOAuth2 }

func (CodexSource) Issuer() string { return CodexIssuer }

func (CodexSource) ClientID() string { return CodexClientID }

func (CodexSource) Scopes() []string {
	return []string{"openid", "email", "profile", "offline_access"}
}

func (CodexSource) RedirectPath() string { return CodexRedirectPath }

func (CodexSource) DefaultPort() int { return CodexDefaultPort }

func (CodexSource) AuthEndpoint() string { return CodexIssuer + "/oauth/authorize" }

func (CodexSource) TokenEndpoint() string { return CodexIssuer + "/oauth/token" }
