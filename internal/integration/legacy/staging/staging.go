// Package staging loads validated legacy snapshot packages into the raw
// integration staging schema and reconciles staging against the manifest
// (Architecture V1 §24.3, §24.5; Slice 3).
//
// Raw means raw: every cell is stored as the exact legacy text (or NULL) in
// its spec's raw_<column>; nothing is normalised, merged, converted,
// deduplicated or reinterpreted. All writes run through one
// integration-owned platform transaction runner, so PostgreSQL itself
// rejects any write outside the integration schema.
//
// # Batch lifecycle (technical states only)
//
//	VALIDATED          manifest, layout, every file checksum, count and row
//	                   encoding verified offline; batch and entities registered
//	LOADING            at least one entity transaction committed
//	LOADED             every entity LOADED and declared COMPLETE
//	LOADED_INCOMPLETE  every entity LOADED, at least one declared not COMPLETE
//	FAILED             an entity failed; failure_reason says why
//
// Entities are PENDING, LOADED or FAILED. Each entity loads in its own
// transaction that re-reads the file, stages its rows, recomputes the
// staging fidelity checksum from PostgreSQL and marks the entity LOADED only
// if file checksum, counts and fidelity checksum all equal the manifest
// (also enforced by a CHECK constraint). A failed entity rolls back
// completely and is then recorded FAILED in a separate transaction; the batch
// can never become LOADED while any entity is not LOADED (trigger guard).
//
// # Idempotency
//
// The declared snapshot_id and the content identity (snapshot.ContentIdentity)
// are both unique. Re-loading the same package resumes or replays without
// duplicating rows; the same snapshot_id with different content is CONFLICT;
// identical content under another snapshot_id reports the existing batch and
// stages nothing new. File names are never identity.
//
// # Synthetic only (fail closed)
//
// Slice 3 stages synthetic packages only. A package whose manifest declares
// any other data_origin is BLOCKED (POLICY_UNRESOLVED) and nothing is
// written; no option, flag or alternative caller path exists. Lifting this
// requires a future Owner-approved staging/anonymisation policy and
// environment (TAD (E0) #18, D-23).
package staging

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// Batch states.
const (
	StateValidated        = "VALIDATED"
	StateLoading          = "LOADING"
	StateLoaded           = "LOADED"
	StateLoadedIncomplete = "LOADED_INCOMPLETE"
	StateFailed           = "FAILED"
)

// Entity states.
const (
	EntityPending = "PENDING"
	EntityLoaded  = "LOADED"
	EntityFailed  = "FAILED"
)

// ErrWrongOwner reports a runner that does not belong to the integration module.
var ErrWrongOwner = errors.New("staging: runner must be owned by the integration module")

// ErrFidelity reports staged rows that no longer reproduce their recorded
// row checksums or contiguous ordinals.
var ErrFidelity = errors.New("staging: staged rows do not reproduce the source rows")

// maxParams keeps one INSERT below PostgreSQL's 65535 bind-parameter limit.
const maxParams = 60000

// Loader stages snapshot packages.
type Loader struct {
	runner      *db.Runner
	clock       timeutil.Clock
	toolVersion string
}

// NewLoader returns a loader over an integration-owned runner.
func NewLoader(r *db.Runner, c timeutil.Clock, toolVersion string) (*Loader, error) {
	if r == nil || r.Module() != integration.Module {
		return nil, ErrWrongOwner
	}
	if c == nil || toolVersion == "" {
		return nil, errors.New("staging: clock and tool version are required")
	}
	return &Loader{runner: r, clock: c, toolVersion: toolVersion}, nil
}

// BlockedNonSyntheticMessage is the bounded explanation returned for every
// non-synthetic package. Slice 3 fails closed: there is no option, flag or
// caller path that stages non-synthetic data. Staging such data requires a
// future Owner-approved staging/anonymisation policy and environment (TAD (E0)
// #18, D-23), which this code does not define.
const BlockedNonSyntheticMessage = "non-synthetic snapshot refused: staging non-synthetic data requires a future Owner-approved staging/anonymisation policy and environment (TAD (E0) #18, D-23); nothing was written"

