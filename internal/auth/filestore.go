package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const authFileName = "auth.json"

type FileStore struct {
	path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func DefaultAuthPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "apex", authFileName), nil
}

func DefaultFileStore() (*FileStore, error) {
	path, err := DefaultAuthPath()
	if err != nil {
		return nil, err
	}
	return NewFileStore(path), nil
}

func (s *FileStore) Path() string { return s.path }

func (s *FileStore) Load(ctx context.Context) (*AuthFile, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return NewAuthFile(), nil
	}
	if err != nil {
		return nil, err
	}

	file, err := decodeAuthFile(data)
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (s *FileStore) Save(ctx context.Context, file *AuthFile) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if file == nil {
		return fmt.Errorf("auth file is nil")
	}
	if *file == nil {
		*file = AuthFile{}
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".auth-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if runtime.GOOS != "windows" {
		if err := tmp.Chmod(0o600); err != nil {
			_ = tmp.Close()
			return err
		}
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}

func (s *FileStore) Delete(ctx context.Context, sourceID CredentialSourceID) error {
	file, err := s.Load(ctx)
	if err != nil {
		return err
	}
	delete(*file, canonicalSourceID(sourceID))
	return s.Save(ctx, file)
}

func decodeAuthFile(data []byte) (AuthFile, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if providersRaw, ok := raw["providers"]; ok {
		return decodeLegacyAuthFile(providersRaw)
	}

	var file AuthFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if file == nil {
		file = AuthFile{}
	}
	return file, nil
}

func decodeLegacyAuthFile(providersRaw json.RawMessage) (AuthFile, error) {
	var legacy map[CredentialSourceID]legacySourceAuth
	if err := json.Unmarshal(providersRaw, &legacy); err != nil {
		return nil, err
	}
	file := AuthFile{}
	for id, auth := range legacy {
		file[canonicalSourceID(id)] = auth.toSourceAuth()
	}
	return file, nil
}

type legacySourceAuth struct {
	Kind         AuthKind  `json:"kind"`
	Issuer       string    `json:"issuer,omitempty"`
	ClientID     string    `json:"client_id,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	LastRefresh  time.Time `json:"last_refresh,omitempty"`
	Claims       Claims    `json:"claims,omitempty"`
}

func (a legacySourceAuth) toSourceAuth() SourceAuth {
	authKind := a.Kind
	switch authKind {
	case "oauth2":
		authKind = AuthKindOAuth2
	case "api_key":
		authKind = AuthKindAPIKey
	}
	return SourceAuth{
		Type:         authKind,
		AccessToken:  a.AccessToken,
		RefreshToken: a.RefreshToken,
		IDToken:      a.IDToken,
		Expires:      unixTime(a.ExpiresAt),
		LastRefresh:  unixTime(a.LastRefresh),
		AccountID:    a.Claims.AccountID,
		Email:        a.Claims.Email,
		PlanType:     a.Claims.PlanType,
		Issuer:       a.Issuer,
		ClientID:     a.ClientID,
	}
}

func canonicalSourceID(id CredentialSourceID) CredentialSourceID {
	switch id {
	case "openai", "openai-codex", "chatgpt":
		return "codex"
	}
	return id
}
