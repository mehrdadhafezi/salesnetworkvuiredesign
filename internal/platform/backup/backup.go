// Package backup is the R3 logical backup / restore toolchain for
// DISPOSABLE staging and development databases and their evidence stores.
//
// What it is and is not:
//
//   - A logical backup (pg_dump custom format) taken from ONE exported
//     snapshot: the row counts, checksums and inventory recorded in the
//     manifest describe exactly the state that was dumped.
//   - Not a physical backup and not point-in-time recovery: no WAL is
//     archived. Data written after the snapshot is not in the backup.
//   - Never for production: the environment must be development or staging,
//     declared explicitly and attested by the database (the
//     worker.environment.<env> flags introduced in R2).
//
// Safety rules (enforced, tested):
//
//   - destinations must lie under an explicit allowlist, must not exist, are
//     written as "<dest>.partial" and renamed only when complete; nothing is
//     ever overwritten or deleted except tool-created, marker-carrying
//     disposable restores (Discard);
//   - restores go only into a NEW database named crm_restore_*, created and
//     marked by the tool; an existing database is refused;
//   - credentials reach pg_dump / pg_restore only through PG* environment
//     variables of the child process, never through arguments, logs,
//     manifests or reports;
//   - backups are encrypted with age unless the environment is development
//     and plaintext was explicitly allowed;
//   - a restored database is QUARANTINED: every feature flag (including the
//     environment attestation and the worker job switches) is set to false,
//     so no worker or business-write surface can run on it until an operator
//     re-validates and re-attests it;
//   - evidence bytes are verified against the restored receipt rows; a
//     missing or corrupt object fails verification and is never "fixed" by
//     changing a receipt's state.
package backup

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// FormatVersion is the manifest format.
const FormatVersion = 1

// Errors.
var (
	ErrRefused   = errors.New("backup: refused")
	ErrIntegrity = errors.New("backup: integrity check failed")
	ErrMismatch  = errors.New("backup: restored state does not match the backup")
)

// Environments a backup may be taken from.
const (
	Development = "development"
	Staging     = "staging"
)

var (
	restoreName = regexp.MustCompile(`^crm_restore_[a-z0-9_]{3,40}$`)
	markerRe    = regexp.MustCompile(`^crm-backup:restore:[0-9a-f]{16}:(INCOMPLETE|COMPLETE|FAILED)$`)
)

// Tools locates the PostgreSQL client and age binaries.
type Tools struct {
	PgBinDir string // empty: PATH
	Age      string // empty: "age" on PATH
}

func (t Tools) pg(name string) string {
	if t.PgBinDir == "" {
		return name
	}
	return filepath.Join(t.PgBinDir, name)
}

func (t Tools) age() string {
	if t.Age == "" {
		return "age"
	}
	return t.Age
}

// pgEnv turns a DSN into PG* variables for a child process (no DSN, user
// or password ever appears on a command line).
func pgEnv(dsn string) ([]string, string, error) {
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		return nil, "", fmt.Errorf("%w: a postgres:// URL is required", ErrRefused)
	}
	db := strings.TrimPrefix(u.Path, "/")
	if db == "" {
		return nil, "", fmt.Errorf("%w: the URL must name a database", ErrRefused)
	}
	env := []string{"PGHOST=" + u.Hostname(), "PGDATABASE=" + db, "PGCONNECT_TIMEOUT=10"}
	if p := u.Port(); p != "" {
		env = append(env, "PGPORT="+p)
	}
	if u.User != nil {
		env = append(env, "PGUSER="+u.User.Username())
		if pw, ok := u.User.Password(); ok {
			env = append(env, "PGPASSWORD="+pw)
		}
	}
	if m := u.Query().Get("sslmode"); m != "" {
		env = append(env, "PGSSLMODE="+m)
	}
	// Only the PG* variables: the child inherits no other secret.
	base := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.TempDir(), "LC_ALL=C"}
	return append(base, env...), db, nil
}

// withDatabase returns dsn pointing at database name.
func withDatabase(dsn, name string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("%w: invalid URL", ErrRefused)
	}
	u.Path = "/" + name
	return u.String(), nil
}

// underAllowed checks that path (cleaned, its existing parent resolved)
// lies under one of the absolute allowlisted roots.
func underAllowed(path string, roots []string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: path must be absolute", ErrRefused)
	}
	clean := filepath.Clean(path)
	parent, err := filepath.EvalSymlinks(filepath.Dir(clean))
	if err != nil {
		return "", fmt.Errorf("%w: parent directory must exist", ErrRefused)
	}
	resolved := filepath.Join(parent, filepath.Base(clean))
	for _, r := range roots {
		if !filepath.IsAbs(r) {
			continue
		}
		rr, err := filepath.EvalSymlinks(filepath.Clean(r))
		if err != nil {
			continue
		}
		if rel, err := filepath.Rel(rr, resolved); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("%w: %s is not under an allowed root", ErrRefused, filepath.Base(clean))
}

func mustNotExist(paths ...string) error {
	for _, p := range paths {
		if _, err := os.Lstat(p); err == nil {
			return fmt.Errorf("%w: %s already exists (never overwritten)", ErrRefused, filepath.Base(p))
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: cannot inspect %s", ErrRefused, filepath.Base(p))
		}
	}
	return nil
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// redact keeps diagnostic output short and free of anything that looks like
// a credential.
var secretish = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token)\S*|postgres(ql)?://\S+`)

func redact(s string) string {
	s = secretish.ReplaceAllString(s, "[redacted]")
	if len(s) > 400 {
		s = s[:400]
	}
	return strings.TrimSpace(s)
}

// attest checks that the database attests exactly env.
func attest(ctx context.Context, tx pgx.Tx, env string) error {
	rows, err := tx.Query(ctx, `SELECT flag_key FROM platform_featureflags.feature_flag WHERE enabled AND flag_key LIKE 'worker.environment.%'`)
	if err != nil {
		return fmt.Errorf("%w: environment attestation unreadable", ErrRefused)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return err
		}
		got = append(got, k)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(got) != 1 || got[0] != "worker.environment."+env {
		return fmt.Errorf("%w: the database does not attest exactly %q (found %v)", ErrRefused, env, got)
	}
	return nil
}

func sha256File(p string) (string, int64, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func writeJSON(p string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func since(t time.Time) float64 { return float64(time.Since(t).Milliseconds()) / 1000 }

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// runCmd runs a child process with env, streaming stdin/stdout; stderr is
// captured and redacted into the error.
func runCmd(ctx context.Context, name string, args, env []string, stdin io.Reader, stdout io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	var stderr strings.Builder
	cmd.Stderr = &limited{w: &stderr, n: 4096}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %s", filepath.Base(name), redact(stderr.String()))
	}
	return nil
}

type limited struct {
	w io.Writer
	n int
}

func (l *limited) Write(p []byte) (int, error) {
	if l.n <= 0 {
		return len(p), nil
	}
	q := p
	if len(q) > l.n {
		q = q[:l.n]
	}
	l.n -= len(q)
	_, _ = l.w.Write(q)
	return len(p), nil
}
