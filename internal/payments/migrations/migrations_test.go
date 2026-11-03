package migrations

import (
	"flag"
	"os"
	"regexp"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

var update = flag.Bool("update", false, "rewrite the rendered payments migration")

const (
	bodyFile     = "body/00001_payments.sql"
	renderedFile = "sql/00001_payments.sql"
)

// TestRenderedMigration proves the committed migration is exactly the module
// template applied to the body.
func TestRenderedMigration(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := migrations.RenderModuleInfra(payments.Module, string(body))
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.WriteFile(renderedFile, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(renderedFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s is stale: regenerate with -args -update", renderedFile)
	}
}

func TestSourceValidates(t *testing.T) {
	names, err := migrations.Validate(Source())
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 3 || names[0] != "00001_payments.sql" || names[1] != "00002_payments_append_only_guards.sql" ||
		names[2] != "00003_payments_controlled_writes.sql" {
		t.Fatalf("migration files = %v", names)
	}
	if Source().Table == migrations.PlatformTable {
		t.Fatal("payments must use its own version table")
	}
}

// No raw gateway secret, URL, path or PII; history append-only; no DELETE.
func TestNoSecretsAndAppendOnly(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	src := regexp.MustCompile(`(?m)--.*$`).ReplaceAll(body, nil)
	for _, bad := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(phone|card_number|proof_data|authority|ref_id|url|path|file_name)\b\s+text`),
		regexp.MustCompile(`(?im)^\s*GRANT[^;]*\bDELETE\b`),
		regexp.MustCompile(`(?im)^\s*GRANT[^;]*\bUPDATE\b[^;]*\battempt_state_history\b`),
		regexp.MustCompile(`(?i)\bbilling\.`),
		regexp.MustCompile(`(?i)\b(float|real|double precision|numeric)\b`),
	} {
		if bad.Match(src) {
			t.Fatalf("forbidden construct: %s", bad.String())
		}
	}
}
