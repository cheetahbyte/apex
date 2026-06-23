package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

	var file AuthFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if file.Version == 0 {
		file.Version = 1
	}
	if file.Providers == nil {
		file.Providers = map[ProviderID]ProviderAuth{}
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
	if file.Version == 0 {
		file.Version = 1
	}
	if file.Providers == nil {
		file.Providers = map[ProviderID]ProviderAuth{}
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

func (s *FileStore) Delete(ctx context.Context, providerID ProviderID) error {
	file, err := s.Load(ctx)
	if err != nil {
		return err
	}
	delete(file.Providers, providerID)
	if file.ActiveProvider == providerID {
		file.ActiveProvider = ""
	}
	return s.Save(ctx, file)
}
