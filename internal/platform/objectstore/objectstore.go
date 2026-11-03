// Package objectstore is the durable object storage port used for payment
// evidence (Architecture V1 §15.1: quarantine and evidence storage share one
// durability class). Keys are server-generated only: a key is a prefix and a
// hex / uuid name, so no client value can ever become a path. The filesystem
// implementation writes to a temporary file, fsyncs it, verifies the content
// hash, renames it into place and fsyncs the directory before reporting
// success: a returned nil means the bytes are durable.
package objectstore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var (
	// ErrNotFound reports a missing object.
	ErrNotFound = errors.New("objectstore: not found")
	// ErrInvalidKey reports a key outside the server-generated grammar.
	ErrInvalidKey = errors.New("objectstore: invalid key")
	// ErrHashMismatch reports bytes whose SHA-256 differs from the expected one.
	ErrHashMismatch = errors.New("objectstore: content hash mismatch")
)

var keyPattern = regexp.MustCompile(`^(quarantine/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|evidence/[0-9a-f]{64})$`)

// ValidKey reports whether key is in the server-generated grammar.
func ValidKey(key string) bool { return keyPattern.MatchString(key) }

// Info describes a stored object; SHA256 is computed from the stored bytes.
type Info struct {
	Size   int64
	SHA256 [32]byte
}

// Store is the storage port.
type Store interface {
	// Put stores exactly the bytes of r under key, verifying their SHA-256
	// equals want. An existing object with the same hash is success (an
	// idempotent re-put); a different existing object is ErrHashMismatch.
	Put(ctx context.Context, key string, r io.Reader, want [32]byte) error
	// Stat hashes the stored object; ErrNotFound when absent.
	Stat(ctx context.Context, key string) (Info, error)
	// CopyIfAbsent copies src to dst unless dst already exists. The copy is
	// verified against want before it is made visible.
	CopyIfAbsent(ctx context.Context, src, dst string, want [32]byte) error
	// Delete removes an object; deleting a missing object is not an error.
	Delete(ctx context.Context, key string) error
}

// FS is the filesystem implementation.
type FS struct{ root string }

// NewFS returns a store rooted at an existing absolute directory.
func NewFS(root string) (*FS, error) {
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("objectstore: root must be absolute")
	}
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("objectstore: root is not a directory")
	}
	return &FS{root: filepath.Clean(root)}, nil
}

func (f *FS) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", ErrInvalidKey
	}
	return filepath.Join(f.root, filepath.FromSlash(key)), nil
}

func hashFile(p string) (Info, error) {
	fh, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return Info{}, ErrNotFound
	}
	if err != nil {
		return Info{}, err
	}
	defer fh.Close()
	h := sha256.New()
	n, err := io.Copy(h, fh)
	if err != nil {
		return Info{}, err
	}
	var out Info
	out.Size = n
	copy(out.SHA256[:], h.Sum(nil))
	return out, nil
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (f *FS) write(dst string, r io.Reader, want [32]byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	if info, err := hashFile(dst); err == nil {
		if info.SHA256 != want {
			return ErrHashMismatch
		}
		return nil
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), r); err != nil {
		return err
	}
	var got [32]byte
	copy(got[:], h.Sum(nil))
	if got != want {
		return ErrHashMismatch
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return err
	}
	ok = true
	return syncDir(filepath.Dir(dst))
}

// Put implements Store.
func (f *FS) Put(ctx context.Context, key string, r io.Reader, want [32]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := f.path(key)
	if err != nil {
		return err
	}
	return f.write(p, r, want)
}

// Stat implements Store.
func (f *FS) Stat(ctx context.Context, key string) (Info, error) {
	if err := ctx.Err(); err != nil {
		return Info{}, err
	}
	p, err := f.path(key)
	if err != nil {
		return Info{}, err
	}
	return hashFile(p)
}

// CopyIfAbsent implements Store.
func (f *FS) CopyIfAbsent(ctx context.Context, src, dst string, want [32]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sp, err := f.path(src)
	if err != nil {
		return err
	}
	dp, err := f.path(dst)
	if err != nil {
		return err
	}
	in, err := os.Open(sp)
	if errors.Is(err, os.ErrNotExist) {
		if info, statErr := hashFile(dp); statErr == nil && info.SHA256 == want {
			return nil
		}
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	defer in.Close()
	return f.write(dp, in, want)
}

// Delete implements Store.
func (f *FS) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return syncDir(filepath.Dir(p))
}
