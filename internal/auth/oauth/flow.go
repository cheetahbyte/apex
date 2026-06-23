package oauth

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type Source interface {
	ClientID() string
	Scopes() []string
	RedirectPath() string
	DefaultPort() int
	AuthEndpoint() string
	TokenEndpoint() string
}

type Flow struct {
	Client      *Client
	OpenBrowser bool
}

func NewFlow(client *Client) *Flow {
	if client == nil {
		client = NewClient(nil)
	}
	return &Flow{Client: client, OpenBrowser: true}
}

func (f *Flow) Login(ctx context.Context, source Source) (TokenResponse, string, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return TokenResponse{}, "", err
	}
	state, err := GenerateState()
	if err != nil {
		return TokenResponse{}, "", err
	}
	server, err := StartCallbackServer(source.DefaultPort(), source.RedirectPath())
	if err != nil {
		return TokenResponse{}, "", err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL := BuildAuthorizeURL(source.AuthEndpoint(), source.ClientID(), server.RedirectURI(), source.Scopes(), pkce.Challenge, state)
	if f.OpenBrowser {
		_ = OpenBrowser(authURL)
	}

	result, err := server.Wait(ctx)
	if err != nil {
		return TokenResponse{}, authURL, err
	}
	if result.Err != "" {
		return TokenResponse{}, authURL, fmt.Errorf("oauth callback error: %s", result.Err)
	}
	if result.State != state {
		return TokenResponse{}, authURL, fmt.Errorf("oauth state mismatch")
	}
	if result.Code == "" {
		return TokenResponse{}, authURL, fmt.Errorf("oauth callback missing code")
	}
	tokens, err := f.Client.ExchangeCode(ctx, source.TokenEndpoint(), CodeExchangeRequest{
		GrantType:    "authorization_code",
		ClientID:     source.ClientID(),
		Code:         result.Code,
		CodeVerifier: pkce.Verifier,
		RedirectURI:  server.RedirectURI(),
	})
	return tokens, authURL, err
}

func BuildAuthorizeURL(endpoint, clientID, redirectURI string, scopes []string, challenge, state string) string {
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", clientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("scope", strings.Join(scopes, " "))
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	values.Set("state", state)
	return endpoint + "?" + values.Encode()
}

func OpenBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}
