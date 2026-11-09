package backup

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
)

// TableState is the recorded state of one table.
type TableState struct {
	Rows int64 `json:"rows"`
	// Checksum is md5 over the sorted md5 of every row's text form: it is
	// independent of physical order and comparable across dump / restore.
	Checksum string `json:"checksum"`
	Owner    string `json:"owner"`
}

// Migration is the state of one migration version table.
type Migration struct {
	Applied    int64 `json:"applied"`
	MaxVersion int64 `json:"max_version"`
}

// Inventory is the logical state of a database at one snapshot. It holds
// counts, digests and names only: no row content.
type Inventory struct {
	ServerVersion string                `json:"server_version"`
	Migrations    map[string]Migration  `json:"migrations"`
	Tables        map[string]TableState `json:"tables"`
	// Constraints counts constraints per schema and type (c, f, p, u, x).
	Constraints map[string]int64 `json:"constraints"`
	// NotValidated counts constraints that are not validated.
	NotValidated int64 `json:"not_validated"`
	// Grants counts table privileges per crm_* grantee.
	Grants map[string]int64 `json:"grants"`
	// States counts selected lifecycle states (UNKNOWN, reconciliation,
	// receipts, uploads, undispatched outbox events) per table.
	States map[string]int64 `json:"states"`
	// Flags is the feature-flag state (key → enabled).
	Flags map[string]bool `json:"flags"`
}

// stateColumns are the lifecycle columns summarised in States when present.
var stateColumns = [][3]string{
	{"integration", "writeback_command", "state"},
	{"integration", "provider_operation", "state"},
	{"payments", "receipt_evidence", "state"},
	{"payments", "upload_object", "state"},
	{"payments", "submitted_attempt", "state"},
}

func ident(s string) string { return pgx.Identifier{s}.Sanitize() }

