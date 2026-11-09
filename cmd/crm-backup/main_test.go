package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func get(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

// Refusals happen before any connection, and no credential is echoed.
func TestRefusalsAndRedaction(t *testing.T) {
	root := t.TempDir()
	secret := "postgres://crm_backup:Sup3rS3cret@db.staging.internal:5432/crm?sslmode=require"
	cases := map[string]map[string]string{
		"production": {"CRM_BACKUP_ENV": "production", "CRM_BACKUP_SOURCE_DATABASE_URL": secret, "CRM_BACKUP_DEST": filepath.Join(root, "b"),
			"CRM_BACKUP_ALLOWED_ROOTS": root, "CRM_BACKUP_AGE_RECIPIENTS": "/dev/null"},
		"no encryption": {"CRM_BACKUP_ENV": "staging", "CRM_BACKUP_SOURCE_DATABASE_URL": secret, "CRM_BACKUP_DEST": filepath.Join(root, "b"),
			"CRM_BACKUP_ALLOWED_ROOTS": root},
		"outside allowlist": {"CRM_BACKUP_ENV": "staging", "CRM_BACKUP_SOURCE_DATABASE_URL": secret, "CRM_BACKUP_DEST": "/tmp/elsewhere-r3",
			"CRM_BACKUP_ALLOWED_ROOTS": root, "CRM_BACKUP_AGE_RECIPIENTS": "/dev/null"},
		"relative dest": {"CRM_BACKUP_ENV": "staging", "CRM_BACKUP_SOURCE_DATABASE_URL": secret, "CRM_BACKUP_DEST": "backups/b",
			"CRM_BACKUP_ALLOWED_ROOTS": root, "CRM_BACKUP_AGE_RECIPIENTS": "/dev/null"},
	}
	for name, env := range cases {
		var out, errOut bytes.Buffer
		if c := run(context.Background(), []string{"backup"}, get(env), &out, &errOut); c != 2 {
			t.Fatalf("%s: exit %d (%s)", name, c, errOut.String())
		}
		for _, leak := range []string{"Sup3rS3cret", "db.staging.internal", "crm_backup:"} {
			if strings.Contains(out.String()+errOut.String(), leak) {
				t.Fatalf("%s leaks %q", name, leak)
			}
		}
	}
	var o, e bytes.Buffer
	if run(context.Background(), []string{"restore", root}, get(map[string]string{"CRM_RESTORE_TARGET": "crm_main"}), &o, &e) != 2 {
		t.Fatal("a non-disposable restore target must be refused")
	}
	if run(context.Background(), []string{"discard", "crm_main"}, get(nil), &o, &e) != 2 {
		t.Fatal("discard of a non-restore database must be refused")
	}
	if run(context.Background(), nil, get(nil), &o, &e) != 2 {
		t.Fatal("usage")
	}
}

func TestMetricsTextfile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "crm_backup.prom")
	env := get(map[string]string{"CRM_BACKUP_METRICS_FILE": p})
	metrics(env, "backup", true, map[string]float64{"last_duration_seconds": 1.5})
	metrics(env, "backup", false, nil)
	raw, _ := os.ReadFile(p)
	s := string(raw)
	if !strings.Contains(s, "crm_backup_last_run_success 0") || !strings.Contains(s, "crm_backup_last_success_timestamp_seconds ") ||
		!strings.Contains(s, "crm_backup_last_duration_seconds 1.5") {
		t.Fatalf("textfile:\n%s", s)
	}
}
