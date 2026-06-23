package auth

import (
	"context"
	"time"
)

type ProviderID string

type AuthKind string

const (
	AuthKindAPIKey AuthKind = "api_key"
	AuthKindOAuth2 AuthKind = "oauth2"
)

type Provider interface {
	ID() ProviderID
	DisplayName() string
	AuthKind() AuthKind
}

type OAuthProvider interface {
	Provider
	Issuer() string
	ClientID() string
	Scopes() []string
	RedirectPath() string
	DefaultPort() int
	AuthEndpoint() string
	TokenEndpoint() string
}

type TokenSource interface {
	Token(ctx context.Context, providerID ProviderID) (string, error)
	Refresh(ctx context.Context, providerID ProviderID) (string, error)
}

type Store interface {
	Load(ctx context.Context) (*AuthFile, error)
	Save(ctx context.Context, file *AuthFile) error
	Delete(ctx context.Context, providerID ProviderID) error
}

type AuthFile struct {
	Version        int                         `json:"version"`
	ActiveProvider ProviderID                  `json:"active_provider,omitempty"`
	Providers      map[ProviderID]ProviderAuth `json:"providers"`
}

type ProviderAuth struct {
	Kind         AuthKind  `json:"kind"`
	Issuer       string    `json:"issuer,omitempty"`
	ClientID     string    `json:"client_id,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	LastRefresh  time.Time `json:"last_refresh,omitempty"`
	Claims       Claims    `json:"claims,omitempty"`
}

type Claims struct {
	Email     string `json:"email,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	PlanType  string `json:"plan_type,omitempty"`
}

func NewAuthFile() *AuthFile {
	return &AuthFile{Version: 1, Providers: map[ProviderID]ProviderAuth{}}
}
