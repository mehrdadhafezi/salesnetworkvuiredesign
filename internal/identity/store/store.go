// Package store holds the identity module's SQL. It is private to identity
// (architecture lint: foreign-store). Every function takes a transaction
// owned by the identity module; read functions require a read-only one.
package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

var (
	// ErrWrongOwner reports a transaction not owned by identity.
	ErrWrongOwner = errors.New("identity/store: transaction is not owned by identity")
	// ErrNotReadOnly reports a read path given a writable transaction.
	ErrNotReadOnly = errors.New("identity/store: read path requires a read-only transaction")
	// ErrReadOnly reports a write path given a read-only transaction.
	ErrReadOnly = errors.New("identity/store: write path requires a writable transaction")
)

// Change classifies what an upsert did.
type Change string

const (
	Inserted   Change = "inserted"
	Refreshed  Change = "refreshed"  // same legacy key, different content: shadow refresh
	Reappeared Change = "reappeared" // was ABSENT_IN_LATEST, present again
	Rebatched  Change = "rebatched"  // identical content seen in a newer batch
	Unchanged  Change = "unchanged"
)

func writable(tx db.Tx) error {
	if tx.Module() != identity.Module {
		return ErrWrongOwner
	}
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	return nil
}

// ReadTx checks a read path transaction.
func ReadTx(tx db.Tx) error {
	if tx.Module() != identity.Module {
		return ErrWrongOwner
	}
	if !tx.ReadOnly() {
		return ErrNotReadOnly
	}
	return nil
}

