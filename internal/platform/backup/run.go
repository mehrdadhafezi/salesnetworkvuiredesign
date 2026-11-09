package backup

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// FileEntry is one file of a backup.
type FileEntry struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Object is one evidence object in the archive.
type Object struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// EvidenceSet describes the evidence archive.
type EvidenceSet struct {
	Archive FileEntry         `json:"archive"`
	Objects map[string]Object `json:"objects"`
	// Order documents the consistency strategy: the database snapshot is
	// taken FIRST, the objects copied AFTER (evidence objects are immutable
	// and never deleted while referenced, so every object the snapshot
	// references is present unless storage lost it; objects added after
	// the snapshot are harmless extras).
	Order string `json:"order"`
}

// Manifest describes one backup. It contains no credential, host, DSN or
// row content.
type Manifest struct {
	Format          int          `json:"format"`
	ID              string       `json:"id"`
	Complete        bool         `json:"complete"`
	CreatedAt       time.Time    `json:"created_at"`
	Environment     string       `json:"environment"`
	SourceDatabase  string       `json:"source_database"`
	ToolRevision    string       `json:"tool_revision"`
	PgDumpVersion   string       `json:"pg_dump_version"`
	Kind            string       `json:"kind"`
	Consistency     string       `json:"consistency"`
	Encrypted       bool         `json:"encrypted"`
	RecipientsSHA   string       `json:"recipients_sha256,omitempty"`
	Database        FileEntry    `json:"database"`
	PlainSHA256     string       `json:"database_plain_sha256"`
	Inventory       Inventory    `json:"inventory"`
	Evidence        *EvidenceSet `json:"evidence,omitempty"`
	DurationSeconds float64      `json:"duration_seconds"`
}

// BackupConfig configures a backup.
type BackupConfig struct {
	Environment  string
	SourceURL    string
	Dest         string
	AllowedRoots []string
	EvidenceRoot string
	// Recipients is an age recipients file. Empty requires AllowPlaintext,
	// which is accepted only for development.
	Recipients     string
	AllowPlaintext bool
	Tools          Tools
}

func toolRevision() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				return s.Value
			}
		}
	}
	return "unknown"
}

func toolVersion(ctx context.Context, bin string) string {
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// encryptTo starts age writing to path, returning its stdin; plaintext
// writes path directly.
type sink struct {
	w    io.WriteCloser
	wait func() error
}

func openSink(ctx context.Context, path, recipients string, t Tools) (sink, error) {
	if recipients == "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return sink{}, err
		}
		// Sync must happen while the file is open: callers Close before wait.
		return sink{w: syncCloser{f}, wait: func() error { return nil }}, nil
	}
	cmd := exec.CommandContext(ctx, t.age(), "-R", recipients, "-o", path)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	var stderr strings.Builder
	cmd.Stderr = &limited{w: &stderr, n: 2048}
	in, err := cmd.StdinPipe()
	if err != nil {
		return sink{}, err
	}
	if err := cmd.Start(); err != nil {
		return sink{}, fmt.Errorf("age: %s", redact(err.Error()))
	}
	return sink{w: in, wait: func() error {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("age failed: %s", redact(stderr.String()))
		}
		return nil
	}}, nil
}

// syncCloser flushes a plaintext file to stable storage before closing it.
type syncCloser struct{ f *os.File }

func (s syncCloser) Write(p []byte) (int, error) { return s.f.Write(p) }

func (s syncCloser) Close() error {
	return errors.Join(s.f.Sync(), s.f.Close())
}

// openSource decrypts path (or opens it) and returns a reader.
func openSource(ctx context.Context, path string, encrypted bool, identity string, t Tools) (io.ReadCloser, func() error, error) {
	if !encrypted {
		f, err := os.Open(path)
		return f, func() error { return nil }, err
	}
	if identity == "" {
		return nil, nil, fmt.Errorf("%w: the backup is encrypted; an age identity is required", ErrRefused)
	}
	cmd := exec.CommandContext(ctx, t.age(), "-d", "-i", identity, path)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	var stderr strings.Builder
	cmd.Stderr = &limited{w: &stderr, n: 2048}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return out, func() error {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("%w: decryption failed: %s", ErrIntegrity, redact(stderr.String()))
		}
		return nil
	}, nil
}

