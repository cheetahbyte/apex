package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cheetahbyte/apex/internal/auth/oauth"
)

const refreshWindow = 5 * time.Minute

type Manager struct {
	store   Store
	sources map[CredentialSourceID]CredentialSource
	client  *oauth.Client
	mu      sync.Mutex
}

func NewManager(store Store, sources []CredentialSource, client *oauth.Client) *Manager {
	byID := make(map[CredentialSourceID]CredentialSource, len(sources))
	for _, source := range sources {
		byID[source.ID()] = source
	}
	if client == nil {
		client = oauth.NewClient(nil)
	}
	return &Manager{store: store, sources: byID, client: client}
}

func (m *Manager) Source(id CredentialSourceID) (CredentialSource, bool) {
	id = canonicalSourceID(id)
	s, ok := m.sources[id]
	return s, ok
}

func (m *Manager) Sources() []CredentialSource {
	out := make([]CredentialSource, 0, len(m.sources))
	for _, source := range m.sources {
		out = append(out, source)
	}
	return out
}

func (m *Manager) StoreLogin(ctx context.Context, source OAuthCredentialSource, tokens oauth.TokenResponse) error {
	file, err := m.store.Load(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	expiresAt := expiryFromTokenResponse(tokens, now)
	claims := Claims{}
	if tokens.IDToken != "" {
		claims, _ = ClaimsFromJWT(tokens.IDToken)
	}
	(*file)[source.ID()] = SourceAuth{
		Type:         AuthKindOAuth2,
		Issuer:       source.Issuer(),
		ClientID:     source.ClientID(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		IDToken:      tokens.IDToken,
		Expires:      unixTime(expiresAt),
		LastRefresh:  unixTime(now),
		AccountID:    claims.AccountID,
		Email:        claims.Email,
		PlanType:     claims.PlanType,
	}
	return m.store.Save(ctx, file)
}

func (m *Manager) StoreAPIKey(ctx context.Context, sourceID CredentialSourceID, apiKey string) error {
	if sourceID == "" {
		return fmt.Errorf("credential source is required")
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("api key is required")
	}
	file, err := m.store.Load(ctx)
	if err != nil {
		return err
	}
	(*file)[canonicalSourceID(sourceID)] = SourceAuth{
		Type: AuthKindAPIKey,
		Key:  apiKey,
	}
	return m.store.Save(ctx, file)
}

func (m *Manager) APIKey(ctx context.Context, sourceID CredentialSourceID) (string, error) {
	file, err := m.store.Load(ctx)
	if err != nil {
		return "", err
	}

	sourceID = canonicalSourceID(sourceID)
	auth, ok := (*file)[sourceID]
	if !ok || auth.Key == "" {
		return "", fmt.Errorf("no api key found for source %s", sourceID)
	}
	if auth.Type != AuthKindAPIKey {
		return "", fmt.Errorf("credential source %s is not an api key source", sourceID)
	}
	return auth.Key, nil
}

func (m *Manager) Token(ctx context.Context, sourceID CredentialSourceID) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, auth, source, err := m.loadOAuth(ctx, sourceID)
	if err != nil {
		return "", err
	}
	if !needsRefresh(auth, time.Now().UTC()) {
		return auth.AccessToken, nil
	}
	updated, err := m.refreshLocked(ctx, source, auth)
	if err != nil {
		return auth.AccessToken, err
	}
	(*file)[source.ID()] = updated
	if err := m.store.Save(ctx, file); err != nil {
		return "", err
	}
	return updated.AccessToken, nil
}

func (m *Manager) Refresh(ctx context.Context, sourceID CredentialSourceID) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, auth, source, err := m.loadOAuth(ctx, sourceID)
	if err != nil {
		return "", err
	}
	updated, err := m.refreshLocked(ctx, source, auth)
	if err != nil {
		return "", err
	}
	(*file)[source.ID()] = updated
	if err := m.store.Save(ctx, file); err != nil {
		return "", err
	}
	return updated.AccessToken, nil
}

func (m *Manager) Status(ctx context.Context, sourceID CredentialSourceID) (SourceAuth, bool, error) {
	file, err := m.store.Load(ctx)
	if err != nil {
		return SourceAuth{}, false, err
	}
	auth, ok := (*file)[canonicalSourceID(sourceID)]
	return auth, ok, nil
}

func (m *Manager) Statuses(ctx context.Context) (AuthFile, error) {
	file, err := m.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	return *file, nil
}

func (m *Manager) Logout(ctx context.Context, sourceID CredentialSourceID) error {
	return m.store.Delete(ctx, sourceID)
}

func (m *Manager) loadOAuth(ctx context.Context, sourceID CredentialSourceID) (*AuthFile, SourceAuth, OAuthCredentialSource, error) {
	file, err := m.store.Load(ctx)
	if err != nil {
		return nil, SourceAuth{}, nil, err
	}
	if sourceID == "" {
		return nil, SourceAuth{}, nil, fmt.Errorf("no active credential source")
	}
	sourceID = canonicalSourceID(sourceID)
	source, ok := m.sources[sourceID]
	if !ok {
		return nil, SourceAuth{}, nil, fmt.Errorf("unknown credential source %q", sourceID)
	}
	oauthSource, ok := source.(OAuthCredentialSource)
	if !ok {
		return nil, SourceAuth{}, nil, fmt.Errorf("credential source %q does not support OAuth", sourceID)
	}
	auth, ok := (*file)[sourceID]
	if !ok || auth.AccessToken == "" {
		return nil, SourceAuth{}, nil, fmt.Errorf("not logged in to %s", sourceID)
	}
	return file, auth, oauthSource, nil
}

func (m *Manager) refreshLocked(ctx context.Context, source OAuthCredentialSource, current SourceAuth) (SourceAuth, error) {
	if current.RefreshToken == "" {
		return SourceAuth{}, fmt.Errorf("refresh token missing for %s", source.ID())
	}
	resp, err := m.client.Refresh(ctx, source.TokenEndpoint(), oauth.RefreshRequest{
		GrantType:    "refresh_token",
		ClientID:     source.ClientID(),
		RefreshToken: current.RefreshToken,
	})
	if err != nil {
		return SourceAuth{}, err
	}
	now := time.Now().UTC()
	if resp.AccessToken != "" {
		current.AccessToken = resp.AccessToken
	}
	if resp.RefreshToken != "" {
		current.RefreshToken = resp.RefreshToken
	}
	if resp.IDToken != "" {
		current.IDToken = resp.IDToken
		if claims, err := ClaimsFromJWT(resp.IDToken); err == nil {
			current.AccountID = claims.AccountID
			current.Email = claims.Email
			current.PlanType = claims.PlanType
		}
	}
	current.Expires = unixTime(expiryFromTokenResponse(resp, now))
	current.LastRefresh = unixTime(now)
	return current, nil
}

func needsRefresh(auth SourceAuth, now time.Time) bool {
	if auth.AccessToken == "" {
		return true
	}
	if expiresAt := auth.ExpiresAt(); !expiresAt.IsZero() {
		return !expiresAt.After(now.Add(refreshWindow))
	}
	if exp, ok, err := JWTExpiration(auth.AccessToken); err == nil && ok {
		return !exp.After(now.Add(refreshWindow))
	}
	return false
}

func expiryFromTokenResponse(resp oauth.TokenResponse, now time.Time) time.Time {
	if resp.ExpiresIn > 0 {
		return now.Add(time.Duration(resp.ExpiresIn) * time.Second)
	}
	if exp, ok, err := JWTExpiration(resp.AccessToken); err == nil && ok {
		return exp
	}
	return time.Time{}
}
