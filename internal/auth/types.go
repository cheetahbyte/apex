package auth

import (
	"context"
	"time"
)

type CredentialSourceID string

type AuthKind string

const (
	AuthKindAPIKey AuthKind = "api"
	AuthKindOAuth2 AuthKind = "oauth"
)

type CredentialSource interface {
	ID() CredentialSourceID
	DisplayName() string
	AuthKind() AuthKind
}

type OAuthCredentialSource interface {
	CredentialSource
	Issuer() string
	ClientID() string
	Scopes() []string
	RedirectPath() string
	DefaultPort() int
	AuthEndpoint() string
	TokenEndpoint() string
}

type TokenSource interface {
	Token(ctx context.Context, sourceID CredentialSourceID) (string, error)
	Refresh(ctx context.Context, sourceID CredentialSourceID) (string, error)
}

type Store interface {
	Load(ctx context.Context) (*AuthFile, error)
	Save(ctx context.Context, file *AuthFile) error
	Delete(ctx context.Context, sourceID CredentialSourceID) error
}

type AuthFile map[CredentialSourceID]SourceAuth

type SourceAuth struct {
	Type          AuthKind `json:"type"`
	Key           string   `json:"key,omitempty"`
	AccessToken   string   `json:"access,omitempty"`
	RefreshToken  string   `json:"refresh,omitempty"`
	IDToken       string   `json:"id,omitempty"`
	Expires       int64    `json:"expires,omitempty"`
	LastRefresh   int64    `json:"lastRefresh,omitempty"`
	AccountID     string   `json:"accountId,omitempty"`
	Email         string   `json:"email,omitempty"`
	PlanType      string   `json:"planType,omitempty"`
	Issuer        string   `json:"issuer,omitempty"`
	ClientID      string   `json:"clientId,omitempty"`
	TokenEndpoint string   `json:"tokenEndpoint,omitempty"`
}

type Claims struct {
	Email     string `json:"email,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	PlanType  string `json:"plan_type,omitempty"`
}

func NewAuthFile() *AuthFile {
	file := AuthFile{}
	return &file
}

func (a SourceAuth) ExpiresAt() time.Time {
	if a.Expires == 0 {
		return time.Time{}
	}
	return time.Unix(a.Expires, 0).UTC()
}

func unixTime(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UTC().Unix()
}
