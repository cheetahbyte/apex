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
	file.ActiveProvider = "openai-codex"
	file.Providers["openai-codex"] = ProviderAuth{Kind: AuthKindOAuth2, AccessToken: "token"}
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
	if loaded.Providers["openai-codex"].AccessToken != "token" {
		t.Fatal("stored token not loaded")
	}

	if err := store.Delete(ctx, "openai-codex"); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Providers["openai-codex"]; ok {
		t.Fatal("provider not deleted")
	}
}
