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
