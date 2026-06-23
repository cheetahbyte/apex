package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cheetahbyte/apex/internal/auth/oauth"
)

const refreshWindow = 5 * time.Minute

type Manager struct {
	store     Store
	providers map[ProviderID]Provider
	client    *oauth.Client
	mu        sync.Mutex
}

func NewManager(store Store, providers []Provider, client *oauth.Client) *Manager {
	byID := make(map[ProviderID]Provider, len(providers))
	for _, provider := range providers {
		byID[provider.ID()] = provider
	}
	if client == nil {
		client = oauth.NewClient(nil)
	}
	return &Manager{store: store, providers: byID, client: client}
}

func (m *Manager) Provider(id ProviderID) (Provider, bool) {
	p, ok := m.providers[id]
	return p, ok
}

func (m *Manager) Providers() []Provider {
	out := make([]Provider, 0, len(m.providers))
	for _, provider := range m.providers {
		out = append(out, provider)
	}
	return out
}

func (m *Manager) StoreLogin(ctx context.Context, provider OAuthProvider, tokens oauth.TokenResponse) error {
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
	file.Providers[provider.ID()] = ProviderAuth{
		Kind:         AuthKindOAuth2,
		Issuer:       provider.Issuer(),
		ClientID:     provider.ClientID(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		IDToken:      tokens.IDToken,
		ExpiresAt:    expiresAt,
		LastRefresh:  now,
		Claims:       claims,
	}
	file.ActiveProvider = provider.ID()
	return m.store.Save(ctx, file)
}

func (m *Manager) Token(ctx context.Context, providerID ProviderID) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, auth, provider, err := m.loadOAuth(ctx, providerID)
	if err != nil {
		return "", err
	}
	if !needsRefresh(auth, time.Now().UTC()) {
		return auth.AccessToken, nil
	}
	updated, err := m.refreshLocked(ctx, provider, auth)
	if err != nil {
		return auth.AccessToken, err
	}
	file.Providers[provider.ID()] = updated
	if err := m.store.Save(ctx, file); err != nil {
		return "", err
	}
	return updated.AccessToken, nil
}

func (m *Manager) Refresh(ctx context.Context, providerID ProviderID) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, auth, provider, err := m.loadOAuth(ctx, providerID)
	if err != nil {
		return "", err
	}
	updated, err := m.refreshLocked(ctx, provider, auth)
	if err != nil {
		return "", err
	}
	file.Providers[provider.ID()] = updated
	if err := m.store.Save(ctx, file); err != nil {
		return "", err
	}
	return updated.AccessToken, nil
}

func (m *Manager) Status(ctx context.Context, providerID ProviderID) (ProviderAuth, bool, error) {
	file, err := m.store.Load(ctx)
	if err != nil {
		return ProviderAuth{}, false, err
	}
	if providerID == "" {
		providerID = file.ActiveProvider
	}
	auth, ok := file.Providers[providerID]
	return auth, ok, nil
}

func (m *Manager) Logout(ctx context.Context, providerID ProviderID) error {
	return m.store.Delete(ctx, providerID)
}

func (m *Manager) loadOAuth(ctx context.Context, providerID ProviderID) (*AuthFile, ProviderAuth, OAuthProvider, error) {
	file, err := m.store.Load(ctx)
	if err != nil {
		return nil, ProviderAuth{}, nil, err
	}
	if providerID == "" {
		providerID = file.ActiveProvider
	}
	if providerID == "" {
		return nil, ProviderAuth{}, nil, fmt.Errorf("no active auth provider")
	}
	provider, ok := m.providers[providerID]
	if !ok {
		return nil, ProviderAuth{}, nil, fmt.Errorf("unknown auth provider %q", providerID)
	}
	oauthProvider, ok := provider.(OAuthProvider)
	if !ok {
		return nil, ProviderAuth{}, nil, fmt.Errorf("provider %q does not support OAuth", providerID)
	}
	auth, ok := file.Providers[providerID]
	if !ok || auth.AccessToken == "" {
		return nil, ProviderAuth{}, nil, fmt.Errorf("not logged in to %s", providerID)
	}
	return file, auth, oauthProvider, nil
}

func (m *Manager) refreshLocked(ctx context.Context, provider OAuthProvider, current ProviderAuth) (ProviderAuth, error) {
	if current.RefreshToken == "" {
		return ProviderAuth{}, fmt.Errorf("refresh token missing for %s", provider.ID())
	}
	resp, err := m.client.Refresh(ctx, provider.TokenEndpoint(), oauth.RefreshRequest{
		GrantType:    "refresh_token",
		ClientID:     provider.ClientID(),
		RefreshToken: current.RefreshToken,
	})
	if err != nil {
		return ProviderAuth{}, err
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
			current.Claims = claims
		}
	}
	current.ExpiresAt = expiryFromTokenResponse(resp, now)
	current.LastRefresh = now
	return current, nil
}

func needsRefresh(auth ProviderAuth, now time.Time) bool {
	if auth.AccessToken == "" {
		return true
	}
	if !auth.ExpiresAt.IsZero() {
		return !auth.ExpiresAt.After(now.Add(refreshWindow))
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
