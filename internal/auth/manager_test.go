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

type testOAuthSource struct{ tokenEndpoint string }

func (p testOAuthSource) ID() CredentialSourceID { return "test" }
func (p testOAuthSource) DisplayName() string    { return "Test" }
func (p testOAuthSource) AuthKind() AuthKind     { return AuthKindOAuth2 }
func (p testOAuthSource) Issuer() string         { return "https://issuer.example" }
func (p testOAuthSource) ClientID() string       { return "client" }
func (p testOAuthSource) Scopes() []string       { return []string{"openid"} }
func (p testOAuthSource) RedirectPath() string   { return "/callback" }
func (p testOAuthSource) DefaultPort() int       { return 1455 }
func (p testOAuthSource) AuthEndpoint() string   { return "https://issuer.example/authorize" }
func (p testOAuthSource) TokenEndpoint() string  { return p.tokenEndpoint }

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
	(*file)["test"] = SourceAuth{
		Type:         AuthKindOAuth2,
		AccessToken:  "old-access",
		RefreshToken: "refresh",
		Expires:      unixTime(time.Now().Add(-time.Hour)),
	}
	if err := store.Save(context.Background(), file); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(store, []CredentialSource{testOAuthSource{tokenEndpoint: server.URL}}, oauth.NewClient(server.Client()))
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
	if (*loaded)["test"].RefreshToken != "new-refresh" {
		t.Fatal("rotated refresh token not stored")
	}
}

func TestManagerStoreAPIKeyAndBearerToken(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "auth.json"))
	manager := NewManager(store, nil, nil)

	if err := manager.StoreAPIKey(context.Background(), "openrouter", "  sk-test  "); err != nil {
		t.Fatal(err)
	}
	key, err := manager.APIKey(context.Background(), "openrouter")
	if err != nil {
		t.Fatal(err)
	}
	if key != "sk-test" {
		t.Fatalf("expected trimmed key, got %q", key)
	}
	token, err := manager.BearerToken(context.Background(), "openrouter")
	if err != nil {
		t.Fatal(err)
	}
	if token != "sk-test" {
		t.Fatalf("expected bearer token from api key, got %q", token)
	}
}
