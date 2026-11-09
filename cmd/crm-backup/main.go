// Command crm-backup is the R3 logical backup / restore tool for DISPOSABLE
// staging and development databases and their evidence stores.
//
//	crm-backup backup                 consistent backup into a new directory
//	crm-backup verify  <dir>          digests, decryption, archive structure, objects
//	crm-backup restore <dir>          restore into a NEW crm_restore_* database (quarantined)
//	crm-backup discard <database>     drop a tool-created crm_restore_* database (+ evidence dir)
//	crm-backup retention-plan <root>  DRY RUN: list backups a retention policy would remove
//
// Configuration (environment; credentials are never printed):
//
//	CRM_BACKUP_ENV                   development | staging (production is refused)
//	CRM_BACKUP_SOURCE_DATABASE_URL   source database (a read-all login)
//	CRM_BACKUP_DEST                  new backup directory (absolute)
//	CRM_BACKUP_ALLOWED_ROOTS         ':'-separated absolute roots for every destination
//	CRM_BACKUP_EVIDENCE_ROOT         evidence store root to include (optional)
//	CRM_BACKUP_AGE_RECIPIENTS        age recipients file (required outside development)
//	CRM_BACKUP_ALLOW_PLAINTEXT=1     development only
//	CRM_BACKUP_AGE_IDENTITY          age identity file (verify / restore)
//	CRM_BACKUP_PG_BIN                directory of pg_dump / pg_restore (optional)
//	CRM_BACKUP_METRICS_FILE          Prometheus textfile to update (optional)
//	CRM_RESTORE_ADMIN_DATABASE_URL   maintenance DB of the target cluster (CREATE DATABASE)
//	CRM_RESTORE_TARGET               new database name (crm_restore_*)
//	CRM_RESTORE_EVIDENCE_DEST        new evidence directory (when the backup has evidence)
//	CRM_RETENTION_KEEP / CRM_RETENTION_MIN_AGE   retention-plan policy
//
// Exit: 0 success · 1 failure (integrity, mismatch, infrastructure) · 2 refused / usage.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/envfile"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/backup"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], envfile.Getenv(os.Getenv), os.Stdout, os.Stderr))
}

func roots(get func(string) string) []string {
	var out []string
	for _, r := range strings.Split(get("CRM_BACKUP_ALLOWED_ROOTS"), ":") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}

func code(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, backup.ErrRefused):
		return 2
	}
	return 1
}