// Backup takes a consistent logical backup (and optionally the evidence
// store) into a new directory.
func Backup(ctx context.Context, cfg BackupConfig) (Manifest, error) {
	start := time.Now()
	m := Manifest{Format: FormatVersion, ID: newID(), CreatedAt: start.UTC(), Environment: cfg.Environment, ToolRevision: toolRevision(),
		Kind:        "logical (pg_dump custom format); not physical, not point-in-time recovery",
		Consistency: "pg_dump --snapshot of the REPEATABLE READ transaction that captured the inventory"}
	switch cfg.Environment {
	case Development, Staging:
	default:
		return m, fmt.Errorf("%w: backups are enabled only for development and staging (R3); %q refused", ErrRefused, cfg.Environment)
	}
	if cfg.Recipients == "" && !(cfg.AllowPlaintext && cfg.Environment == Development) {
		return m, fmt.Errorf("%w: an age recipients file is required (plaintext only in development, explicitly allowed)", ErrRefused)
	}
	dest, err := underAllowed(cfg.Dest, cfg.AllowedRoots)
	if err != nil {
		return m, err
	}
	partial := dest + ".partial"
	if err := mustNotExist(dest, partial); err != nil {
		return m, err
	}
	env, dbname, err := pgEnv(cfg.SourceURL)
	if err != nil {
		return m, err
	}
	m.SourceDatabase = dbname
	if cfg.Recipients != "" {
		s, _, err := sha256File(cfg.Recipients)
		if err != nil {
			return m, fmt.Errorf("%w: recipients file unreadable", ErrRefused)
		}
		m.RecipientsSHA, m.Encrypted = s, true
	}
	conn, err := pgx.Connect(ctx, cfg.SourceURL)
	if err != nil {
		return m, fmt.Errorf("%w: source database unavailable", ErrRefused)
	}
	defer conn.Close(context.WithoutCancel(ctx))
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return m, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := attest(ctx, tx, cfg.Environment); err != nil {
		return m, err
	}
	if m.Inventory, err = Capture(ctx, tx); err != nil {
		return m, fmt.Errorf("inventory: %w", err)
	}
	var snapshot string
	if err := tx.QueryRow(ctx, `SELECT pg_export_snapshot()`).Scan(&snapshot); err != nil {
		return m, err
	}
	if err := os.Mkdir(partial, 0o700); err != nil {
		return m, fmt.Errorf("%w: cannot create the backup directory", ErrRefused)
	}
	m.PgDumpVersion = toolVersion(ctx, cfg.Tools.pg("pg_dump"))
	name := "database.dump"
	if m.Encrypted {
		name += ".age"
	}
	sk, err := openSink(ctx, filepath.Join(partial, name), cfg.Recipients, cfg.Tools)
	if err != nil {
		return m, err
	}
	plain := sha256.New()
	dumpErr := runCmd(ctx, cfg.Tools.pg("pg_dump"), []string{"--format=custom", "--no-password", "--snapshot=" + snapshot}, env, nil,
		io.MultiWriter(plain, sk.w))
	closeErr := sk.w.Close()
	waitErr := sk.wait()
	if err := errors.Join(dumpErr, closeErr, waitErr); err != nil {
		return m, fmt.Errorf("database dump: %w", err)
	}
	_ = tx.Rollback(ctx) // the snapshot is no longer needed
	m.PlainSHA256 = hex.EncodeToString(plain.Sum(nil))
	sum, size, err := sha256File(filepath.Join(partial, name))
	if err != nil {
		return m, err
	}
	m.Database = FileEntry{Name: name, SHA256: sum, Size: size}
	if cfg.EvidenceRoot != "" {
		ev, err := archiveEvidence(ctx, cfg.EvidenceRoot, partial, cfg.Recipients, cfg.Tools)
		if err != nil {
			return m, fmt.Errorf("evidence: %w", err)
		}
		m.Evidence = ev
	}
	m.Complete = true
	m.DurationSeconds = since(start)
	if err := writeJSON(filepath.Join(partial, "manifest.json"), m); err != nil {
		return m, err
	}
	ms, _, err := sha256File(filepath.Join(partial, "manifest.json"))
	if err != nil {
		return m, err
	}
	if err := os.WriteFile(filepath.Join(partial, "manifest.sha256"), []byte(ms+"  manifest.json\n"), 0o600); err != nil {
		return m, err
	}
	if err := os.Rename(partial, dest); err != nil {
		return m, err
	}
	return m, syncDir(filepath.Dir(dest))
}