// LockImport serialises identity import commands.
func LockImport(ctx context.Context, tx db.Tx) error {
	if err := writable(tx); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('crm.identity.import'))`)
	return err
}

// UpsertUser inserts or refreshes the shadow account of one legacy key.
func UpsertUser(ctx context.Context, tx db.Tx, u api.UserRecord, now time.Time) (string, Change, error) {
	if err := writable(tx); err != nil {
		return "", "", err
	}
	var (
		id, batch, presence string
		sha                 []byte
	)
	err := tx.QueryRow(ctx, `SELECT user_id::text, source_row_sha256, source_batch_id::text, presence
		FROM identity.user_account WHERE source_system = $1 AND source_key = $2 FOR UPDATE`,
		u.Source.System, u.Source.Key).Scan(&id, &sha, &batch, &presence)
	if errors.Is(err, pgx.ErrNoRows) {
		if id, err = ids.NewUUID(); err != nil {
			return "", "", err
		}
		_, err = tx.Exec(ctx, `INSERT INTO identity.user_account (user_id, source_system, source_spec, source_key,
			first_batch_id, source_batch_id, source_row_sha256, imported_at, refreshed_at, presence,
			login_evidence, display_name, registered_raw, account_state, login_enabled, version)
			VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $7, 'PRESENT', $8, $9, $10, 'SHADOW_IMPORTED', false, 1)`,
			id, u.Source.System, u.Source.Spec, u.Source.Key, u.Source.BatchID, u.Source.RowSHA256, now,
			u.Login, u.DisplayName, u.RegisteredRaw)
		return id, Inserted, err
	}
	if err != nil {
		return "", "", err
	}
	switch {
	case !bytes.Equal(sha, u.Source.RowSHA256):
		_, err = tx.Exec(ctx, `UPDATE identity.user_account SET source_spec = $2, source_batch_id = $3,
			source_row_sha256 = $4, refreshed_at = $5, presence = 'PRESENT', login_evidence = $6,
			display_name = $7, registered_raw = $8, version = version + 1 WHERE user_id = $1`,
			id, u.Source.Spec, u.Source.BatchID, u.Source.RowSHA256, now, u.Login, u.DisplayName, u.RegisteredRaw)
		return id, Refreshed, err
	case presence != "PRESENT":
		_, err = tx.Exec(ctx, `UPDATE identity.user_account SET presence = 'PRESENT', source_batch_id = $2,
			refreshed_at = $3, version = version + 1 WHERE user_id = $1`, id, u.Source.BatchID, now)
		return id, Reappeared, err
	case batch != u.Source.BatchID:
		_, err = tx.Exec(ctx, `UPDATE identity.user_account SET source_batch_id = $2, refreshed_at = $3
			WHERE user_id = $1`, id, u.Source.BatchID, now)
		return id, Rebatched, err
	}
	return id, Unchanged, nil
}

// MarkAbsentUsers flags accounts of system whose key is not in present.
func MarkAbsentUsers(ctx context.Context, tx db.Tx, system string, present []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `UPDATE identity.user_account SET presence = 'ABSENT_IN_LATEST', refreshed_at = $3,
		version = version + 1 WHERE source_system = $1 AND presence = 'PRESENT' AND NOT (source_key = ANY($2))`,
		system, present, now)
	return tag.RowsAffected(), err
}

// UserIDsByKey maps legacy keys of system to native user ids (any tx).
func UserIDsByKey(ctx context.Context, tx db.Tx, system string, keys []string) (map[string]string, error) {
	if tx.Module() != identity.Module {
		return nil, ErrWrongOwner
	}
	rows, err := tx.Query(ctx, `SELECT source_key, user_id::text FROM identity.user_account
		WHERE source_system = $1 AND source_key = ANY($2)`, system, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, id string
		if err := rows.Scan(&k, &id); err != nil {
			return nil, err
		}
		out[k] = id
	}
	return out, rows.Err()
}

// GrantBasisLegacyProfileRole is the only Slice 4 grant basis.
const GrantBasisLegacyProfileRole = "LEGACY_PROFILE_ROLE"

// UpsertShadowGrant records one role evidence row as a SHADOW grant.
func UpsertShadowGrant(ctx context.Context, tx db.Tx, e api.RoleEvidence, userID string, now time.Time) (Change, error) {
	if err := writable(tx); err != nil {
		return "", err
	}
	mapping := "UNMAPPED"
	var role *string
	if e.RoleKey != nil {
		s := string(*e.RoleKey)
		role, mapping = &s, "MAPPED"
	}
	var (
		id, curUser, curValue, batch, presence string
		curRole                                *string
	)
	err := tx.QueryRow(ctx, `SELECT grant_id::text, user_id::text, role_key, legacy_role_value, source_batch_id::text, presence
		FROM identity.role_grant WHERE basis = $1 AND evidence_key = $2 FOR UPDATE`,
		GrantBasisLegacyProfileRole, e.Evidence.Key).Scan(&id, &curUser, &curRole, &curValue, &batch, &presence)
	if errors.Is(err, pgx.ErrNoRows) {
		if id, err = ids.NewUUID(); err != nil {
			return "", err
		}
		_, err = tx.Exec(ctx, `INSERT INTO identity.role_grant (grant_id, user_id, role_key, legacy_role_value,
			mapping_state, basis, status, evidence_spec, evidence_key, first_batch_id, source_batch_id,
			imported_at, refreshed_at, presence, version)
			VALUES ($1, $2, $3, $4, $5, $6, 'SHADOW', $7, $8, $9, $9, $10, $10, 'PRESENT', 1)`,
			id, userID, role, e.LegacyValue, mapping, GrantBasisLegacyProfileRole, e.Evidence.Spec, e.Evidence.Key,
			e.Evidence.BatchID, now)
		return Inserted, err
	}
	if err != nil {
		return "", err
	}
	same := curUser == userID && curValue == e.LegacyValue && ((curRole == nil) == (role == nil)) && (curRole == nil || *curRole == *role)
	switch {
	case !same:
		_, err = tx.Exec(ctx, `UPDATE identity.role_grant SET user_id = $2, role_key = $3, legacy_role_value = $4,
			mapping_state = $5, evidence_spec = $6, source_batch_id = $7, refreshed_at = $8, presence = 'PRESENT',
			version = version + 1 WHERE grant_id = $1`,
			id, userID, role, e.LegacyValue, mapping, e.Evidence.Spec, e.Evidence.BatchID, now)
		return Refreshed, err
	case presence != "PRESENT":
		_, err = tx.Exec(ctx, `UPDATE identity.role_grant SET presence = 'PRESENT', source_batch_id = $2, refreshed_at = $3,
			version = version + 1 WHERE grant_id = $1`, id, e.Evidence.BatchID, now)
		return Reappeared, err
	case batch != e.Evidence.BatchID:
		_, err = tx.Exec(ctx, `UPDATE identity.role_grant SET source_batch_id = $2, refreshed_at = $3 WHERE grant_id = $1`,
			id, e.Evidence.BatchID, now)
		return Rebatched, err
	}
	return Unchanged, nil
}

// WithdrawGrant marks the shadow grant of one evidence row ABSENT_IN_LATEST
// when its evidence no longer resolves to an imported account, so a stale
// association never keeps evaluating as authority.
func WithdrawGrant(ctx context.Context, tx db.Tx, evidenceKey string, batchID string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `UPDATE identity.role_grant SET presence = 'ABSENT_IN_LATEST', source_batch_id = $3, refreshed_at = $4,
		version = version + 1 WHERE basis = $1 AND evidence_key = $2 AND presence = 'PRESENT'`,
		GrantBasisLegacyProfileRole, evidenceKey, batchID, now)
	return tag.RowsAffected(), err
}

// MarkAbsentGrants flags shadow grants whose evidence key is not in present.
func MarkAbsentGrants(ctx context.Context, tx db.Tx, present []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `UPDATE identity.role_grant SET presence = 'ABSENT_IN_LATEST', refreshed_at = $3,
		version = version + 1 WHERE basis = $1 AND presence = 'PRESENT' AND NOT (evidence_key = ANY($2))`,
		GrantBasisLegacyProfileRole, present, now)
	return tag.RowsAffected(), err
}

// PriorRun reports whether batchID was imported before.
func PriorRun(ctx context.Context, tx db.Tx, batchID string) (bool, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT count(*) FROM identity.import_run WHERE batch_id = $1`, batchID).Scan(&n)
	return n > 0, err
}

