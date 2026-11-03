package objectstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const q = "quarantine/0f8fad5b-d9cb-469f-a165-70867728950e"

func TestFSPutStatCopyDelete(t *testing.T) {
	ctx := context.Background()
	s, err := NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("%PDF-1.4 synthetic receipt")
	sum := sha256.Sum256(body)
	if err := s.Put(ctx, q, bytes.NewReader(body), sum); err != nil {
		t.Fatal(err)
	}
	// Idempotent re-put; a different body under the same key is refused.
	if err := s.Put(ctx, q, bytes.NewReader(body), sum); err != nil {
		t.Fatalf("re-put: %v", err)
	}
	other := sha256.Sum256([]byte("x"))
	if err := s.Put(ctx, q, bytes.NewReader([]byte("x")), other); !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("overwrite: %v", err)
	}
	// A declared hash that does not match the bytes never becomes visible.
	q2 := "quarantine/1f8fad5b-d9cb-469f-a165-70867728950e"
	if err := s.Put(ctx, q2, bytes.NewReader(body), other); !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("mismatch: %v", err)
	}
	if _, err := s.Stat(ctx, q2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("mismatched object visible: %v", err)
	}
	info, err := s.Stat(ctx, q)
	if err != nil || info.SHA256 != sum || info.Size != int64(len(body)) {
		t.Fatalf("stat: %+v %v", info, err)
	}
	ev := "evidence/" + hexOf(sum)
	if err := s.CopyIfAbsent(ctx, q, ev, sum); err != nil {
		t.Fatal(err)
	}
	if err := s.CopyIfAbsent(ctx, q, ev, sum); err != nil {
		t.Fatalf("copy is idempotent: %v", err)
	}
	if err := s.Delete(ctx, q); err != nil {
		t.Fatal(err)
	}
	// The evidence copy is enough once the quarantine object is gone.
	if err := s.CopyIfAbsent(ctx, q, ev, sum); err != nil {
		t.Fatalf("copy after quarantine purge: %v", err)
	}
	if err := s.Delete(ctx, q); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
}

func TestFSRejectsClientPaths(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, _ := NewFS(root)
	for _, k := range []string{"../etc/passwd", "/etc/passwd", "quarantine/../../x", "evidence/abc", "receipts/a.pdf", ""} {
		if _, err := s.Stat(ctx, k); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("%q accepted: %v", k, err)
		}
	}
	if _, err := NewFS("relative"); err == nil {
		t.Fatal("relative root accepted")
	}
	if _, err := NewFS(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing root accepted")
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatal("rejected keys touched the store")
	}
}

func hexOf(b [32]byte) string {
	const h = "0123456789abcdef"
	out := make([]byte, 64)
	for i, v := range b {
		out[2*i], out[2*i+1] = h[v>>4], h[v&15]
	}
	return string(out)
}