// syntheticOnly reports whether p may be staged. It checks the parsed
// manifest, the exact manifest bytes that would be stored (and their recorded
// checksum) and the manifest file actually present in the package directory,
// so a caller that alters or re-points the in-memory Package cannot bypass
// the block. Any inconsistency fails closed.
func syntheticOnly(p *snapshot.Package) bool {
	if p == nil || p.Manifest.DataOrigin != snapshot.OriginSynthetic || len(p.ManifestBytes) == 0 || p.Dir == "" {
		return false
	}
	sum := sha256.Sum256(p.ManifestBytes)
	if hex.EncodeToString(sum[:]) != p.ManifestSHA256 {
		return false
	}
	onDisk, err := os.ReadFile(filepath.Join(p.Dir, snapshot.ManifestFile))
	if err != nil || !bytes.Equal(onDisk, p.ManifestBytes) {
		return false
	}
	stored, err := snapshot.ParseManifest(p.ManifestBytes)
	return err == nil && stored.DataOrigin == snapshot.OriginSynthetic && stored.SnapshotID == p.Manifest.SnapshotID
}

// EntityResult is the per-entity load/reconciliation result (no row values).
type EntityResult struct {
	snapshot.EntityReport
	State         string `json:"state"`
	StagedRows    *int64 `json:"staged_rows"`
	StagingSHA256 string `json:"staging_sha256,omitempty"`
}

// Report is the machine-readable outcome of load or reconcile.
type Report struct {
	Command        string            `json:"command"`
	SnapshotID     string            `json:"snapshot_id"`
	BatchID        string            `json:"batch_id,omitempty"`
	LegacyVersion  string            `json:"legacy_version,omitempty"`
	DataOrigin     string            `json:"data_origin,omitempty"`
	ContentSHA256  string            `json:"content_sha256,omitempty"`
	ManifestSHA256 string            `json:"manifest_sha256,omitempty"`
	BatchState     string            `json:"batch_state,omitempty"`
	Outcome        result.Status     `json:"outcome"`
	Reason         *reasoncodes.Code `json:"reason_code,omitempty"`
	Problem        string            `json:"problem,omitempty"`
	Replay         bool              `json:"replay"`
	Entities       []EntityResult    `json:"entities"`
	Warnings       []string          `json:"warnings"`
}

func (r *Report) set(s result.Status, code reasoncodes.Code, problem string) {
	r.Outcome, r.Reason, r.Problem = s, &code, problem
}

func newReport(cmd string, p *snapshot.Package) Report {
	return Report{
		Command: cmd, SnapshotID: p.Manifest.SnapshotID, LegacyVersion: p.Manifest.Legacy.Version,
		DataOrigin: string(p.Manifest.DataOrigin), ContentSHA256: p.Manifest.ContentIdentity(),
		ManifestSHA256: p.ManifestSHA256, Entities: []EntityResult{}, Warnings: []string{},
	}
}

func (l *Loader) now() time.Time { return l.clock.Now().Time() }

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err) // inputs are validated lowercase hex SHA-256
	}
	return b
}