func syncDir(p string) error {
	d, err := os.Open(p)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// archiveEvidence writes every regular file under root/quarantine and
// root/evidence into one (encrypted) tar, recording each object's digest.
func archiveEvidence(ctx context.Context, root, dir, recipients string, t Tools) (*EvidenceSet, error) {
	name := "evidence.tar"
	if recipients != "" {
		name += ".age"
	}
	ev := &EvidenceSet{Objects: map[string]Object{}, Order: "database snapshot first, evidence objects copied after"}
	sk, err := openSink(ctx, filepath.Join(dir, name), recipients, t)
	if err != nil {
		return nil, err
	}
	tw := tar.NewWriter(sk.w)
	var walkErr error
	for _, sub := range []string{"quarantine", "evidence"} {
		base := filepath.Join(root, sub)
		if _, err := os.Stat(base); errors.Is(err, os.ErrNotExist) {
			continue
		}
		walkErr = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !d.Type().IsRegular() {
				return fmt.Errorf("non-regular object %s refused", filepath.Base(p))
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			key := filepath.ToSlash(rel)
			if strings.Contains(filepath.Base(p), ".tmp") {
				return nil // in-flight temporary writes are not objects
			}
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			defer f.Close()
			st, err := f.Stat()
			if err != nil {
				return err
			}
			if err := tw.WriteHeader(&tar.Header{Name: key, Mode: 0o600, Size: st.Size(), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg}); err != nil {
				return err
			}
			h := sha256.New()
			n, err := io.Copy(io.MultiWriter(tw, h), f)
			if err != nil {
				return err
			}
			ev.Objects[key] = Object{SHA256: hex.EncodeToString(h.Sum(nil)), Size: n}
			return nil
		})
		if walkErr != nil {
			break
		}
	}
	err = errors.Join(walkErr, tw.Close(), sk.w.Close(), sk.wait())
	if err != nil {
		return nil, err
	}
	sum, size, err := sha256File(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	ev.Archive = FileEntry{Name: name, SHA256: sum, Size: size}
	return ev, nil
}

// ReadManifest loads and integrity-checks a backup's manifest.
func ReadManifest(dir string) (Manifest, error) {
	var m Manifest
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return m, fmt.Errorf("%w: no manifest (incomplete or interrupted backup)", ErrIntegrity)
	}
	want, err := os.ReadFile(filepath.Join(dir, "manifest.sha256"))
	if err != nil {
		return m, fmt.Errorf("%w: no manifest checksum", ErrIntegrity)
	}
	got := sha256.Sum256(raw)
	if strings.Fields(string(want))[0] != hex.EncodeToString(got[:]) {
		return m, fmt.Errorf("%w: manifest checksum mismatch", ErrIntegrity)
	}
	if err := json.Unmarshal(raw, &m); err != nil || m.Format != FormatVersion || !m.Complete {
		return m, fmt.Errorf("%w: manifest unreadable or incomplete", ErrIntegrity)
	}
	return m, nil
}

// VerifyReport is the result of Verify.
type VerifyReport struct {
	ID              string  `json:"id"`
	DatabaseOK      bool    `json:"database_ok"`
	ArchiveEntries  int     `json:"archive_entries"`
	ObjectsVerified int     `json:"objects_verified"`
	DurationSeconds float64 `json:"duration_seconds"`
}

// Verify checks a backup end to end: manifest, file digests, decryption,
// archive structure (pg_restore --list) and every evidence object's digest.
func Verify(ctx context.Context, dir, identity string, t Tools) (VerifyReport, error) {
	start := time.Now()
	var r VerifyReport
	m, err := ReadManifest(dir)
	if err != nil {
		return r, err
	}
	r.ID = m.ID
	if sum, _, err := sha256File(filepath.Join(dir, m.Database.Name)); err != nil || sum != m.Database.SHA256 {
		return r, fmt.Errorf("%w: database file digest", ErrIntegrity)
	}
	src, wait, err := openSource(ctx, filepath.Join(dir, m.Database.Name), m.Encrypted, identity, t)
	if err != nil {
		return r, err
	}
	plain := sha256.New()
	var listing strings.Builder
	listErr := runCmd(ctx, t.pg("pg_restore"), []string{"--list"}, []string{"PATH=" + os.Getenv("PATH"), "LC_ALL=C"}, io.TeeReader(src, plain),
		&limited{w: &listing, n: 1 << 20})
	_, _ = io.Copy(io.Discard, src)
	src.Close()
	if err := errors.Join(listErr, wait()); err != nil {
		return r, fmt.Errorf("%w: database archive unreadable: %v", ErrIntegrity, err)
	}
	if hex.EncodeToString(plain.Sum(nil)) != m.PlainSHA256 {
		return r, fmt.Errorf("%w: decrypted database digest", ErrIntegrity)
	}
	for _, line := range strings.Split(listing.String(), "\n") {
		if strings.Contains(line, " TABLE DATA ") {
			r.ArchiveEntries++
		}
	}
	r.DatabaseOK = true
	if m.Evidence != nil {
		n, err := walkArchive(ctx, dir, m, identity, t, nil)
		if err != nil {
			return r, err
		}
		r.ObjectsVerified = n
	}
	r.DurationSeconds = since(start)
	return r, nil
}

