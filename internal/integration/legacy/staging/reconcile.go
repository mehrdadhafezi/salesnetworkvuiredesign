package staging

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// ErrBatchNotFound reports an unknown snapshot_id.
var ErrBatchNotFound = errors.New("staging: no batch for this snapshot_id")

// Reconcile recomputes, in a read-only transaction, every entity's staged
// row count and staging fidelity checksum and compares them with the
// manifest values recorded at registration. It never writes.
func (l *Loader) Reconcile(ctx context.Context, reg *spec.Registry, snapshotID string) (Report, error) {
	rep := Report{Command: "reconcile", SnapshotID: snapshotID, Entities: []EntityResult{}, Warnings: []string{}}
	err := l.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, func(ctx context.Context, tx db.Tx) error {
		var batchID string
		err := tx.QueryRow(ctx, `SELECT batch_id::text FROM integration.import_batch WHERE snapshot_id = $1`, snapshotID).Scan(&batchID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBatchNotFound
		}
		if err != nil {
			return err
		}
		return fillReconciliation(ctx, tx, reg, &rep, batchID)
	})
	if errors.Is(err, ErrBatchNotFound) {
		rep.set(result.Failed, reasoncodes.SourceUnavailable, "no staged batch for snapshot_id "+snapshotID)
		return rep, nil
	}
	return finish(&rep, err)
}

// reconcileInto replaces the report's entity results with a read-only
// reconciliation of the committed batch.
func (l *Loader) reconcileInto(ctx context.Context, rep *Report, batchID, cmd string) (Report, error) {
	prior := map[string]snapshot.EntityReport{}
	for _, e := range rep.Entities {
		prior[e.SpecID] = e.EntityReport
	}
	err := l.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, func(ctx context.Context, tx db.Tx) error {
		rep.Entities = rep.Entities[:0]
		return fillReconciliation(ctx, tx, nil, rep, batchID)
	})
	if err != nil {
		return finish(rep, err)
	}
	// Counts and file checksums come from the persisted observations; this
	// run's scan adds only what is not persisted (content checksum, NULL
	// counts) and only when it actually measured them.
	for i := range rep.Entities {
		if p, ok := prior[rep.Entities[i].SpecID]; ok && p.ParsedRows != nil {
			rep.Entities[i].ActualContentSHA = p.ActualContentSHA
			rep.Entities[i].NullInNotNull = p.NullInNotNull
		}
	}
	rep.Command = cmd
	return *rep, nil
}

type entityRow struct {
	specID, sourceTable, completeness, state string
	expectedRows, expectedBytes              int64
	expectedFile, expectedContent            []byte
	parsedRows, actualBytes, stagedRows      *int64
	actualFile                               []byte
	reasonCode, failureReason                *string
	warnings                                 []byte
}

