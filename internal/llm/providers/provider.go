package providers

import (
	"github.com/cheetahbyte/apex/internal/config"
)

type ClientType string

const (
	ClientTypeOpenAICompatible ClientType = "openai-compatible"
)

type AuthType string

const (
	AuthTypeNone      AuthType = "none"
	AuthTypeAPIKey    AuthType = "api-key"
	AuthTypeOAuthPKCE AuthType = "oauth-pkce"
)

type Provider struct {
	ID           string
	DisplayName  string
	Aliases      []string
	Client       ClientSpec
	Auth         AuthSpec
	DefaultModel string
	ToolMode     config.ToolMode
}

type ClientSpec struct {
	Type    ClientType
	BaseURL string
	Headers map[string]string
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
	Issuer        string
	ClientID      string
	Scopes        []string
	RedirectPath  string
	DefaultPort   int
	AuthEndpoint  string
	TokenEndpoint string
}
