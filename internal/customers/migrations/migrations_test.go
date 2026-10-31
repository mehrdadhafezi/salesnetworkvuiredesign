package migrations

import (
	"flag"
	"os"
	"regexp"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

var update = flag.Bool("update", false, "rewrite the rendered customers migration")

const (
	bodyFile     = "body/00001_customers.sql"
	renderedFile = "sql/00001_customers.sql"
)

// TestRenderedMigration proves the committed migration is exactly the module
// template applied to the body.
func TestRenderedMigration(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := migrations.RenderModuleInfra(customers.Module, string(body))
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
	if len(names) != 1 || names[0] != "00001_customers.sql" {
		t.Fatalf("migration files = %v", names)
	}
	if Source().Table == migrations.PlatformTable {
		t.Fatal("customers must use its own version table")
	}
}

// Phone is never a key (Architecture V1 §10): no UNIQUE / PRIMARY KEY may
// involve a contact or identifier value, and no DELETE is ever granted.
func TestNoPhoneKeysAndNoDelete(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)UNIQUE\s*\([^)]*normalized_value`),
		regexp.MustCompile(`(?i)UNIQUE\s+INDEX[^;]*normalized_value`),
		regexp.MustCompile(`(?i)(phone|email|normalized_value|external_value|raw_value)[^,\n]*PRIMARY KEY`),
		regexp.MustCompile(`(?i)PRIMARY KEY\s*\([^)]*(phone|email|normalized_value|external_value)`),
		regexp.MustCompile(`(?im)^\s*GRANT[^;]*\bDELETE\b`),
		regexp.MustCompile(`(?i)UNIQUE\s*\(\s*scheme\s*,\s*external_value\s*\)`),
	} {
		if bad.Match(body) {
			t.Fatalf("forbidden customer identity construct: %s", bad.String())
		}
	}
}