// Load validates the package offline, then stages it. It returns a report in
// every case; err is non-nil only for infrastructure faults, and then the
// report outcome says FAILED or UNKNOWN honestly.
//
// Only synthetic packages are staged. Every other package is BLOCKED with
// POLICY_UNRESOLVED before any validation or database access; the loader
// enforces this itself and offers no override.
func (l *Loader) Load(ctx context.Context, p *snapshot.Package) (Report, error) {
	if !syntheticOnly(p) {
		rep := Report{Command: "load", Entities: []EntityResult{}, Warnings: []string{}}
		if p != nil {
			rep.SnapshotID, rep.LegacyVersion, rep.DataOrigin = p.Manifest.SnapshotID, p.Manifest.Legacy.Version, string(p.Manifest.DataOrigin)
		}
		rep.set(result.Blocked, reasoncodes.PolicyUnresolved, BlockedNonSyntheticMessage)
		return rep, nil
	}
	rep := newReport("load", p)
	// 1. Offline validation of every entity before anything is written.
	pre, err := p.Validate()
	if err != nil {
		rep.set(result.Failed, reasoncodes.SourceUnavailable, err.Error())
		return rep, nil
	}
	for _, e := range pre {
		rep.Entities = append(rep.Entities, EntityResult{EntityReport: e, State: EntityPending})
		if e.Status == result.Failed {
			rep.set(result.Failed, reasoncodes.ContractInvalid, "snapshot validation failed for "+e.SpecID+": "+e.Problem+" — nothing was written")
		}
	}
	if rep.Outcome == result.Failed {
		return rep, nil
	}

	// 2. Register (or find) the batch.
	batchID, existing, err := l.register(ctx, p, &rep)
	if err != nil || rep.Outcome == result.Conflict {
		return finish(&rep, err)
	}
	rep.BatchID = batchID
	if existing == StateLoaded || existing == StateLoadedIncomplete {
		rep.Replay = true
		return l.reconcileInto(ctx, &rep, batchID, "load")
	}

	// 3. Load each entity in its own transaction.
	for i := range rep.Entities {
		er := &rep.Entities[i]
		e := p.Manifest.Entities[indexOf(p.Manifest.Entities, er.SpecID)]
		if err := l.loadEntity(ctx, p, batchID, e, er); err != nil {
			l.recordFailure(ctx, batchID, e.SpecID, err)
			rep.BatchState = StateFailed
			if errors.Is(err, db.ErrCommitOutcomeUnknown) {
				rep.set(result.Unknown, reasoncodes.OutcomeUnknown, "commit outcome unknown for "+e.SpecID+"; run reconcile before any retry")
			} else {
				rep.set(result.Failed, reasoncodes.ContractInvalid, "entity "+e.SpecID+" failed and was rolled back: "+err.Error())
			}
			return rep, nil
		}
	}

	// 4. Promote the batch (trigger-guarded).
	if err := l.finalize(ctx, batchID); err != nil {
		return finish(&rep, err)
	}
	return l.reconcileInto(ctx, &rep, batchID, "load")
}

func finish(rep *Report, err error) (Report, error) {
	if err == nil {
		return *rep, nil
	}
	if errors.Is(err, db.ErrCommitOutcomeUnknown) {
		rep.set(result.Unknown, reasoncodes.OutcomeUnknown, "commit outcome unknown; run reconcile before any retry")
	} else if rep.Outcome != result.Conflict {
		rep.set(result.Failed, reasoncodes.SourceUnavailable, err.Error())
	}
	return *rep, err
}

func indexOf(es []snapshot.Entity, id string) int {
	for i, e := range es {
		if e.SpecID == id {
			return i
		}
	}
	return -1
}

