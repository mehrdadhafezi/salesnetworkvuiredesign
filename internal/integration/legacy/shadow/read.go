// Package shadow is the Slice 4 legacy → target shadow import orchestration
// (Architecture V1 §24.4). It reads Slice 3 raw staging in integration-owned
// READ ONLY transactions, maps raw legacy values (mapping.go: the only place
// with legacy role/status knowledge) into neutral command records, and calls
// each module's published import command. It never writes identity or
// organization tables itself and never writes two modules in one
// transaction: identity commits first, organization resolves accounts
// through the published identity API and commits second.
//
// It reads staging only: never WordPress, MySQL, snapshot files or any live
// system. wp_usermeta meta_value is never read (its row scope is an
// unresolved Owner decision, see spec.ExtractionScopeUnresolved).
package shadow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/staging"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Spec ids consumed by Slice 4.
const (
	SpecUser           = "legacy.wp_user.v1"
	SpecUsermeta       = "legacy.wp_usermeta.v1"
	SpecProfile        = "legacy.hr_profile.v1"
	SpecPosition       = "legacy.hr_position.v1"
	SpecLevel          = "legacy.hr_level.v1"
	SpecRoleMapping    = "legacy.hr_position_role_mapping.v1"
	SpecDepartment     = "legacy.hr_department.v1"
	SpecUnit           = "legacy.hr_unit.v1"
	SpecTeam           = "legacy.hr_team.v1"
	SpecAssignment     = "legacy.hr_assignment.v1"
	SpecAssignmentHist = "legacy.hr_assignment_history.v1"
)

// ConsumedSpecs lists every staged spec Slice 4 reads, in a stable order.
func ConsumedSpecs() []string {
	return []string{SpecUser, SpecUsermeta, SpecProfile, SpecPosition, SpecLevel, SpecRoleMapping,
		SpecDepartment, SpecUnit, SpecTeam, SpecAssignment, SpecAssignmentHist}
}

// Errors.
var (
	// ErrBatchNotFound reports an unknown snapshot id.
	ErrBatchNotFound = errors.New("shadow: no staged batch for this snapshot_id")
	// ErrBatchNotLoaded reports a batch that is not LOADED / LOADED_INCOMPLETE.
	ErrBatchNotLoaded = errors.New("shadow: staged batch is not loaded")
)

// Row is one staged row: legacy key, staged checksum and raw columns.
type Row struct {
	Key string
	SHA []byte
	Raw map[string]*string // source column name → raw value (NULL = nil)
}

// Get returns a raw column value.
func (r Row) Get(col string) *string { return r.Raw[col] }

// Entity is one staged entity of the batch.
type Entity struct {
	SpecID       string
	State        string // staging entity state
	Completeness string
	Rows         []Row
}

// Usable reports whether the entity was staged (LOADED).
func (e Entity) Usable() bool { return e.State == staging.EntityLoaded }

// Complete reports a LOADED entity whose manifest declares COMPLETE.
func (e Entity) Complete() bool { return e.Usable() && e.Completeness == "COMPLETE" }

// Batch is one staged snapshot as Slice 4 reads it.
type Batch struct {
	BatchID       string
	SnapshotID    string
	State         string
	LegacyVersion string
	TablePrefix   *string
	Entities      map[string]Entity
	// UsermetaRows / UsermetaCapabilityRows count wp_usermeta rows and rows
	// whose meta_key is the prefixed capabilities key. meta_value is never read.
	UsermetaRows           int64
	UsermetaCapabilityRows int64
}

// Entity returns the staged entity for specID (zero value if absent).
func (b *Batch) Entity(specID string) Entity {
	if e, ok := b.Entities[specID]; ok {
		return e
	}
	return Entity{SpecID: specID, State: "ABSENT"}
}

// ReadBatch reads every Slice 4 entity of the staged snapshot.
func ReadBatch(ctx context.Context, staged *db.Runner, reg *spec.Registry, snapshotID string) (*Batch, error) {
	b := &Batch{SnapshotID: snapshotID, Entities: map[string]Entity{}}
	err := staged.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, func(ctx context.Context, tx db.Tx) error {
		err := tx.QueryRow(ctx, `SELECT batch_id::text, state, legacy_version, source_table_prefix FROM integration.import_batch WHERE snapshot_id = $1`,
			snapshotID).Scan(&b.BatchID, &b.State, &b.LegacyVersion, &b.TablePrefix)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBatchNotFound
		}
		if err != nil {
			return err
		}
		if b.State != staging.StateLoaded && b.State != staging.StateLoadedIncomplete {
			return fmt.Errorf("%w: state %s", ErrBatchNotLoaded, b.State)
		}
		rows, err := tx.Query(ctx, `SELECT spec_id, state, completeness FROM integration.import_entity WHERE batch_id = $1`, b.BatchID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var e Entity
			if err := rows.Scan(&e.SpecID, &e.State, &e.Completeness); err != nil {
				rows.Close()
				return err
			}
			b.Entities[e.SpecID] = e
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, id := range append(append(ConsumedSpecs(), CustomerSpecs()...), SalesSpecs()...) {
			e, ok := b.Entities[id]
			if !ok || !e.Usable() {
				continue
			}
			s, err := reg.ForVersion(id, b.LegacyVersion)
			if err != nil {
				return err
			}
			if id == SpecUsermeta {
				if err := countUsermeta(ctx, tx, s, b); err != nil {
					return err
				}
				continue
			}
			if e.Rows, err = readRows(ctx, tx, s, b.BatchID); err != nil {
				return err
			}
			b.Entities[id] = e
		}
		return nil
	})
	return b, err
}

func readRows(ctx context.Context, tx db.Tx, s spec.Spec, batchID string) ([]Row, error) {
	cols := make([]string, len(s.Columns))
	for i, c := range s.Columns {
		cols[i] = c.StagingColumn() // validated spec identifiers only
	}
	rows, err := tx.Query(ctx, `SELECT source_key, row_sha256, `+strings.Join(cols, ", ")+
		` FROM integration.`+s.StagingTable+` WHERE batch_id = $1 ORDER BY source_ordinal`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		vals := make([]*string, len(cols))
		dst := []any{new(string), new([]byte)}
		for i := range vals {
			dst = append(dst, &vals[i])
		}
		if err := rows.Scan(dst...); err != nil {
			return nil, err
		}
		r := Row{Key: *dst[0].(*string), SHA: *dst[1].(*[]byte), Raw: map[string]*string{}}
		for i, c := range s.Columns {
			r.Raw[c.Name] = vals[i]
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// countUsermeta counts rows and capability-key rows. It selects meta_key
// only; meta_value is never read, parsed or copied.
func countUsermeta(ctx context.Context, tx db.Tx, s spec.Spec, b *Batch) error {
	capKey := ""
	if b.TablePrefix != nil {
		capKey = *b.TablePrefix + "capabilities"
	}
	return tx.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE raw_meta_key = $2)
		FROM integration.`+s.StagingTable+` WHERE batch_id = $1`, b.BatchID, capKey).
		Scan(&b.UsermetaRows, &b.UsermetaCapabilityRows)
}
