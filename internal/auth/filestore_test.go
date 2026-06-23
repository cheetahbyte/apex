package auth

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileStoreSaveLoadDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	store := NewFileStore(path)
	ctx := context.Background()

	file := NewAuthFile()
	(*file)["openai"] = SourceAuth{Type: AuthKindOAuth2, AccessToken: "token"}
	if err := store.Save(ctx, file); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("expected 0600, got %o", mode)
		}
	}

	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if (*loaded)["openai"].AccessToken != "token" {
		t.Fatal("stored token not loaded")
	}

	if err := store.Delete(ctx, "openai"); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := (*loaded)["openai"]; ok {
		t.Fatal("credential source not deleted")
	}
}

func TestFileStoreLoadMigratesLegacyProvidersShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	store := NewFileStore(path)
	data := []byte(`{
  "version": 1,
  "active_provider": "openai-codex",
  "providers": {
    "openai-codex": {
      "kind": "oauth2",
      "access_token": "old-access",
      "refresh_token": "old-refresh",
      "expires_at": "2030-01-02T03:04:05Z",
      "claims": {"email":"me@example.com","account_id":"acct_123"}
    }
  }
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	auth := (*loaded)["openai"]
	if auth.Type != AuthKindOAuth2 || auth.AccessToken != "old-access" || auth.RefreshToken != "old-refresh" {
		t.Fatalf("legacy auth not migrated: %+v", auth)
	}
	if auth.Email != "me@example.com" || auth.AccountID != "acct_123" {
		t.Fatalf("legacy claims not flattened: %+v", auth)
	}
}