// register inserts the batch and its PENDING entities, or finds the existing
// batch for the same snapshot_id / content identity.
func (l *Loader) register(ctx context.Context, p *snapshot.Package, rep *Report) (string, string, error) {
	m := p.Manifest
	content := m.ContentIdentity()
	// Store the exact manifest bytes and their checksum (the evidence), not a
	// re-serialisation, so validate/inspect/load/reconcile agree.
	manifestBytes := p.ManifestBytes
	var batchID, state string
	err := l.runner.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		var enc string
		if err := tx.QueryRow(ctx, "SHOW server_encoding").Scan(&enc); err != nil {
			return err
		}
		if enc != "UTF8" {
			return fmt.Errorf("staging database encoding is %s; UTF8 is required for raw fidelity", enc)
		}
		// Serialise registration of the same declared and content identity
		// across concurrent loaders (fixed lock order: snapshot, then content).
		for _, k := range []string{"crm.integration.snapshot:" + m.SnapshotID, "crm.integration.content:" + content} {
			if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", k); err != nil {
				return err
			}
		}
		var foundSnapshot, foundContent string
		err := tx.QueryRow(ctx,
			`SELECT batch_id::text, snapshot_id, encode(content_sha256, 'hex'), state FROM integration.import_batch
			  WHERE snapshot_id = $1 OR content_sha256 = $2 ORDER BY (snapshot_id = $1) DESC LIMIT 1`,
			m.SnapshotID, mustHex(content)).Scan(&batchID, &foundSnapshot, &foundContent, &state)
		switch {
		case err == nil:
			if foundSnapshot == m.SnapshotID && foundContent != content {
				rep.set(result.Conflict, reasoncodes.LinkageConflict,
					"snapshot_id already staged with different content (batch "+batchID+"); nothing was written")
				return nil
			}
			if foundSnapshot != m.SnapshotID {
				rep.Warnings = append(rep.Warnings, "identical content is already staged as snapshot_id "+foundSnapshot+"; no new rows staged")
			}
			return nil
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		id, err := ids.NewUUID()
		if err != nil {
			return err
		}
		extracted, err := time.Parse(time.RFC3339Nano, m.ExtractedAt)
		if err != nil {
			return err
		}
		now := l.now()
		if _, err := tx.Exec(ctx, `INSERT INTO integration.import_batch
			(batch_id, snapshot_id, content_sha256, manifest_sha256, manifest_format_version, manifest_json,
			 legacy_application, legacy_version, legacy_build, source_id, source_timezone, source_table_prefix,
			 data_origin, extracted_at, extraction_tool, extraction_tool_version, import_tool_version,
			 state, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10,NULLIF($11,''),NULLIF($12,''),$13,$14,$15,$16,$17,$18,$19,$19)`,
			id, m.SnapshotID, mustHex(content), mustHex(p.ManifestSHA256), m.FormatVersion, string(manifestBytes),
			m.Legacy.Application, m.Legacy.Version, m.Legacy.Build, m.Source.ID, m.Source.Timezone, m.Source.TablePrefix,
			string(m.DataOrigin), extracted.UTC(), m.Tool.Name, m.Tool.Version, l.toolVersion,
			StateValidated, now); err != nil {
			return err
		}
		for _, e := range m.Entities {
			s, err := p.Spec(e.SpecID)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO integration.import_entity
				(batch_id, spec_id, source_table, staging_table, file_path, completeness, expected_rows, expected_bytes,
				 expected_file_sha256, expected_content_sha256, state, updated_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
				id, e.SpecID, s.SourceTable, s.StagingTable, e.File, string(e.Completeness), e.RowCount, e.ByteSize,
				mustHex(e.FileSHA256), mustHex(e.ContentSHA256), EntityPending, now); err != nil {
				return err
			}
		}
		batchID, state = id, StateValidated
		return nil
	})
	if err != nil {
		return "", "", err
	}
	rep.BatchState = state
	return batchID, state, nil
}