// walkArchive verifies every archived object against the manifest and,
// when extract is set, writes it under extract.
func walkArchive(ctx context.Context, dir string, m Manifest, identity string, t Tools, extract *string) (int, error) {
	if sum, _, err := sha256File(filepath.Join(dir, m.Evidence.Archive.Name)); err != nil || sum != m.Evidence.Archive.SHA256 {
		return 0, fmt.Errorf("%w: evidence archive digest", ErrIntegrity)
	}
	src, wait, err := openSource(ctx, filepath.Join(dir, m.Evidence.Archive.Name), m.Encrypted, identity, t)
	if err != nil {
		return 0, err
	}
	defer src.Close()
	tr := tar.NewReader(src)
	seen := map[string]bool{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("%w: evidence archive truncated or corrupt", ErrIntegrity)
		}
		want, ok := m.Evidence.Objects[h.Name]
		if !ok || !keyRe(h.Name) || h.Typeflag != tar.TypeReg {
			return 0, fmt.Errorf("%w: unexpected archive entry", ErrIntegrity)
		}
		hs := sha256.New()
		var w io.Writer = hs
		var f *os.File
		if extract != nil {
			p := filepath.Join(*extract, filepath.FromSlash(h.Name))
			if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
				return 0, err
			}
			if f, err = os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600); err != nil {
				return 0, err
			}
			w = io.MultiWriter(hs, f)
		}
		_, cerr := io.Copy(w, tr)
		if f != nil {
			cerr = errors.Join(cerr, f.Sync(), f.Close())
		}
		if cerr != nil {
			return 0, cerr
		}
		if hex.EncodeToString(hs.Sum(nil)) != want.SHA256 {
			return 0, fmt.Errorf("%w: archived object digest mismatch", ErrIntegrity)
		}
		seen[h.Name] = true
	}
	if err := wait(); err != nil {
		return 0, err
	}
	if len(seen) != len(m.Evidence.Objects) {
		return 0, fmt.Errorf("%w: evidence archive is missing objects (%d of %d)", ErrIntegrity, len(seen), len(m.Evidence.Objects))
	}
	return len(seen), nil
}

// RestoreConfig configures a restore.
type RestoreConfig struct {
	// AdminURL connects to the maintenance database of the TARGET cluster
	// with a role that may CREATE DATABASE. The owner roles referenced by
	// the backup must exist in that cluster.
	AdminURL     string
	Target       string // new database, crm_restore_*
	BackupDir    string
	Identity     string // age identity file (encrypted backups)
	EvidenceDest string // new directory for evidence objects (optional)
	AllowedRoots []string
	Tools        Tools
}

// RestoreReport is the measured result of a restore.
type RestoreReport struct {
	BackupID          string             `json:"backup_id"`
	Target            string             `json:"target"`
	Status            string             `json:"status"` // COMPLETE or FAILED
	Differences       []string           `json:"differences"`
	EvidenceChecked   int                `json:"evidence_checked"`
	EvidenceFindings  []EvidenceFinding  `json:"evidence_findings"`
	QuarantinedFlags  []string           `json:"quarantined_flags"`
	TablesCompared    int                `json:"tables_compared"`
	TimingsSeconds    map[string]float64 `json:"timings_seconds"`
	MeasuredRecoveryS float64            `json:"measured_recovery_seconds"`
}

func marker(id, status string) string { return "crm-backup:restore:" + id + ":" + status }

