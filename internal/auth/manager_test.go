package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cheetahbyte/apex/internal/auth/oauth"
)

type testOAuthProvider struct{ tokenEndpoint string }

func (p testOAuthProvider) ID() ProviderID        { return "test" }
func (p testOAuthProvider) DisplayName() string   { return "Test" }
func (p testOAuthProvider) AuthKind() AuthKind    { return AuthKindOAuth2 }
func (p testOAuthProvider) Issuer() string        { return "https://issuer.example" }
func (p testOAuthProvider) ClientID() string      { return "client" }
func (p testOAuthProvider) Scopes() []string      { return []string{"openid"} }
func (p testOAuthProvider) RedirectPath() string  { return "/callback" }
func (p testOAuthProvider) DefaultPort() int      { return 1455 }
func (p testOAuthProvider) AuthEndpoint() string  { return "https://issuer.example/authorize" }
func (p testOAuthProvider) TokenEndpoint() string { return p.tokenEndpoint }

func TestManagerTokenRefreshesExpiredToken(t *testing.T) {
	var refreshCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&refreshCalls, 1)
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected content type %q", r.Header.Get("Content-Type"))
		}
		var req oauth.RefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.ClientID != "client" || req.RefreshToken != "refresh" || req.GrantType != "refresh_token" {
			t.Fatalf("unexpected refresh request %+v", req)
		}
		_ = json.NewEncoder(w).Encode(oauth.TokenResponse{AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresIn: 3600})
	}))
	defer server.Close()

	store := NewFileStore(filepath.Join(t.TempDir(), "auth.json"))
	file := NewAuthFile()
	file.ActiveProvider = "test"
	file.Providers["test"] = ProviderAuth{
		Kind:         AuthKindOAuth2,
		AccessToken:  "old-access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(-time.Hour),
	}
	if err := store.Save(context.Background(), file); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(store, []Provider{testOAuthProvider{tokenEndpoint: server.URL}}, oauth.NewClient(server.Client()))
	token, err := manager.Token(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if token != "new-access" {
		t.Fatalf("unexpected token %q", token)
	}
	if atomic.LoadInt32(&refreshCalls) != 1 {
		t.Fatalf("expected one refresh, got %d", refreshCalls)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Providers["test"].RefreshToken != "new-refresh" {
		t.Fatal("rotated refresh token not stored")
	}
}
