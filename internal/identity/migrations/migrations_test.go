package migrations

import (
	"flag"
	"os"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

var update = flag.Bool("update", false, "rewrite the rendered identity migration")

const (
	bodyFile     = "body/00001_identity.sql"
	renderedFile = "sql/00001_identity.sql"
)

// TestRenderedMigration proves the committed migration is exactly the module
// template applied to the body.
func TestRenderedMigration(t *testing.T) {
	body, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := migrations.RenderModuleInfra(identity.Module, string(body))
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
	if len(names) != 3 || names[0] != "00001_identity.sql" || names[1] != "00002_identity_customer_capabilities.sql" || names[2] != "00003_identity_sales_capabilities.sql" {
		t.Fatalf("migration files = %v", names)
	}
	if Source().Table == migrations.PlatformTable {
		t.Fatal("identity must use its own version table")
	}
}
