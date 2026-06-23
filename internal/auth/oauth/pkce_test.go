package oauth

import (
	"net/url"
	"strings"
	"testing"
)

func TestGeneratePKCE(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if len(pkce.Verifier) < 43 || len(pkce.Verifier) > 128 {
		t.Fatalf("verifier length out of range: %d", len(pkce.Verifier))
	}
	if strings.Contains(pkce.Verifier, "=") || strings.Contains(pkce.Challenge, "=") {
		t.Fatal("PKCE values must be unpadded base64url")
	}
	if pkce.Challenge == "" || pkce.Challenge == pkce.Verifier {
		t.Fatal("challenge must be derived from verifier")
	}
}

func TestBuildAuthorizeURL(t *testing.T) {
	raw := BuildAuthorizeURL(
		"https://auth.example.com/oauth/authorize",
		"client_123",
		"http://localhost:1455/auth/callback",
		[]string{"openid", "email", "offline_access"},
		"challenge",
		"state",
	)
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("response_type") != "code" {
		t.Fatal("missing response_type")
	}
	if query.Get("client_id") != "client_123" {
		t.Fatal("missing client_id")
	}
	if query.Get("redirect_uri") != "http://localhost:1455/auth/callback" {
		t.Fatal("missing redirect_uri")
	}
	if query.Get("scope") != "openid email offline_access" {
		t.Fatalf("unexpected scope %q", query.Get("scope"))
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatal("missing S256 challenge method")
	}
	if query.Get("state") != "state" {
		t.Fatal("missing state")
	}
}
