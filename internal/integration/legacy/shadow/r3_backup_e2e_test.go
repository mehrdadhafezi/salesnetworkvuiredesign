package shadow_test

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	paymentscommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/command"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/backup"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/objectstore"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// R3 backup / restore rehearsal over PostgreSQL with multi-module SYNTHETIC
// data (identity, organization, billing, payments, integration write-back
// incl. UNKNOWN, outbox / inbox, feature flags) and real evidence objects
// on a temporary filesystem store. Requires CRM_TEST_BACKUP=1 (pg_dump /
// pg_restore of the server's major version and age on PATH).

type r3 struct {
	*r2
	keyFile, recipients, roots string
}

func requireBackupTools(t *testing.T) {
	t.Helper()
	if os.Getenv("CRM_TEST_BACKUP") != "1" {
		t.Skip("CRM_TEST_BACKUP not set; skipping the backup / restore rehearsal")
	}
	for _, b := range []string{"pg_dump", "pg_restore", "age", "age-keygen"} {
		if _, err := exec.LookPath(b); err != nil {
			t.Fatalf("CRM_TEST_BACKUP=1 requires %s on PATH", b)
		}
	}
}

func setupR3(t *testing.T) *r3 {
	t.Helper()
	requireBackupTools(t)
	e := &r3{r2: setupR2(t), roots: t.TempDir()}
	ctx := context.Background()
	keys := t.TempDir()
	e.keyFile, e.recipients = filepath.Join(keys, "key.txt"), filepath.Join(keys, "recipients.txt")
	if out, err := exec.Command("age-keygen", "-o", e.keyFile).CombinedOutput(); err != nil {
		t.Fatalf("age-keygen: %v %s", err, out)
	}
	pub, err := exec.Command("age-keygen", "-y", e.keyFile).Output()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.recipients, pub, 0o600); err != nil {
		t.Fatal(err)
	}
	// Synthetic state across the lifecycle: READY receipts, a STORAGE_ERROR
	// receipt, an UNBOUND upload, write-back commands incl. UNKNOWN.
	seller := e.actor("6")
	a, b := e.upload(t, "6", 61), e.upload(t, "6", 62)
	e.submit(t, batch("r3-key-0001", e.invoices["3001"], 0, seller, item("a", 600, a), item("b", 400, b)))
	e.pass(t, "receipt_finalise")
	c := e.upload(t, "6", 63)
	e.submit(t, batch("r3-key-0002", e.invoices["3005"], 0, seller, item("c", 100, c)))
	e.store.failCopy = true
	e.pass(t, "receipt_finalise")
	e.upload(t, "6", 64) // stays UNBOUND
	e.clock.add(time.Minute)
	e.pass(t, "writeback_intake_payments")
	e.switchOn.Store(true)
	e.legacy.Inject("payment", legacytest.TimeoutAfterCommit)
	e.pass(t, "writeback_deliver")
	// Attest the source as staging (deliberately, as an operator would).
	if _, err := e.admin.Exec(ctx, `INSERT INTO platform_featureflags.feature_flag (flag_key, enabled, updated_at) VALUES
		('worker.environment.staging', true, now()), ('worker.job.receipt_finalise', true, now())`); err != nil {
		t.Fatal(err)
	}
	if e.count(t, `SELECT count(*) FROM integration.writeback_command WHERE state = 'UNKNOWN'`) == 0 ||
		e.count(t, `SELECT count(*) FROM payments.receipt_evidence WHERE state = 'STORAGE_ERROR'`) != 1 ||
		e.count(t, `SELECT count(*) FROM payments.receipt_evidence WHERE state = 'READY'`) != 2 {
		t.Fatal("synthetic fixture did not reach the intended lifecycle states")
	}
	return e
}

func (e *r3) cfg(dest string) backup.BackupConfig {
	return backup.BackupConfig{Environment: backup.Staging, SourceURL: e.DSN, Dest: dest, AllowedRoots: []string{e.roots}, EvidenceRoot: e.root,
		Recipients: e.recipients}
}

func restoreName(t *testing.T) string {
	t.Helper()
	u, _ := ids.NewUUID()
	return "crm_restore_" + strings.ReplaceAll(u, "-", "")[:16]
}

func dsnFor(t *testing.T, base, db string) string {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + db
	return u.String()
}

func (e *r3) restoreCfg(t *testing.T, dir, target string) backup.RestoreConfig {
	return backup.RestoreConfig{AdminURL: pgtest.AdminURL(t), Target: target, BackupDir: dir, Identity: e.keyFile,
		EvidenceDest: filepath.Join(e.roots, target+"-evidence"), AllowedRoots: []string{e.roots}}
}

