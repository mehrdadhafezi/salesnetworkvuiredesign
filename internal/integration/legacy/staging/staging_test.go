package staging_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	integrationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/staging"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

const t0 = "2026-10-29T08:00:00Z"

type env struct {
	*pgtest.Env
	loader *staging.Loader
	admin  *pgx.Conn
}

func setup(t *testing.T) *env {
	t.Helper()
	e := pgtest.New(t) // platform + synthetic modules, real migration runner
	ctx := context.Background()
	if _, err := migrations.Up(ctx, e.DSN, integrationmigrations.Source()); err != nil {
		t.Fatalf("integration migrations: %v", err)
	}
	at, _ := timeutil.ParseRFC3339("2026-10-29T09:00:00Z")
	l, err := staging.NewLoader(e.Runner(t, integration.Module), timeutil.FixedClock{At: at}, "test/1")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.Connect(ctx, e.DSN) // provisioning connection: test assertions/fault injection only
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	return &env{Env: e, loader: l, admin: admin}
}

func (e *env) open(t *testing.T, dir string) *snapshot.Package {
	t.Helper()
	p, err := snapshot.Open(dir, spec.Default())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (e *env) count(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func (e *env) load(t *testing.T, dir string) staging.Report {
	t.Helper()
	rep, err := e.loader.Load(context.Background(), e.open(t, dir))
	if err != nil {
		t.Fatalf("load: %v (report %+v)", err, rep)
	}
	return rep
}

// A. Migrations: the integration schema and every spec's staging table exist
// with exactly provenance + raw text columns; the Down section drops only
// the integration schema.
func TestMigrationsAndRollback(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	for _, s := range spec.Default().Specs() {
		rows, err := e.admin.Query(ctx, `SELECT column_name, data_type, is_nullable FROM information_schema.columns
			WHERE table_schema = 'integration' AND table_name = $1 ORDER BY ordinal_position`, s.StagingTable)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for rows.Next() {
			var name, typ, nullable string
			if err := rows.Scan(&name, &typ, &nullable); err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(name, "raw_") && (typ != "text" || nullable != "YES") {
				t.Errorf("%s.%s is %s nullable=%s; raw columns must be nullable text", s.StagingTable, name, typ, nullable)
			}
			names = append(names, name)
		}
		rows.Close()
		want := []string{"batch_id", "spec_id", "source_ordinal", "source_key", "row_sha256", "imported_at"}
		for _, c := range s.Columns {
			want = append(want, c.StagingColumn())
		}
		if strings.Join(names, ",") != strings.Join(want, ",") {
			t.Errorf("%s columns = %v, want %v", s.StagingTable, names, want)
		}
	}
	// Re-applying is a no-op.
	if applied, err := migrations.Up(ctx, e.DSN, integrationmigrations.Source()); err != nil || len(applied) != 0 {
		t.Fatalf("re-apply: %v %v", applied, err)
	}
	// Rollback strategy = the migration's Down section: drop the staging
	// schema. Proven here on the disposable database only.
	if _, err := e.admin.Exec(ctx, "DROP SCHEMA integration CASCADE"); err != nil {
		t.Fatal(err)
	}
	if n := e.count(t, "SELECT count(*) FROM information_schema.schemata WHERE schema_name IN ('audit','platform_featureflags','syn_alpha','syn_beta')"); n != 4 {
		t.Fatalf("rollback harmed platform/business schemas: %d remain", n)
	}
	if n := e.count(t, "SELECT count(*) FROM crm_schema_migrations WHERE is_applied"); n < 2 {
		t.Fatalf("platform version table harmed: %d", n)
	}
}

// E/F/H. Load, reconcile, raw fidelity, idempotent replay.
func TestLoadReconcileFidelityAndReplay(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	dir := legacytest.Write(t, "synthetic-snapshot-0100", t0, legacytest.Rows())
	rep := e.load(t, dir)
	if rep.Outcome != result.Success || rep.BatchState != staging.StateLoaded || rep.Replay {
		t.Fatalf("load: %+v", rep)
	}
	for _, er := range rep.Entities {
		if er.State != staging.EntityLoaded || er.StagedRows == nil || *er.StagedRows != er.ExpectedRows ||
			er.StagingSHA256 != er.ExpectedContentSHA || er.ActualFileSHA256 != er.ExpectedFileSHA256 || er.Status != result.Success {
			t.Fatalf("entity %s: %+v", er.SpecID, er)
		}
	}
	if n := e.count(t, "SELECT count(*) FROM integration.stg_hr_profile_v1"); n != 3 {
		t.Fatalf("staged hr_profile rows = %d", n)
	}

	// F. Raw fidelity: NULL vs empty string, Unicode, raw meta/JSON, raw timestamps, unfamiliar statuses.
	var nullCount, emptyCount int64
	_ = e.admin.QueryRow(ctx, `SELECT count(*) FILTER (WHERE raw_user_login IS NULL), count(*) FILTER (WHERE raw_user_login = '') FROM integration.stg_wp_user_v1`).Scan(&nullCount, &emptyCount)
	if nullCount != 1 || emptyCount != 1 {
		t.Fatalf("NULL/empty: %d/%d", nullCount, emptyCount)
	}
	var display, created, meta, ctxText, status, key string
	_ = e.admin.QueryRow(ctx, `SELECT raw_display_name, raw_user_registered FROM integration.stg_wp_user_v1 WHERE source_ordinal = 1`).Scan(&display, &created)
	_ = e.admin.QueryRow(ctx, `SELECT raw_meta_value FROM integration.stg_wp_usermeta_v1 WHERE source_ordinal = 1`).Scan(&meta)
	_ = e.admin.QueryRow(ctx, `SELECT raw_context, raw_action, source_key FROM integration.stg_hr_assignment_history_v1 WHERE source_ordinal = 1`).Scan(&ctxText, &status, &key)
	if display != "Dev User 1 — کاربر آزمایشی" || created != "0000-00-00 00:00:00" || meta != `a:1:{s:13:"administrator";b:1;}` ||
		ctxText != "{\"synthetic\":true,\"note\":\"خط\\nدوم\",\"list\":[1,2]}" || status != "synthetic_unfamiliar_status_x" || key != "10" {
		t.Fatalf("raw values changed: %q %q %q %q %q %q", display, created, meta, ctxText, status, key)
	}

	// H. Reconcile (read-only) agrees and reports the persisted observations,
	// never default zeros.
	rec, err := e.loader.Reconcile(ctx, spec.Default(), "synthetic-snapshot-0100")
	if err != nil || rec.Outcome != result.Success || rec.ManifestSHA256 != rep.ManifestSHA256 {
		t.Fatalf("reconcile: %v %+v", err, rec)
	}
	for _, er := range rec.Entities {
		if er.ParsedRows == nil || *er.ParsedRows != er.ExpectedRows || er.ActualBytes == nil || *er.ActualBytes != er.ExpectedBytes ||
			er.ActualFileSHA256 != er.ExpectedFileSHA256 || er.ExpectedBytes <= 0 {
			t.Fatalf("reconcile %s counts: %+v", er.SpecID, er)
		}
	}

	// E. Same snapshot again: replay, no duplicates.
	again := e.load(t, dir)
	if !again.Replay || again.BatchID != rep.BatchID || again.Outcome != result.Success {
		t.Fatalf("replay: %+v", again)
	}
	if n := e.count(t, "SELECT count(*) FROM integration.import_batch"); n != 1 {
		t.Fatalf("batches = %d", n)
	}
	if n := e.count(t, "SELECT count(*) FROM integration.stg_hr_profile_v1"); n != 3 {
		t.Fatalf("rows after replay = %d", n)
	}

	// E. Same snapshot_id, different content → CONFLICT, nothing written.
	changed := legacytest.Rows()
	changed[0].Rows = changed[0].Rows[:2]
	conflict, err := e.loader.Load(ctx, e.open(t, legacytest.Write(t, "synthetic-snapshot-0100", t0, changed)))
	if err != nil || conflict.Outcome != result.Conflict {
		t.Fatalf("conflict: %v %+v", err, conflict)
	}
	// Identical content, different snapshot_id → existing batch, no new rows.
	dup, err := e.loader.Load(ctx, e.open(t, legacytest.Write(t, "synthetic-snapshot-0101", "2027-01-01T00:00:00Z", legacytest.Rows())))
	if err != nil || dup.BatchID != rep.BatchID || len(dup.Warnings) == 0 {
		t.Fatalf("duplicate content: %v %+v", err, dup)
	}
	if n := e.count(t, "SELECT count(*) FROM integration.import_batch"); n != 1 {
		t.Fatalf("batches after conflict/duplicate = %d", n)
	}

	// H. Staging tampered after load (by the provisioning superuser; the owner
	// cannot update staged rows) → reconcile reports CONFLICT explicitly.
	if _, err := e.admin.Exec(ctx, `UPDATE integration.stg_hr_level_v1 SET raw_label = raw_label || 'x' WHERE source_ordinal = 1`); err != nil {
		t.Fatal(err)
	}
	tampered, err := e.loader.Reconcile(ctx, spec.Default(), "synthetic-snapshot-0100")
	if err != nil || tampered.Outcome != result.Partial {
		t.Fatalf("tampered reconcile: %v %+v", err, tampered)
	}
	conflicts := 0
	for _, er := range tampered.Entities {
		if er.Status == result.Conflict {
			conflicts++
			if er.SpecID != "legacy.hr_level.v1" || strings.Contains(er.Problem, "syn-") {
				t.Fatalf("conflict entity %s problem %q", er.SpecID, er.Problem)
			}
		}
	}
	if conflicts != 1 {
		t.Fatalf("conflicts = %d", conflicts)
	}
	// A deleted staged row is a count mismatch → CONFLICT, never SUCCESS.
	if _, err := e.admin.Exec(ctx, `DELETE FROM integration.stg_hr_unit_v1 WHERE source_ordinal = 3`); err != nil {
		t.Fatal(err)
	}
	if r, err := e.loader.Reconcile(ctx, spec.Default(), "synthetic-snapshot-0100"); err != nil || r.Outcome == result.Success {
		t.Fatalf("deleted row reconcile: %v %+v", err, r)
	}
	if r, err := e.loader.Reconcile(ctx, spec.Default(), "synthetic-snapshot-0999"); err != nil || r.Outcome != result.Failed {
		t.Fatalf("unknown snapshot reconcile: %v %+v", err, r)
	}
}

// The batch stores the exact manifest bytes and their hash, so validate,
// load and reconcile report the same manifest checksum for any valid
// formatting of the manifest.
func TestStoredManifestIsTheOriginalEvidence(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	dir := legacytest.Write(t, "synthetic-snapshot-0150", t0, legacytest.Rows())
	path := filepath.Join(dir, snapshot.ManifestFile)
	raw, _ := os.ReadFile(path)
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, compact.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	p := e.open(t, dir)
	rep, err := e.loader.Load(ctx, p)
	if err != nil || rep.Outcome != result.Success || rep.ManifestSHA256 != p.ManifestSHA256 {
		t.Fatalf("load: %v %+v", err, rep)
	}
	var stored string
	var storedSHA []byte
	if err := e.admin.QueryRow(ctx, "SELECT manifest_json, manifest_sha256 FROM integration.import_batch").Scan(&stored, &storedSHA); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(compact.Bytes())
	if stored != compact.String() || !bytes.Equal(storedSHA, sum[:]) {
		t.Fatal("stored manifest must be the exact original bytes and their checksum")
	}
	rec, err := e.loader.Reconcile(ctx, spec.Default(), "synthetic-snapshot-0150")
	if err != nil || rec.ManifestSHA256 != p.ManifestSHA256 {
		t.Fatalf("reconcile manifest checksum %s, validate %s (%v)", rec.ManifestSHA256, p.ManifestSHA256, err)
	}
}

// G/J. A failing entity rolls back, is recorded FAILED, the batch cannot be
// LOADED, and a retry after the fault resumes deterministically.
func TestFailedEntityRollsBackAndRetryResumes(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	dir := legacytest.Write(t, "synthetic-snapshot-0200", t0, legacytest.Rows())
	// Fault injection: the owner temporarily loses INSERT on one staging table.
	if _, err := e.admin.Exec(ctx, "REVOKE INSERT ON integration.stg_hr_team_v1 FROM crm_owner_integration"); err != nil {
		t.Fatal(err)
	}
	rep, err := e.loader.Load(ctx, e.open(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Outcome != result.Failed || rep.BatchState != staging.StateFailed {
		t.Fatalf("failed load: %+v", rep)
	}
	if n := e.count(t, "SELECT count(*) FROM integration.stg_hr_team_v1"); n != 0 {
		t.Fatalf("failed entity left %d rows", n)
	}
	var state, entityState string
	_ = e.admin.QueryRow(ctx, "SELECT state FROM integration.import_batch").Scan(&state)
	_ = e.admin.QueryRow(ctx, "SELECT state FROM integration.import_entity WHERE spec_id = 'legacy.hr_team.v1'").Scan(&entityState)
	if state != staging.StateFailed || entityState != staging.EntityFailed {
		t.Fatalf("states: batch %s entity %s", state, entityState)
	}
	rec, _ := e.loader.Reconcile(ctx, spec.Default(), "synthetic-snapshot-0200")
	if rec.Outcome == result.Success {
		t.Fatalf("a failed batch must not reconcile as SUCCESS: %+v", rec)
	}
	// Retry after the fault is removed.
	if _, err := e.admin.Exec(ctx, "GRANT INSERT ON integration.stg_hr_team_v1 TO crm_owner_integration"); err != nil {
		t.Fatal(err)
	}
	retry := e.load(t, dir)
	if retry.Outcome != result.Success || retry.BatchState != staging.StateLoaded || retry.BatchID != rep.BatchID {
		t.Fatalf("retry: %+v", retry)
	}
	for _, s := range spec.Default().Specs() {
		if n := e.count(t, "SELECT count(*) FROM integration."+s.StagingTable); n != 3 {
			t.Fatalf("%s rows after retry = %d", s.StagingTable, n)
		}
	}
}

// G. Database guards: an entity cannot be LOADED with unreconciled counts or
// checksums, and a batch cannot be LOADED while any entity is not LOADED.
func TestDatabaseRefusesDishonestStates(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	rep, err := e.loader.Load(ctx, e.open(t, legacytest.Write(t, "synthetic-snapshot-0300", t0, legacytest.Rows())))
	if err != nil || rep.Outcome != result.Success {
		t.Fatalf("load: %v %+v", err, rep)
	}
	runner := e.Runner(t, integration.Module)
	err = runner.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE integration.import_entity SET staged_rows = staged_rows + 1 WHERE spec_id = 'legacy.hr_unit.v1'`)
		return err
	})
	if !isCheckViolation(err) {
		t.Fatalf("entity LOADED with wrong count accepted: %v", err)
	}
	err = runner.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE integration.import_batch SET state = 'LOADING', loaded_at = NULL`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE integration.import_entity SET state = 'PENDING', loaded_at = NULL WHERE spec_id = 'legacy.hr_unit.v1'`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE integration.import_batch SET state = 'LOADED', loaded_at = now()`)
		return err
	})
	if !isCheckViolation(err) {
		t.Fatalf("batch LOADED with a PENDING entity accepted: %v", err)
	}
	// Staged rows are append-only for the owner.
	err = runner.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE integration.stg_wp_user_v1 SET raw_user_login = 'x'`)
		return err
	})
	if !isPrivilege(err) {
		t.Fatalf("staged rows must not be updatable by the owner: %v", err)
	}
	err = runner.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM integration.stg_wp_user_v1`)
		return err
	})
	if !isPrivilege(err) {
		t.Fatalf("staged rows must not be deletable by the owner: %v", err)
	}
}