// loadEntity stages one entity in one transaction.
func (l *Loader) loadEntity(ctx context.Context, p *snapshot.Package, batchID string, e snapshot.Entity, er *EntityResult) error {
	s, err := p.Spec(e.SpecID)
	if err != nil {
		return err
	}
	return l.runner.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		var state string
		if err := tx.QueryRow(ctx, `SELECT state FROM integration.import_entity WHERE batch_id = $1 AND spec_id = $2 FOR UPDATE`,
			batchID, e.SpecID).Scan(&state); err != nil {
			return err
		}
		if state == EntityLoaded {
			er.State = EntityLoaded
			return nil // committed earlier (resume); reconcile re-verifies it
		}
		now := l.now()
		ins := newInserter(s, batchID, now)
		rep, err := p.ScanEntity(e, func(r snapshot.Row) error { return ins.add(ctx, tx, r) })
		if err != nil {
			return err
		}
		if rep.Status == result.Failed {
			return fmt.Errorf("%s", rep.Problem)
		}
		if err := ins.flush(ctx, tx); err != nil {
			return err
		}
		staged, sum, err := stagingChecksum(ctx, tx, s, batchID)
		if err != nil {
			return err
		}
		if staged != e.RowCount || sum != e.ContentSHA256 {
			return fmt.Errorf("staging fidelity check failed: staged %d rows, checksum match %t", staged, sum == e.ContentSHA256)
		}
		warnings, _ := json.Marshal(rep.Warnings)
		var code *string
		if rep.Reason != nil {
			c := string(*rep.Reason)
			code = &c
		}
		if _, err := tx.Exec(ctx, `UPDATE integration.import_entity SET state = $3, parsed_rows = $4, staged_rows = $5,
			actual_file_sha256 = $6, staging_sha256 = $7, reason_code = $8, failure_reason = NULL, warnings = $9,
			loaded_at = $10, updated_at = $10, actual_bytes = $11 WHERE batch_id = $1 AND spec_id = $2`,
			batchID, e.SpecID, EntityLoaded, *rep.ParsedRows, staged, mustHex(rep.ActualFileSHA256), mustHex(sum),
			code, string(warnings), now, *rep.ActualBytes); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE integration.import_batch SET state = $2, failure_reason = NULL, updated_at = $3
			WHERE batch_id = $1 AND state IN ('VALIDATED', 'LOADING', 'FAILED')`, batchID, StateLoading, now); err != nil {
			return err
		}
		er.EntityReport = rep
		er.State = EntityLoaded
		return nil
	})
}

// recordFailure marks the entity and batch FAILED after the entity
// transaction rolled back. Best effort: if it cannot be recorded, the batch
// simply stays not LOADED, which is still honest.
func (l *Loader) recordFailure(ctx context.Context, batchID, specID string, cause error) {
	reason := truncate(cause.Error(), 900)
	_ = l.runner.InTx(context.WithoutCancel(ctx), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		now := l.now()
		if _, err := tx.Exec(ctx, `UPDATE integration.import_entity SET state = $3, failure_reason = $4, loaded_at = NULL, updated_at = $5
			WHERE batch_id = $1 AND spec_id = $2 AND state <> 'LOADED'`, batchID, specID, EntityFailed, reason, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE integration.import_batch SET state = $2, failure_reason = $3, loaded_at = NULL, updated_at = $4
			WHERE batch_id = $1 AND state NOT IN ('LOADED', 'LOADED_INCOMPLETE')`, batchID, StateFailed, truncate("entity "+specID+": "+reason, 1000), now)
		return err
	})
}