// metrics updates the textfile atomically (write temp, rename).
func metrics(get func(string) string, op string, ok bool, extra map[string]float64) {
	path := get("CRM_BACKUP_METRICS_FILE")
	if path == "" {
		return
	}
	prev := map[string]string{}
	if raw, err := os.ReadFile(path); err == nil {
		for _, l := range strings.Split(string(raw), "\n") {
			if f := strings.Fields(l); len(f) == 2 && !strings.HasPrefix(l, "#") {
				prev[f[0]] = f[1]
			}
		}
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	b := "0"
	if ok {
		b = "1"
		prev["crm_"+op+"_last_success_timestamp_seconds"] = now
	}
	prev["crm_"+op+"_last_run_timestamp_seconds"] = now
	prev["crm_"+op+"_last_run_success"] = b
	for k, v := range extra {
		prev["crm_"+op+"_"+k] = strconv.FormatFloat(v, 'g', -1, 64)
	}
	var sb strings.Builder
	keys := make([]string, 0, len(prev))
	for k := range prev {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		fmt.Fprintf(&sb, "# TYPE %s gauge\n%s %s\n", k, k, prev[k])
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, []byte(sb.String()), 0o644) == nil {
		_ = os.Rename(tmp, path)
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func run(ctx context.Context, args []string, get func(string) string, stdout, stderr io.Writer) int {
	usage := func() int {
		fmt.Fprintln(stderr, "usage: crm-backup backup | verify <dir> | restore <dir> | discard <database> | retention-plan <root>")
		return 2
	}
	if len(args) == 0 {
		return usage()
	}
	tools := backup.Tools{PgBinDir: get("CRM_BACKUP_PG_BIN")}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	fail := func(op string, err error) int {
		fmt.Fprintf(stderr, "crm-backup %s: %v\n", op, err)
		return code(err)
	}
	switch args[0] {
	case "backup":
		if len(args) != 1 {
			return usage()
		}
		m, err := backup.Backup(ctx, backup.BackupConfig{Environment: get("CRM_BACKUP_ENV"), SourceURL: get("CRM_BACKUP_SOURCE_DATABASE_URL"),
			Dest: get("CRM_BACKUP_DEST"), AllowedRoots: roots(get), EvidenceRoot: get("CRM_BACKUP_EVIDENCE_ROOT"),
			Recipients: get("CRM_BACKUP_AGE_RECIPIENTS"), AllowPlaintext: get("CRM_BACKUP_ALLOW_PLAINTEXT") == "1", Tools: tools})
		metrics(get, "backup", err == nil, map[string]float64{"last_duration_seconds": m.DurationSeconds})
		if err != nil {
			return fail("backup", err)
		}
		_ = enc.Encode(map[string]any{"id": m.ID, "dir": filepath.Base(get("CRM_BACKUP_DEST")), "tables": len(m.Inventory.Tables),
			"encrypted": m.Encrypted, "evidence_objects": evidenceCount(m), "duration_seconds": m.DurationSeconds})
		return 0
	case "verify":
		if len(args) != 2 {
			return usage()
		}
		r, err := backup.Verify(ctx, args[1], get("CRM_BACKUP_AGE_IDENTITY"), tools)
		metrics(get, "backup_verify", err == nil, nil)
		if err != nil {
			return fail("verify", err)
		}
		_ = enc.Encode(r)
		return 0
	case "restore":
		if len(args) != 2 {
			return usage()
		}
		r, err := backup.Restore(ctx, backup.RestoreConfig{AdminURL: get("CRM_RESTORE_ADMIN_DATABASE_URL"), Target: get("CRM_RESTORE_TARGET"),
			BackupDir: args[1], Identity: get("CRM_BACKUP_AGE_IDENTITY"), EvidenceDest: get("CRM_RESTORE_EVIDENCE_DEST"), AllowedRoots: roots(get),
			Tools: tools})
		metrics(get, "restore_rehearsal", err == nil, map[string]float64{"last_recovery_seconds": r.MeasuredRecoveryS})
		_ = enc.Encode(r)
		if err != nil {
			return fail("restore", err)
		}
		return 0
	case "discard":
		if len(args) != 2 {
			return usage()
		}
		if err := backup.Discard(ctx, get("CRM_RESTORE_ADMIN_DATABASE_URL"), args[1], get("CRM_RESTORE_EVIDENCE_DEST"), roots(get)); err != nil {
			return fail("discard", err)
		}
		_ = enc.Encode(map[string]string{"discarded": args[1]})
		return 0
	case "retention-plan":
		if len(args) != 2 {
			return usage()
		}
		keep, err := strconv.Atoi(get("CRM_RETENTION_KEEP"))
		if err != nil || keep < 1 {
			fmt.Fprintln(stderr, "crm-backup retention-plan: CRM_RETENTION_KEEP must be >= 1")
			return 2
		}
		minAge, err := time.ParseDuration(get("CRM_RETENTION_MIN_AGE"))
		if err != nil || minAge < 0 {
			fmt.Fprintln(stderr, "crm-backup retention-plan: CRM_RETENTION_MIN_AGE must be a duration")
			return 2
		}
		c, err := backup.PlanRetention(args[1], keep, minAge, time.Now())
		if err != nil {
			return fail("retention-plan", err)
		}
		_ = enc.Encode(map[string]any{"dry_run": true, "deletion": "not implemented: needs an approved retention decision", "candidates": c})
		return 0
	}
	return usage()
}

func evidenceCount(m backup.Manifest) int {
	if m.Evidence == nil {
		return 0
	}
	return len(m.Evidence.Objects)
}