func TestIncompleteEntityAndTamperedFile(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	rows := legacytest.Rows()
	rows[0].Completeness = snapshot.Incomplete
	rep := e.load(t, legacytest.Write(t, "synthetic-snapshot-0400", t0, rows))
	if rep.Outcome != result.Partial || rep.BatchState != staging.StateLoadedIncomplete {
		t.Fatalf("incomplete: %+v", rep)
	}

	// A tampered file fails validation before anything is written.
	bad := legacytest.Write(t, "synthetic-snapshot-0402", t0, legacytest.Rows())
	f := filepath.Join(bad, snapshot.EntityFile("legacy.hr_level.v1"))
	b, _ := os.ReadFile(f)
	_ = os.WriteFile(f, []byte(strings.Replace(string(b), "syn-", "syx-", 1)), 0o600)
	failed, err := e.loader.Load(ctx, e.open(t, bad))
	if err != nil || failed.Outcome != result.Failed {
		t.Fatalf("tampered: %v %+v", err, failed)
	}
	if n := e.count(t, "SELECT count(*) FROM integration.import_batch WHERE snapshot_id = 'synthetic-snapshot-0402'"); n != 0 {
		t.Fatal("a failed validation wrote a batch")
	}
}

func unclassifiedPackage(t *testing.T, snapshotID string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "p")
	h := legacytest.Header(snapshotID, t0)
	h.DataOrigin = snapshot.OriginUnclassified
	if _, err := snapshot.Write(dir, spec.Default(), h, legacytest.Rows()); err != nil {
		t.Fatal(err)
	}
	return dir
}

