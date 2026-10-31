package migrations

import (
	"flag"
	"os"
	"regexp"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

var update = flag.Bool("update", false, "rewrite the rendered cases migration")

const (
	bodyFile     = "body/00001_cases.sql"
	renderedFile = "sql/00001_cases.sql"
)

// TestRenderedMigration proves the committed migration is exactly the module
// template applied to the body.
func TestRenderedMigration(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := migrations.RenderModuleInfra(cases.Module, string(body))
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
	if len(names) != 2 || names[0] != "00001_cases.sql" || names[1] != "00002_cases_append_only_guards.sql" {
		t.Fatalf("migration files = %v", names)
	}
	if Source().Table == migrations.PlatformTable {
		t.Fatal("cases must use its own version table")
	}
}

// No Case key involves a phone, name or username, and nothing is deletable.
func TestNoPhoneKeysAndNoDelete(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(phone|customer_name|national_id|user_login|username)\b`),
		regexp.MustCompile(`(?im)^\s*GRANT[^;]*\bDELETE\b`),
		regexp.MustCompile(`(?im)^\s*GRANT[^;]*\bUPDATE\b[^;]*\b(case_state_history|attribution_snapshot)\b`),
	} {
		if bad.Match(stripComments(body)) {
			t.Fatalf("forbidden construct: %s", bad.String())
		}
	}
}

func stripComments(b []byte) []byte {
	return regexp.MustCompile(`(?m)--.*$`).ReplaceAll(b, nil)
}
