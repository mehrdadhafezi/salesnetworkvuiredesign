package backup

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// R3: pg_restore --list may exit after reading only the table of contents.
// The plaintext digest must still cover every byte of the dump, so a valid
// backup larger than the subprocess read-ahead verifies.
func TestVerifyHashesBytesPgRestoreDidNotRead(t *testing.T) {
	bin := t.TempDir()
	// A pg_restore that reads 4 KiB and exits successfully.
	script := "#!/bin/sh\nhead -c 4096 >/dev/null\nexit 0\n"
	if err := os.WriteFile(filepath.Join(bin, "pg_restore"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	dump := make([]byte, 8<<20)
	if _, err := rand.Read(dump); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "database.dump"), dump, 0o600); err != nil {
		t.Fatal(err)
	}
	fileSHA, size, err := sha256File(filepath.Join(dir, "database.dump"))
	if err != nil {
		t.Fatal(err)
	}
	plain := sha256.Sum256(dump)
	m := Manifest{Format: FormatVersion, Complete: true, ID: "r3-verify-test",
		Database: FileEntry{Name: "database.dump", SHA256: fileSHA, Size: size}, PlainSHA256: hex.EncodeToString(plain[:])}
	if err := writeJSON(filepath.Join(dir, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	ms, _, err := sha256File(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.sha256"), []byte(ms+"  manifest.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Verify(context.Background(), dir, "", Tools{PgBinDir: bin})
	if err != nil {
		t.Fatalf("valid backup refused: %v", err)
	}
	if !r.DatabaseOK {
		t.Fatal("database not reported OK")
	}
}