// Capture records the inventory inside tx (which must be REPEATABLE READ so
// that every query and an exported snapshot see the same state).
func Capture(ctx context.Context, tx pgx.Tx) (Inventory, error) {
	inv := Inventory{Migrations: map[string]Migration{}, Tables: map[string]TableState{}, Constraints: map[string]int64{},
		Grants: map[string]int64{}, States: map[string]int64{}, Flags: map[string]bool{}}
	if err := tx.QueryRow(ctx, `SELECT current_setting('server_version')`).Scan(&inv.ServerVersion); err != nil {
		return inv, err
	}
	type tbl struct{ schema, name, owner string }
	var tables []tbl
	rows, err := tx.Query(ctx, `SELECT n.nspname, c.relname, pg_get_userbyid(c.relowner) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p') AND n.nspname NOT IN ('pg_catalog', 'information_schema') AND n.nspname NOT LIKE 'pg_toast%'
		AND n.nspname NOT LIKE 'pg_temp%' ORDER BY 1, 2`)
	if err != nil {
		return inv, err
	}
	for rows.Next() {
		var t tbl
		if err := rows.Scan(&t.schema, &t.name, &t.owner); err != nil {
			rows.Close()
			return inv, err
		}
		tables = append(tables, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return inv, err
	}
	for _, t := range tables {
		key := t.schema + "." + t.name
		var st TableState
		st.Owner = t.owner
		q := `SELECT count(*), coalesce(md5(string_agg(md5(x::text), '' ORDER BY md5(x::text))), '') FROM ` + ident(t.schema) + `.` + ident(t.name) + ` AS x`
		if err := tx.QueryRow(ctx, q).Scan(&st.Rows, &st.Checksum); err != nil {
			return inv, fmt.Errorf("table %s: %w", key, err)
		}
		inv.Tables[key] = st
		if t.schema == "public" && strings.HasPrefix(t.name, "crm_schema_migrations") {
			var m Migration
			if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE is_applied), coalesce(max(version_id) FILTER (WHERE is_applied), 0) FROM `+
				ident(t.schema)+`.`+ident(t.name)).Scan(&m.Applied, &m.MaxVersion); err != nil {
				return inv, fmt.Errorf("migrations %s: %w", key, err)
			}
			inv.Migrations[t.name] = m
		}
	}
	rows, err = tx.Query(ctx, `SELECT n.nspname || ':' || c.contype::text, count(*), count(*) FILTER (WHERE NOT c.convalidated)
		FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema') AND n.nspname NOT LIKE 'pg_toast%' GROUP BY 1`)
	if err != nil {
		return inv, err
	}
	for rows.Next() {
		var k string
		var n, nv int64
		if err := rows.Scan(&k, &n, &nv); err != nil {
			rows.Close()
			return inv, err
		}
		inv.Constraints[k] = n
		inv.NotValidated += nv
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT grantee, count(*) FROM information_schema.role_table_grants WHERE grantee LIKE 'crm\_%' GROUP BY 1`)
	if err != nil {
		return inv, err
	}
	for rows.Next() {
		var g string
		var n int64
		if err := rows.Scan(&g, &n); err != nil {
			rows.Close()
			return inv, err
		}
		inv.Grants[g] = n
	}
	rows.Close()
	for _, sc := range stateColumns {
		key := sc[0] + "." + sc[1]
		if _, ok := inv.Tables[key]; !ok {
			continue
		}
		rows, err := tx.Query(ctx, `SELECT `+ident(sc[2])+`::text, count(*) FROM `+ident(sc[0])+`.`+ident(sc[1])+` GROUP BY 1`)
		if err != nil {
			return inv, err
		}
		for rows.Next() {
			var v string
			var n int64
			if err := rows.Scan(&v, &n); err != nil {
				rows.Close()
				return inv, err
			}
			inv.States[key+"."+sc[2]+"="+v] = n
		}
		rows.Close()
	}
	for key := range inv.Tables {
		if strings.HasSuffix(key, ".outbox_event") {
			var n int64
			parts := strings.SplitN(key, ".", 2)
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM `+ident(parts[0])+`.outbox_event WHERE dispatch_state <> 'dispatched'`).Scan(&n); err != nil {
				return inv, err
			}
			inv.States[key+".undispatched"] = n
		}
	}
	if _, ok := inv.Tables["platform_featureflags.feature_flag"]; ok {
		rows, err := tx.Query(ctx, `SELECT flag_key, enabled FROM platform_featureflags.feature_flag`)
		if err != nil {
			return inv, err
		}
		for rows.Next() {
			var k string
			var on bool
			if err := rows.Scan(&k, &on); err != nil {
				rows.Close()
				return inv, err
			}
			inv.Flags[k] = on
		}
		rows.Close()
	}
	return inv, nil
}

// Diff lists every difference between a recorded and an observed inventory
// (empty: identical). The server version is reported only when the major
// version differs.
func Diff(want, got Inventory) []string {
	var d []string
	major := func(v string) string { return strings.SplitN(v, ".", 2)[0] }
	if major(want.ServerVersion) != major(got.ServerVersion) {
		d = append(d, "server major version "+want.ServerVersion+" → "+got.ServerVersion)
	}
	for _, k := range sortedKeys(want.Migrations) {
		if want.Migrations[k] != got.Migrations[k] {
			d = append(d, "migrations "+k)
		}
	}
	for _, k := range sortedKeys(got.Migrations) {
		if _, ok := want.Migrations[k]; !ok {
			d = append(d, "unexpected migrations "+k)
		}
	}
	for _, k := range sortedKeys(want.Tables) {
		g, ok := got.Tables[k]
		switch {
		case !ok:
			d = append(d, "missing table "+k)
		case g.Rows != want.Tables[k].Rows:
			d = append(d, fmt.Sprintf("rows %s %d → %d", k, want.Tables[k].Rows, g.Rows))
		case g.Checksum != want.Tables[k].Checksum:
			d = append(d, "checksum "+k)
		case g.Owner != want.Tables[k].Owner:
			d = append(d, "owner "+k)
		}
	}
	for _, k := range sortedKeys(got.Tables) {
		if _, ok := want.Tables[k]; !ok {
			d = append(d, "unexpected table "+k)
		}
	}
	for _, k := range sortedKeys(want.Constraints) {
		if want.Constraints[k] != got.Constraints[k] {
			d = append(d, "constraints "+k)
		}
	}
	if want.NotValidated != got.NotValidated {
		d = append(d, "not-validated constraints")
	}
	for _, k := range sortedKeys(want.Grants) {
		if want.Grants[k] != got.Grants[k] {
			d = append(d, "grants "+k)
		}
	}
	for _, k := range sortedKeys(want.States) {
		if want.States[k] != got.States[k] {
			d = append(d, "state "+k)
		}
	}
	return d
}

