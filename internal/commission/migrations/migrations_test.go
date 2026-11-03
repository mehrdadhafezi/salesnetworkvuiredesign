package migrations

import (
	"flag"
	"os"
	"regexp"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

var update = flag.Bool("update", false, "rewrite the rendered commission migration")

const (
	bodyFile     = "body/00001_commission.sql"
	renderedFile = "sql/00001_commission.sql"
)

// TestRenderedMigration proves the committed migration is exactly the module
// template applied to the body.
func TestRenderedMigration(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := migrations.RenderModuleInfra(commission.Module, string(body))
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
	if len(names) != 2 || names[0] != "00001_commission.sql" || names[1] != "00002_commission_append_only_guards.sql" {
		t.Fatalf("migration files = %v", names)
	}
	if Source().Table == migrations.PlatformTable {
		t.Fatal("commission must use its own version table")
	}
}

// No PII, URL, path or token column; no float or money column; no legacy
// table literal; INSERT/SELECT only; no other schema is referenced (commission
// never reads or writes an owner schema).
func TestNoPIIAndInsertOnly(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	src := regexp.MustCompile(`(?m)--.*$`).ReplaceAll(body, nil)
	for _, bad := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b\w*(phone|address|postcode|national|name_|customer_name|email|url|path|token|secret|authority|ref_id)\w*\b\s+(text|bytea)`),
		regexp.MustCompile(`(?im)^\s*GRANT[^;]*\b(DELETE|UPDATE|TRUNCATE)\b`),
		regexp.MustCompile(`(?i)\b(cases|custody|billing|payments|customers|identity|organization|integration|finance|biawin|fulfillment|reporting)\.`),
		regexp.MustCompile(`(?i)\b(float|real|double precision|money)\b`),
		// numeric is allowed only as the overflow-free sum inside the balance
		// trigger; never as a stored column type.
		regexp.MustCompile(`(?im)^\s+\w+\s+numeric\b.*,\s*$`),
		regexp.MustCompile(`(?i)\b(mobile|email|national_id|postal)\w*\b`),
		regexp.MustCompile(`(?i)\b(wp_|sn_)\w+`),
	} {
		if bad.Match(src) {
			t.Fatalf("forbidden construct: %s", bad.String())
		}
	}
}
