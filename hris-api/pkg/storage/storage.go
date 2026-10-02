// Package storage keeps uploaded file contents. Local disk is the only
// backend for now; keys are opaque random names, never user input.
package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type Storage interface {
	Put(ctx context.Context, key string, r io.Reader) (int64, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

var validKey = regexp.MustCompile(`^[a-f0-9]{32,64}$`)

type Local struct{ Dir string }

func (l Local) path(key string) (string, error) {
	if !validKey.MatchString(key) {
		return "", errors.New("invalid storage key")
	}
	return filepath.Join(l.Dir, key[:2], key), nil
}

func (l Local) Put(_ context.Context, key string, r io.Reader) (int64, error) {
	p, err := l.path(key)
	if err != nil {
		return 0, err
	}
	if err = os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return 0, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(p)
	}
	return n, err
}

func (l Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := l.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p) // #nosec G304 -- key is validated hex, joined under Dir
}

func (l Local) Delete(_ context.Context, key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	return os.Remove(p)
}