// Restore restores a verified backup into a NEW disposable database (and
// evidence directory), verifies it against the manifest and quarantines it.
func Restore(ctx context.Context, cfg RestoreConfig) (RestoreReport, error) {
	start := time.Now()
	rep := RestoreReport{Target: cfg.Target, Status: "FAILED", TimingsSeconds: map[string]float64{}, Differences: []string{},
		EvidenceFindings: []EvidenceFinding{}, QuarantinedFlags: []string{}}
	if !restoreName.MatchString(cfg.Target) {
		return rep, fmt.Errorf("%w: the target must be a new database named crm_restore_*", ErrRefused)
	}
	var evDest string
	if cfg.EvidenceDest != "" {
		var err error
		if evDest, err = underAllowed(cfg.EvidenceDest, cfg.AllowedRoots); err != nil {
			return rep, err
		}
		if err := mustNotExist(evDest, evDest+".partial"); err != nil {
			return rep, err
		}
	}
	t0 := time.Now()
	vr, err := Verify(ctx, cfg.BackupDir, cfg.Identity, cfg.Tools)
	rep.TimingsSeconds["verify"] = since(t0)
	if err != nil {
		return rep, err
	}
	m, _ := ReadManifest(cfg.BackupDir)
	rep.BackupID = vr.ID
	if m.Evidence != nil && evDest == "" {
		return rep, fmt.Errorf("%w: the backup carries evidence; an evidence destination is required", ErrRefused)
	}
	admin, err := pgx.Connect(ctx, cfg.AdminURL)
	if err != nil {
		return rep, fmt.Errorf("%w: target cluster unavailable", ErrRefused)
	}
	defer admin.Close(context.WithoutCancel(ctx))
	var exists bool
	if err := admin.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, cfg.Target).Scan(&exists); err != nil {
		return rep, err
	}
	if exists {
		return rep, fmt.Errorf("%w: database %s already exists (a restore never overwrites)", ErrRefused, cfg.Target)
	}
	t0 = time.Now()
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+ident(cfg.Target)); err != nil {
		return rep, fmt.Errorf("create database: %w", err)
	}
	if _, err := admin.Exec(ctx, `COMMENT ON DATABASE `+ident(cfg.Target)+` IS '`+marker(vr.ID, "INCOMPLETE")+`'`); err != nil {
		return rep, err
	}
	targetURL, err := withDatabase(cfg.AdminURL, cfg.Target)
	if err != nil {
		return rep, err
	}
	env, _, err := pgEnv(targetURL)
	if err != nil {
		return rep, err
	}
	src, wait, err := openSource(ctx, filepath.Join(cfg.BackupDir, m.Database.Name), m.Encrypted, cfg.Identity, cfg.Tools)
	if err != nil {
		return rep, err
	}
	rerr := runCmd(ctx, cfg.Tools.pg("pg_restore"), []string{"--no-password", "--exit-on-error", "--single-transaction", "--dbname=" + cfg.Target},
		env, src, io.Discard)
	_, _ = io.Copy(io.Discard, src)
	src.Close()
	if err := errors.Join(rerr, wait()); err != nil {
		return rep, fmt.Errorf("pg_restore: %w", err)
	}
	rep.TimingsSeconds["database_restore"] = since(t0)
	target, err := pgx.Connect(ctx, targetURL)
	if err != nil {
		return rep, err
	}
	defer target.Close(context.WithoutCancel(ctx))
	t0 = time.Now()
	tx, err := target.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return rep, err
	}
	got, err := Capture(ctx, tx)
	_ = tx.Rollback(ctx)
	if err != nil {
		return rep, err
	}
	rep.Differences = Diff(m.Inventory, got)
	if rep.Differences == nil {
		rep.Differences = []string{}
	}
	rep.TablesCompared = len(m.Inventory.Tables)
	rep.TimingsSeconds["reconcile"] = since(t0)
	if m.Evidence != nil {
		t0 = time.Now()
		partial := evDest + ".partial"
		if err := os.Mkdir(partial, 0o700); err != nil {
			return rep, err
		}
		if _, err := walkArchive(ctx, cfg.BackupDir, m, cfg.Identity, cfg.Tools, &partial); err != nil {
			return rep, err
		}
		if err := os.WriteFile(filepath.Join(partial, ".crm-restore"), []byte(marker(vr.ID, "EVIDENCE")+"\n"), 0o600); err != nil {
			return rep, err
		}
		if err := os.Rename(partial, evDest); err != nil {
			return rep, err
		}
		etx, err := target.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
		if err != nil {
			return rep, err
		}
		rep.EvidenceFindings, rep.EvidenceChecked, err = CheckEvidence(ctx, etx, evDest)
		_ = etx.Rollback(ctx)
		if err != nil {
			return rep, err
		}
		if rep.EvidenceFindings == nil {
			rep.EvidenceFindings = []EvidenceFinding{}
		}
		rep.TimingsSeconds["evidence_restore"] = since(t0)
	}
	// Quarantine: nothing on the restored copy may run or authorise until an
	// operator re-validates and re-attests it.
	rows, err := target.Query(ctx, `UPDATE platform_featureflags.feature_flag SET enabled = false, updated_at = now() WHERE enabled RETURNING flag_key`)
	if err != nil {
		return rep, fmt.Errorf("quarantine: %w", err)
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return rep, err
		}
		rep.QuarantinedFlags = append(rep.QuarantinedFlags, k)
	}
	rows.Close()
	sort.Strings(rep.QuarantinedFlags)
	status := "COMPLETE"
	if len(rep.Differences) > 0 || len(rep.EvidenceFindings) > 0 {
		status = "FAILED"
	}
	if _, err := admin.Exec(ctx, `COMMENT ON DATABASE `+ident(cfg.Target)+` IS '`+marker(vr.ID, status)+`'`); err != nil {
		return rep, err
	}
	rep.Status = status
	rep.MeasuredRecoveryS = since(start)
	if status != "COMPLETE" {
		return rep, fmt.Errorf("%w: %d differences, %d evidence findings", ErrMismatch, len(rep.Differences), len(rep.EvidenceFindings))
	}
	return rep, nil
}