// InsertRun records one import run.
func InsertRun(ctx context.Context, tx db.Tx, runID, batchID, snapshotID, outcome string, replay bool, counts map[string]int64, now time.Time) error {
	if err := writable(tx); err != nil {
		return err
	}
	b, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO identity.import_run (run_id, batch_id, snapshot_id, outcome, replay, counts, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, runID, batchID, snapshotID, outcome, replay, b, now)
	return err
}

// --- read model ------------------------------------------------------------

// User is one account row as read back.
type User struct {
	UserID       string
	DisplayName  *string
	AccountState string
	LoginEnabled bool
	Source       api.Provenance
}

// UserSort is an allowlisted sort.
type UserSort struct {
	Key  string // public sort key
	Desc bool
}

// UserSorts maps public sort keys to their order. Values are never
// interpolated from input: only these constants reach SQL.
var UserSorts = map[string]UserSort{
	"display_name":  {Key: "display_name"},
	"-display_name": {Key: "-display_name", Desc: true},
}

// UserFilter is the allowlisted filter set.
type UserFilter struct {
	Query    *string // case-insensitive substring of the display name
	Presence *string // PRESENT | ABSENT_IN_LATEST
}

const userColumns = `user_id::text, display_name, account_state, login_enabled, source_system, source_spec,
	source_key, source_batch_id::text, imported_at, refreshed_at, presence`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.UserID, &u.DisplayName, &u.AccountState, &u.LoginEnabled, &u.Source.System, &u.Source.Spec,
		&u.Source.Key, &u.Source.BatchID, &u.Source.ImportedAt, &u.Source.RefreshedAt, &u.Source.Presence)
	u.Source.ImportedAt, u.Source.RefreshedAt = u.Source.ImportedAt.UTC(), u.Source.RefreshedAt.UTC()
	return u, err
}

