package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Store abstracts blob storage so M8 can swap to S3/GCS without touching handlers.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader) (Meta, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type Meta struct {
	Key  string
	Size int64
}

type Local struct {
	Root string
}

func NewLocal(root string) (*Local, error) {
	if root == "" {
		return nil, errors.New("storage root is empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Local{Root: root}, nil
}

func (l *Local) path(key string) string {
	return filepath.Join(l.Root, filepath.FromSlash(key))
}

func (l *Local) Put(_ context.Context, key string, r io.Reader) (Meta, error) {
	p := l.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return Meta{}, err
	}
	f, err := os.Create(p)
	if err != nil {
		return Meta{}, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return Meta{}, err
	}
	return Meta{Key: key, Size: n}, nil
}

func (l *Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(l.path(key))
}

func (l *Local) Delete(_ context.Context, key string) error {
	err := os.Remove(l.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
