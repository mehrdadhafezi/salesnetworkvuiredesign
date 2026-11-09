package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// R3: the plaintext (development-only) sink is closed BEFORE wait() by every
// caller; closing must sync and succeed, and wait must not fail afterwards.
func TestPlaintextSinkCloseThenWait(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.dump")
	sk, err := openSink(context.Background(), path, "", Tools{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sk.w.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := sk.w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := sk.wait(); err != nil {
		t.Fatalf("wait after close: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "payload" {
		t.Fatalf("content = %q, %v", got, err)
	}
}
