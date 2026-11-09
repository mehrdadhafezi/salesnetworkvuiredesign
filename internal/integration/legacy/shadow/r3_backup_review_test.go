package shadow_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/backup"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// R3 review remediation (PR #43, Codex findings P1-1 and P2-1).
//
// These run real pg_dump / pg_restore / age against PostgreSQL with a
// database large enough that `pg_restore --list` stops reading long before
// the end of the dump (it only needs the table of contents). Requires
// CRM_TEST_BACKUP=1, like the other R3 rehearsal tests.

// addBulk adds ~9.6 MB of incompressible synthetic bytes to the source.
func addBulk(t *testing.T, e *r3) {
	t.Helper()
	if _, err := e.admin.Exec(context.Background(), `CREATE SCHEMA r3_bulk;
		CREATE TABLE r3_bulk.blob (id int PRIMARY KEY, b bytea NOT NULL);
		INSERT INTO r3_bulk.blob SELECT g, decode(md5(random()::text) || md5(random()::text) || md5(random()::text) || md5(random()::text), 'hex')
		FROM generate_series(1, 150000) g`); err != nil {
		t.Fatal(err)
	}
}

// attestOnly makes the source attest exactly env (as an operator would).
func attestOnly(t *testing.T, e *r3, env string) {
	t.Helper()
	if _, err := e.admin.Exec(context.Background(), `UPDATE platform_featureflags.feature_flag SET enabled = false WHERE flag_key LIKE 'worker.environment.%';
		INSERT INTO platform_featureflags.feature_flag (flag_key, enabled, updated_at) VALUES ('worker.environment.`+env+`', true, now())
		ON CONFLICT (flag_key) DO UPDATE SET enabled = true`); err != nil {
		t.Fatal(err)
	}
}

