package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetenv(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "db_url")
	if err := os.WriteFile(ok, []byte("postgres://synthetic\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	open := filepath.Join(dir, "world")
	if err := os.WriteFile(open, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"A": "direct", "A_FILE": ok, "B_FILE": ok, "C_FILE": open, "D_FILE": filepath.Join(dir, "missing"), "E_FILE": dir}
	g := Getenv(func(k string) string { return env[k] })
	if g("A") != "direct" || g("B") != "postgres://synthetic" || g("C") != "" || g("D") != "" || g("E") != "" || g("F") != "" {
		t.Fatalf("A=%q B=%q C=%q D=%q E=%q", g("A"), g("B"), g("C"), g("D"), g("E"))
	}
}