// assertNothingWritten proves a refused load left no batch, no entity and no
// staged row in any staging table.
func (e *env) assertNothingWritten(t *testing.T) {
	t.Helper()
	if n := e.count(t, "SELECT count(*) FROM integration.import_batch"); n != 0 {
		t.Fatalf("import_batch rows = %d", n)
	}
	if n := e.count(t, "SELECT count(*) FROM integration.import_entity"); n != 0 {
		t.Fatalf("import_entity rows = %d", n)
	}
	for _, s := range spec.Default().Specs() {
		if n := e.count(t, "SELECT count(*) FROM integration."+s.StagingTable); n != 0 {
			t.Fatalf("%s rows = %d", s.StagingTable, n)
		}
	}
}

func assertBlocked(t *testing.T, rep staging.Report, err error) {
	t.Helper()
	if err != nil || rep.Outcome != result.Blocked || rep.Reason == nil || *rep.Reason != reasoncodes.PolicyUnresolved ||
		rep.Problem != staging.BlockedNonSyntheticMessage || rep.BatchID != "" {
		t.Fatalf("want BLOCKED/POLICY_UNRESOLVED: %v %+v", err, rep)
	}
}

// Slice 3 fails closed: a non-synthetic package is BLOCKED and writes nothing.
func TestNonSyntheticPackageIsBlocked(t *testing.T) {
	e := setup(t)
	rep, err := e.loader.Load(context.Background(), e.open(t, unclassifiedPackage(t, "unclassified-snapshot-0401")))
	assertBlocked(t, rep, err)
	e.assertNothingWritten(t)
	// The block is stable on repeat.
	rep, err = e.loader.Load(context.Background(), e.open(t, unclassifiedPackage(t, "unclassified-snapshot-0401")))
	assertBlocked(t, rep, err)
	e.assertNothingWritten(t)
}