// reseal rewrites the backup's manifest so its FILE digests match the
// (tampered) files: only the decrypted-content digest can then catch it.
func reseal(t *testing.T, dir string) {
	t.Helper()
	m, err := backup.ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.Database.SHA256, m.Database.Size = fileSHA(t, filepath.Join(dir, m.Database.Name))
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	sum, _ := fileSHA(t, filepath.Join(dir, "manifest.json"))
	if err := os.WriteFile(filepath.Join(dir, "manifest.sha256"), []byte(sum+"  manifest.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func fileSHA(t *testing.T, p string) (string, int64) {
	t.Helper()
	out, err := exec.Command("sha256sum", p).Output()
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(out))[0], fi.Size()
}

// flipTail changes one byte near the END of b: a region pg_restore --list
// never reads.
func flipTail(b []byte) []byte {
	b[len(b)-16] ^= 0xff
	return b
}

func restoredBulk(t *testing.T, target string) (n int, sum string) {
	t.Helper()
	ctx := context.Background()
	rc, err := pgx.Connect(ctx, dsnFor(t, pgtest.AdminURL(t), target))
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close(ctx)
	if err := rc.QueryRow(ctx, `SELECT count(*), md5(string_agg(md5(b), '' ORDER BY id)) FROM r3_bulk.blob`).Scan(&n, &sum); err != nil {
		t.Fatal(err)
	}
	return n, sum
}

// P1-1: a valid large ENCRYPTED backup verifies and restores, and a change
// in the tail of the decrypted dump is rejected.
func TestR3LargeEncryptedBackupVerifiesRestoresAndRejectsTailTamper(t *testing.T) {
	e := setupR3(t)
	addBulk(t, e)
	ctx := context.Background()
	var wantN int
	var wantSum string
	if err := e.admin.QueryRow(ctx, `SELECT count(*), md5(string_agg(md5(b), '' ORDER BY id)) FROM r3_bulk.blob`).Scan(&wantN, &wantSum); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(e.roots, "large-encrypted")
	m, err := backup.Backup(ctx, e.cfg(dir))
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if !m.Encrypted || m.Database.Size < 8<<20 {
		t.Fatalf("expected a large encrypted dump, got encrypted=%v size=%d", m.Encrypted, m.Database.Size)
	}
	if _, err := backup.Verify(ctx, dir, e.keyFile, backup.Tools{}); err != nil {
		t.Fatalf("a valid large backup must verify: %v", err)
	}
	target := restoreName(t)
	discard(t, e, target)
	rep, err := backup.Restore(ctx, e.restoreCfg(t, dir, target))
	if err != nil || rep.Status != "COMPLETE" || len(rep.Differences) != 0 {
		t.Fatalf("a valid large backup must restore: %+v %v", rep, err)
	}
	if n, sum := restoredBulk(t, target); n != wantN || sum != wantSum {
		t.Fatalf("restored bulk rows %d/%s, want %d/%s", n, sum, wantN, wantSum)
	}

	// Tail tamper: decrypt, change a byte near the end, re-encrypt to the
	// same recipients and reseal the file digests. Only the digest over
	// EVERY decrypted byte can detect it.
	bad := filepath.Join(e.roots, "large-tail-tamper")
	if err := os.CopyFS(bad, os.DirFS(dir)); err != nil {
		t.Fatal(err)
	}
	enc := filepath.Join(bad, m.Database.Name)
	plain, err := exec.Command("age", "-d", "-i", e.keyFile, enc).Output()
	if err != nil {
		t.Fatal(err)
	}
	pf := filepath.Join(t.TempDir(), "plain.dump")
	if err := os.WriteFile(pf, flipTail(plain), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(enc); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("age", "-R", e.recipients, "-o", enc, pf).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	reseal(t, bad)
	if _, err := backup.Verify(ctx, bad, e.keyFile, backup.Tools{}); !errors.Is(err, backup.ErrIntegrity) ||
		!strings.Contains(err.Error(), "decrypted database digest") {
		t.Fatalf("a tampered tail must be rejected by the content digest: %v", err)
	}
	t2 := restoreName(t)
	discard(t, e, t2)
	if _, err := backup.Restore(ctx, e.restoreCfg(t, bad, t2)); !errors.Is(err, backup.ErrIntegrity) {
		t.Fatalf("restore of a tampered backup must be refused: %v", err)
	}
}

// P1-1 + P2-1: the explicitly permitted development PLAINTEXT path backs up,
// verifies and restores a large database, and rejects a tampered tail.
func TestR3DevelopmentPlaintextLargeBackupRoundTrip(t *testing.T) {
	e := setupR3(t)
	addBulk(t, e)
	attestOnly(t, e, backup.Development)
	ctx := context.Background()
	cfg := e.cfg(filepath.Join(e.roots, "dev-plain"))
	cfg.Environment, cfg.Recipients, cfg.AllowPlaintext = backup.Development, "", true
	m, err := backup.Backup(ctx, cfg)
	if err != nil {
		t.Fatalf("development plaintext backup: %v", err)
	}
	if m.Encrypted || m.Database.Name != "database.dump" || m.Database.Size < 8<<20 || !m.Complete {
		t.Fatalf("manifest: %+v", m.Database)
	}
	if _, err := os.Stat(cfg.Dest + ".partial"); !os.IsNotExist(err) {
		t.Fatal("a completed backup must not leave a .partial directory")
	}
	if _, err := backup.Verify(ctx, cfg.Dest, "", backup.Tools{}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	target := restoreName(t)
	discard(t, e, target)
	rc := e.restoreCfg(t, cfg.Dest, target)
	rc.Identity = ""
	rep, err := backup.Restore(ctx, rc)
	if err != nil || rep.Status != "COMPLETE" || len(rep.Differences) != 0 {
		t.Fatalf("restore: %+v %v", rep, err)
	}
	if strings.Join(rep.QuarantinedFlags, ",") != "worker.environment.development,worker.job.receipt_finalise" {
		t.Fatalf("quarantine: %v", rep.QuarantinedFlags)
	}
	bad := filepath.Join(e.roots, "dev-plain-tamper")
	if err := os.CopyFS(bad, os.DirFS(cfg.Dest)); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(bad, "database.dump")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, flipTail(b), 0o600); err != nil {
		t.Fatal(err)
	}
	reseal(t, bad)
	if _, err := backup.Verify(ctx, bad, "", backup.Tools{}); !errors.Is(err, backup.ErrIntegrity) ||
		!strings.Contains(err.Error(), "decrypted database digest") {
		t.Fatalf("a tampered tail must be rejected: %v", err)
	}
}

// P2-1: a backup whose dump or encryption fails never publishes a
// manifest; the name stays blocked by the .partial directory.
func TestR3FailedBackupPublishesNoManifest(t *testing.T) {
	e := setupR3(t)
	ctx := context.Background()
	bin := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(bin, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o700); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// A pg_dump that streams 256 KiB and then fails.
	write("pg_dump", `[ "$1" = --version ] && { echo "pg_dump (fake) 0"; exit 0; }
head -c 262144 /dev/zero; exit 1
`)
	// An age that consumes its input and then fails.
	failingAge := write("age-fail", "cat >/dev/null; exit 1\n")

	check := func(name string, cfg backup.BackupConfig) {
		t.Helper()
		if _, err := backup.Backup(ctx, cfg); err == nil {
			t.Fatalf("%s: a failed backup must return an error", name)
		}
		if _, err := os.Stat(cfg.Dest); !os.IsNotExist(err) {
			t.Fatalf("%s: a failed backup must not publish %s", name, cfg.Dest)
		}
		if _, err := os.Stat(filepath.Join(cfg.Dest+".partial", "manifest.json")); !os.IsNotExist(err) {
			t.Fatalf("%s: a failed backup must not write a manifest", name)
		}
		if _, err := backup.Verify(ctx, cfg.Dest+".partial", e.keyFile, backup.Tools{}); !errors.Is(err, backup.ErrIntegrity) {
			t.Fatalf("%s: the partial directory must never verify: %v", name, err)
		}
		if _, err := backup.Backup(ctx, cfg); !errors.Is(err, backup.ErrRefused) {
			t.Fatalf("%s: the name must stay blocked by the .partial directory: %v", name, err)
		}
	}
	enc := e.cfg(filepath.Join(e.roots, "fail-dump-encrypted"))
	enc.Tools = backup.Tools{PgBinDir: bin}
	check("encrypted, pg_dump fails", enc)
	age := e.cfg(filepath.Join(e.roots, "fail-age"))
	age.Tools = backup.Tools{Age: failingAge}
	check("age fails", age)
	attestOnly(t, e, backup.Development)
	plain := e.cfg(filepath.Join(e.roots, "fail-dump-plain"))
	plain.Environment, plain.Recipients, plain.AllowPlaintext = backup.Development, "", true
	plain.Tools = backup.Tools{PgBinDir: bin}
	check("plaintext, pg_dump fails", plain)
}