// Discard drops a database created by Restore (and its evidence directory).
// Anything without the tool's marker is refused.
func Discard(ctx context.Context, adminURL, target, evidenceDir string, roots []string) error {
	if !restoreName.MatchString(target) {
		return fmt.Errorf("%w: only crm_restore_* databases can be discarded", ErrRefused)
	}
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return fmt.Errorf("%w: cluster unavailable", ErrRefused)
	}
	defer admin.Close(context.WithoutCancel(ctx))
	var comment *string
	err = admin.QueryRow(ctx, `SELECT shobj_description(oid, 'pg_database') FROM pg_database WHERE datname = $1`, target).Scan(&comment)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s does not exist", ErrRefused, target)
	}
	if err != nil {
		return err
	}
	if comment == nil || !markerRe.MatchString(*comment) {
		return fmt.Errorf("%w: %s was not created by a crm-backup restore", ErrRefused, target)
	}
	if evidenceDir != "" {
		dir, err := underAllowed(evidenceDir, roots)
		if err != nil {
			return err
		}
		mk, err := os.ReadFile(filepath.Join(dir, ".crm-restore"))
		if err != nil || !strings.HasPrefix(string(mk), "crm-backup:restore:") {
			return fmt.Errorf("%w: the evidence directory carries no restore marker", ErrRefused)
		}
	}
	if _, err := admin.Exec(ctx, `DROP DATABASE `+ident(target)+` WITH (FORCE)`); err != nil {
		return err
	}
	if evidenceDir != "" {
		dir, _ := underAllowed(evidenceDir, roots)
		return os.RemoveAll(dir)
	}
	return nil
}

// RetentionCandidate is a backup a retention policy would remove.
type RetentionCandidate struct {
	Dir     string    `json:"dir"`
	ID      string    `json:"id"`
	Created time.Time `json:"created_at"`
}

// PlanRetention lists the complete backups under root beyond the newest
// keep and older than minAge. It is a DRY RUN only: deleting backups needs
// an approved retention decision and is not implemented.
func PlanRetention(root string, keep int, minAge time.Duration, now time.Time) ([]RetentionCandidate, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var all []RetentionCandidate
	for _, e := range entries {
		if !e.IsDir() || strings.HasSuffix(e.Name(), ".partial") {
			continue
		}
		m, err := ReadManifest(filepath.Join(root, e.Name()))
		if err != nil {
			continue
		}
		all = append(all, RetentionCandidate{Dir: e.Name(), ID: m.ID, Created: m.CreatedAt})
	}
	sort.Slice(all, func(a, b int) bool { return all[a].Created.After(all[b].Created) })
	var out []RetentionCandidate
	for i, c := range all {
		if i >= keep && now.Sub(c.Created) > minAge {
			out = append(out, c)
		}
	}
	return out, nil
}