func (l *Loader) finalize(ctx context.Context, batchID string) error {
	return l.runner.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		var incomplete int64
		if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE completeness <> 'COMPLETE') FROM integration.import_entity WHERE batch_id = $1`,
			batchID).Scan(&incomplete); err != nil {
			return err
		}
		target := StateLoaded
		if incomplete > 0 {
			target = StateLoadedIncomplete
		}
		now := l.now()
		_, err := tx.Exec(ctx, `UPDATE integration.import_batch SET state = $2, failure_reason = NULL, loaded_at = $3, updated_at = $3
			WHERE batch_id = $1 AND state NOT IN ('LOADED', 'LOADED_INCOMPLETE')`, batchID, target, now)
		return err
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// inserter batches multi-row INSERTs. Identifiers come only from validated
// specs; every value is a bind parameter.
type inserter struct {
	s       spec.Spec
	batchID string
	at      time.Time
	head    string
	perRow  int
	maxRows int
	args    []any
	rows    int
}

func newInserter(s spec.Spec, batchID string, at time.Time) *inserter {
	cols := []string{"batch_id", "spec_id", "source_ordinal", "source_key", "row_sha256", "imported_at"}
	for _, c := range s.Columns {
		cols = append(cols, c.StagingColumn())
	}
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = pgx.Identifier{c}.Sanitize()
	}
	per := len(cols)
	maxRows := maxParams / per
	if maxRows > 1000 {
		maxRows = 1000
	}
	return &inserter{
		s: s, batchID: batchID, at: at, perRow: per, maxRows: maxRows,
		head: "INSERT INTO " + pgx.Identifier{integration.Module.Schema(), s.StagingTable}.Sanitize() + " (" + strings.Join(quoted, ", ") + ") VALUES ",
	}
}

func (in *inserter) add(ctx context.Context, tx db.Tx, r snapshot.Row) error {
	in.args = append(in.args, in.batchID, in.s.ID, r.Ordinal, r.Key, snapshot.RowSHA256(r.Canonical), in.at)
	for _, c := range r.Cells {
		if c.Null {
			in.args = append(in.args, nil)
		} else {
			in.args = append(in.args, c.Text)
		}
	}
	in.rows++
	if in.rows >= in.maxRows {
		return in.flush(ctx, tx)
	}
	return nil
}

func (in *inserter) flush(ctx context.Context, tx db.Tx) error {
	if in.rows == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(in.head)
	n := 1
	for r := 0; r < in.rows; r++ {
		if r > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('(')
		for c := 0; c < in.perRow; c++ {
			if c > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, "$%d", n)
			if c == 5 {
				b.WriteString("::timestamptz")
			}
			n++
		}
		b.WriteByte(')')
	}
	if _, err := tx.Exec(ctx, b.String(), in.args...); err != nil {
		return err
	}
	in.args, in.rows = in.args[:0], 0
	return nil
}

// stagingChecksum re-encodes the staged raw values in source-ordinal order
// and recomputes the content checksum, also checking that ordinals are
// exactly 1..n and that every stored row_sha256 matches its re-encoded row.
func stagingChecksum(ctx context.Context, tx db.Tx, s spec.Spec, batchID string) (int64, string, error) {
	cols := make([]string, len(s.Columns))
	for i, c := range s.Columns {
		cols[i] = pgx.Identifier{c.StagingColumn()}.Sanitize()
	}
	q := "SELECT source_ordinal, row_sha256, " + strings.Join(cols, ", ") + " FROM " +
		pgx.Identifier{integration.Module.Schema(), s.StagingTable}.Sanitize() + " WHERE batch_id = $1 ORDER BY source_ordinal"
	rows, err := tx.Query(ctx, q, batchID)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	h := snapshot.NewContentHasher(s.ID)
	vals := make([]*string, len(s.Columns))
	dest := make([]any, 0, len(s.Columns)+2)
	var ordinal int64
	var rowSHA []byte
	dest = append(dest, &ordinal, &rowSHA)
	for i := range vals {
		dest = append(dest, &vals[i])
	}
	var n int64
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return 0, "", err
		}
		n++
		if ordinal != n {
			return 0, "", fmt.Errorf("%w: ordinals are not contiguous at %d", ErrFidelity, n)
		}
		cells := make([]snapshot.Cell, len(vals))
		for i, v := range vals {
			if v == nil {
				cells[i] = snapshot.Null()
			} else {
				cells[i] = snapshot.Text(*v)
			}
		}
		canon, err := snapshot.EncodeRow(cells)
		if err != nil {
			return 0, "", err
		}
		if hex.EncodeToString(snapshot.RowSHA256(canon)) != hex.EncodeToString(rowSHA) {
			return 0, "", fmt.Errorf("%w: ordinal %d does not match its row checksum", ErrFidelity, n)
		}
		h.AddRow(canon)
	}
	if err := rows.Err(); err != nil {
		return 0, "", err
	}
	return n, h.Sum(), nil
}