func fillReconciliation(ctx context.Context, tx db.Tx, reg *spec.Registry, rep *Report, batchID string) error {
	if reg == nil {
		reg = spec.Default()
	}
	var state, legacyVersion, origin string
	var content, manifest []byte
	if err := tx.QueryRow(ctx, `SELECT state, legacy_version, data_origin, content_sha256, manifest_sha256
		FROM integration.import_batch WHERE batch_id = $1`, batchID).Scan(&state, &legacyVersion, &origin, &content, &manifest); err != nil {
		return err
	}
	rep.BatchID, rep.BatchState, rep.LegacyVersion, rep.DataOrigin = batchID, state, legacyVersion, origin
	rep.ContentSHA256, rep.ManifestSHA256 = hex.EncodeToString(content), hex.EncodeToString(manifest)

	rows, err := tx.Query(ctx, `SELECT spec_id, source_table, completeness, state, expected_rows, expected_bytes,
		expected_file_sha256, expected_content_sha256, parsed_rows, actual_bytes, actual_file_sha256, staged_rows,
		reason_code, failure_reason, warnings
		FROM integration.import_entity WHERE batch_id = $1 ORDER BY spec_id`, batchID)
	if err != nil {
		return err
	}
	var ents []entityRow
	for rows.Next() {
		var e entityRow
		if err := rows.Scan(&e.specID, &e.sourceTable, &e.completeness, &e.state, &e.expectedRows, &e.expectedBytes,
			&e.expectedFile, &e.expectedContent, &e.parsedRows, &e.actualBytes, &e.actualFile, &e.stagedRows,
			&e.reasonCode, &e.failureReason, &e.warnings); err != nil {
			rows.Close()
			return err
		}
		ents = append(ents, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	statuses := make([]result.Status, 0, len(ents))
	for _, e := range ents {
		s, err := reg.ForVersion(e.specID, legacyVersion)
		if err != nil {
			return err
		}
		er := EntityResult{EntityReport: snapshot.EntityReport{
			SpecID: e.specID, SourceTable: e.sourceTable, Classification: string(s.Classification()),
			Completeness: snapshot.Completeness(e.completeness), ExpectedRows: e.expectedRows, ExpectedBytes: e.expectedBytes,
			ExpectedFileSHA256: hex.EncodeToString(e.expectedFile), ExpectedContentSHA: hex.EncodeToString(e.expectedContent),
			ParsedRows: e.parsedRows, ActualBytes: e.actualBytes, Warnings: []string{},
		}, State: e.state}
		if e.actualFile != nil {
			er.ActualFileSHA256 = hex.EncodeToString(e.actualFile)
		}
		if len(e.warnings) > 0 {
			_ = json.Unmarshal(e.warnings, &er.Warnings)
		}
		staged, sum, err := stagingChecksum(ctx, tx, s, batchID)
		if errors.Is(err, ErrFidelity) {
			code := reasoncodes.LinkageConflict
			er.Status, er.Reason, er.Problem = result.Conflict, &code, err.Error()
			statuses = append(statuses, er.Status)
			rep.Entities = append(rep.Entities, er)
			continue
		}
		if err != nil {
			return err
		}
		er.StagedRows, er.StagingSHA256 = &staged, sum
		switch e.state {
		case EntityLoaded:
			if staged != e.expectedRows || sum != er.ExpectedContentSHA {
				code := reasoncodes.LinkageConflict
				er.Status, er.Reason = result.Conflict, &code
				er.Problem = fmt.Sprintf("staging does not match the manifest (staged %d of %d rows, checksum match %t)", staged, e.expectedRows, sum == er.ExpectedContentSHA)
			} else {
				er.Status = snapshot.CompletenessStatus(er.Completeness)
				if er.Status != result.Success {
					code := reasoncodes.CoverageIncomplete
					er.Reason, er.Problem = &code, "manifest declares completeness "+e.completeness
				}
			}
		case EntityFailed:
			code := reasoncodes.ContractInvalid
			if e.reasonCode != nil {
				if c, err := reasoncodes.ParseCode(*e.reasonCode); err == nil {
					code = c
				}
			}
			er.Status, er.Reason = result.Failed, &code
			if e.failureReason != nil {
				er.Problem = *e.failureReason
			}
		default:
			code := reasoncodes.CoverageIncomplete
			er.Status, er.Reason, er.Problem = result.Unknown, &code, "entity is not loaded ("+e.state+")"
		}
		if e.state != EntityLoaded && staged != 0 {
			code := reasoncodes.LinkageConflict
			er.Status, er.Reason, er.Problem = result.Conflict, &code, fmt.Sprintf("entity is %s but %d staged rows exist", e.state, staged)
		}
		statuses = append(statuses, er.Status)
		rep.Entities = append(rep.Entities, er)
	}
	if len(statuses) == 0 {
		rep.set(result.Failed, reasoncodes.ContractInvalid, "batch has no entities")
		return nil
	}
	agg, err := result.Aggregate(statuses)
	if err != nil {
		return err
	}
	rep.Outcome, rep.Reason, rep.Problem = agg, nil, ""
	if agg != result.Success {
		code := reasoncodes.CoverageIncomplete
		switch agg {
		case result.Conflict:
			code = reasoncodes.LinkageConflict
		case result.Failed:
			code = reasoncodes.ContractInvalid
		}
		rep.Reason = &code
	}
	if (state == StateLoaded) != (agg == result.Success) && state != StateLoadedIncomplete {
		rep.Warnings = append(rep.Warnings, "batch state "+state+" disagrees with reconciled outcome "+agg.String())
	}
	return nil
}
