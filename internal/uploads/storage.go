package uploads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrInvalidPath is returned for a relative path that would leave the storage root.
var ErrInvalidPath = errors.New("invalid file path")

// Storage is where uploaded bytes live. LocalStorage keeps them on disk; an S3 or GCS
// implementation only has to satisfy this interface and be set on UploadConfig.Storage.
type Storage interface {
	Save(ctx context.Context, relativePath string, content io.Reader) error
	Delete(ctx context.Context, relativePath string) error
	// URL is the public address of a stored file.
	URL(relativePath string) string
}

// LocalStorage stores files under Root and serves them from BaseURL.
type LocalStorage struct {
	Root    string
	BaseURL string
}

func NewLocalStorage(root, baseURL string) *LocalStorage {
	return &LocalStorage{Root: root, BaseURL: baseURL}
}

// Resolve maps a relative path to an absolute one that is guaranteed to stay inside
// Root, so "../" segments cannot escape it.
func (s *LocalStorage) Resolve(relativePath string) (string, error) {
	root := filepath.Clean(s.Root)
	full := filepath.Join(root, filepath.Clean("/"+filepath.ToSlash(relativePath)))

	if full == root || !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", ErrInvalidPath
	}
	return full, nil
}

func (s *LocalStorage) Save(_ context.Context, relativePath string, content io.Reader) error {
	full, err := s.Resolve(relativePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	dst, err := os.Create(full)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	if _, err := io.Copy(dst, content); err != nil {
		dst.Close()
		os.Remove(full)
		return fmt.Errorf("write file: %w", err)
	}
	if err := dst.Close(); err != nil {
		os.Remove(full)
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}

// Delete removes a file. A file that is already gone is not an error.
func (s *LocalStorage) Delete(_ context.Context, relativePath string) error {
	full, err := s.Resolve(relativePath)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func (s *LocalStorage) URL(relativePath string) string {
	rel := strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(relativePath)), "/")
	if rel == "" {
		return ""
	}
	return strings.TrimRight(s.BaseURL, "/") + "/" + rel
}