func discard(t *testing.T, e *r3, target string) {
	t.Helper()
	t.Cleanup(func() {
		_ = backup.Discard(context.Background(), pgtest.AdminURL(t), target, filepath.Join(e.roots, target+"-evidence"), []string{e.roots})
	})
}

func (e *r3) sourceInventory(t *testing.T) backup.Inventory {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	inv, err := backup.Capture(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

// Rehearsal steps 1–15.
func TestR3BackupRestoreRehearsal(t *testing.T) {
	e := setupR3(t)
	ctx := context.Background()
	before := e.sourceInventory(t)
	dir := filepath.Join(e.roots, "backup-0001")
	m, err := backup.Backup(ctx, e.cfg(dir))
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if !m.Encrypted || m.Evidence == nil || len(m.Evidence.Objects) < 4 || len(m.Inventory.Tables) < 50 || len(m.Inventory.Migrations) < 10 {
		t.Fatalf("manifest: encrypted=%v objects=%d tables=%d migrations=%d", m.Encrypted, len(m.Evidence.Objects), len(m.Inventory.Tables),
			len(m.Inventory.Migrations))
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "manifest.json"))
	for _, leak := range []string{"crm_ci_only", "password", "postgres://", "synthetic receipt"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("manifest leaks %q", leak)
		}
	}
	if fi, _ := os.Stat(dir); fi.Mode().Perm() != 0o700 {
		t.Fatalf("backup directory mode %v", fi.Mode().Perm())
	}
	vr, err := backup.Verify(ctx, dir, e.keyFile, backup.Tools{})
	if err != nil || !vr.DatabaseOK || vr.ObjectsVerified != len(m.Evidence.Objects) || vr.ArchiveEntries < 50 {
		t.Fatalf("verify: %+v %v", vr, err)
	}
	target := restoreName(t)
	discard(t, e, target)
	rep, err := backup.Restore(ctx, e.restoreCfg(t, dir, target))
	if err != nil || rep.Status != "COMPLETE" || len(rep.Differences) != 0 || len(rep.EvidenceFindings) != 0 || rep.EvidenceChecked < 4 {
		t.Fatalf("restore: %+v %v", rep, err)
	}
	if rep.MeasuredRecoveryS <= 0 || rep.TimingsSeconds["database_restore"] <= 0 {
		t.Fatalf("recovery time must be measured: %+v", rep.TimingsSeconds)
	}
	if strings.Join(rep.QuarantinedFlags, ",") != "worker.environment.staging,worker.job.receipt_finalise" {
		t.Fatalf("quarantine: %v", rep.QuarantinedFlags)
	}
	// The restored copy: UNKNOWN survives, nothing enabled, owners intact.
	rc, err := pgx.Connect(ctx, dsnFor(t, pgtest.AdminURL(t), target))
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close(ctx)
	var unknown, enabled, owners int
	if err := rc.QueryRow(ctx, `SELECT (SELECT count(*) FROM integration.writeback_command WHERE state = 'UNKNOWN'),
		(SELECT count(*) FROM platform_featureflags.feature_flag WHERE enabled),
		(SELECT count(DISTINCT tableowner) FROM pg_tables WHERE schemaname = 'payments')`).Scan(&unknown, &enabled, &owners); err != nil {
		t.Fatal(err)
	}
	if unknown != int(before.States["integration.writeback_command.state=UNKNOWN"]) || unknown == 0 || enabled != 0 || owners != 1 {
		t.Fatalf("restored: unknown=%d enabled=%d payments owners=%d", unknown, enabled, owners)
	}
	// The source is untouched (no existing database was overwritten).
	if d := backup.Diff(before, e.sourceInventory(t)); len(d) != 0 {
		t.Fatalf("the source changed: %v", d)
	}
	// Replay after restore (13): the restored copy's services can start,
	// and the finaliser on it finalises the STORAGE_ERROR receipt from the
	// restored quarantine object, without duplicating evidence.
	login := e.LoginDSN(t, payments.Module)
	pool := pgtest.OpenPool(t, dsnFor(t, login, target))
	runner, _ := pool.Runner(payments.Module)
	st, err := newFS(filepath.Join(e.roots, target+"-evidence"))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := paymentscommand.New(runner, st, e.billingQ, e.clock)
	if err != nil {
		t.Fatal(err)
	}
	if r, err := svc.Finalise(ctx, 10); err != nil || r.Ready != 1 {
		t.Fatalf("finalise on the restored copy: %+v %v", r, err)
	}
	// Refusals: existing target, invalid name, existing destination,
	// outside the allowlist, production, plaintext in staging.
	if _, err := backup.Restore(ctx, e.restoreCfg(t, dir, target)); !errors.Is(err, backup.ErrRefused) {
		t.Fatalf("restore into an existing database: %v", err)
	}
	src := e.restoreCfg(t, dir, "crm_test_main")
	if _, err := backup.Restore(ctx, src); !errors.Is(err, backup.ErrRefused) {
		t.Fatal("a non-disposable target name must be refused")
	}
	if _, err := backup.Backup(ctx, e.cfg(dir)); !errors.Is(err, backup.ErrRefused) {
		t.Fatal("an existing destination must be refused")
	}
	out := e.cfg(filepath.Join(t.TempDir(), "elsewhere"))
	if _, err := backup.Backup(ctx, out); !errors.Is(err, backup.ErrRefused) {
		t.Fatal("a destination outside the allowlist must be refused")
	}
	prod := e.cfg(filepath.Join(e.roots, "prod"))
	prod.Environment = "production"
	if _, err := backup.Backup(ctx, prod); !errors.Is(err, backup.ErrRefused) {
		t.Fatal("production must be refused")
	}
	plain := e.cfg(filepath.Join(e.roots, "plain"))
	plain.Recipients, plain.AllowPlaintext = "", true
	if _, err := backup.Backup(ctx, plain); !errors.Is(err, backup.ErrRefused) {
		t.Fatal("plaintext outside development must be refused")
	}
	dev := e.cfg(filepath.Join(e.roots, "dev"))
	dev.Environment = backup.Development
	if _, err := backup.Backup(ctx, dev); !errors.Is(err, backup.ErrRefused) {
		t.Fatal("an environment the database does not attest must be refused")
	}
	// Discard refuses anything the tool did not create.
	if err := backup.Discard(ctx, pgtest.AdminURL(t), "crm_restore_notmine", "", nil); !errors.Is(err, backup.ErrRefused) {
		t.Fatal("discard of an unknown database must be refused")
	}
	srcName := strings.TrimPrefix(mustURL(t, e.DSN).Path, "/")
	if err := backup.Discard(ctx, pgtest.AdminURL(t), srcName, "", nil); !errors.Is(err, backup.ErrRefused) {
		t.Fatal("discard of the source database must be refused")
	}
}

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// Integrity and object-store consistency: corrupt / truncated / interrupted
// backups are refused; a database restored without matching objects fails
// verification and no receipt becomes READY because of it.
func TestR3BackupIntegrityAndEvidenceConsistency(t *testing.T) {
	e := setupR3(t)
	ctx := context.Background()
	dir := filepath.Join(e.roots, "backup-ok")
	m, err := backup.Backup(ctx, e.cfg(dir))
	if err != nil {
		t.Fatal(err)
	}
	clone := func(name string) string {
		d := filepath.Join(e.roots, name)
		if err := os.CopyFS(d, os.DirFS(dir)); err != nil {
			t.Fatal(err)
		}
		return d
	}
	// Corrupted database file.
	bad := clone("corrupt-db")
	flip(t, filepath.Join(bad, m.Database.Name))
	if _, err := backup.Verify(ctx, bad, e.keyFile, backup.Tools{}); !errors.Is(err, backup.ErrIntegrity) {
		t.Fatalf("corrupt database file: %v", err)
	}
	// Truncated evidence archive.
	trunc := clone("trunc-evidence")
	p := filepath.Join(trunc, m.Evidence.Archive.Name)
	fi, _ := os.Stat(p)
	if err := os.Truncate(p, fi.Size()/2); err != nil {
		t.Fatal(err)
	}
	if _, err := backup.Verify(ctx, trunc, e.keyFile, backup.Tools{}); !errors.Is(err, backup.ErrIntegrity) {
		t.Fatalf("truncated archive: %v", err)
	}
	// Tampered manifest.
	tm := clone("tampered-manifest")
	raw, _ := os.ReadFile(filepath.Join(tm, "manifest.json"))
	if err := os.WriteFile(filepath.Join(tm, "manifest.json"), []byte(strings.Replace(string(raw), `"rows": `, `"rows": 1`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := backup.Verify(ctx, tm, e.keyFile, backup.Tools{}); !errors.Is(err, backup.ErrIntegrity) {
		t.Fatalf("tampered manifest: %v", err)
	}
	// Interrupted backup: a .partial directory without a manifest is never
	// a backup, and the destination name stays blocked.
	part := filepath.Join(e.roots, "interrupted")
	if err := os.Mkdir(part+".partial", 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := backup.Verify(ctx, part+".partial", e.keyFile, backup.Tools{}); !errors.Is(err, backup.ErrIntegrity) {
		t.Fatal("an interrupted backup must not verify")
	}
	if _, err := backup.Backup(ctx, e.cfg(part)); !errors.Is(err, backup.ErrRefused) {
		t.Fatal("a destination with a leftover .partial must be refused")
	}
	// Wrong key.
	other := filepath.Join(t.TempDir(), "other.txt")
	if out, err := exec.Command("age-keygen", "-o", other).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if _, err := backup.Verify(ctx, dir, other, backup.Tools{}); !errors.Is(err, backup.ErrIntegrity) {
		t.Fatalf("wrong key: %v", err)
	}
	// Database without matching objects: a READY receipt's evidence object
	// and the STORAGE_ERROR receipt's quarantine object are missing from the
	// evidence copy, and an UNBOUND upload's object is corrupt.
	gappy := filepath.Join(e.roots, "gappy-root")
	if err := os.CopyFS(gappy, os.DirFS(e.root)); err != nil {
		t.Fatal(err)
	}
	ready := e.q(t, `SELECT evidence_ref FROM payments.receipt_evidence WHERE state = 'READY' ORDER BY receipt_id LIMIT 1`)
	storage := e.q(t, `SELECT quarantine_ref FROM payments.receipt_evidence WHERE state = 'STORAGE_ERROR'`)
	unbound := e.q(t, `SELECT quarantine_ref FROM payments.upload_object WHERE state = 'UNBOUND' LIMIT 1`)
	for _, k := range []string{ready, storage} {
		if err := os.Remove(filepath.Join(gappy, filepath.FromSlash(k))); err != nil {
			t.Fatal(err)
		}
	}
	flip(t, filepath.Join(gappy, filepath.FromSlash(unbound)))
	gcfg := e.cfg(filepath.Join(e.roots, "backup-gappy"))
	gcfg.EvidenceRoot = gappy
	if _, err := backup.Backup(ctx, gcfg); err != nil {
		t.Fatal(err)
	}
	target := restoreName(t)
	discard(t, e, target)
	rep, err := backup.Restore(ctx, e.restoreCfg(t, gcfg.Dest, target))
	if !errors.Is(err, backup.ErrMismatch) || rep.Status != "FAILED" {
		t.Fatalf("restore without matching objects must fail: %+v %v", rep, err)
	}
	problems := map[string]int{}
	for _, f := range rep.EvidenceFindings {
		problems[f.Kind+":"+f.State+":"+f.Problem]++
	}
	if problems["receipt:READY:MISSING"] != 1 || problems["receipt:STORAGE_ERROR:MISSING"] != 1 || problems["upload:UNBOUND:CORRUPT"] != 1 {
		t.Fatalf("findings: %v", problems)
	}
	// Replay on the failed copy: the finaliser proves the missing quarantine
	// bytes NOT_RECOVERABLE; it never fabricates READY.
	login := e.LoginDSN(t, payments.Module)
	pool := pgtest.OpenPool(t, dsnFor(t, login, target))
	runner, _ := pool.Runner(payments.Module)
	st, _ := newFS(filepath.Join(e.roots, target+"-evidence"))
	svc, _ := paymentscommand.New(runner, st, e.billingQ, e.clock)
	if r, err := svc.Finalise(ctx, 10); err != nil || r.Ready != 0 || r.NotRecoverable != 1 {
		t.Fatalf("replay on a copy without the bytes: %+v %v", r, err)
	}
	// Retention is a dry run.
	cands, err := backup.PlanRetention(e.roots, 1, 0, time.Now().Add(time.Hour))
	if err != nil || len(cands) == 0 {
		t.Fatalf("retention plan: %v %v", cands, err)
	}
	for _, c := range cands {
		if _, err := os.Stat(filepath.Join(e.roots, c.Dir, "manifest.json")); err != nil {
			t.Fatal("planning retention must never delete a backup")
		}
	}
	if keepAll, _ := backup.PlanRetention(e.roots, 100, 0, time.Now()); len(keepAll) != 0 {
		t.Fatal("nothing is a candidate while under the keep count")
	}
}

func flip(t *testing.T, p string) {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil || len(b) < 2 {
		t.Fatalf("flip %s: %v", p, err)
	}
	b[len(b)/2] ^= 0xff
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func newFS(root string) (*objectstore.FS, error) { return objectstore.NewFS(root) }
