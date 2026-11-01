package migrations

import (
	"flag"
	"os"
	"regexp"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

var update = flag.Bool("update", false, "rewrite the rendered billing migration")

const (
	bodyFile     = "body/00001_billing.sql"
	renderedFile = "sql/00001_billing.sql"
)

// TestRenderedMigration proves the committed migration is exactly the module
// template applied to the body.
func TestRenderedMigration(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := migrations.RenderModuleInfra(billing.Module, string(body))
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
	if len(names) != 1 || names[0] != "00001_billing.sql" {
		t.Fatalf("migration files = %v", names)
	}
	if Source().Table == migrations.PlatformTable {
		t.Fatal("billing must use its own version table")
	}
}

// No phone/PII column, no DELETE grant, no payment truth in billing.
func TestNoPIIAndNoDelete(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	src := regexp.MustCompile(`(?m)--.*$`).ReplaceAll(body, nil)
	for _, bad := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(phone|customer_name|address|postal|card_number|access_token|national_id|receipt_url)\b`),
		regexp.MustCompile(`(?im)^\s*GRANT[^;]*\bDELETE\b`),
		regexp.MustCompile(`(?i)\bpayments\.`),
		regexp.MustCompile(`(?i)\b(float|real|double precision|numeric)\b`),
	} {
		if bad.Match(src) {
			t.Fatalf("forbidden construct: %s", bad.String())
		}
	}
}