// ListUsers returns one keyset page (limit+1 rows signal more).
func ListUsers(ctx context.Context, tx db.Tx, f UserFilter, sort UserSort, afterValue, afterID *string, limit int) ([]User, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	op, dir := ">", "ASC"
	if sort.Desc {
		op, dir = "<", "DESC"
	}
	q := `SELECT ` + userColumns + ` FROM identity.user_account
		WHERE ($1::text IS NULL OR display_name ILIKE '%' || $1 || '%')
		  AND ($2::text IS NULL OR presence = $2)
		  AND ($3::text IS NULL OR (COALESCE(display_name, ''), user_id) ` + op + ` ($3, $4::uuid))
		ORDER BY COALESCE(display_name, '') ` + dir + `, user_id ` + dir + ` LIMIT $5`
	var query *string
	if f.Query != nil {
		s := escapeLike(*f.Query)
		query = &s
	}
	rows, err := tx.Query(ctx, q, query, f.Presence, afterValue, afterID, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// GetUser returns one account.
func GetUser(ctx context.Context, tx db.Tx, id string) (User, bool, error) {
	if err := ReadTx(tx); err != nil {
		return User{}, false, err
	}
	u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM identity.user_account WHERE user_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	return u, err == nil, err
}

// Grants lists every stored grant.
func Grants(ctx context.Context, tx db.Tx) ([]api.Grant, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT user_id::text, role_key, legacy_role_value, status, basis
		FROM identity.role_grant WHERE presence = 'PRESENT' ORDER BY user_id, evidence_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []api.Grant
	for rows.Next() {
		var g api.Grant
		var role *string
		var status string
		if err := rows.Scan(&g.UserID, &role, &g.LegacyValue, &status, &g.Basis); err != nil {
			return nil, err
		}
		if role != nil {
			r := api.Role(*role)
			g.RoleKey = &r
		}
		g.Status = api.GrantStatus(status)
		out = append(out, g)
	}
	return out, rows.Err()
}

// Counts returns reconciliation counts of the stored model.
func Counts(ctx context.Context, tx db.Tx) (map[string]int64, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	out := map[string]int64{}
	queries := map[string]string{
		"users":                 `SELECT count(*) FROM identity.user_account`,
		"users_present":         `SELECT count(*) FROM identity.user_account WHERE presence = 'PRESENT'`,
		"users_login_enabled":   `SELECT count(*) FROM identity.user_account WHERE login_enabled`,
		"role_grants":           `SELECT count(*) FROM identity.role_grant`,
		"role_grants_mapped":    `SELECT count(*) FROM identity.role_grant WHERE mapping_state = 'MAPPED'`,
		"role_grants_unmapped":  `SELECT count(*) FROM identity.role_grant WHERE mapping_state = 'UNMAPPED'`,
		"role_grants_active":    `SELECT count(*) FROM identity.role_grant WHERE status <> 'SHADOW'`,
		"roles":                 `SELECT count(*) FROM identity.role`,
		"capabilities":          `SELECT count(*) FROM identity.capability`,
		"role_capability_pairs": `SELECT count(*) FROM identity.role_capability`,
	}
	for k, q := range queries {
		var n int64
		if err := tx.QueryRow(ctx, q).Scan(&n); err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
		out[k] = n
	}
	return out, nil
}

// Vocabulary returns the stored capability and role-bundle reference data.
func Vocabulary(ctx context.Context, tx db.Tx) (caps map[string]string, bundles map[string][]string, err error) {
	if err := ReadTx(tx); err != nil {
		return nil, nil, err
	}
	caps, bundles = map[string]string{}, map[string][]string{}
	rows, err := tx.Query(ctx, `SELECT capability_key, description FROM identity.capability`)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var k, d string
		if err := rows.Scan(&k, &d); err != nil {
			rows.Close()
			return nil, nil, err
		}
		caps[k] = d
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT r.role_key, rc.capability_key FROM identity.role r
		LEFT JOIN identity.role_capability rc USING (role_key) ORDER BY r.role_key, rc.capability_key`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r string
		var c *string
		if err := rows.Scan(&r, &c); err != nil {
			return nil, nil, err
		}
		if _, ok := bundles[r]; !ok {
			bundles[r] = []string{}
		}
		if c != nil {
			bundles[r] = append(bundles[r], *c)
		}
	}
	return caps, bundles, rows.Err()
}

func escapeLike(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == '%' || c == '_' || c == '\\' {
			b = append(b, '\\')
		}
		b = append(b, s[i])
	}
	return string(b)
}