// Direct Loader callers cannot bypass the block: Load has no options, and
// altering the in-memory Package (origin, bytes, checksum) or handing it a
// nil/zero Package still fails closed with nothing written.
func TestLoaderCannotBeBypassed(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	unclassified := unclassifiedPackage(t, "unclassified-snapshot-0402")
	synthetic := legacytest.Write(t, "synthetic-snapshot-0403", t0, legacytest.Rows())

	// 1. Flip the parsed origin to synthetic; the stored manifest bytes still say unclassified.
	p := e.open(t, unclassified)
	p.Manifest.DataOrigin = snapshot.OriginSynthetic
	rep, err := e.loader.Load(ctx, p)
	assertBlocked(t, rep, err)

	// 2. Swap in synthetic manifest bytes without their checksum.
	p = e.open(t, unclassified)
	p.Manifest.DataOrigin = snapshot.OriginSynthetic
	p.ManifestBytes = e.open(t, synthetic).ManifestBytes
	rep, err = e.loader.Load(ctx, p)
	assertBlocked(t, rep, err)

	// 3. Swap in synthetic bytes and their checksum from another package.
	s := e.open(t, synthetic)
	p = e.open(t, unclassified)
	p.Manifest.DataOrigin = snapshot.OriginSynthetic
	p.ManifestBytes, p.ManifestSHA256 = s.ManifestBytes, s.ManifestSHA256
	rep, err = e.loader.Load(ctx, p)
	assertBlocked(t, rep, err)

	// 4. Unparseable or empty manifest bytes, nil and zero-value packages.
	p = e.open(t, unclassified)
	p.Manifest.DataOrigin = snapshot.OriginSynthetic
	p.ManifestBytes = nil
	rep, err = e.loader.Load(ctx, p)
	assertBlocked(t, rep, err)
	rep, err = e.loader.Load(ctx, nil)
	assertBlocked(t, rep, err)
	// A fully consistent synthetic Package re-pointed at the unclassified
	// package's directory (identical table files) is refused: the manifest on
	// disk must equal the bytes that would be stored.
	p = e.open(t, synthetic)
	p.Dir = unclassified
	rep, err = e.loader.Load(ctx, p)
	assertBlocked(t, rep, err)
	rep, err = e.loader.Load(ctx, &snapshot.Package{})
	assertBlocked(t, rep, err)

	e.assertNothingWritten(t)

	// 5. The genuine synthetic package still loads.
	ok, err := e.loader.Load(ctx, e.open(t, synthetic))
	if err != nil || ok.Outcome != result.Success || ok.BatchState != staging.StateLoaded {
		t.Fatalf("synthetic load: %v %+v", err, ok)
	}
}