// EvidenceFinding is one evidence-consistency problem. It names the row
// identity and the problem only, never content.
type EvidenceFinding struct {
	Kind, ID, State, Problem string
}

// CheckEvidence verifies, for every receipt and upload row, that the bytes
// its state requires exist under root with the recorded SHA-256:
//
//   - READY receipts need evidence/<sha256>;
//   - QUARANTINED / STORAGE_ERROR / FINALIZING receipts and UNBOUND / BOUND
//     uploads need their quarantine object;
//   - NOT_RECOVERABLE receipts and EXPIRING / PURGED uploads need nothing.
//
// It never changes a row: a missing or corrupt object is a finding.
func CheckEvidence(ctx context.Context, tx pgx.Tx, root string) ([]EvidenceFinding, int, error) {
	var findings []EvidenceFinding
	checked := 0
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT to_regclass('payments.receipt_evidence') IS NOT NULL`).Scan(&exists); err != nil || !exists {
		return nil, 0, err
	}
	check := func(kind, id, state, key string, sum []byte) {
		checked++
		got, err := objectSHA(root, key)
		switch {
		case err != nil && os.IsNotExist(err):
			findings = append(findings, EvidenceFinding{kind, id, state, "MISSING"})
		case err != nil:
			findings = append(findings, EvidenceFinding{kind, id, state, "UNREADABLE"})
		case got != hex.EncodeToString(sum):
			findings = append(findings, EvidenceFinding{kind, id, state, "CORRUPT"})
		}
	}
	rows, err := tx.Query(ctx, `SELECT receipt_id::text, state, quarantine_ref, coalesce(evidence_ref, ''), content_sha256 FROM payments.receipt_evidence ORDER BY 1`)
	if err != nil {
		return nil, 0, err
	}
	type rec struct {
		id, state, q, e string
		sum             []byte
	}
	var recs []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.id, &r.state, &r.q, &r.e, &r.sum); err != nil {
			rows.Close()
			return nil, 0, err
		}
		recs = append(recs, r)
	}
	rows.Close()
	for _, r := range recs {
		switch r.state {
		case "READY":
			key := r.e
			if key == "" {
				key = "evidence/" + hex.EncodeToString(r.sum)
			}
			check("receipt", r.id, r.state, key, r.sum)
		case "QUARANTINED", "STORAGE_ERROR", "FINALIZING":
			check("receipt", r.id, r.state, r.q, r.sum)
		}
	}
	rows, err = tx.Query(ctx, `SELECT upload_id::text, state, quarantine_ref, content_sha256 FROM payments.upload_object WHERE state IN ('UNBOUND', 'BOUND') ORDER BY 1`)
	if err != nil {
		return nil, 0, err
	}
	var ups []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.id, &r.state, &r.q, &r.sum); err != nil {
			rows.Close()
			return nil, 0, err
		}
		ups = append(ups, r)
	}
	rows.Close()
	for _, u := range ups {
		check("upload", u.id, u.state, u.q, u.sum)
	}
	return findings, checked, nil
}

var keyRe = func(k string) bool {
	return k != "" && !strings.Contains(k, "..") && !strings.HasPrefix(k, "/") && len(k) < 200
}

func objectSHA(root, key string) (string, error) {
	if !keyRe(key) {
		return "", fmt.Errorf("invalid key")
	}
	s, _, err := sha256File(filepath.Join(root, filepath.FromSlash(key)))
	return s, err
}
