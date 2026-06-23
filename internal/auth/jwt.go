package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type jwtPayload struct {
	Exp     int64  `json:"exp"`
	Email   string `json:"email"`
	Profile struct {
		Email string `json:"email"`
	} `json:"https://api.openai.com/profile"`
	Auth struct {
		ChatGPTAccountID string `json:"chatgpt_account_id"`
		ChatGPTPlanType  string `json:"chatgpt_plan_type"`
		UserID           string `json:"user_id"`
		ChatGPTUserID    string `json:"chatgpt_user_id"`
	} `json:"https://api.openai.com/auth"`
}

func JWTExpiration(token string) (time.Time, bool, error) {
	payload, err := decodeJWTPayload(token)
	if err != nil {
		return time.Time{}, false, err
	}
	if payload.Exp == 0 {
		return time.Time{}, false, nil
	}
	return time.Unix(payload.Exp, 0).UTC(), true, nil
}

func ClaimsFromJWT(token string) (Claims, error) {
	payload, err := decodeJWTPayload(token)
	if err != nil {
		return Claims{}, err
	}
	email := payload.Email
	if email == "" {
		email = payload.Profile.Email
	}
	return Claims{
		Email:     email,
		AccountID: payload.Auth.ChatGPTAccountID,
		PlanType:  payload.Auth.ChatGPTPlanType,
	}, nil
}

func decodeJWTPayload(token string) (jwtPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[1] == "" {
		return jwtPayload{}, fmt.Errorf("invalid JWT format")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtPayload{}, err
	}
	var payload jwtPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return jwtPayload{}, err
	}
	return payload, nil
}
