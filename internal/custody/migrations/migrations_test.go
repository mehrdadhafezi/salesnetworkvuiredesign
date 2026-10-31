package migrations

import (
	"flag"
	"os"
	"regexp"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

var update = flag.Bool("update", false, "rewrite the rendered custody migration")

const (
	bodyFile     = "body/00001_custody.sql"
	renderedFile = "sql/00001_custody.sql"
)

// TestRenderedMigration proves the committed migration is exactly the module
// template applied to the body.
func TestRenderedMigration(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := migrations.RenderModuleInfra(custody.Module, string(body))
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
	if len(names) != 2 || names[0] != "00001_custody.sql" || names[1] != "00002_custody_append_only_guards.sql" {
		t.Fatalf("migration files = %v", names)
	}
	if Source().Table == migrations.PlatformTable {
		t.Fatal("custody must use its own version table")
	}
}

// Custody events are INSERT-only and nothing is deletable.
func TestEventsAppendOnlyAndNoDelete(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	src := regexp.MustCompile(`(?m)--.*$`).ReplaceAll(body, nil)
	for _, bad := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bphone\b`),
		regexp.MustCompile(`(?im)^\s*GRANT[^;]*\bDELETE\b`),
		regexp.MustCompile(`(?im)^\s*GRANT[^;]*\bUPDATE\b[^;]*\bcustody_event\b`),
	} {
		if bad.Match(src) {
			t.Fatalf("forbidden construct: %s", bad.String())
		}
	}
}