// J. Concurrent loads of the same package stage it exactly once.
func TestConcurrentLoadsAreDeterministic(t *testing.T) {
	e := setup(t)
	dir := legacytest.Write(t, "synthetic-snapshot-0500", t0, legacytest.Rows())
	var wg sync.WaitGroup
	reps := make([]staging.Report, 4)
	errs := make([]error, 4)
	for i := range reps {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			reps[i], errs[i] = e.loader.Load(context.Background(), e.open(t, dir))
		}(i)
	}
	wg.Wait()
	for i := range reps {
		if errs[i] != nil || reps[i].Outcome != result.Success {
			t.Fatalf("loader %d: %v %+v", i, errs[i], reps[i])
		}
	}
	if n := e.count(t, "SELECT count(*) FROM integration.import_batch"); n != 1 {
		t.Fatalf("batches = %d", n)
	}
	if n := e.count(t, "SELECT count(*) FROM integration.stg_wp_usermeta_v1"); n != 3 {
		t.Fatalf("rows = %d", n)
	}
}

// I. Ownership: the integration owner cannot write other modules, and other
// modules cannot write integration.
func TestOwnershipIsolation(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	in := e.Runner(t, integration.Module)
	for name, q := range map[string]string{
		"audit":        `INSERT INTO audit.audit_event (event_id, occurred_at, subject_domain, subject_id, action) VALUES (gen_random_uuid(), now(), 'x', 'y', 'syn.test')`,
		"featureflags": `INSERT INTO platform_featureflags.feature_flag (flag_key, enabled, updated_at) VALUES ('syn.flag', true, now())`,
		"syn_alpha":    `INSERT INTO syn_alpha.outbox_event (event_id, event_type, event_version, payload, occurred_at, recorded_at) VALUES (gen_random_uuid(), 'syn.e', 1, '{}', now(), now())`,
	} {
		err := in.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error { _, err := tx.Exec(ctx, q); return err })
		if !isPrivilege(err) {
			t.Errorf("integration wrote %s: %v", name, err)
		}
	}
	alpha := e.Runner(t, pgtest.Alpha)
	err := alpha.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO integration.import_batch (batch_id) VALUES (gen_random_uuid())`)
		return err
	})
	if !isPrivilege(err) {
		t.Fatalf("syn_alpha wrote integration: %v", err)
	}
	// Another module's login cannot act as integration, and a shared login
	// that can assume both integration and another owner role is refused
	// before any SQL runs.
	foreign, err := e.Pool(t, owner.Module("syn_beta")).Runner(integration.Module)
	if err != nil {
		t.Fatal(err)
	}
	if err := foreign.InTx(ctx, db.TxOptions{}, func(context.Context, db.Tx) error { return nil }); !errors.Is(err, db.ErrSessionRoleNotIsolated) {
		t.Fatalf("foreign login used as integration: %v", err)
	}
	shared, err := pgtest.OpenPool(t, e.LoginDSN(t, integration.Module, pgtest.Alpha)).Runner(integration.Module)
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.InTx(ctx, db.TxOptions{}, func(context.Context, db.Tx) error { return nil }); !errors.Is(err, db.ErrSessionRoleNotIsolated) {
		t.Fatalf("shared two-owner login accepted: %v", err)
	}
	if _, err := staging.NewLoader(alpha, timeutil.FixedClock{}, "x"); !errors.Is(err, staging.ErrWrongOwner) {
		t.Fatalf("loader over a non-integration runner: %v", err)
	}
}

func isPrivilege(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && (pg.Code == "42501" || pg.Code == "3F000")
}

func isCheckViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23514"
}
