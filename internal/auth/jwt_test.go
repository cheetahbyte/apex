package auth

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestJWTExpirationAndClaims(t *testing.T) {
	payload := `{"exp":1893456000,"https://api.openai.com/profile":{"email":"a@example.com"},"https://api.openai.com/auth":{"chatgpt_account_id":"acc","chatgpt_plan_type":"plus"}}`
	token := "e30." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".sig"
	expires, ok, err := JWTExpiration(token)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !expires.Equal(time.Unix(1893456000, 0).UTC()) {
		t.Fatalf("unexpected expiration %v ok=%v", expires, ok)
	}
	claims, err := ClaimsFromJWT(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Email != "a@example.com" || claims.AccountID != "acc" || claims.PlanType != "plus" {
		t.Fatalf("unexpected claims %+v", claims)
	}
}

func TestClaimsFromJWTAccountIDVariants(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		want    string
	}{
		{name: "top-level", payload: `{"chatgpt_account_id":"acct_top"}`, want: "acct_top"},
		{name: "auth-claim", payload: `{"https://api.openai.com/auth":{"chatgpt_account_id":"acct_auth"}}`, want: "acct_auth"},
		{name: "organizations", payload: `{"organizations":[{"id":"acct_org"}]}`, want: "acct_org"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := "e30." + base64.RawURLEncoding.EncodeToString([]byte(tc.payload)) + ".sig"
			claims, err := ClaimsFromJWT(token)
			if err != nil {
				t.Fatal(err)
			}
			if claims.AccountID != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, claims.AccountID)
			}
		})
	}
}
