package providers

import (
	"github.com/cheetahbyte/apex/internal/config"
)

type ClientType string

const (
	ClientTypeOpenAICompatible ClientType = "openai-compatible"
	ClientTypeCodex            ClientType = "codex"
)

type AuthType string

const (
	AuthTypeNone      AuthType = "none"
	AuthTypeAPIKey    AuthType = "api-key"
	AuthTypeOAuthPKCE AuthType = "oauth-pkce"
)

type Provider struct {
	ID          string
	DisplayName string
	Aliases     []string
	Client      ClientSpec
	Auth        AuthSpec
	ToolMode    config.ToolMode
}

type ClientSpec struct {
	Type                 ClientType
	BaseURL              string
	Headers              map[string]string
	SupportsModelListing bool
	ModelsPath           string
}

type AuthSpec struct {
	Type       AuthType
	Prompts    []PromptSpec
	OAuth      *OAuthSpec
	DefaultKey string
}

type PromptSpec struct {
	Name     string
	Label    string
	Secret   bool
	Required bool
}

type OAuthSpec struct {
	Issuer          string
	ClientID        string
	Scopes          []string
	AuthorizeParams map[string]string
	RedirectPath    string
	DefaultPort     int
	AuthEndpoint    string
	TokenEndpoint   string
}

type ModelSpec struct {
	ID          string
	DisplayName string
	Context     int
}
