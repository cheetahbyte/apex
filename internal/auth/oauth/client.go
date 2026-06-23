package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	http *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{http: httpClient}
}

type CodeExchangeRequest struct {
	GrantType    string
	ClientID     string
	Code         string
	CodeVerifier string
	RedirectURI  string
}

type RefreshRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (c *Client) ExchangeCode(ctx context.Context, endpoint string, req CodeExchangeRequest) (TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", req.GrantType)
	form.Set("client_id", req.ClientID)
	form.Set("code", req.Code)
	form.Set("code_verifier", req.CodeVerifier)
	form.Set("redirect_uri", req.RedirectURI)
	return c.postForm(ctx, endpoint, form)
}

func (c *Client) Refresh(ctx context.Context, endpoint string, req RefreshRequest) (TokenResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TokenResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return TokenResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	return c.doTokenRequest(httpReq)
}

func (c *Client) postForm(ctx context.Context, endpoint string, form url.Values) (TokenResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")
	return c.doTokenRequest(httpReq)
}

func (c *Client) doTokenRequest(req *http.Request) (TokenResponse, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return TokenResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return TokenResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TokenResponse{}, fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, sanitizedErrorBody(body))
	}
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return TokenResponse{}, err
	}
	if tokenResp.AccessToken == "" {
		return TokenResponse{}, fmt.Errorf("token endpoint response missing access_token")
	}
	return tokenResp, nil
}

func sanitizedErrorBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if len(text) > 512 {
		text = text[:512] + "..."
	}
	for _, key := range []string{"access_token", "refresh_token", "id_token", "code", "code_verifier"} {
		text = redactJSONKey(text, key)
	}
	return text
}

func redactJSONKey(text, key string) string {
	re := regexp.MustCompile(`(?i)("` + regexp.QuoteMeta(key) + `"\s*:\s*")[^"]*(")`)
	return re.ReplaceAllString(text, `${1}<redacted>${2}`)
}
